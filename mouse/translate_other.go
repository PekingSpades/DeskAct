//go:build !(cgo && darwin)
// +build !cgo !darwin

package mouse

// translateClientToScreen is a no-op on non-Darwin platforms because the
// per-window mouse C entry points accept client-relative coordinates
// directly (Windows: MAKELPARAM; X11: XSendEvent x/y).
func translateClientToScreen(_ uint64, x, y int) (int, int, bool) {
	return x, y, false
}
