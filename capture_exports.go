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
// that are occluded by other windows. Use CaptureWindowEx if you also need
// to know which backend produced the image or to distinguish OK from
// partial results.
func CaptureWindow(req CaptureWindowRequest) (*image.RGBA, error) {
	return screenshot.CaptureWindow(req)
}

// CaptureWindowResult is the structured return of CaptureWindowEx.
type CaptureWindowResult = screenshot.CaptureWindowResult

// CaptureWindowEx is the structured form of CaptureWindow. Prefer this when
// you want to record which backend ran or distinguish OK vs partial
// results. When the caller explicitly asked for a non-default backend
// (e.g. CaptureBackendWGC) and that backend fails, no fallback is
// performed — the error surfaces truthfully so callers can verify the
// requested code path.
func CaptureWindowEx(req CaptureWindowRequest) CaptureWindowResult {
	return screenshot.CaptureWindowEx(req)
}
