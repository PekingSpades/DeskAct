package screenshot

import (
	"image"

	cap "github.com/PekingSpades/DeskAct/capture"
)

// CaptureWindow takes a screenshot of a specific window, including pixels
// that are occluded by other windows. Each platform picks a default backend
// that handles occlusion correctly:
//
//   - Windows: WGC (Windows.Graphics.Capture) with PrintWindow fallback.
//   - macOS:   CGWindowListCreateImage with kCGWindowImageBoundsIgnoreFraming.
//   - Linux:   XComposite redirect + XGetImage of the offscreen pixmap.
//
// The default is selected when req.Options.Backend is the empty
// CaptureBackendDefault. Callers can override (e.g. force
// CaptureBackendPrintWindow on Windows) when WGC misbehaves on a particular
// window.
func CaptureWindow(req cap.WindowRequest) (*image.RGBA, error) {
	return captureWindowPlatform(req)
}
