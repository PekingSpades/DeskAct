package deskact

import "github.com/PekingSpades/DeskAct/window"

type WindowOptions = window.WindowOptions
type WindowInfo = window.WindowInfo
type WindowDisplayRegion = window.DisplayRegion
type WindowPlatformInfo = window.PlatformInfo

var ErrWindowUnsupported = window.ErrUnsupported

func DefaultWindowOptions() WindowOptions {
	return window.DefaultWindowOptions()
}

func ListWindows(options WindowOptions) ([]WindowInfo, error) {
	return window.List(options)
}
