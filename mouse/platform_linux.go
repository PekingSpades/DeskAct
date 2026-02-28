//go:build linux
// +build linux

package mouse

/*
#include "mouse.h"
*/
import "C"

import "sync"

const scrollPixelsPerLine = 120

var scrollAccum struct {
	mu sync.Mutex
	x  int
	y  int
}

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
		return C.MMMouseButton(7 + idx), nil
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
		return C.int(-delta.X), C.int(delta.Y), C.MM_SCROLL_UNIT_LINE, nil
	case ScrollUnitPixel:
		scrollAccum.mu.Lock()
		defer scrollAccum.mu.Unlock()
		scrollAccum.x += delta.X
		scrollAccum.y += delta.Y
		xTicks := scrollAccum.x / scrollPixelsPerLine
		yTicks := scrollAccum.y / scrollPixelsPerLine
		scrollAccum.x -= xTicks * scrollPixelsPerLine
		scrollAccum.y -= yTicks * scrollPixelsPerLine
		return C.int(-xTicks), C.int(yTicks), C.MM_SCROLL_UNIT_LINE, nil
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
		PixelScrollEmulated: true,
	}
}
