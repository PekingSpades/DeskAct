//go:build windows

package display

import "testing"

func TestDefaultCaptureOptionsOnWindows(t *testing.T) {
	options := DefaultCaptureOptions()
	if options.Backend != CaptureBackendDXGI {
		t.Fatalf("expected default backend %q, got %q", CaptureBackendDXGI, options.Backend)
	}
}
