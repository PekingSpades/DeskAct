package display

import cap "github.com/PekingSpades/DeskAct/capture"

type CaptureBackend = cap.CaptureBackend

const (
	CaptureBackendDefault          = cap.CaptureBackendDefault
	CaptureBackendGDI              = cap.CaptureBackendGDI
	CaptureBackendDXGI             = cap.CaptureBackendDXGI
	CaptureBackendScreenCaptureKit = cap.CaptureBackendScreenCaptureKit
	CaptureBackendCGDisplay        = cap.CaptureBackendCGDisplay
)

type CaptureOptions = cap.CaptureOptions

var (
	ErrCaptureBackendUnavailable  = cap.ErrCaptureBackendUnavailable
	ErrWindowExclusionUnsupported = cap.ErrWindowExclusionUnsupported
)
