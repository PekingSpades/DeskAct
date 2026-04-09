package screenshot

import (
	"errors"
	"testing"

	cap "github.com/PekingSpades/DeskAct/capture"
)

func TestBackendUnavailableErrorWrapsSentinel(t *testing.T) {
	err := backendUnavailableError(cap.CaptureBackendDXGI, "backend %q is unavailable", cap.CaptureBackendDXGI)
	if !errors.Is(err, cap.ErrCaptureBackendUnavailable) {
		t.Fatalf("expected ErrCaptureBackendUnavailable, got %v", err)
	}
}

func TestWindowExclusionUnsupportedErrorWrapsSentinel(t *testing.T) {
	err := windowExclusionUnsupportedError(cap.CaptureBackendScreenCaptureKit)
	if !errors.Is(err, cap.ErrWindowExclusionUnsupported) {
		t.Fatalf("expected ErrWindowExclusionUnsupported, got %v", err)
	}
}

func TestNormalizeRequestedBackend(t *testing.T) {
	if got := normalizeRequestedBackend(cap.CaptureBackendDefault, cap.CaptureBackendDXGI); got != cap.CaptureBackendDXGI {
		t.Fatalf("expected default backend to normalize to %q, got %q", cap.CaptureBackendDXGI, got)
	}
	if got := normalizeRequestedBackend(cap.CaptureBackendGDI, cap.CaptureBackendDXGI); got != cap.CaptureBackendGDI {
		t.Fatalf("expected explicit backend to be preserved, got %q", got)
	}
}

func TestHasExcludedWindowIDs(t *testing.T) {
	if hasExcludedWindowIDs(cap.CaptureOptions{}) {
		t.Fatal("expected empty options to report no excluded window IDs")
	}
	if !hasExcludedWindowIDs(cap.CaptureOptions{ExcludedWindowIDs: []uint64{42}}) {
		t.Fatal("expected excluded window IDs to be detected")
	}
}
