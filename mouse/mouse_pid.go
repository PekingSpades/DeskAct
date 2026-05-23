package mouse

/*
#include "mouse_c_pid.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"

	cap "github.com/PekingSpades/DeskAct/capture"
)

// WindowTarget identifies a window for non-preemptive mouse injection.
//
// The fields each platform actually uses:
//
//   - Windows: WindowID is the HWND (uintptr cast).
//   - macOS:   PID is the owning process id; WindowID is informational only.
//   - Linux X11: WindowID is the X11 Window XID.
//
// Callers running cross-platform code should fill both fields when they
// know them — the implementation only reads what its platform needs.
type WindowTarget struct {
	WindowID uint64
	PID      int32
}

// ErrMouseWindowMissing is returned when the platform-relevant identifier on
// WindowTarget is zero.
var ErrMouseWindowMissing = errors.New("mouse window target missing platform identifier")

// guardSession returns capture.ErrUnsupported when the running session
// cannot service per-window mouse injection (Wayland on Linux today).
func guardSession() error {
	if runtime.GOOS == "linux" && isWaylandSession() {
		return fmt.Errorf("%w: wayland session — per-window mouse injection requires X11", cap.ErrUnsupported)
	}
	return nil
}

// MoveWithWindow posts a mouse-move event to a specific window without
// touching the user's real cursor or focus. (x, y) are window-client
// coordinates on Windows and X11; on macOS they are screen coordinates
// (callers translate client to screen using window.Bounds before calling).
func MoveWithWindow(t WindowTarget, x, y int, settings MouseSettings) error {
	if err := guardSession(); err != nil {
		return err
	}
	if !targetReady(t) {
		return wrapMouseError(MouseOpMove, ErrMouseWindowMissing, 0, 0, 0, "", 0)
	}
	cx, cy := translateForCall(t, x, y)
	rc := C.mouseMovePidGo(toCID(t), C.int(cx), C.int(cy))
	MilliSleep(settings.Sleep)
	if int(rc) != 0 {
		return wrapMouseError(MouseOpMove, ErrMouseActionFailed, 0, 0, 0, pidErrorDetail(int(rc)), int(rc))
	}
	return nil
}

// ClickWithWindow synthesizes a single press+release of the supplied button
// on the target window.
func ClickWithWindow(t WindowTarget, x, y int, button MouseButton, settings MouseSettings) error {
	if err := guardSession(); err != nil {
		return err
	}
	if !targetReady(t) {
		return wrapMouseError(MouseOpClick, ErrMouseWindowMissing, button, 0, 1, "", 0)
	}
	cButton, err := mouseButtonToC(button)
	if err != nil {
		return wrapMouseError(MouseOpClick, err, button, 0, 1, "", 0)
	}
	cx, cy := translateForCall(t, x, y)
	rc := C.mouseClickPidGo(toCID(t), C.int(cx), C.int(cy), cButton)
	MilliSleep(settings.Sleep)
	if int(rc) != 0 {
		return wrapMouseError(MouseOpClick, ErrMouseActionFailed, button, 0, 1, pidErrorDetail(int(rc)), int(rc))
	}
	return nil
}

// ToggleWithWindow presses (down=true) or releases (down=false) the button
// on the target window without affecting the user's real input devices.
func ToggleWithWindow(t WindowTarget, x, y int, button MouseButton, down bool, settings MouseSettings) error {
	if err := guardSession(); err != nil {
		return err
	}
	if !targetReady(t) {
		return wrapMouseError(MouseOpToggle, ErrMouseWindowMissing, button, 0, 0, "", 0)
	}
	cButton, err := mouseButtonToC(button)
	if err != nil {
		return wrapMouseError(MouseOpToggle, err, button, 0, 0, "", 0)
	}
	downC := C.int(0)
	if down {
		downC = 1
	}
	cx, cy := translateForCall(t, x, y)
	rc := C.mouseTogglePidGo(toCID(t), C.int(cx), C.int(cy), cButton, downC)
	MilliSleep(settings.Sleep)
	if int(rc) != 0 {
		return wrapMouseError(MouseOpToggle, ErrMouseActionFailed, button, 0, 0, pidErrorDetail(int(rc)), int(rc))
	}
	return nil
}

// ScrollWithWindow posts a scroll event to the target window. dx>0 scrolls
// right; dy>0 scrolls up. The unit is interpreted per-platform; X11 always
// treats values as line ticks regardless of unit.
func ScrollWithWindow(t WindowTarget, x, y, dx, dy int, unit ScrollUnit, settings MouseSettings) error {
	if err := guardSession(); err != nil {
		return err
	}
	if !targetReady(t) {
		return wrapMouseError(MouseOpScroll, ErrMouseWindowMissing, 0, unit, 0, "", 0)
	}
	cUnit := C.MMScrollUnit(C.MM_SCROLL_UNIT_LINE)
	if unit == ScrollUnitPixel {
		cUnit = C.MMScrollUnit(C.MM_SCROLL_UNIT_PIXEL)
	}
	cx, cy := translateForCall(t, x, y)
	// On macOS the scroll wheel event carries a logical location set via
	// CGEventSetLocation (in mouse_c_macos_pid.h). The real system cursor is
	// NOT moved — non-preemption holds.
	rc := C.mouseScrollPidGo(toCID(t), C.int(cx), C.int(cy), C.int(dx), C.int(dy), cUnit)
	MilliSleep(settings.Sleep)
	if int(rc) != 0 {
		return wrapMouseError(MouseOpScroll, ErrMouseActionFailed, 0, unit, 0, pidErrorDetail(int(rc)), int(rc))
	}
	return nil
}

// translateForCall returns the (x, y) that the platform's C entry point
// expects. On macOS the C side wants global screen coordinates, so we map
// the supplied window-client coords through the target window's bounds.
// On Windows/X11 the C side wants client coords; we pass through.
func translateForCall(t WindowTarget, clientX, clientY int) (int, int) {
	if runtime.GOOS != "darwin" {
		return clientX, clientY
	}
	if sx, sy, ok := translateClientToScreen(t.WindowID, clientX, clientY); ok {
		return sx, sy
	}
	return clientX, clientY
}

func targetReady(t WindowTarget) bool {
	switch runtime.GOOS {
	case "darwin":
		return t.PID != 0
	default:
		return t.WindowID != 0
	}
}

// toCID returns the uintptr the platform's C entry points expect:
// HWND/XID on Windows/X11, PID on macOS.
func toCID(t WindowTarget) C.uintptr_t {
	if runtime.GOOS == "darwin" {
		return C.uintptr_t(t.PID)
	}
	return C.uintptr_t(t.WindowID)
}

func pidErrorDetail(code int) string {
	switch code {
	case 0:
		return ""
	case -1:
		return "missing target window/pid"
	case -2:
		return "unsupported mouse button"
	case -3:
		return "post failed"
	case -4:
		return "display unavailable or event creation failed"
	}
	return ""
}
