package main

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	deskact "github.com/PekingSpades/DeskAct"
)

type state struct {
	reader   *bufio.Reader
	windows  []deskact.WindowInfo
	selected int // index into windows; -1 if none
}

func main() {
	fmt.Println("========================================")
	fmt.Println("DeskAct Non-Preemptive Window Ops Example")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("========================================")

	s := &state{reader: bufio.NewReader(os.Stdin), selected: -1}

	if yes, info := promptDPIInit(s.reader); yes {
		fmt.Printf("DPI init: %s\n", info)
	}

	for {
		s.printSelected()
		fmt.Println()
		fmt.Println("Choose an operation:")
		fmt.Println("  1) List windows")
		fmt.Println("  2) Select window")
		fmt.Println("  3) Screenshot selected window")
		fmt.Println("  4) Click in selected window (x, y, button)")
		fmt.Println("  5) Type text in selected window")
		fmt.Println("  6) Move selected window (x, y)")
		fmt.Println("  7) Resize selected window (w, h)")
		fmt.Println("  8) Focus / Raise / Minimize / Restore selected window")
		fmt.Println("  9) Close selected window")
		fmt.Println("  q) Quit")
		fmt.Print("> ")
		choice := readLine(s.reader)
		switch choice {
		case "1":
			s.cmdList()
		case "2":
			s.cmdSelect()
		case "3":
			s.cmdScreenshot()
		case "4":
			s.cmdClick()
		case "5":
			s.cmdType()
		case "6":
			s.cmdMove()
		case "7":
			s.cmdResize()
		case "8":
			s.cmdLifecycle()
		case "9":
			s.cmdClose()
		case "q", "quit", "exit":
			return
		case "":
			continue
		default:
			fmt.Printf("unknown choice: %q\n", choice)
		}
	}
}

func (s *state) printSelected() {
	if s.selected < 0 || s.selected >= len(s.windows) {
		fmt.Println("\n[ no window selected ]")
		return
	}
	w := s.windows[s.selected]
	fmt.Printf("\n[ selected: idx=%d id=0x%x pid=%d %q (%dx%d at %d,%d) ]\n",
		s.selected, w.ID, w.PID, w.Title, w.Bounds.W, w.Bounds.H, w.Bounds.X, w.Bounds.Y)
}

func (s *state) cmdList() {
	opts := deskact.DefaultWindowOptions()
	wins, err := deskact.ListWindows(opts)
	if err != nil {
		fmt.Printf("list failed: %v\n", err)
		return
	}
	s.windows = wins
	for i, w := range wins {
		marker := "  "
		if i == s.selected {
			marker = "->"
		}
		fmt.Printf("%s %3d  id=0x%-10x pid=%-6d %4dx%-4d @(%d,%d)  %q\n",
			marker, i, w.ID, w.PID, w.Bounds.W, w.Bounds.H, w.Bounds.X, w.Bounds.Y, w.Title)
	}
	if len(wins) == 0 {
		fmt.Println("(no windows)")
	}
}

func (s *state) cmdSelect() {
	if len(s.windows) == 0 {
		s.cmdList()
	}
	fmt.Print("Window index: ")
	line := readLine(s.reader)
	idx, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || idx < 0 || idx >= len(s.windows) {
		fmt.Printf("invalid index %q\n", line)
		return
	}
	s.selected = idx
}

func (s *state) target() (deskact.WindowInfo, bool) {
	if s.selected < 0 || s.selected >= len(s.windows) {
		fmt.Println("no window selected — use option 2 first")
		return deskact.WindowInfo{}, false
	}
	return s.windows[s.selected], true
}

func (s *state) mouseTarget(w deskact.WindowInfo) deskact.MouseWindowTarget {
	return deskact.MouseWindowTarget{WindowID: w.ID, PID: int32(w.PID)}
}

func (s *state) delayBeforeOp() time.Duration {
	fmt.Print("Delay before operation (seconds, default 0): ")
	line := readLine(s.reader)
	line = strings.TrimSpace(line)
	if line == "" {
		return 0
	}
	v, err := strconv.ParseFloat(line, 64)
	if err != nil || v < 0 {
		fmt.Printf("invalid delay %q, using 0\n", line)
		return 0
	}
	d := time.Duration(v * float64(time.Second))
	if d > 0 {
		fmt.Printf("sleeping %s ...\n", d)
		time.Sleep(d)
	}
	return d
}

func (s *state) cmdScreenshot() {
	w, ok := s.target()
	if !ok {
		return
	}
	backend := promptBackend(s.reader)
	s.delayBeforeOp()
	req := deskact.CaptureWindowRequest{
		WindowID: w.ID,
		PID:      int32(w.PID),
		Options:  deskact.CaptureOptions{Backend: backend},
	}
	start := time.Now()
	img, err := deskact.CaptureWindow(req)
	elapsed := time.Since(start)
	if err != nil && img == nil {
		fmt.Printf("op=screenshot win=%d ok=false elapsed=%s err=%v\n", w.ID, elapsed, err)
		return
	}
	if err != nil {
		fmt.Printf("op=screenshot win=%d partial=true elapsed=%s warn=%v\n", w.ID, elapsed, err)
	}
	fname := fmt.Sprintf("windowops-%d-%x.png", time.Now().Unix(), w.ID)
	if e := savePNG(fname, img); e != nil {
		fmt.Printf("op=screenshot win=%d ok=false save_err=%v\n", w.ID, e)
		return
	}
	fmt.Printf("op=screenshot win=%d ok=true elapsed=%s png=%s size=%dx%d backend=%q\n",
		w.ID, elapsed, fname, img.Bounds().Dx(), img.Bounds().Dy(), backend)
}

