//go:build !cgo || (!darwin && !linux && !windows && !openbsd && !netbsd && !freebsd)

package screenshot

import (
	"image"

	cap "github.com/PekingSpades/DeskAct/capture"
)

func captureWindowPlatform(req cap.WindowRequest) (*image.RGBA, error) {
	return nil, cap.ErrUnsupported
}
