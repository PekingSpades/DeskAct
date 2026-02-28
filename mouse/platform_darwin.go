//go:build darwin
// +build darwin

package mouse

/*
#include "mouse.h"

void dragMouse(MMPointInt32 point, const MMMouseButton button);
bool smoothlyDragMouse(MMPointInt32 startPoint, MMPointInt32 endPoint, const MMMouseButton button, double lowSpeed, double highSpeed);
*/
import "C"

func mouseButtonToC(button MouseButton) (C.MMMouseButton, error) {
	switch button {
	case MouseButtonLeft:
		return C.MMMouseButton(C.MM_BUTTON_LEFT), nil
	case MouseButtonRight:
		return C.MMMouseButton(C.MM_BUTTON_RIGHT), nil
	case MouseButtonMiddle:
		return C.MMMouseButton(C.MM_BUTTON_MIDDLE), nil
	case MouseButtonBack:
		return C.MMMouseButton(C.MM_BUTTON_BACK), nil
	case MouseButtonForward:
		return C.MMMouseButton(C.MM_BUTTON_FORWARD), nil
	}
	if idx, ok := button.OtherIndex(); ok {
		if idx < 1 {
			return 0, ErrMouseInvalidButton
		}
		return C.MMMouseButton(2 + idx), nil
	}
	return 0, ErrMouseInvalidButton
}

func dragTo(fromX, fromY, toX, toY int, button MouseButton, settings MouseSettings) error {
	cbtn, err := mouseButtonToC(button)
	if err != nil {
		return wrapMouseError(MouseOpDrag, err, button, 0, 0, "", 0)
	}
	cx := C.int32_t(toX)
	cy := C.int32_t(toY)
	C.dragMouse(C.MMPointInt32Make(cx, cy), cbtn)
	MilliSleep(settings.Sleep)
	return nil
}

func dragSmoothTo(fromX, fromY, toX, toY int, button MouseButton, settings MouseSettings) error {
	cbtn, err := mouseButtonToC(button)
	if err != nil {
		return wrapMouseError(MouseOpDrag, err, button, 0, 0, "", 0)
	}
	cx := C.int32_t(toX)
	cy := C.int32_t(toY)
	startPt := C.MMPointInt32Make(C.int32_t(fromX), C.int32_t(fromY))
	low := C.double(settings.MoveSmoothLow)
	high := C.double(settings.MoveSmoothHigh)

	cbool := C.smoothlyDragMouse(startPt, C.MMPointInt32Make(cx, cy), cbtn, low, high)
	MilliSleep(settings.Sleep + settings.MoveSmoothDelay)
	if !bool(cbool) {
		return wrapMouseError(MouseOpDrag, ErrMouseActionFailed, button, 0, 0, "smooth drag returned false", 0)
	}
	return nil
}

func scrollDeltaToC(delta ScrollDelta) (C.int, C.int, C.MMScrollUnit, error) {
	switch delta.Unit {
	case ScrollUnitLine:
		return C.int(delta.X), C.int(delta.Y), C.MM_SCROLL_UNIT_LINE, nil
	case ScrollUnitPixel:
		return C.int(delta.X), C.int(delta.Y), C.MM_SCROLL_UNIT_PIXEL, nil
	default:
		return 0, 0, 0, ErrMouseInvalidScrollUnit
	}
}

func mouseCapabilities() MouseCapabilities {
	return MouseCapabilities{
		Buttons: []MouseButton{
			MouseButtonLeft,
			MouseButtonRight,
			MouseButtonMiddle,
			MouseButtonBack,
			MouseButtonForward,
		},
		MaxOtherButtons:     -1,
		ScrollUnits:         []ScrollUnit{ScrollUnitLine, ScrollUnitPixel},
		PixelScrollEmulated: false,
	}
}
