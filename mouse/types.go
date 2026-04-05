package mouse

import "fmt"

// MouseButton identifies a logical mouse button.
type MouseButton int

const (
	MouseButtonLeft MouseButton = iota + 1
	MouseButtonRight
	MouseButtonMiddle
	MouseButtonBack
	MouseButtonForward
)

// MouseButtonCenter is an alias for MouseButtonMiddle.
const MouseButtonCenter MouseButton = MouseButtonMiddle

const mouseButtonOtherBase MouseButton = 1000

// MouseButtonOther returns an extra mouse button by index (1-based).
func MouseButtonOther(index int) MouseButton {
	return MouseButton(mouseButtonOtherBase + MouseButton(index))
}

// OtherIndex returns the 1-based index for extra mouse buttons.
func (b MouseButton) OtherIndex() (int, bool) {
	if b >= mouseButtonOtherBase {
		return int(b - mouseButtonOtherBase), true
	}
	return 0, false
}

func (b MouseButton) String() string {
	switch b {
	case MouseButtonLeft:
		return "left"
	case MouseButtonRight:
		return "right"
	case MouseButtonMiddle:
		return "middle"
	case MouseButtonBack:
		return "back"
	case MouseButtonForward:
		return "forward"
	}
	if idx, ok := b.OtherIndex(); ok {
		return fmt.Sprintf("other(%d)", idx)
	}
	return fmt.Sprintf("button(%d)", int(b))
}

// ScrollUnit defines the scroll measurement unit.
type ScrollUnit int

const (
	ScrollUnitLine ScrollUnit = iota
	ScrollUnitPixel
)

func (u ScrollUnit) String() string {
	switch u {
	case ScrollUnitLine:
		return "line"
	case ScrollUnitPixel:
		return "pixel"
	default:
		return fmt.Sprintf("scrollUnit(%d)", int(u))
	}
}

// ScrollDelta represents a scroll amount in a specific unit.
// Positive X scrolls right; positive Y scrolls up.
type ScrollDelta struct {
	X    int
	Y    int
	Unit ScrollUnit
}

// ScrollDeltaLines builds a line-based scroll delta.
func ScrollDeltaLines(x, y int) ScrollDelta {
	return ScrollDelta{X: x, Y: y, Unit: ScrollUnitLine}
}

// ScrollDeltaPixels builds a pixel-based scroll delta.
func ScrollDeltaPixels(x, y int) ScrollDelta {
	return ScrollDelta{X: x, Y: y, Unit: ScrollUnitPixel}
}

// MouseCapabilities describes supported mouse features for the platform.
type MouseCapabilities struct {
	Buttons             []MouseButton
	MaxOtherButtons     int
	ScrollUnits         []ScrollUnit
	PixelScrollEmulated bool
}

// CurrentMouseCapabilities returns the current platform's mouse capabilities.
func CurrentMouseCapabilities() MouseCapabilities {
	return mouseCapabilities()
}

// SupportsButton reports whether the platform supports the provided button.
func (c MouseCapabilities) SupportsButton(btn MouseButton) bool {
	if btn == 0 {
		return false
	}
	if idx, ok := btn.OtherIndex(); ok {
		if idx < 1 {
			return false
		}
		if c.MaxOtherButtons < 0 {
			return true
		}
		return idx <= c.MaxOtherButtons
	}
	for _, b := range c.Buttons {
		if b == btn {
			return true
		}
	}
	return false
}

// SupportsScrollUnit reports whether the platform supports the scroll unit.
func (c MouseCapabilities) SupportsScrollUnit(unit ScrollUnit) bool {
	for _, u := range c.ScrollUnits {
		if u == unit {
			return true
		}
	}
	return false
}
