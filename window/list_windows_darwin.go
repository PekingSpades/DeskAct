//go:build darwin
// +build darwin

package window

/*
#cgo darwin LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdbool.h>
#include <stdlib.h>

typedef struct {
	uint32_t windowID;
	int32_t ownerPID;
	int32_t layer;
	int32_t isOnscreen;
	double  alpha;
	int32_t x;
	int32_t y;
	int32_t w;
	int32_t h;
	char*   title;
	char*   ownerName;
} WindowInfoC;

typedef struct {
	WindowInfoC* items;
	int32_t count;
} WindowListC;

static char* CopyCFString(CFStringRef str) {
	if (!str) {
		return NULL;
	}
	CFIndex length = CFStringGetLength(str);
	CFIndex maxSize = CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8) + 1;
	char* buffer = (char*)calloc((size_t)maxSize, sizeof(char));
	if (!buffer) {
		return NULL;
	}
	if (CFStringGetCString(str, buffer, maxSize, kCFStringEncodingUTF8)) {
		return buffer;
	}
	free(buffer);
	return NULL;
}

static int32_t GetInt32(CFDictionaryRef dict, CFStringRef key, int32_t defValue) {
	CFNumberRef num = (CFNumberRef)CFDictionaryGetValue(dict, key);
	if (!num) {
		return defValue;
	}
	int32_t value = defValue;
	if (CFNumberGetValue(num, kCFNumberSInt32Type, &value)) {
		return value;
	}
	return defValue;
}

static double GetDouble(CFDictionaryRef dict, CFStringRef key, double defValue) {
	CFNumberRef num = (CFNumberRef)CFDictionaryGetValue(dict, key);
	if (!num) {
		return defValue;
	}
	double value = defValue;
	if (CFNumberGetValue(num, kCFNumberDoubleType, &value)) {
		return value;
	}
	return defValue;
}

static int32_t GetOnscreen(CFDictionaryRef dict) {
	CFBooleanRef onScreen = (CFBooleanRef)CFDictionaryGetValue(dict, kCGWindowIsOnscreen);
	if (!onScreen) {
		return 0;
	}
	return CFBooleanGetValue(onScreen) ? 1 : 0;
}

static WindowListC getWindowList(bool includeOffscreen, bool includeAllLayers) {
	WindowListC list = {0};
	CGWindowListOption opts = kCGWindowListExcludeDesktopElements;
	if (includeOffscreen) {
		opts |= kCGWindowListOptionAll;
	} else {
		opts |= kCGWindowListOptionOnScreenOnly;
	}
	CFArrayRef infoList = CGWindowListCopyWindowInfo(opts, kCGNullWindowID);
	if (!infoList) {
		return list;
	}

	CFIndex count = CFArrayGetCount(infoList);
	if (count <= 0) {
		CFRelease(infoList);
		return list;
	}

	WindowInfoC* items = (WindowInfoC*)calloc((size_t)count, sizeof(WindowInfoC));
	if (!items) {
		CFRelease(infoList);
		return list;
	}

	int32_t outCount = 0;
	for (CFIndex i = 0; i < count; i++) {
		CFDictionaryRef dict = (CFDictionaryRef)CFArrayGetValueAtIndex(infoList, i);
		if (!dict) {
			continue;
		}

		int32_t layer = GetInt32(dict, kCGWindowLayer, 0);
		if (!includeAllLayers && layer != 0) {
			continue;
		}

		int32_t onScreen = GetOnscreen(dict);
		if (!includeOffscreen && !onScreen) {
			continue;
		}

		CFDictionaryRef boundsDict = (CFDictionaryRef)CFDictionaryGetValue(dict, kCGWindowBounds);
		if (!boundsDict) {
			continue;
		}
		CGRect bounds;
		if (!CGRectMakeWithDictionaryRepresentation(boundsDict, &bounds)) {
			continue;
		}
		if (bounds.size.width <= 0 || bounds.size.height <= 0) {
			continue;
		}

		WindowInfoC* item = &items[outCount++];
		item->windowID = (uint32_t)GetInt32(dict, kCGWindowNumber, 0);
		item->ownerPID = GetInt32(dict, kCGWindowOwnerPID, 0);
		item->layer = layer;
		item->isOnscreen = onScreen;
		item->alpha = GetDouble(dict, kCGWindowAlpha, 1.0);
		item->x = (int32_t)bounds.origin.x;
		item->y = (int32_t)bounds.origin.y;
		item->w = (int32_t)bounds.size.width;
		item->h = (int32_t)bounds.size.height;
		item->title = CopyCFString((CFStringRef)CFDictionaryGetValue(dict, kCGWindowName));
		item->ownerName = CopyCFString((CFStringRef)CFDictionaryGetValue(dict, kCGWindowOwnerName));
	}

	CFRelease(infoList);

	list.items = items;
	list.count = outCount;
	return list;
}

static void freeWindowList(WindowListC list) {
	if (!list.items) {
		return;
	}
	for (int32_t i = 0; i < list.count; i++) {
		free(list.items[i].title);
		free(list.items[i].ownerName);
	}
	free(list.items);
}
*/
import "C"

import "unsafe"

// DarwinPlatformInfo contains macOS-specific metadata.
type DarwinPlatformInfo struct {
	WindowID  uint32
	OwnerName string
	Layer     int32
	Alpha     float64
	Onscreen  bool
}

// Platform returns the platform name.
func (d *DarwinPlatformInfo) Platform() string {
	return "darwin"
}

func listWindows(options WindowOptions) ([]WindowInfo, error) {
	includeOffscreen := options.IncludeOffscreen || options.IncludeMinimized
	list := C.getWindowList(C.bool(includeOffscreen), C.bool(options.IncludeAllLayers))
	defer C.freeWindowList(list)

	if list.count == 0 || list.items == nil {
		return nil, nil
	}

	count := int(list.count)
	items := unsafe.Slice(list.items, count)
	windows := make([]WindowInfo, 0, count)

	for i := 0; i < count; i++ {
		item := items[i]
		bounds := makeRect(int(item.x), int(item.y), int(item.w), int(item.h))
		if bounds.W <= 0 || bounds.H <= 0 {
			continue
		}

		title := C.GoString(item.title)
		ownerName := C.GoString(item.ownerName)
		onScreen := item.isOnscreen != 0

		windows = append(windows, WindowInfo{
			ID:          uint64(item.windowID),
			PID:         int(item.ownerPID),
			Title:       title,
			Bounds:      bounds,
			IsVisible:   onScreen,
			IsMinimized: false,
			platform: &DarwinPlatformInfo{
				WindowID:  uint32(item.windowID),
				OwnerName: ownerName,
				Layer:     item.layer,
				Alpha:     float64(item.alpha),
				Onscreen:  onScreen,
			},
		})
	}

	return windows, nil
}
