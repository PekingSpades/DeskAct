//go:build linux
// +build linux

package screenshot

import (
	"image"

	cap "github.com/PekingSpades/DeskAct/capture"
)

// captureWindowPlatform on Linux is implemented in a later commit (XComposite
// + XGetImage). Until then it returns ErrUnsupported.
func captureWindowPlatform(req cap.WindowRequest) (*image.RGBA, error) {
	return nil, cap.ErrUnsupported
}
