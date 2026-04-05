//go:build windows
// +build windows

package window

import (
	"github.com/PekingSpades/DeskAct/display"
	"golang.org/x/sys/windows"
	"runtime"
)

func attachDisplayRegions(windows []WindowInfo, displays []*display.Display, options WindowOptions) {
	coordsPhysical := coordsArePhysical(options)
	for i := range windows {
		if !coordsPhysical {
			if physBounds, ok := windowPhysicalBounds(windows[i]); ok {
				windows[i].DisplayRegions = windowRegionsFromPhysical(physBounds, displays)
				continue
			}
		}
		windows[i].DisplayRegions = windowRegions(windows[i].Bounds, displays, coordsPhysical)
	}
}

func windowRegionsFromPhysical(bounds display.Rect, displays []*display.Display) []DisplayRegion {
	regions := make([]DisplayRegion, 0, len(displays))
	for _, d := range displays {
		pi, ok := d.GetPlatformInfo().(*display.WindowsPlatformInfo)
		if !ok {
			continue
		}
		physDisplay := makeRect(pi.PhysicalOrigin.X, pi.PhysicalOrigin.Y, d.Size().W, d.Size().H)
		intersect, ok := intersectRect(bounds, physDisplay)
		if !ok {
			continue
		}
		relPhys := makeRect(
			intersect.X-pi.PhysicalOrigin.X,
			intersect.Y-pi.PhysicalOrigin.Y,
			intersect.W,
			intersect.H,
		)
		if relPhys.W <= 0 || relPhys.H <= 0 {
			continue
		}
		regions = append(regions, DisplayRegion{
			DisplayIndex: d.Index(),
			DisplayID:    d.ID(),
			PhysicalRect: relPhys,
		})
	}
	return regions
}

func windowPhysicalBounds(info WindowInfo) (display.Rect, bool) {
	pi, ok := info.platform.(*WindowsPlatformInfo)
	if !ok {
		return display.Rect{}, false
	}
	hwnd := windows.HWND(uintptr(pi.HWND))
	return physicalWindowRect(hwnd)
}

func physicalWindowRect(hwnd windows.HWND) (display.Rect, bool) {
	// Thread DPI awareness is thread-scoped; lock to keep set/read/restore on the same OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	prev := setThreadDPIAwarenessContext(dpiAwarenessContextPerMonitorAwareV2)
	if prev != 0 {
		defer setThreadDPIAwarenessContext(prev)
	}
	if rect, ok := dwmExtendedFrameBounds(hwnd); ok {
		return rect, true
	}
	return windowRect(hwnd)
}

const dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(3) // -4

var (
	user32DPI                        = windows.NewLazySystemDLL("user32.dll")
	procSetThreadDpiAwarenessContext = user32DPI.NewProc("SetThreadDpiAwarenessContext")
)

func setThreadDPIAwarenessContext(ctx uintptr) uintptr {
	if procSetThreadDpiAwarenessContext.Find() != nil {
		return 0
	}
	prev, _, _ := procSetThreadDpiAwarenessContext.Call(ctx)
	return prev
}
