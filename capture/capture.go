package capture

import "errors"

type CaptureBackend string

const (
	CaptureBackendDefault          CaptureBackend = ""
	CaptureBackendGDI              CaptureBackend = "gdi"
	CaptureBackendDXGI             CaptureBackend = "dxgi"
	CaptureBackendScreenCaptureKit CaptureBackend = "screencapturekit"
	CaptureBackendCGDisplay        CaptureBackend = "cgdisplay"

	// Window-targeted backends introduced by the non-preemptive window ops work.
	CaptureBackendWGC          CaptureBackend = "wgc"
	CaptureBackendPrintWindow  CaptureBackend = "printwindow"
	CaptureBackendCGWindowList CaptureBackend = "cgwindowlist"
	CaptureBackendXComposite   CaptureBackend = "xcomposite"
)

type CaptureOptions struct {
	WaylandToken      uint64
	Backend           CaptureBackend
	ExcludedWindowIDs []uint64
}

type Request struct {
	DisplayID int
	X         int
	Y         int
	Width     int
	Height    int
	Options   CaptureOptions
}

// WindowRequest describes a per-window capture request. WindowID carries the
// platform-native handle (HWND on Windows, CGWindowID on macOS, X11 Window on
// Linux). PID is required on macOS when the caller wants the AX-trusted path;
// other platforms may leave it zero.
type WindowRequest struct {
	WindowID uint64
	PID      int32
	Options  CaptureOptions
}

var (
	ErrCaptureBackendUnavailable  = errors.New("capture backend unavailable")
	ErrWindowExclusionUnsupported = errors.New("window exclusion is unsupported by the selected capture backend")

	// Sentinels introduced for the per-window capture / non-preemptive ops API.
	ErrUnsupported      = errors.New("operation unsupported on this platform or session")
	ErrWindowNotFound   = errors.New("target window not found")
	ErrCaptureFailed    = errors.New("window capture failed")
	ErrPermissionDenied = errors.New("operating system permission required")
)
