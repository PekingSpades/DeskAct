package capture

import "errors"

type CaptureBackend string

const (
	CaptureBackendDefault          CaptureBackend = ""
	CaptureBackendGDI              CaptureBackend = "gdi"
	CaptureBackendDXGI             CaptureBackend = "dxgi"
	CaptureBackendScreenCaptureKit CaptureBackend = "screencapturekit"
	CaptureBackendCGDisplay        CaptureBackend = "cgdisplay"
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

var (
	ErrCaptureBackendUnavailable  = errors.New("capture backend unavailable")
	ErrWindowExclusionUnsupported = errors.New("window exclusion is unsupported by the selected capture backend")
)
