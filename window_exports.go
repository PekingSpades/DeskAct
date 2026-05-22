package deskact

import (
	"github.com/PekingSpades/DeskAct/display"
	"github.com/PekingSpades/DeskAct/window"
)

type WindowOptions = window.WindowOptions
type WindowInfo = window.WindowInfo
type WindowDisplayRegion = window.DisplayRegion
type WindowPlatformInfo = window.PlatformInfo

var ErrWindowUnsupported = window.ErrUnsupported

func DefaultWindowOptions() WindowOptions {
	return window.DefaultWindowOptions()
}

func ListWindows(options WindowOptions) ([]WindowInfo, error) {
	return window.List(options)
}

// Per-window operations (non-preemptive — they target a specific window
// rather than the focused one). id is HWND on Windows, CGWindowID on
// macOS, X11 Window XID on Linux. pid is required on macOS to bind the
// AXUIElement to the owning application; other platforms ignore it.

func WindowMove(id uint64, pid int32, x, y int) error {
	return window.Move(id, pid, x, y)
}

func WindowResize(id uint64, pid int32, w, h int) error {
	return window.Resize(id, pid, w, h)
}

func WindowMoveResize(id uint64, pid int32, x, y, w, h int) error {
	return window.MoveResize(id, pid, x, y, w, h)
}

func WindowRaise(id uint64, pid int32) error {
	return window.Raise(id, pid)
}

func WindowFocus(id uint64, pid int32) error {
	return window.Focus(id, pid)
}

func WindowMinimize(id uint64, pid int32) error {
	return window.Minimize(id, pid)
}

func WindowRestore(id uint64, pid int32) error {
	return window.Restore(id, pid)
}

func WindowClose(id uint64, pid int32) error {
	return window.Close(id, pid)
}

func WindowBounds(id uint64, pid int32) (display.Rect, error) {
	return window.Bounds(id, pid)
}
