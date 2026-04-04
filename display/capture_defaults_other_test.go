//go:build !windows && !darwin

package display

import "testing"

func TestDefaultCaptureOptionsOnOtherPlatforms(t *testing.T) {
	options := DefaultCaptureOptions()
	if options.Backend != CaptureBackendDefault {
		t.Fatalf("expected default backend %q, got %q", CaptureBackendDefault, options.Backend)
	}
	if options.WaylandToken != 0 {
		t.Fatalf("expected zero Wayland token, got %d", options.WaylandToken)
	}
	if len(options.ExcludedWindowIDs) != 0 {
		t.Fatalf("expected no excluded window IDs, got %v", options.ExcludedWindowIDs)
	}
}
