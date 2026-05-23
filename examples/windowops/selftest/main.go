package main

// windowops/selftest — unattended scenario runner.
//
// Reads a JSON scenario from stdin, runs each step, writes a JSON report
// to stdout (plus PNGs into the scenario's outDir). Designed to run
// inside a VM over RDP/SSH/VNC without any human interaction.

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	deskact "github.com/PekingSpades/DeskAct"
)

type Scenario struct {
	OutDir       string       `json:"outDir"`
	WindowMatch  WindowMatch  `json:"windowMatch"`
	InitDPIAware bool         `json:"initDpiAware"`
	Steps        []ScenarioOp `json:"steps"`
}

type WindowMatch struct {
	TitleContains string `json:"titleContains"`
	PID           int    `json:"pid"`
	WindowID      uint64 `json:"windowId"`
	Index         int    `json:"index"`
}

type ScenarioOp struct {
	Op      string `json:"op"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	W       int    `json:"w"`
	H       int    `json:"h"`
	DX      int    `json:"dx"`
	DY      int    `json:"dy"`
	Button  string `json:"button"`
	Text    string `json:"text"`
	DelayMs int    `json:"delayMs"`
	Ms      int    `json:"ms"` // used by op=sleep
	Backend string `json:"backend"`
	Sub     string `json:"sub"`
	Name    string `json:"name"`
}

type Report struct {
	Platform string       `json:"platform"`
	Target   TargetReport `json:"target"`
	Steps    []StepReport `json:"steps"`
	OK       bool         `json:"ok"`
	Errors   []string     `json:"errors,omitempty"`
}

type TargetReport struct {
	WindowID uint64 `json:"windowId"`
	PID      int    `json:"pid"`
	Title    string `json:"title"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	W        int    `json:"w"`
	H        int    `json:"h"`
}

