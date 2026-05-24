//go:build windows
// +build windows

package window

import (
	"fmt"
	"unsafe"

	"github.com/PekingSpades/DeskAct/capture"
	"github.com/PekingSpades/DeskAct/display"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const (
	swpNoZorder     = 0x0004
	swpNoActivate   = 0x0010
	swpShowWindow   = 0x0040
	swpFrameChanged = 0x0020

	swShowNoActivate = 4
	swMinimize       = 6
	swRestore        = 9
	swShow           = 5

	hwndTop = 0
)

var (
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procShowWindow          = user32.NewProc("ShowWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procIsWindow            = user32.NewProc("IsWindow")
	procBringWindowToTop    = user32.NewProc("BringWindowToTop")
)

func ensureHwnd(id uint64) (windows.HWND, error) {
	hwnd := windows.HWND(uintptr(id))
	rc, _, _ := procIsWindow.Call(uintptr(hwnd))
	if rc == 0 {
		return 0, fmt.Errorf("%w: hwnd=0x%x", capture.ErrWindowNotFound, id)
	}
	return hwnd, nil
}

func moveWindow(id uint64, pid int32, x, y int) error {
	hwnd, err := ensureHwnd(id)
	if err != nil {
		return err
	}
	if windowHasPlacementState(hwnd) {
		rect, err := restoredPlacementBounds(hwnd)
		if err != nil {
			return err
		}
		return setRestoredPlacementBounds(hwnd, x, y, rect.W, rect.H)
	}
	rect, ok := windowRect(hwnd)
	if !ok {
		return fmt.Errorf("%w: bounds unavailable for hwnd=0x%x", capture.ErrCaptureFailed, id)
	}
	return setWindowPos(hwnd, x, y, rect.W, rect.H, swpNoZorder|swpNoActivate)
}

func resizeWindow(id uint64, pid int32, w, h int) error {
	hwnd, err := ensureHwnd(id)
	if err != nil {
		return err
	}
	if windowHasPlacementState(hwnd) {
		rect, err := restoredPlacementBounds(hwnd)
		if err != nil {
			return err
		}
		return setRestoredPlacementBounds(hwnd, rect.X, rect.Y, w, h)
	}
	rect, ok := windowRect(hwnd)
	if !ok {
		return fmt.Errorf("%w: bounds unavailable for hwnd=0x%x", capture.ErrCaptureFailed, id)
	}
	return setWindowPos(hwnd, rect.X, rect.Y, w, h, swpNoZorder|swpNoActivate)
}

func moveResizeWindow(id uint64, pid int32, x, y, w, h int) error {
	hwnd, err := ensureHwnd(id)
	if err != nil {
		return err
	}
	if windowHasPlacementState(hwnd) {
		return setRestoredPlacementBounds(hwnd, x, y, w, h)
	}
	return setWindowPos(hwnd, x, y, w, h, swpNoZorder|swpNoActivate)
}

func raiseWindow(id uint64, pid int32) error {
	hwnd, err := ensureHwnd(id)
	if err != nil {
		return err
	}
	if rc, _, e := procBringWindowToTop.Call(uintptr(hwnd)); rc == 0 {
		return fmt.Errorf("BringWindowToTop failed: %v", e)
	}
	return nil
}

func focusWindow(id uint64, pid int32) error {
	hwnd, err := ensureHwnd(id)
	if err != nil {
		return err
	}
	if rc, _, e := procSetForegroundWindow.Call(uintptr(hwnd)); rc == 0 {
		return fmt.Errorf("SetForegroundWindow failed: %v", e)
	}
	return nil
}

func minimizeWindow(id uint64, pid int32) error {
	hwnd, err := ensureHwnd(id)
	if err != nil {
		return err
	}
	procShowWindow.Call(uintptr(hwnd), swMinimize)
	return nil
}

func restoreWindow(id uint64, pid int32) error {
	hwnd, err := ensureHwnd(id)
	if err != nil {
		return err
	}
	procShowWindow.Call(uintptr(hwnd), swRestore)
	return nil
}

func closeWindow(id uint64, pid int32) error {
	hwnd, err := ensureHwnd(id)
	if err != nil {
		return err
	}
	rc, _, e := procPostMessageW.Call(uintptr(hwnd), uintptr(win.WM_CLOSE), 0, 0)
	if rc == 0 {
		return fmt.Errorf("PostMessage WM_CLOSE failed: %v", e)
	}
	return nil
}

func boundsWindow(id uint64, pid int32) (display.Rect, error) {
	hwnd, err := ensureHwnd(id)
	if err != nil {
		return display.Rect{}, err
	}
	if windowHasPlacementState(hwnd) {
		return restoredPlacementBounds(hwnd)
	}
	rect, ok := windowRect(hwnd)
	if !ok {
		return display.Rect{}, fmt.Errorf("%w: bounds unavailable for hwnd=0x%x", capture.ErrCaptureFailed, id)
	}
	return rect, nil
}

func windowHasPlacementState(hwnd windows.HWND) bool {
	wh := win.HWND(hwnd)
	return win.IsIconic(wh) || win.IsZoomed(wh)
}

func restoredPlacementBounds(hwnd windows.HWND) (display.Rect, error) {
	wp, err := getWindowPlacement(hwnd)
	if err != nil {
		return display.Rect{}, err
	}
	rect := wp.RcNormalPosition
	w := int(rect.Right - rect.Left)
	h := int(rect.Bottom - rect.Top)
	if w <= 0 || h <= 0 {
		return display.Rect{}, fmt.Errorf("%w: restored bounds unavailable for hwnd=0x%x", capture.ErrCaptureFailed, uintptr(hwnd))
	}
	return makeRect(int(rect.Left), int(rect.Top), w, h), nil
}

func setRestoredPlacementBounds(hwnd windows.HWND, x, y, w, h int) error {
	if w <= 0 || h <= 0 {
		return fmt.Errorf("%w: invalid restored bounds %dx%d for hwnd=0x%x", capture.ErrCaptureFailed, w, h, uintptr(hwnd))
	}
	wp, err := getWindowPlacement(hwnd)
	if err != nil {
		return err
	}
	wp.RcNormalPosition.Left = int32(x)
	wp.RcNormalPosition.Top = int32(y)
	wp.RcNormalPosition.Right = int32(x + w)
	wp.RcNormalPosition.Bottom = int32(y + h)
	if !win.SetWindowPlacement(win.HWND(hwnd), &wp) {
		return fmt.Errorf("SetWindowPlacement failed for hwnd=0x%x", uintptr(hwnd))
	}
	return nil
}

func getWindowPlacement(hwnd windows.HWND) (win.WINDOWPLACEMENT, error) {
	var wp win.WINDOWPLACEMENT
	wp.Length = uint32(unsafe.Sizeof(wp))
	if !win.GetWindowPlacement(win.HWND(hwnd), &wp) {
		return wp, fmt.Errorf("%w: GetWindowPlacement failed for hwnd=0x%x", capture.ErrCaptureFailed, uintptr(hwnd))
	}
	return wp, nil
}

func setWindowPos(hwnd windows.HWND, x, y, w, h int, flags uintptr) error {
	rc, _, e := procSetWindowPos.Call(
		uintptr(hwnd),
		uintptr(hwndTop),
		uintptr(int32(x)),
		uintptr(int32(y)),
		uintptr(int32(w)),
		uintptr(int32(h)),
		flags,
	)
	if rc == 0 {
		return fmt.Errorf("SetWindowPos failed: %v", e)
	}
	return nil
}
