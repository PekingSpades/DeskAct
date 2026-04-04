//go:build darwin

package display

import cap "github.com/PekingSpades/DeskAct/capture"

// DefaultCaptureOptions returns the default capture options.
func DefaultCaptureOptions() CaptureOptions {
	return CaptureOptions{
		Backend: cap.CaptureBackendScreenCaptureKit,
	}
}
