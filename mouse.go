package deskact

/*
#include "base/os.h"
#include "mouse/mouse_c.h"
*/
import "C"

import (
	"fmt"
	"runtime"
	"syscall"
)

const defaultMouseButton = "left"
const defaultScrollDirection = "down"

func normalizeMouseButton(button string) string {
	if button == "" {
		return defaultMouseButton
	}
	return button
}

func normalizeScrollDirection(direction string) string {
	if direction == "" {
		return defaultScrollDirection
	}
	return direction
}

// CheckMouse check the mouse button.
func CheckMouse(btn string) C.MMMouseButton {
	m1 := map[string]C.MMMouseButton{
		"left":       C.LEFT_BUTTON,
		"center":     C.CENTER_BUTTON,
		"right":      C.RIGHT_BUTTON,
		"wheelDown":  C.WheelDown,
		"wheelUp":    C.WheelUp,
		"wheelLeft":  C.WheelLeft,
		"wheelRight": C.WheelRight,
	}
	if v, ok := m1[btn]; ok {
		return v
	}

	return C.LEFT_BUTTON
}

// MouseButtonString converts a C.MMMouseButton to a readable name.
func MouseButtonString(btn C.MMMouseButton) string {
	m1 := map[C.MMMouseButton]string{
		C.LEFT_BUTTON:   "left",
		C.CENTER_BUTTON: "center",
		C.RIGHT_BUTTON:  "right",
		C.WheelDown:     "wheelDown",
		C.WheelUp:       "wheelUp",
		C.WheelLeft:     "wheelLeft",
		C.WheelRight:    "wheelRight",
	}
	if v, ok := m1[btn]; ok {
		return v
	}

	return fmt.Sprintf("button%d", btn)
}

// Move move the mouse to (x, y) using absolute coordinates.
func Move(x, y int, settings MouseSettings) {
	cx := C.int32_t(x)
	cy := C.int32_t(y)
	C.moveMouse(C.MMPointInt32Make(cx, cy))

	MilliSleep(settings.Sleep)
}

// Drag drag the mouse to (x, y).
// It's not valid now, use the DragSmooth().
func Drag(x, y int, button string, settings MouseSettings) {
	cx := C.int32_t(x)
	cy := C.int32_t(y)

	btn := CheckMouse(normalizeMouseButton(button))
	C.dragMouse(C.MMPointInt32Make(cx, cy), btn)
	MilliSleep(settings.Sleep)
}

// DragSmooth drag the mouse like smooth to (x, y).
func DragSmooth(x, y int, settings MouseSettings) {
	Toggle(defaultMouseButton, true, false, settings)
	MilliSleep(50)
	MoveSmooth(x, y, settings)
	Toggle(defaultMouseButton, false, false, settings)
}

// MoveSmooth move the mouse smooth.
func MoveSmooth(x, y int, settings MouseSettings) bool {
	cx := C.int32_t(x)
	cy := C.int32_t(y)

	low := C.double(settings.MoveSmoothLow)
	high := C.double(settings.MoveSmoothHigh)

	cbool := C.smoothlyMoveMouse(C.MMPointInt32Make(cx, cy), low, high)
	MilliSleep(settings.Sleep + settings.MoveSmoothDelay)

	return bool(cbool)
}

// MoveArgs get the mouse relative args.
func MoveArgs(x, y int) (int, int) {
	mx, my := Location()
	mx = mx + x
	my = my + y

	return mx, my
}

// MoveRelative move mouse with relative.
func MoveRelative(x, y int, settings MouseSettings) {
	mx, my := MoveArgs(x, y)
	Move(mx, my, settings)
}

// MoveSmoothRelative move mouse smooth with relative.
func MoveSmoothRelative(x, y int, settings MouseSettings) {
	mx, my := MoveArgs(x, y)
	MoveSmooth(mx, my, settings)
}

// Location get the mouse location position return x, y.
func Location() (int, int) {
	pos := C.location()
	return int(pos.x), int(pos.y)
}

