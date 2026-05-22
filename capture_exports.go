package deskact

import (
	"image"

	cap "github.com/PekingSpades/DeskAct/capture"
	"github.com/PekingSpades/DeskAct/screenshot"
)

// CaptureBackend, CaptureOptions, ErrCaptureBackendUnavailable, and the
// display-oriented CaptureBackend* constants are exported by
// display_exports.go via aliasing. Below we only add the new types,
// constants, and entry points introduced by the per-window screenshot
// work.

type CaptureRequest = cap.Request
type CaptureWindowRequest = cap.WindowRequest

const (
	CaptureBackendWGC          = cap.CaptureBackendWGC
	CaptureBackendPrintWindow  = cap.CaptureBackendPrintWindow
	CaptureBackendCGWindowList = cap.CaptureBackendCGWindowList
	CaptureBackendXComposite   = cap.CaptureBackendXComposite
)

var (
	ErrCaptureUnsupported      = cap.ErrUnsupported
	ErrCaptureWindowNotFound   = cap.ErrWindowNotFound
	ErrCaptureFailed           = cap.ErrCaptureFailed
	ErrCapturePermissionDenied = cap.ErrPermissionDenied
)

// CaptureScreen takes a screenshot of a display region (existing behavior).
func CaptureScreen(req CaptureRequest) (*image.RGBA, error) {
	return screenshot.Capture(req)
}

// CaptureWindow takes a screenshot of a specific window, including pixels
// that are occluded by other windows.
func CaptureWindow(req CaptureWindowRequest) (*image.RGBA, error) {
	return screenshot.CaptureWindow(req)
}
