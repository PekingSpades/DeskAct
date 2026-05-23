//go:build !(cgo && darwin)
// +build !cgo !darwin

package mouse

// warpSystemCursor is only meaningful on macOS where the per-pid scroll
// event is delivered to the window under the system cursor. Other
// platforms post the event with explicit client coordinates and need no
// cursor movement.
func warpSystemCursor(_, _ int) {}
