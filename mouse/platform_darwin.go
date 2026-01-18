//go:build darwin
// +build darwin

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
		if idx < 1 {
			return 0, ErrMouseInvalidButton
		}
		return C.MMMouseButton(2 + idx), nil
	}
	return 0, ErrMouseInvalidButton
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
