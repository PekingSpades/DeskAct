package window

import (
	"fmt"

	cap "github.com/PekingSpades/DeskAct/capture"
	"github.com/PekingSpades/DeskAct/display"
)

// ErrUnsupported is returned when a per-window listing or ops entry point
// is invoked on a session that cannot service it (Wayland today, missing
// DISPLAY, etc.). It wraps capture.ErrUnsupported so callers can use
// errors.Is(err, capture.ErrUnsupported) against any per-window error
// from the deskact surface (capture, window, mouse, keyboard) and not
// have to special-case window.ErrUnsupported separately. Docs in
// AGENTS.md describe this as the single sentinel for "the platform
// can't do this".
var ErrUnsupported = fmt.Errorf("%w: window operation not supported on this platform", cap.ErrUnsupported)

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
