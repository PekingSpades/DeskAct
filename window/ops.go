package window

import (
	"github.com/PekingSpades/DeskAct/capture"
	"github.com/PekingSpades/DeskAct/display"
)

// Target identifies a window for the per-window ops below. Carries both
// the platform-native ID (HWND on Windows, CGWindowID on macOS, X11 XID
// on Linux) and the owning PID (used by macOS AX; ignored elsewhere).
type Target struct {
	ID  uint64
	PID int32
}

// FromInfo extracts a Target from a WindowInfo. Convenience for the
// common case of "the window I just enumerated".
func FromInfo(w WindowInfo) Target {
	return Target{ID: w.ID, PID: int32(w.PID)}
}

// Move sets the top-left position of a window in virtual desktop pixels.
// id is platform-specific (HWND on Windows, CGWindowID on macOS, X11 Window
// XID on Linux). pid is required on macOS and ignored on the others.
//
// Move accepts the full (id, pid, x, y) signature rather than the plan's
// (id, x, y) so the macOS AX path has both identifiers. Callers that
// already hold a Target should prefer MoveTarget for symmetry with
// FromInfo.
func Move(id uint64, pid int32, x, y int) error {
	return moveWindow(id, pid, x, y)
}

// MoveTarget is the WindowTarget form of Move.
func MoveTarget(t Target, x, y int) error {
	return Move(t.ID, t.PID, x, y)
}

// Resize sets a window's outer width and height in virtual desktop pixels.
func Resize(id uint64, pid int32, w, h int) error {
	return resizeWindow(id, pid, w, h)
}

// ResizeTarget is the WindowTarget form of Resize.
func ResizeTarget(t Target, w, h int) error {
	return Resize(t.ID, t.PID, w, h)
}

// MoveResize sets both position and size in one operation when supported by
// the platform (Windows SetWindowPos, X11 _NET_MOVERESIZE_WINDOW). Falls
// back to Move + Resize on macOS where the AX API exposes them separately.
func MoveResize(id uint64, pid int32, x, y, w, h int) error {
	return moveResizeWindow(id, pid, x, y, w, h)
}

// MoveResizeTarget is the WindowTarget form of MoveResize.
func MoveResizeTarget(t Target, x, y, w, h int) error {
	return MoveResize(t.ID, t.PID, x, y, w, h)
}

// Raise moves the window to the top of the Z-order without activating it
// (no focus change). On macOS the AX API has no non-activating raise, so
// Raise is equivalent to Focus there.
func Raise(id uint64, pid int32) error {
	return raiseWindow(id, pid)
}

// RaiseTarget is the WindowTarget form of Raise.
func RaiseTarget(t Target) error { return Raise(t.ID, t.PID) }

// Focus brings the window to the top and gives it keyboard focus.
func Focus(id uint64, pid int32) error {
	return focusWindow(id, pid)
}

// FocusTarget is the WindowTarget form of Focus.
func FocusTarget(t Target) error { return Focus(t.ID, t.PID) }

// Minimize iconifies the window.
func Minimize(id uint64, pid int32) error {
	return minimizeWindow(id, pid)
}

// MinimizeTarget is the WindowTarget form of Minimize.
func MinimizeTarget(t Target) error { return Minimize(t.ID, t.PID) }

// Restore un-minimizes the window if iconified; no-op if already shown.
func Restore(id uint64, pid int32) error {
	return restoreWindow(id, pid)
}

// RestoreTarget is the WindowTarget form of Restore.
func RestoreTarget(t Target) error { return Restore(t.ID, t.PID) }

// Close requests the window to close. Equivalent to clicking the close
// button — the application may refuse.
func Close(id uint64, pid int32) error {
	return closeWindow(id, pid)
}

// CloseTarget is the WindowTarget form of Close.
func CloseTarget(t Target) error { return Close(t.ID, t.PID) }

// Bounds returns the current outer bounds of the window in virtual desktop
// pixels. Returns capture.ErrWindowNotFound if the window cannot be located.
func Bounds(id uint64, pid int32) (display.Rect, error) {
	return boundsWindow(id, pid)
}

// BoundsTarget is the WindowTarget form of Bounds.
func BoundsTarget(t Target) (display.Rect, error) { return Bounds(t.ID, t.PID) }

// errOpUnsupported reuses the shared capture-level sentinel for symmetry
// with the per-window screenshot path.
func errOpUnsupported() error {
	return capture.ErrUnsupported
}
