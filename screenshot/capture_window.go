package screenshot

import (
	"image"

	cap "github.com/PekingSpades/DeskAct/capture"
)

// CaptureWindowResult pairs the captured image with metadata about how it
// was produced. BackendUsed tells the caller whether the requested backend
// (or the default) actually ran — useful for proving "WGC succeeded" rather
// than "the default fell back to PrintWindow". Partial=true means the image
// was returned but is known to be imperfect (e.g. PrintWindow blank-frame
// from a window that wouldn't render), with Err holding the reason. OK
// results have Partial=false and Err=nil.
type CaptureWindowResult struct {
	Image       *image.RGBA
	BackendUsed cap.CaptureBackend
	Partial     bool
	Err         error
}

// CaptureWindow takes a screenshot of a specific window, including pixels
// that are occluded by other windows. Each platform picks a default backend
// that handles occlusion correctly:
//
//   - Windows: WGC (Windows.Graphics.Capture) with PrintWindow fallback
//     when the caller requests CaptureBackendDefault. An explicit
//     CaptureBackendWGC request does NOT fall back — failure surfaces
//     as an error.
//   - macOS:   CGWindowListCreateImage with kCGWindowImageBoundsIgnoreFraming.
//   - Linux:   XComposite redirect + XGetImage of the offscreen pixmap.
//
// The default is selected when req.Options.Backend is the empty
// CaptureBackendDefault. Callers can override (e.g. force
// CaptureBackendPrintWindow on Windows) when WGC misbehaves on a particular
// window.
func CaptureWindow(req cap.WindowRequest) (*image.RGBA, error) {
	res := CaptureWindowEx(req)
	return res.Image, res.Err
}

// CaptureWindowEx is the structured form of CaptureWindow. Prefer this when
// you want to know which backend produced the image or distinguish OK from
// "partial / known imperfect" results.
func CaptureWindowEx(req cap.WindowRequest) CaptureWindowResult {
	return captureWindowPlatformEx(req)
}
