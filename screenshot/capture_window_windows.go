//go:build windows
// +build windows

package screenshot

import (
	"image"

	cap "github.com/PekingSpades/DeskAct/capture"
)

// captureWindowPlatform on Windows is implemented in a later commit (WGC +
// PrintWindow fallback). Until then it returns ErrUnsupported.
func captureWindowPlatform(req cap.WindowRequest) (*image.RGBA, error) {
	return nil, cap.ErrUnsupported
}
