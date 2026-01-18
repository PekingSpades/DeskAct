//go:build windows
// +build windows

package display

import (
	"syscall"
	"unsafe"
)

// DPI awareness constants.
const (
	DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = ^uintptr(3) // -4
	DPI_AWARENESS_CONTEXT_UNAWARE              = ^uintptr(0) // -1
	PROCESS_PER_MONITOR_DPI_AWARE              = 2

	// Error codes.
	E_ACCESSDENIED = 0x80070005
)

// IsDPIAware returns whether the process is DPI aware.
func IsDPIAware() bool {
	user32 := syscall.NewLazyDLL("user32.dll")
	if getProc := user32.NewProc("GetThreadDpiAwarenessContext"); getProc.Find() == nil {
		ctx, _, _ := getProc.Call()
		if ctx != 0 {
			if eqProc := user32.NewProc("AreDpiAwarenessContextsEqual"); eqProc.Find() == nil {
				isUnaware, _, _ := eqProc.Call(ctx, DPI_AWARENESS_CONTEXT_UNAWARE)
				if isUnaware == 0 {
					return true
				}
				return false
			}
		}
	}

	shcore := syscall.NewLazyDLL("shcore.dll")
	if getProc := shcore.NewProc("GetProcessDpiAwareness"); getProc.Find() == nil {
		var awareness uint32
		ret, _, _ := getProc.Call(0, uintptr(unsafe.Pointer(&awareness)))
		if ret == 0 && awareness >= 1 {
			return true
		}
	}

	return false
}

// InitDPIAwareness sets the process DPI awareness to Per-Monitor V2.
func InitDPIAwareness() bool {
	user32 := syscall.NewLazyDLL("user32.dll")

	// Try SetProcessDpiAwarenessContext first (Windows 10 1703+).
	if proc := user32.NewProc("SetProcessDpiAwarenessContext"); proc.Find() == nil {
		ret, _, _ := proc.Call(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)
		if ret != 0 {
			return true // Success
		}
		// Failed - might be already set or access denied.
		if IsDPIAware() {
			return true
		}
	}

	// Fallback to SetProcessDpiAwareness (Windows 8.1+).
	shcore := syscall.NewLazyDLL("shcore.dll")
	if proc := shcore.NewProc("SetProcessDpiAwareness"); proc.Find() == nil {
		ret, _, _ := proc.Call(PROCESS_PER_MONITOR_DPI_AWARE)
		// S_OK = 0, E_ACCESSDENIED means already set.
		if ret == 0 || ret == E_ACCESSDENIED {
			if IsDPIAware() {
				return true
			}
			if ret == 0 {
				return true
			}
		}
	}

	// Fallback to SetProcessDPIAware (Vista+).
	if proc := user32.NewProc("SetProcessDPIAware"); proc.Find() == nil {
		ret, _, _ := proc.Call()
		if ret != 0 {
			return true
		}
	}

	// All methods failed - process is DPI unaware.
	return false
}
