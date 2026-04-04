package screenshot

import (
	"fmt"

	cap "github.com/PekingSpades/DeskAct/capture"
)

func backendUnavailableError(backend cap.CaptureBackend, format string, args ...any) error {
	detail := fmt.Sprintf(format, args...)
	if detail == "" {
		detail = fmt.Sprintf("backend %q is unavailable", backend)
	}
	return fmt.Errorf("%w: %s", cap.ErrCaptureBackendUnavailable, detail)
}

func windowExclusionUnsupportedError(backend cap.CaptureBackend) error {
	if backend == cap.CaptureBackendDefault {
		return fmt.Errorf("%w: the default backend does not support excluded window IDs", cap.ErrWindowExclusionUnsupported)
	}
	return fmt.Errorf("%w: backend %q does not support excluded window IDs", cap.ErrWindowExclusionUnsupported, backend)
}

func normalizeRequestedBackend(requested, defaultBackend cap.CaptureBackend) cap.CaptureBackend {
	if requested == cap.CaptureBackendDefault {
		return defaultBackend
	}
	return requested
}

func hasExcludedWindowIDs(options cap.CaptureOptions) bool {
	return len(options.ExcludedWindowIDs) > 0
}
