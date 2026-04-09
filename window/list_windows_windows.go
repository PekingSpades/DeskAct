//go:build windows
// +build windows

package window

import (
	"unsafe"

	"github.com/PekingSpades/DeskAct/display"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const (
	dwmwaExtendedFrameBounds = 9
	dwmwaCloaked             = 14
)

// WindowsPlatformInfo contains Windows-specific metadata.
type WindowsPlatformInfo struct {
	HWND      uint64
	ClassName string
	Style     uint32
	ExStyle   uint32
	Cloaked   bool
}

// Platform returns the platform name.
func (w *WindowsPlatformInfo) Platform() string {
	return "windows"
}

type enumContext struct {
	options WindowOptions
	windows []WindowInfo
}

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	procGetWindowTextW    = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW = user32.NewProc("GetWindowTextLengthW")
)

func listWindows(options WindowOptions) ([]WindowInfo, error) {
	ctx := &enumContext{options: options}
	callback := windows.NewCallback(enumWindowsProc)
	if err := windows.EnumWindows(callback, unsafe.Pointer(ctx)); err != nil {
		return nil, err
	}
	return ctx.windows, nil
}

func enumWindowsProc(hwnd windows.HWND, lparam uintptr) uintptr {
	ctx := (*enumContext)(unsafe.Pointer(lparam))
	info, ok := buildWindowInfo(hwnd, ctx.options)
	if ok {
		ctx.windows = append(ctx.windows, info)
	}
	return 1
}

func buildWindowInfo(hwnd windows.HWND, options WindowOptions) (WindowInfo, bool) {
	if hwnd == 0 {
		return WindowInfo{}, false
	}

	isVisible := windows.IsWindowVisible(hwnd)
	if !options.IncludeInvisible && !isVisible {
		return WindowInfo{}, false
	}

	isMinimized := win.IsIconic(win.HWND(hwnd))
	if !options.IncludeMinimized && isMinimized {
		return WindowInfo{}, false
	}

	exStyle := uint32(win.GetWindowLongPtr(win.HWND(hwnd), win.GWL_EXSTYLE))
	if !options.IncludeToolWindows && (exStyle&win.WS_EX_TOOLWINDOW) != 0 {
		return WindowInfo{}, false
	}

	cloaked := windowCloaked(hwnd)
	if !options.IncludeCloaked && cloaked {
		return WindowInfo{}, false
	}

	bounds, ok := windowBounds(hwnd, options)
	if !ok || bounds.W <= 0 || bounds.H <= 0 {
		return WindowInfo{}, false
	}

	title := windowTitle(hwnd)
	pid := windowPID(hwnd)
	className := windowClassName(hwnd)
	style := uint32(win.GetWindowLongPtr(win.HWND(hwnd), win.GWL_STYLE))

	return WindowInfo{
		ID:          uint64(uintptr(hwnd)),
		PID:         int(pid),
		Title:       title,
		Bounds:      bounds,
		IsVisible:   isVisible && !cloaked,
		IsMinimized: isMinimized,
		platform: &WindowsPlatformInfo{
			HWND:      uint64(uintptr(hwnd)),
			ClassName: className,
			Style:     style,
			ExStyle:   exStyle,
			Cloaked:   cloaked,
		},
	}, true
}

func windowBounds(hwnd windows.HWND, options WindowOptions) (display.Rect, bool) {
	if options.DPIAware {
		if rect, ok := dwmExtendedFrameBounds(hwnd); ok {
			return rect, true
		}
	}
	return windowRect(hwnd)
}

func windowRect(hwnd windows.HWND) (display.Rect, bool) {
	var rect win.RECT
	if !win.GetWindowRect(win.HWND(hwnd), &rect) {
		return display.Rect{}, false
	}
	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	if width <= 0 || height <= 0 {
		return display.Rect{}, false
	}
	return makeRect(int(rect.Left), int(rect.Top), width, height), true
}

func dwmExtendedFrameBounds(hwnd windows.HWND) (display.Rect, bool) {
	var rect win.RECT
	err := windows.DwmGetWindowAttribute(hwnd, dwmwaExtendedFrameBounds, unsafe.Pointer(&rect), uint32(unsafe.Sizeof(rect)))
	if err != nil {
		return display.Rect{}, false
	}
	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	if width <= 0 || height <= 0 {
		return display.Rect{}, false
	}
	return makeRect(int(rect.Left), int(rect.Top), width, height), true
}

func windowCloaked(hwnd windows.HWND) bool {
	var cloaked uint32
	err := windows.DwmGetWindowAttribute(hwnd, dwmwaCloaked, unsafe.Pointer(&cloaked), uint32(unsafe.Sizeof(cloaked)))
	if err != nil {
		return false
	}
	return cloaked != 0
}

func windowPID(hwnd windows.HWND) uint32 {
	var pid uint32
	_, _ = windows.GetWindowThreadProcessId(hwnd, &pid)
	return pid
}

func windowClassName(hwnd windows.HWND) string {
	buf := make([]uint16, 256)
	n, _ := windows.GetClassName(hwnd, &buf[0], int32(len(buf)))
	if n == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

func windowTitle(hwnd windows.HWND) string {
	length, _, _ := procGetWindowTextLenW.Call(uintptr(hwnd))
	if length == 0 {
		return ""
	}
	buf := make([]uint16, int(length)+1)
	_, _, _ = procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf)
}
