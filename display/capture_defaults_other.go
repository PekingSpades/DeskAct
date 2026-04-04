//go:build !windows && !darwin

package display

// DefaultCaptureOptions returns the default capture options.
func DefaultCaptureOptions() CaptureOptions {
	return CaptureOptions{}
}
