//go:build cgo && darwin
// +build cgo,darwin

package mouse

/*
#cgo darwin LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdint.h>

// Resolve a CGWindowID to its (x, y, w, h) screen bounds via
// CGWindowListCreateDescriptionFromArray. Returns 0 on success, -1 if the
// window does not exist or its bounds are unparseable.
static int mouse_window_bounds(uint32_t winID, int *x, int *y, int *w, int *h) {
    CFNumberRef num = CFNumberCreate(NULL, kCFNumberSInt32Type, &winID);
    const void *vals[] = { num };
    CFArrayRef ids = CFArrayCreate(NULL, vals, 1, &kCFTypeArrayCallBacks);
    CFArrayRef list = CGWindowListCreateDescriptionFromArray(ids);
    CFRelease(num);
    CFRelease(ids);
    if (!list || CFArrayGetCount(list) == 0) {
        if (list) CFRelease(list);
        return -1;
    }
    CFDictionaryRef d = (CFDictionaryRef)CFArrayGetValueAtIndex(list, 0);
    CFDictionaryRef bd = (CFDictionaryRef)CFDictionaryGetValue(d, kCGWindowBounds);
    CGRect r;
    int ok = bd && CGRectMakeWithDictionaryRepresentation(bd, &r);
    CFRelease(list);
    if (!ok) return -1;
    *x = (int)r.origin.x;
    *y = (int)r.origin.y;
    *w = (int)r.size.width;
    *h = (int)r.size.height;
    return 0;
}
*/
import "C"

func translateClientToScreen(windowID uint64, clientX, clientY int) (int, int, bool) {
	if windowID == 0 {
		return clientX, clientY, false
	}
	var x, y, w, h C.int
	if C.mouse_window_bounds(C.uint32_t(windowID), &x, &y, &w, &h) != 0 {
		return clientX, clientY, false
	}
	return int(x) + clientX, int(y) + clientY, true
}
