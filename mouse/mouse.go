package mouse

/*
#include "mouse_c.h"
*/
import "C"

// Move moves the mouse to (x, y) using absolute coordinates.
func Move(x, y int, settings MouseSettings) error {
	cx := C.int32_t(x)
	cy := C.int32_t(y)
	C.moveMouse(C.MMPointInt32Make(cx, cy))

	MilliSleep(settings.Sleep)
	return nil
}

// Drag drags the mouse to (x, y) from the current position.
func Drag(x, y int, button MouseButton, settings MouseSettings) error {
	if err := Toggle(button, true, false, settings); err != nil {
		return err
	}
	MilliSleep(50)
	if err := Move(x, y, settings); err != nil {
		_ = Toggle(button, false, false, settings)
		return err
	}
	return Toggle(button, false, false, settings)
}

// DragSmooth drags the mouse smoothly to (x, y) from the current position.
func DragSmooth(x, y int, button MouseButton, settings MouseSettings) error {
	if err := Toggle(button, true, false, settings); err != nil {
		return err
	}
	MilliSleep(50)
	if err := MoveSmooth(x, y, settings); err != nil {
		_ = Toggle(button, false, false, settings)
		return err
	}
	return Toggle(button, false, false, settings)
}

// MoveSmooth smoothly moves the mouse to (x, y).
func MoveSmooth(x, y int, settings MouseSettings) error {
	cx := C.int32_t(x)
	cy := C.int32_t(y)

	low := C.double(settings.MoveSmoothLow)
	high := C.double(settings.MoveSmoothHigh)

	cbool := C.smoothlyMoveMouse(C.MMPointInt32Make(cx, cy), low, high)
	MilliSleep(settings.Sleep + settings.MoveSmoothDelay)
	if !bool(cbool) {
		return wrapMouseError(MouseOpMoveSmooth, ErrMouseActionFailed, 0, 0, 0, "smooth move returned false", 0)
	}
	return nil
}

// MoveArgs get the mouse relative args.
func MoveArgs(x, y int) (int, int) {
	mx, my := Location()
	mx = mx + x
	my = my + y

	return mx, my
}

// MoveRelative moves the mouse with relative coordinates.
func MoveRelative(x, y int, settings MouseSettings) error {
	mx, my := MoveArgs(x, y)
	return Move(mx, my, settings)
}

// MoveSmoothRelative moves the mouse smoothly with relative coordinates.
func MoveSmoothRelative(x, y int, settings MouseSettings) error {
	mx, my := MoveArgs(x, y)
	return MoveSmooth(mx, my, settings)
}

// Location returns the mouse location position.
func Location() (int, int) {
	pos := C.location()
	return int(pos.x), int(pos.y)
}

// Click clicks the mouse button and returns error.
func Click(button MouseButton, double bool, settings MouseSettings) error {
	defer MilliSleep(settings.Sleep)

	clickCount := 1
	if double {
		clickCount = 2
	}

	cbtn, err := mouseButtonToC(button)
	if err != nil {
		return wrapMouseError(MouseOpClick, err, button, 0, clickCount, "", 0)
	}

	if code := C.multiClickErr(cbtn, C.int(clickCount)); code != 0 {
		return mouseActionError(MouseOpClick, button, clickCount, int(code))
	}

	return nil
}

// MultiClick performs multiple clicks and returns error.
func MultiClick(button MouseButton, clickCount int, settings MouseSettings) error {
	if clickCount < 1 {
		return nil
	}

	defer MilliSleep(settings.Sleep)

	cbtn, err := mouseButtonToC(button)
	if err != nil {
		return wrapMouseError(MouseOpMultiClick, err, button, 0, clickCount, "", 0)
	}

	if code := C.multiClickErr(cbtn, C.int(clickCount)); code != 0 {
		return mouseActionError(MouseOpMultiClick, button, clickCount, int(code))
	}

	return nil
}

// MoveClick moves and clicks the mouse.
func MoveClick(x, y int, button MouseButton, double bool, settings MouseSettings) error {
	if err := Move(x, y, settings); err != nil {
		return err
	}
	MilliSleep(50)
	return Click(button, double, settings)
}

// MovesClick moves smoothly and clicks the mouse.
func MovesClick(x, y int, button MouseButton, double bool, settings MouseSettings) error {
	if err := MoveSmooth(x, y, settings); err != nil {
		return err
	}
	MilliSleep(50)
	return Click(button, double, settings)
}

// Toggle toggles a mouse button up or down.
func Toggle(button MouseButton, down bool, sleepAfter bool, settings MouseSettings) error {
	cbtn, err := mouseButtonToC(button)
	if err != nil {
		return wrapMouseError(MouseOpToggle, err, button, 0, 0, "", 0)
	}
	if code := C.toggleMouseErr(C.bool(down), cbtn); code != 0 {
		return mouseActionError(MouseOpToggle, button, 0, int(code))
	}
	if sleepAfter {
		MilliSleep(settings.Sleep)
	}

	return nil
}

// Scroll scrolls the mouse by a delta.
func Scroll(delta ScrollDelta, settings MouseSettings) error {
	caps := CurrentMouseCapabilities()
	if !caps.SupportsScrollUnit(delta.Unit) {
		return wrapMouseError(MouseOpScroll, ErrMouseUnsupportedScrollUnit, 0, delta.Unit, 0, "", 0)
	}
	cx, cy, unit, err := scrollDeltaToC(delta)
	if err != nil {
		return wrapMouseError(MouseOpScroll, err, 0, delta.Unit, 0, "", 0)
	}
	if cx != 0 || cy != 0 {
		C.scrollMouseXY(cx, cy, unit)
	}
	MilliSleep(settings.Sleep + settings.ScrollDelay)
	return nil
}

// ScrollLines scrolls using line-based deltas.
func ScrollLines(x, y int, settings MouseSettings) error {
	return Scroll(ScrollDeltaLines(x, y), settings)
}

// ScrollPixels scrolls using pixel-based deltas (best-effort on some platforms).
func ScrollPixels(x, y int, settings MouseSettings) error {
	return Scroll(ScrollDeltaPixels(x, y), settings)
}

// ScrollSmooth scrolls smoothly using repeated deltas.
func ScrollSmooth(delta ScrollDelta, settings MouseSettings) error {
	if settings.ScrollSmoothCount <= 0 {
		return nil
	}
	for i := 0; i < settings.ScrollSmoothCount; i++ {
		if err := Scroll(delta, settings); err != nil {
			return err
		}
		MilliSleep(settings.ScrollSmoothInterval)
	}
	MilliSleep(settings.Sleep)
	return nil
}
