//go:build darwin

package display

import "testing"

func TestDefaultCaptureOptionsOnDarwin(t *testing.T) {
	options := DefaultCaptureOptions()
	if options.Backend != CaptureBackendCGDisplay {
		t.Fatalf("expected default backend %q, got %q", CaptureBackendCGDisplay, options.Backend)
	}
}