// Click clicks the mouse button and returns error.
func Click(button string, double bool, settings MouseSettings) error {
	btn := CheckMouse(normalizeMouseButton(button))

	defer MilliSleep(settings.Sleep)

	clickCount := 1
	if double {
		clickCount = 2
	}

	if code := C.multiClickErr(btn, C.int(clickCount)); code != 0 {
		return formatClickError(int(code), btn, clickCount)
	}

	return nil
}

func formatClickError(code int, button C.MMMouseButton, clickCount int) error {
	btnName := MouseButtonString(button)
	detail := ""

	switch runtime.GOOS {
	case "windows":
		if code != 0 {
			detail = syscall.Errno(code).Error()
		}
	case "darwin":
		cgErrors := map[int]string{
			0:    "kCGErrorSuccess",
			1000: "kCGErrorFailure",
			1001: "kCGErrorIllegalArgument",
			1002: "kCGErrorInvalidConnection",
			1003: "kCGErrorInvalidContext",
			1004: "kCGErrorCannotComplete",
			1005: "kCGErrorNotImplemented",
			1006: "kCGErrorRangeCheck",
			1007: "kCGErrorTypeCheck",
			1008: "kCGErrorNoCurrentPoint",
			1010: "kCGErrorInvalidOperation",
		}
		if v, ok := cgErrors[code]; ok {
			detail = v
		}
	default:
		if code == 1 {
			detail = "XTestFakeButtonEvent returned false"
		}
	}

	if detail != "" {
		return fmt.Errorf("click failed (%s, count=%d): %s (code=%d)", btnName, clickCount, detail, code)
	}

	return fmt.Errorf("click failed (%s, count=%d), code=%d", btnName, clickCount, code)
}

// MultiClick performs multiple clicks and returns error.
func MultiClick(button string, clickCount int, settings MouseSettings) error {
	if clickCount < 1 {
		return nil
	}

	btn := CheckMouse(normalizeMouseButton(button))
	defer MilliSleep(settings.Sleep)

	if code := C.multiClickErr(btn, C.int(clickCount)); code != 0 {
		return formatClickError(int(code), btn, clickCount)
	}

	return nil
}

// MoveClick move and click the mouse.
func MoveClick(x, y int, button string, double bool, settings MouseSettings) error {
	Move(x, y, settings)
	MilliSleep(50)
	return Click(button, double, settings)
}

// MovesClick move smooth and click the mouse.
func MovesClick(x, y int, button string, double bool, settings MouseSettings) error {
	MoveSmooth(x, y, settings)
	MilliSleep(50)
	return Click(button, double, settings)
}

// Toggle toggle the mouse.
func Toggle(button string, down bool, sleepAfter bool, settings MouseSettings) error {
	btn := CheckMouse(normalizeMouseButton(button))
	C.toggleMouseErr(C.bool(down), btn)
	if sleepAfter {
		MilliSleep(settings.Sleep)
	}

	return nil
}

// Scroll scroll the mouse to (x, y).
func Scroll(x, y int, settings MouseSettings) {
	cx := C.int(x)
	cy := C.int(y)

	C.scrollMouseXY(cx, cy)
	MilliSleep(settings.Sleep + settings.ScrollDelay)
}

// ScrollDir scroll the mouse with direction.
func ScrollDir(x int, direction string, settings MouseSettings) {
	d := normalizeScrollDirection(direction)

	if d == "down" {
		Scroll(0, -x, settings)
	}
	if d == "up" {
		Scroll(0, x, settings)
	}

	if d == "left" {
		Scroll(x, 0, settings)
	}
	if d == "right" {
		Scroll(-x, 0, settings)
	}
}

// ScrollSmooth scroll the mouse smooth.
func ScrollSmooth(to int, settings MouseSettings) {
	i := 0
	num := settings.ScrollSmoothCount
	tm := settings.ScrollSmoothInterval
	tox := settings.ScrollSmoothX

	for {
		Scroll(tox, to, settings)
		MilliSleep(tm)
		i++
		if i == num {
			break
		}
	}
	MilliSleep(settings.Sleep)
}

// ScrollRelative scroll mouse with relative.
func ScrollRelative(x, y int, settings MouseSettings) {
	mx, my := MoveArgs(x, y)
	Scroll(mx, my, settings)
}