func (s *state) cmdClick() {
	w, ok := s.target()
	if !ok {
		return
	}
	x := promptInt(s.reader, "Client x", 0)
	y := promptInt(s.reader, "Client y", 0)
	button := promptButton(s.reader)
	s.delayBeforeOp()
	start := time.Now()
	err := deskact.ClickWithWindow(s.mouseTarget(w), x, y, button, deskact.DefaultMouseSettings())
	fmt.Printf("op=click win=%d at=(%d,%d) btn=%s elapsed=%s err=%v\n",
		w.ID, x, y, button, time.Since(start), err)
}

func (s *state) cmdType() {
	w, ok := s.target()
	if !ok {
		return
	}
	fmt.Print("Text to type: ")
	text := readLine(s.reader)
	s.delayBeforeOp()
	start := time.Now()
	settings := deskact.DefaultKeyboardSettings()
	for _, r := range text {
		if err := deskact.KeyTapWithPID(string(r), int(w.ID), nil, settings); err != nil {
			fmt.Printf("op=type win=%d ok=false at=%q err=%v\n", w.ID, string(r), err)
			return
		}
	}
	fmt.Printf("op=type win=%d ok=true text=%q elapsed=%s\n", w.ID, text, time.Since(start))
}

func (s *state) cmdMove() {
	w, ok := s.target()
	if !ok {
		return
	}
	x := promptInt(s.reader, "New x", w.Bounds.X)
	y := promptInt(s.reader, "New y", w.Bounds.Y)
	s.delayBeforeOp()
	start := time.Now()
	err := deskact.WindowMove(w.ID, int32(w.PID), x, y)
	fmt.Printf("op=move win=%d at=(%d,%d) elapsed=%s err=%v\n", w.ID, x, y, time.Since(start), err)
}

func (s *state) cmdResize() {
	w, ok := s.target()
	if !ok {
		return
	}
	cw := promptInt(s.reader, "New width", w.Bounds.W)
	ch := promptInt(s.reader, "New height", w.Bounds.H)
	s.delayBeforeOp()
	start := time.Now()
	err := deskact.WindowResize(w.ID, int32(w.PID), cw, ch)
	fmt.Printf("op=resize win=%d size=(%dx%d) elapsed=%s err=%v\n", w.ID, cw, ch, time.Since(start), err)
}

func (s *state) cmdLifecycle() {
	w, ok := s.target()
	if !ok {
		return
	}
	fmt.Println("  a) Focus    b) Raise    c) Minimize    d) Restore")
	fmt.Print("> ")
	c := strings.TrimSpace(readLine(s.reader))
	s.delayBeforeOp()
	var err error
	op := ""
	switch c {
	case "a":
		op, err = "focus", deskact.WindowFocus(w.ID, int32(w.PID))
	case "b":
		op, err = "raise", deskact.WindowRaise(w.ID, int32(w.PID))
	case "c":
		op, err = "minimize", deskact.WindowMinimize(w.ID, int32(w.PID))
	case "d":
		op, err = "restore", deskact.WindowRestore(w.ID, int32(w.PID))
	default:
		fmt.Printf("unknown sub-op %q\n", c)
		return
	}
	fmt.Printf("op=%s win=%d err=%v\n", op, w.ID, err)
}

func (s *state) cmdClose() {
	w, ok := s.target()
	if !ok {
		return
	}
	s.delayBeforeOp()
	err := deskact.WindowClose(w.ID, int32(w.PID))
	fmt.Printf("op=close win=%d err=%v\n", w.ID, err)
}

// ---------- prompts / helpers ----------

func readLine(r *bufio.Reader) string {
	s, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, os.ErrClosed) {
		// EOF or read error — treat as quit
		if s == "" {
			os.Exit(0)
		}
	}
	return strings.TrimRight(s, "\r\n")
}

func promptDPIInit(r *bufio.Reader) (bool, string) {
	for {
		fmt.Print("Initialize DPI awareness? (y/n, default n): ")
		s := strings.ToLower(strings.TrimSpace(readLine(r)))
		switch s {
		case "y", "yes":
			result := deskact.InitDPIAwareness()
			return true, fmt.Sprintf("%v", result)
		case "", "n", "no":
			return false, "skipped"
		}
	}
}

func promptInt(r *bufio.Reader, label string, def int) int {
	fmt.Printf("%s [%d]: ", label, def)
	s := strings.TrimSpace(readLine(r))
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		fmt.Printf("invalid integer %q, using %d\n", s, def)
		return def
	}
	return v
}

func promptButton(r *bufio.Reader) deskact.MouseButton {
	fmt.Print("Button [left/right/middle, default left]: ")
	s := strings.ToLower(strings.TrimSpace(readLine(r)))
	switch s {
	case "right", "r":
		return deskact.MouseButtonRight
	case "middle", "m":
		return deskact.MouseButtonMiddle
	}
	return deskact.MouseButtonLeft
}

func promptBackend(r *bufio.Reader) deskact.CaptureBackend {
	fmt.Println("Backend: 1) default  2) wgc  3) printwindow  4) cgwindowlist  5) xcomposite")
	fmt.Print("> ")
	s := strings.TrimSpace(readLine(r))
	switch s {
	case "2":
		return deskact.CaptureBackendWGC
	case "3":
		return deskact.CaptureBackendPrintWindow
	case "4":
		return deskact.CaptureBackendCGWindowList
	case "5":
		return deskact.CaptureBackendXComposite
	}
	return deskact.CaptureBackendDefault
}

func savePNG(name string, img *image.RGBA) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