type StepReport struct {
	Op        string `json:"op"`
	OK        bool   `json:"ok"`
	Err       string `json:"err,omitempty"`
	ElapsedMs int64  `json:"elapsedMs"`
	PNG       string `json:"png,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

func main() {
	scenario, err := readScenario(os.Stdin)
	if err != nil {
		fail("read scenario: %v", err)
	}
	if scenario.OutDir == "" {
		scenario.OutDir = "."
	}
	if err := os.MkdirAll(scenario.OutDir, 0o755); err != nil {
		fail("mkdir outDir %q: %v", scenario.OutDir, err)
	}

	if scenario.InitDPIAware {
		deskact.InitDPIAwareness()
	}

	target, info, err := findTarget(scenario.WindowMatch)
	if err != nil {
		fail("find target: %v", err)
	}

	report := Report{
		Platform: runtime.GOOS,
		Target: TargetReport{
			WindowID: info.ID,
			PID:      info.PID,
			Title:    info.Title,
			X:        info.Bounds.X,
			Y:        info.Bounds.Y,
			W:        info.Bounds.W,
			H:        info.Bounds.H,
		},
	}

	allOK := true
	for i, step := range scenario.Steps {
		sr := runStep(scenario, target, info, i, step)
		report.Steps = append(report.Steps, sr)
		if !sr.OK {
			allOK = false
			report.Errors = append(report.Errors, fmt.Sprintf("step %d (%s): %s", i, step.Op, sr.Err))
		}
	}
	report.OK = allOK

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(report)
	if !allOK {
		os.Exit(2)
	}
}

func readScenario(r *os.File) (Scenario, error) {
	var s Scenario
	dec := json.NewDecoder(r)
	if err := dec.Decode(&s); err != nil {
		return s, err
	}
	return s, nil
}

func findTarget(m WindowMatch) (deskact.MouseWindowTarget, deskact.WindowInfo, error) {
	if m.WindowID != 0 {
		// Try direct lookup via bounds() to confirm it exists; if not, list.
		t := deskact.MouseWindowTarget{WindowID: m.WindowID, PID: int32(m.PID)}
		// We need a WindowInfo for context; list and find it.
		wins, err := deskact.ListWindows(deskact.DefaultWindowOptions())
		if err == nil {
			for _, w := range wins {
				if w.ID == m.WindowID {
					return t, w, nil
				}
			}
		}
		return t, deskact.WindowInfo{ID: m.WindowID, PID: m.PID}, nil
	}

	wins, err := deskact.ListWindows(deskact.DefaultWindowOptions())
	if err != nil {
		return deskact.MouseWindowTarget{}, deskact.WindowInfo{}, err
	}

	matches := []deskact.WindowInfo{}
	for _, w := range wins {
		if m.TitleContains != "" && !strings.Contains(w.Title, m.TitleContains) {
			continue
		}
		if m.PID != 0 && w.PID != m.PID {
			continue
		}
		matches = append(matches, w)
	}
	if len(matches) == 0 {
		return deskact.MouseWindowTarget{}, deskact.WindowInfo{},
			fmt.Errorf("no window matched (titleContains=%q pid=%d)", m.TitleContains, m.PID)
	}
	if m.Index < 0 || m.Index >= len(matches) {
		return deskact.MouseWindowTarget{}, matches[0],
			fmt.Errorf("index %d out of range (got %d matches)", m.Index, len(matches))
	}
	w := matches[m.Index]
	return deskact.MouseWindowTarget{WindowID: w.ID, PID: int32(w.PID)}, w, nil
}

func runStep(s Scenario, t deskact.MouseWindowTarget, w deskact.WindowInfo, idx int, step ScenarioOp) (sr StepReport) {
	// op=sleep uses Ms (matches plan); the generic pre-op DelayMs is skipped
	// for sleep so the step does not sleep twice.
	if step.Op != "sleep" && step.DelayMs > 0 {
		time.Sleep(time.Duration(step.DelayMs) * time.Millisecond)
	}
	start := time.Now()
	sr = StepReport{Op: step.Op}

	defer func() {
		sr.ElapsedMs = time.Since(start).Milliseconds()
	}()

	switch step.Op {
	case "sleep":
		ms := step.Ms
		if ms == 0 {
			ms = step.DelayMs
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		sr.OK = true
	case "screenshot":
		backend := deskact.CaptureBackendDefault
		if step.Backend != "" {
			backend = deskact.CaptureBackend(step.Backend)
		}
		req := deskact.CaptureWindowRequest{
			WindowID: t.WindowID,
			PID:      t.PID,
			Options:  deskact.CaptureOptions{Backend: backend},
		}
		img, err := deskact.CaptureWindow(req)
		if err != nil && img == nil {
			sr.Err = err.Error()
			return sr
		}
		name := step.Name
		if name == "" {
			name = fmt.Sprintf("step-%03d.png", idx)
		}
		path := filepath.Join(s.OutDir, name)
		if e := writePNG(path, img); e != nil {
			sr.Err = fmt.Sprintf("write png: %v", e)
			return sr
		}
		sr.PNG = path
		if err != nil {
			sr.Detail = fmt.Sprintf("partial: %v", err)
		}
		sr.OK = true
	case "click":
		btn := buttonOf(step.Button)
		err := deskact.ClickWithWindow(t, step.X, step.Y, btn, deskact.DefaultMouseSettings())
		if err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "move-mouse":
		err := deskact.MoveWithWindow(t, step.X, step.Y, deskact.DefaultMouseSettings())
		if err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "scroll":
		err := deskact.ScrollWithWindow(t, step.X, step.Y, step.DX, step.DY, deskact.ScrollUnitLine, deskact.DefaultMouseSettings())
		if err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "type":
		for _, r := range step.Text {
			if err := deskact.KeyTapWithWindow(string(r), t.WindowID, int(t.PID), nil, deskact.DefaultKeyboardSettings()); err != nil {
				sr.Err = fmt.Sprintf("KeyTapWithWindow(%q): %v", string(r), err)
				return sr
			}
		}
		sr.OK = true
	case "key":
		if err := deskact.KeyTapWithWindow(step.Text, t.WindowID, int(t.PID), nil, deskact.DefaultKeyboardSettings()); err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "move":
		if err := deskact.WindowMove(w.ID, int32(w.PID), step.X, step.Y); err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "resize":
		if err := deskact.WindowResize(w.ID, int32(w.PID), step.W, step.H); err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "moveresize":
		if err := deskact.WindowMoveResize(w.ID, int32(w.PID), step.X, step.Y, step.W, step.H); err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "focus":
		if err := deskact.WindowFocus(w.ID, int32(w.PID)); err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "raise":
		if err := deskact.WindowRaise(w.ID, int32(w.PID)); err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "minimize":
		if err := deskact.WindowMinimize(w.ID, int32(w.PID)); err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "restore":
		if err := deskact.WindowRestore(w.ID, int32(w.PID)); err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	case "close":
		if err := deskact.WindowClose(w.ID, int32(w.PID)); err != nil {
			sr.Err = err.Error()
			return sr
		}
		sr.OK = true
	default:
		sr.Err = "unknown op: " + step.Op
	}
	return sr
}

func buttonOf(s string) deskact.MouseButton {
	switch strings.ToLower(s) {
	case "right", "r":
		return deskact.MouseButtonRight
	case "middle", "m":
		return deskact.MouseButtonMiddle
	}
	return deskact.MouseButtonLeft
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "selftest: "+format+"\n", args...)
	os.Exit(1)
}
