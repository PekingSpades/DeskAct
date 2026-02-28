//go:build windows
// +build windows

package mouse

/*
#include "mouse.h"
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
		switch idx {
		case 1:
			return C.MMMouseButton(C.MM_BUTTON_BACK), nil
		case 2:
			return C.MMMouseButton(C.MM_BUTTON_FORWARD), nil
		default:
			return 0, ErrMouseUnsupportedButton
		}
	}
	return 0, ErrMouseInvalidButton
}

func dragTo(fromX, fromY, toX, toY int, button MouseButton, settings MouseSettings) error {
	if _, err := mouseButtonToC(button); err != nil {
		return wrapMouseError(MouseOpDrag, err, button, 0, 0, "", 0)
	}
	return Move(toX, toY, settings)
}

func dragSmoothTo(fromX, fromY, toX, toY int, button MouseButton, settings MouseSettings) error {
	if _, err := mouseButtonToC(button); err != nil {
		return wrapMouseError(MouseOpDrag, err, button, 0, 0, "", 0)
	}
	return MoveSmooth(toX, toY, settings)
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
		MaxOtherButtons:     2,
		ScrollUnits:         []ScrollUnit{ScrollUnitLine, ScrollUnitPixel},
		PixelScrollEmulated: true,
	}
}
