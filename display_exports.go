package deskact

import "github.com/PekingSpades/DeskAct/display"

type PlatformInfo = display.PlatformInfo
type Display = display.Display
type DisplayInfo = display.DisplayInfo
type DisplayOptions = display.DisplayOptions
type CaptureBackend = display.CaptureBackend
type CaptureOptions = display.CaptureOptions
type Point = display.Point
type Size = display.Size
type Rect = display.Rect

const DefaultDPIAware = display.DefaultDPIAware

const (
	CaptureBackendDefault          = display.CaptureBackendDefault
	CaptureBackendGDI              = display.CaptureBackendGDI
	CaptureBackendDXGI             = display.CaptureBackendDXGI
	CaptureBackendScreenCaptureKit = display.CaptureBackendScreenCaptureKit
	CaptureBackendCGDisplay        = display.CaptureBackendCGDisplay
)

var (
	ErrCaptureBackendUnavailable  = display.ErrCaptureBackendUnavailable
	ErrWindowExclusionUnsupported = display.ErrWindowExclusionUnsupported
)

func DefaultDisplayOptions() DisplayOptions {
	return display.DefaultDisplayOptions()
}

func DefaultCaptureOptions() CaptureOptions {
	return display.DefaultCaptureOptions()
}

func MainDisplay(options DisplayOptions) *Display {
	return display.MainDisplay(options)
}

func AllDisplays(options DisplayOptions) []*Display {
	return display.AllDisplays(options)
}

func DisplayAt(index int, options DisplayOptions) *Display {
	return display.DisplayAt(index, options)
}

func DisplayCount() int {
	return display.DisplayCount()
}
