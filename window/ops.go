package window

import (
	"github.com/PekingSpades/DeskAct/capture"
	"github.com/PekingSpades/DeskAct/display"
)

// Move sets the top-left position of a window in virtual desktop pixels.
// id is platform-specific (HWND on Windows, CGWindowID on macOS, X11 Window
// XID on Linux). pid is required on macOS and ignored on the others.
func Move(id uint64, pid int32, x, y int) error {
	return moveWindow(id, pid, x, y)
}

// Resize sets a window's outer width and height in virtual desktop pixels.
func Resize(id uint64, pid int32, w, h int) error {
	return resizeWindow(id, pid, w, h)
}

// MoveResize sets both position and size in one operation when supported by
// the platform (Windows SetWindowPos, X11 _NET_MOVERESIZE_WINDOW). Falls
// back to Move + Resize on macOS where the AX API exposes them separately.
func MoveResize(id uint64, pid int32, x, y, w, h int) error {
	return moveResizeWindow(id, pid, x, y, w, h)
}

// Raise moves the window to the top of the Z-order without activating it
// (no focus change). On macOS the AX API has no non-activating raise, so
// Raise is equivalent to Focus there.
func Raise(id uint64, pid int32) error {
	return raiseWindow(id, pid)
}

// Focus brings the window to the top and gives it keyboard focus.
func Focus(id uint64, pid int32) error {
	return focusWindow(id, pid)
}

// Minimize iconifies the window.
func Minimize(id uint64, pid int32) error {
	return minimizeWindow(id, pid)
}

// Restore un-minimizes the window if iconified; no-op if already shown.
func Restore(id uint64, pid int32) error {
	return restoreWindow(id, pid)
}

// Close requests the window to close. Equivalent to clicking the close
// button — the application may refuse.
func Close(id uint64, pid int32) error {
	return closeWindow(id, pid)
}

// Bounds returns the current outer bounds of the window in virtual desktop
// pixels. Returns capture.ErrWindowNotFound if the window cannot be located.
func Bounds(id uint64, pid int32) (display.Rect, error) {
	return boundsWindow(id, pid)
}

// errOpUnsupported reuses the shared capture-level sentinel for symmetry
// with the per-window screenshot path.
func errOpUnsupported() error {
	return capture.ErrUnsupported
}
