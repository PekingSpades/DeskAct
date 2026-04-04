//go:build windows

package screenshot

import (
	"errors"
	"image"

	cap "github.com/PekingSpades/DeskAct/capture"
)

var windowsDXGIManager = newDXGIDuplicationManager(func(displayID int) (dxgiCaptureSession, error) {
	return newDXGIDuplicationSession(displayID)
})

func Capture(req cap.Request) (*image.RGBA, error) {
	if req.Width <= 0 || req.Height <= 0 {
		return nil, errors.New("width or height should be > 0")
	}

	switch normalizeRequestedBackend(req.Options.Backend, cap.CaptureBackendDXGI) {
	case cap.CaptureBackendDXGI:
		return windowsDXGIManager.Capture(req)
	case cap.CaptureBackendGDI:
		return captureGDI(req)
	default:
		backend := normalizeRequestedBackend(req.Options.Backend, cap.CaptureBackendDXGI)
		return nil, backendUnavailableError(backend, "backend %q is not supported on Windows", backend)
	}
}
