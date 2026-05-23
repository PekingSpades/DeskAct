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
	"os/exec"
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
	// op=occlude only. Command line of the auxiliary process spawned to
	// cover the target so the next screenshot exercises occluded capture.
	// Defaults per platform: notepad on Windows, xterm on Linux. macOS:
	// open -na TextEdit. Tokenized by ' '; no shell quoting.
	Cmd string `json:"cmd"`
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
	Op             string `json:"op"`
	OK             bool   `json:"ok"`
	Err            string `json:"err,omitempty"`
	ElapsedMs      int64  `json:"elapsedMs"`
	PNG            string `json:"png,omitempty"`
	Detail         string `json:"detail,omitempty"`
	BackendUsed    string `json:"backendUsed,omitempty"`
	Partial        bool   `json:"partial,omitempty"`
	FallbackReason string `json:"fallbackReason,omitempty"`
}

// spawnedOccluders is the set of (pid, *os.Process) we launched via
// op=occlude so cleanupSpawned can kill them when the selftest exits.
// Leftover occluders are not just untidy — they have the same window
// title as the legitimate target (e.g. "Untitled - Notepad"), so the
// next run's windowMatch could grab the wrong window.
var spawnedOccluders []*os.Process

func cleanupSpawned() {
	for _, p := range spawnedOccluders {
		if p == nil {
			continue
		}
		_ = p.Kill()
		// Drain the child so it doesn't sit as a zombie. Ignored
		// errors are expected when Kill races with normal exit.
		_, _ = p.Wait()
	}
	spawnedOccluders = nil
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
	// Kill any occluders we spawned before exiting so subsequent runs
	// don't accidentally re-target our leftover window (titles like
	// "Untitled - Notepad" overlap with the real target).
	cleanupSpawned()
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
	start := time.Now()
	sr = StepReport{Op: step.Op}

	defer func() {
		// DelayMs is a post-op settle: most window-management ops
		// (move/resize/focus/click) are asynchronous on Windows and X11
		// — the WM applies the change some time after PostMessage /
		// XSendEvent returns. Sleeping AFTER the op gives the WM time
		// to settle before the next step (typically a screenshot or
		// another input event) takes its measurement. Pre-op sleeps
		// would let races bleed across step boundaries. We sleep here
		// BEFORE computing ElapsedMs so the reported wall-clock
		// includes the settle — otherwise a 0ms PostMessage with a
		// 200ms settle would falsely look free. "sleep" is its own
		// kind of post-op delay so we skip the double-sleep.
		if step.Op != "sleep" && step.DelayMs > 0 {
			time.Sleep(time.Duration(step.DelayMs) * time.Millisecond)
		}
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
		res := deskact.CaptureWindowEx(req)
		sr.BackendUsed = string(res.BackendUsed)
		sr.FallbackReason = res.FallbackReason
		if res.Image == nil {
			if res.Err != nil {
				sr.Err = res.Err.Error()
			} else {
				sr.Err = "capture returned no image"
			}
			return sr
		}
		name := step.Name
		if name == "" {
			name = fmt.Sprintf("step-%03d.png", idx)
		}
		path := filepath.Join(s.OutDir, name)
		if e := writePNG(path, res.Image); e != nil {
			sr.Err = fmt.Sprintf("write png: %v", e)
			return sr
		}
		sr.PNG = path
		// Partial: image was returned but is known imperfect (e.g.
		// PrintWindow blank-frame). Surface as not-OK so callers can't
		// mistake "we wrote a PNG" for "the requested capture worked".
		if res.Partial {
			sr.Partial = true
			sr.Detail = fmt.Sprintf("partial: %v", res.Err)
			sr.Err = res.Err.Error()
			return sr
		}
		if res.Err != nil {
			sr.Err = res.Err.Error()
			return sr
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
			if err := deskact.UnicodeTypeWithWindow(r, t.WindowID, int(t.PID)); err != nil {
				sr.Err = fmt.Sprintf("UnicodeTypeWithWindow(%q): %v", string(r), err)
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
	case "occlude":
		// Spawn a foreground window to cover the target. The next
		// screenshot step exercises the "captured under occlusion"
		// path that per-window WGC / XComposite / CGWindowList all
		// promise. After spawning we look up the new window in the
		// system window list and move it to the target's current
		// bounds, so occlusion is actually provable instead of
		// relying on the OS happening to place the new window over
		// the target. cleanupSpawned() kills our children on exit so
		// repeated runs don't accumulate matching-title windows.
		cmd := step.Cmd
		if cmd == "" {
			cmd = defaultOccluderCmd()
		}
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			sr.Err = "occlude: no command resolved"
			return sr
		}
		// Snapshot the existing window set BEFORE spawn so we can
		// pick out the new one even when the spawn goes through a
		// launcher (`open -na TextEdit` on macOS exits immediately;
		// c.Process.Pid is the short-lived `open` shim's PID, not
		// TextEdit's). Matching by PID alone misses that case.
		before := map[uint64]bool{}
		if wins, lerr := deskact.ListWindows(deskact.WindowOptions{}); lerr == nil {
			for _, x := range wins {
				before[x.ID] = true
			}
		}
		c := exec.Command(parts[0], parts[1:]...)
		if err := c.Start(); err != nil {
			sr.Err = fmt.Sprintf("spawn %q: %v", cmd, err)
			return sr
		}
		spawnedOccluders = append(spawnedOccluders, c.Process)
		newPID := c.Process.Pid
		// Poll briefly for a new top-level window to appear. ~2s
		// covers Notepad / xterm / TextEdit cold-start.
		var coverID uint64
		var coverPID int
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			wins, lerr := deskact.ListWindows(deskact.WindowOptions{})
			if lerr == nil {
				// First preference: a window owned by the PID we
				// spawned (works for direct binary launches like
				// notepad.exe and xterm).
				for _, candidate := range wins {
					if candidate.PID == newPID && candidate.ID != w.ID {
						coverID = candidate.ID
						coverPID = candidate.PID
						break
					}
				}
				if coverID != 0 {
					break
				}
				// Second preference: the highest-PID new window
				// that wasn't in the pre-spawn snapshot. This is
				// the macOS-launcher path — `open -na TextEdit`
				// returns immediately but TextEdit comes up under
				// a fresh, higher PID a moment later.
				var bestID uint64
				var bestPID int
				for _, candidate := range wins {
					if before[candidate.ID] || candidate.ID == w.ID {
						continue
					}
					if candidate.PID > bestPID {
						bestPID = candidate.PID
						bestID = candidate.ID
					}
				}
				if bestID != 0 {
					coverID = bestID
					coverPID = bestPID
					// Track the actual app's process so cleanup
					// can kill it. The launcher process we
					// already appended will already have died.
					if p, perr := os.FindProcess(bestPID); perr == nil {
						spawnedOccluders = append(spawnedOccluders, p)
					}
					break
				}
			}
			time.Sleep(150 * time.Millisecond)
		}
		if coverID == 0 {
			sr.Err = fmt.Sprintf("spawned %s pid=%d but no new window registered in ListWindows within 2s — cannot prove occlusion", parts[0], newPID)
			return sr
		}
		// Position the cover over the target's last-known bounds. Use
		// WindowMoveResize so a single op handles both.
		bx, by, bw, bh := w.Bounds.X, w.Bounds.Y, w.Bounds.W, w.Bounds.H
		if err := deskact.WindowMoveResize(coverID, int32(coverPID), bx, by, bw, bh); err != nil {
			sr.Err = fmt.Sprintf("spawned %s pid=%d windowID=0x%x; move-resize over target failed: %v — cannot prove occlusion", parts[0], newPID, coverID, err)
			return sr
		}
		// Raise the cover to make sure it's actually in front.
		_ = deskact.WindowRaise(coverID, int32(coverPID))
		sr.Detail = fmt.Sprintf("spawned %s pid=%d windowID=0x%x covering (%d,%d %dx%d)", parts[0], newPID, coverID, bx, by, bw, bh)
		sr.OK = true
	default:
		sr.Err = "unknown op: " + step.Op
	}
	return sr
}

func defaultOccluderCmd() string {
	switch runtime.GOOS {
	case "windows":
		return "notepad.exe"
	case "linux":
		return "xterm -title deskact-occluder -geometry 100x40+0+0"
	case "darwin":
		return "open -na TextEdit"
	}
	return ""
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
