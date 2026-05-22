//go:build linux
// +build linux

package window

// LinuxPlatformInfo contains X11-specific window metadata. It is attached to
// every WindowInfo returned by List on Linux X11.
type LinuxPlatformInfo struct {
	XID          uint32
	ParentXID    uint32
	Name         string // _NET_WM_NAME (UTF-8) preferred over WM_NAME
	Class        string // WM_CLASS class portion
	Instance     string // WM_CLASS instance portion
	FrameExtents [4]int // L, R, T, B
	States       []string
	Stacking     int // index in _NET_CLIENT_LIST_STACKING (-1 if unknown)
}

// Platform returns the platform name.
func (l *LinuxPlatformInfo) Platform() string {
	return "linux"
}

// IsMinimized reports whether the window has the _NET_WM_STATE_HIDDEN state.
func (l *LinuxPlatformInfo) IsMinimized() bool {
	for _, s := range l.States {
		if s == "_NET_WM_STATE_HIDDEN" {
			return true
		}
	}
	return false
}
