// Package-local Wayland guard used by per-window mouse APIs on Linux.
// Mouse-injection via XSendEvent requires X11; under a Wayland compositor
// (or unknown session type) the per-window mouse entry points return
// capture.ErrUnsupported instead of producing silent failures.

package mouse

import "os"

func isWaylandSession() bool {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	return os.Getenv("XDG_SESSION_TYPE") == "wayland"
}
