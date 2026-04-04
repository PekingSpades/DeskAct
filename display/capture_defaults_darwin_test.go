//go:build darwin

package display

import "testing"

func TestDefaultCaptureOptionsOnDarwin(t *testing.T) {
	options := DefaultCaptureOptions()
	if options.Backend != CaptureBackendScreenCaptureKit {
		t.Fatalf("expected default backend %q, got %q", CaptureBackendScreenCaptureKit, options.Backend)
	}
}
