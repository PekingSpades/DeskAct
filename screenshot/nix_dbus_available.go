//go:build !s390x && !ppc64le && !darwin && !windows && (linux || openbsd || netbsd)

package screenshot

import (
	cap "github.com/PekingSpades/DeskAct/capture"
	"image"
	"os"
)

// Capture returns screen capture of specified desktop region.
// x and y represent distance from the upper-left corner of primary display.
// Y-axis is downward direction. This means coordinates system is similar to Windows OS.
func Capture(req cap.Request) (img *image.RGBA, e error) {
	if req.Options.Backend != cap.CaptureBackendDefault {
		return nil, backendUnavailableError(req.Options.Backend, "backend %q is not supported on Linux/X11/Wayland capture", req.Options.Backend)
	}
	if hasExcludedWindowIDs(req.Options) {
		return nil, windowExclusionUnsupportedError(req.Options.Backend)
	}

	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType == "wayland" {
		return captureDbus(req.X, req.Y, req.Width, req.Height, req.Options.WaylandToken)
	} else {
		return captureXinerama(req.X, req.Y, req.Width, req.Height)
	}
}
