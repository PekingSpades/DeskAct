//go:build s390x || ppc64le || (!(cgo && darwin) && !windows && !linux && !freebsd && !openbsd && !netbsd)

package screenshot

import (
	cap "github.com/PekingSpades/DeskAct/capture"
	"image"
)

// Capture returns screen capture of specified desktop region.
// x and y represent distance from the upper-left corner of primary display.
// Y-axis is downward direction. This means coordinates system is similar to Windows OS.
func Capture(req cap.Request) (*image.RGBA, error) {
	if req.Options.Backend != cap.CaptureBackendDefault {
		return nil, backendUnavailableError(req.Options.Backend, "backend %q is not supported on this platform", req.Options.Backend)
	}
	if hasExcludedWindowIDs(req.Options) {
		return nil, windowExclusionUnsupportedError(req.Options.Backend)
	}
	return nil, errUnsupported()
}
