package window

import (
	"errors"

	"github.com/PekingSpades/DeskAct/display"
)

var ErrUnsupported = errors.New("window listing is not supported on this platform")

// WindowOptions defines options for listing windows.
type WindowOptions struct {
	// DPIAware controls whether coordinates are returned in physical pixels on Windows.
	// On macOS this flag is ignored because display coordinates are already virtual.
	// The caller should set this to match the current process DPI awareness.
	DPIAware bool

	// IncludeMinimized includes minimized windows when supported.
	IncludeMinimized bool
	// IncludeOffscreen includes windows not currently visible on screen (macOS).
	IncludeOffscreen bool
	// IncludeInvisible includes windows that are not visible (Windows).
	IncludeInvisible bool
	// IncludeToolWindows includes tool windows (Windows).
	IncludeToolWindows bool
	// IncludeCloaked includes cloaked windows such as those on other desktops (Windows).
	IncludeCloaked bool
	// IncludeAllLayers includes non-layer-0 windows (macOS).
	IncludeAllLayers bool
}

// DefaultWindowOptions returns the default options for window listing.
func DefaultWindowOptions() WindowOptions {
	return WindowOptions{
		DPIAware: display.DefaultDisplayOptions().DPIAware,
	}
}

// DisplayRegion describes the window's intersection with a display.
type DisplayRegion struct {
	DisplayIndex int
	DisplayID    int
	// PhysicalRect is in physical pixels relative to the display origin.
	PhysicalRect display.Rect
}

// PlatformInfo provides platform-specific window metadata.
type PlatformInfo interface {
	Platform() string
}

// WindowInfo describes a desktop window and its location.
type WindowInfo struct {
	ID             uint64
	PID            int
	Title          string
	Bounds         display.Rect
	IsVisible      bool
	IsMinimized    bool
	DisplayRegions []DisplayRegion
	platform       PlatformInfo
}

// GetPlatformInfo returns platform-specific metadata for the window.
func (w WindowInfo) GetPlatformInfo() PlatformInfo {
	return w.platform
}

// List returns the current desktop windows and their locations.
func List(options WindowOptions) ([]WindowInfo, error) {
	windows, err := listWindows(options)
	if err != nil {
		return nil, err
	}
	if len(windows) == 0 {
		return windows, nil
	}

	displays := display.AllDisplays(display.DisplayOptions{DPIAware: options.DPIAware})
	if len(displays) == 0 {
		return windows, nil
	}

	attachDisplayRegions(windows, displays, options)
	return windows, nil
}

func errUnsupported() error {
	return ErrUnsupported
}
