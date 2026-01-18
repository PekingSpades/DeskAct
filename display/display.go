package display

// PlatformInfo is an interface for platform-specific display information.
type PlatformInfo interface {
	// Platform returns the platform name (e.g., "windows", "darwin", "linux").
	Platform() string
}

// Display represents a physical display/monitor.
type Display struct {
	id       int          // Platform-specific display ID
	index    int          // Display index (0 is main display)
	isMain   bool         // Whether this is the main display
	origin   Rect         // Bounds in platform's coordinate system (position + logical size)
	size     Size         // Physical pixel size
	scale    float64      // Scale factor (physical pixels / virtual points)
	platform PlatformInfo // Platform-specific information (nil if not set)
	dpiAware bool         // Windows-only DPI awareness flag
}

// DisplayInfo contains detailed information about a display.
type DisplayInfo struct {
	ID          int     // Platform-specific ID
	Index       int     // Index
	IsMain      bool    // Whether this is the main display
	Origin      Rect    // Bounds in platform's coordinate system
	Size        Size    // Physical pixel size
	ScaleFactor float64 // Scale factor
}

// ID returns the platform-specific display identifier.
func (d *Display) ID() int {
	return d.id
}

// Index returns the display index (0 is the main display).
func (d *Display) Index() int {
	return d.index
}

// IsMain returns whether this is the main display.
func (d *Display) IsMain() bool {
	return d.isMain
}

// Origin returns the display bounds in platform's coordinate system.
func (d *Display) Origin() Rect {
	return d.origin
}

// Size returns the display physical pixel size.
func (d *Display) Size() Size {
	return d.size
}

// Width returns the display physical pixel width.
func (d *Display) Width() int {
	return d.size.W
}

// Height returns the display physical pixel height.
func (d *Display) Height() int {
	return d.size.H
}

// Scale returns the scale factor (physical pixels / virtual points).
func (d *Display) Scale() float64 {
	return d.scale
}

// GetPlatformInfo returns the platform-specific information.
func (d *Display) GetPlatformInfo() PlatformInfo {
	return d.platform
}

// Info returns detailed display information.
func (d *Display) Info() DisplayInfo {
	return DisplayInfo{
		ID:          d.id,
		Index:       d.index,
		IsMain:      d.isMain,
		Origin:      d.origin,
		Size:        d.size,
		ScaleFactor: d.scale,
	}
}

// Platform-specific methods:
// - ToAbsolute(x, y int) (absX, absY int)
// - ToRelative(absX, absY int) (x, y int, ok bool)
// - Contains(absX, absY int) bool
// - Move(x, y int, settings MouseSettings) error
// - MoveSmooth(x, y int, settings MouseSettings) error
// - Drag(fromX, fromY, toX, toY int, button MouseButton, settings MouseSettings) error
// - DragTo(x, y int, button MouseButton, settings MouseSettings) error
// - CaptureRect(x, y, w, h int, options CaptureOptions) (*image.RGBA, error)
// - MouseLocation() (x, y int, ok bool)
// - ContainsMouse() bool
