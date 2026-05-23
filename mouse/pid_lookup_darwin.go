//go:build darwin

package mouse

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdint.h>

// front_window_for_pid returns the CGWindowID of the on-screen window
// owned by `pid` that appears highest in the window-server's z-order
// (excluding desktop/menu-bar level windows). Returns 0 when the PID
// owns no addressable window. We use CGWindowListCopyWindowInfo with
// kCGWindowListOptionOnScreenOnly + kCGWindowListExcludeDesktopElements
// so we skip the persistent menu-bar / desktop-picture windows.
static uint32_t front_window_for_pid(int pid) {
    CFArrayRef arr = CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID);
    if (arr == NULL) return 0;
    uint32_t result = 0;
    CFIndex n = CFArrayGetCount(arr);
    for (CFIndex i = 0; i < n; i++) {
        CFDictionaryRef d = (CFDictionaryRef)CFArrayGetValueAtIndex(arr, i);
        CFNumberRef pidNum = (CFNumberRef)CFDictionaryGetValue(d, kCGWindowOwnerPID);
        if (pidNum == NULL) continue;
        int owner = 0;
        CFNumberGetValue(pidNum, kCFNumberIntType, &owner);
        if (owner != pid) continue;
        CFNumberRef widNum = (CFNumberRef)CFDictionaryGetValue(d, kCGWindowNumber);
        if (widNum == NULL) continue;
        uint32_t wid = 0;
        CFNumberGetValue(widNum, kCFNumberSInt32Type, &wid);
        // CGWindowList returns front-to-back; first match wins.
        result = wid;
        break;
    }
    CFRelease(arr);
    return result;
}
*/
import "C"

func frontWindowIDByPIDPlatform(pid int) uint64 {
	if pid <= 0 {
		return 0
	}
	return uint64(C.front_window_for_pid(C.int(pid)))
}

// hwndByPIDPlatform / xidByPIDPlatform are Windows / Linux only.
func hwndByPIDPlatform(pid int) uint64 { return 0 }
func xidByPIDPlatform(pid int) uint64  { return 0 }
