//go:build darwin && cgo

package apps

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Cocoa -framework CoreFoundation -framework CoreGraphics
#include <Cocoa/Cocoa.h>
#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <stdlib.h>
#include <string.h>

static char *resolve_alias_path(const char *path) {
	if (!path) {
		return NULL;
	}
	CFURLRef url = CFURLCreateFromFileSystemRepresentation(kCFAllocatorDefault, (const UInt8 *)path, (CFIndex)strlen(path), false);
	if (!url) {
		return NULL;
	}
	Boolean wasAliased = false;
	CFErrorRef error = NULL;
	CFURLRef resolved = CFURLCreateByResolvingAliasFile(kCFAllocatorDefault, url, 0, &wasAliased, &error);
	CFRelease(url);
	if (!resolved) {
		if (error) {
			CFRelease(error);
		}
		return NULL;
	}
	if (!wasAliased) {
		CFRelease(resolved);
		return NULL;
	}
	CFStringRef cfPath = CFURLCopyFileSystemPath(resolved, kCFURLPOSIXPathStyle);
	CFRelease(resolved);
	if (!cfPath) {
		return NULL;
	}
	CFIndex maxSize = CFStringGetMaximumSizeForEncoding(CFStringGetLength(cfPath), kCFStringEncodingUTF8) + 1;
	char *buffer = (char *)malloc(maxSize);
	if (!buffer) {
		CFRelease(cfPath);
		return NULL;
	}
	if (!CFStringGetCString(cfPath, buffer, maxSize, kCFStringEncodingUTF8)) {
		free(buffer);
		buffer = NULL;
	}
	CFRelease(cfPath);
	return buffer;
}

static unsigned char *icon_rgba_for_path(const char *path, int *width, int *height) {
	@autoreleasepool {
		if (!path) {
			return NULL;
		}
		NSString *nsPath = [NSString stringWithUTF8String:path];
		if (!nsPath) {
			return NULL;
		}
		NSImage *image = [[NSWorkspace sharedWorkspace] iconForFile:nsPath];
		if (!image) {
			return NULL;
		}

		NSInteger bestWidth = 0;
		NSInteger bestHeight = 0;
		for (NSImageRep *rep in [image representations]) {
			NSInteger w = [rep pixelsWide];
			NSInteger h = [rep pixelsHigh];
			if (w > bestWidth && h > 0) {
				bestWidth = w;
				bestHeight = h;
			}
		}
		if (bestWidth <= 0 || bestHeight <= 0) {
			NSSize size = [image size];
			bestWidth = (NSInteger)size.width;
			bestHeight = (NSInteger)size.height;
		}
		if (bestWidth <= 0 || bestHeight <= 0) {
			return NULL;
		}

		NSRect rect = NSMakeRect(0, 0, bestWidth, bestHeight);
		CGImageRef cgImage = [image CGImageForProposedRect:&rect context:nil hints:nil];
		if (!cgImage) {
			return NULL;
		}
		CGColorSpaceRef cs = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
		if (!cs) {
			return NULL;
		}

		size_t w = (size_t)bestWidth;
		size_t h = (size_t)bestHeight;
		size_t bytesPerRow = w * 4;
		unsigned char *data = (unsigned char *)malloc(bytesPerRow * h);
		if (!data) {
			CGColorSpaceRelease(cs);
			return NULL;
		}

		CGContextRef ctx = CGBitmapContextCreate(data, w, h, 8, bytesPerRow, cs, kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
		if (!ctx) {
			free(data);
			CGColorSpaceRelease(cs);
			return NULL;
		}
		CGContextDrawImage(ctx, CGRectMake(0, 0, w, h), cgImage);
		CGContextRelease(ctx);
		CGColorSpaceRelease(cs);

		*width = (int)w;
		*height = (int)h;
		return data;
	}
}

static void free_icon_data(void *data) {
	free(data);
}
*/
import "C"

import (
	"errors"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

func DesktopApps() ([]AppInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	desktop := filepath.Join(home, "Desktop")
	entries, err := os.ReadDir(desktop)
	if err != nil {
		return nil, err
	}

	var apps []AppInfo
	var errs []error
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(desktop, name)
		if entry.IsDir() {
			if strings.HasSuffix(strings.ToLower(name), ".app") {
				info, infoErr := appInfoForPath(path)
				if infoErr != nil {
					errs = append(errs, infoErr)
				}
				apps = append(apps, info)
			}
			continue
		}

		resolved, resolveErr := resolveAlias(path)
		if resolveErr != nil {
			errs = append(errs, resolveErr)
		}
		if resolved == "" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(resolved), ".app") {
			continue
		}
		info, infoErr := appInfoForPath(resolved)
		if infoErr != nil {
			errs = append(errs, infoErr)
		}
		apps = append(apps, info)
	}

	sortApps(apps)
	return apps, joinErrors(errs)
}

func InstalledApps() ([]AppInfo, error) {
	roots := []string{
		"/Applications",
		"/Applications/Utilities",
		"/System/Applications",
		"/System/Applications/Utilities",
	}
	home, err := os.UserHomeDir()
	if err == nil {
		roots = append(roots, filepath.Join(home, "Applications"))
	}

	seen := make(map[string]struct{})
	var apps []AppInfo
	var errs []error
	for _, root := range roots {
		bundles, walkErr := findAppBundles(root, 4)
		if walkErr != nil {
			errs = append(errs, walkErr)
			continue
		}
		for _, bundle := range bundles {
			key := strings.ToLower(bundle)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			info, infoErr := appInfoForPath(bundle)
			if infoErr != nil {
				errs = append(errs, infoErr)
			}
			apps = append(apps, info)
		}
	}

	sortApps(apps)
	return apps, joinErrors(errs)
}

func appInfoForPath(path string) (AppInfo, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	icon, err := iconForPath(path)
	return AppInfo{
		Name: name,
		Path: path,
		Icon: icon,
	}, err
}

func resolveAlias(path string) (string, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	resolved := C.resolve_alias_path(cpath)
	if resolved == nil {
		return "", nil
	}
	defer C.free(unsafe.Pointer(resolved))
	return C.GoString(resolved), nil
}

func iconForPath(path string) (*image.RGBA, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var width C.int
	var height C.int
	data := C.icon_rgba_for_path(cpath, &width, &height)
	if data == nil || width <= 0 || height <= 0 {
		return nil, ErrIconNotFound
	}
	defer C.free_icon_data(unsafe.Pointer(data))

	w := int(width)
	h := int(height)
	size := w * h * 4
	src := unsafe.Slice((*byte)(unsafe.Pointer(data)), size)
	pix := make([]byte, size)
	copy(pix, src)

	return &image.RGBA{
		Pix:    pix,
		Stride: w * 4,
		Rect:   image.Rect(0, 0, w, h),
	}, nil
}

func findAppBundles(root string, maxDepth int) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	root = filepath.Clean(root)
	var bundles []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".app") {
			bundles = append(bundles, path)
			return filepath.SkipDir
		}
		if d.IsDir() && relativeDepth(root, path) >= maxDepth {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return bundles, err
	}
	return bundles, nil
}

func relativeDepth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return 0
	}
	if rel == "." {
		return 0
	}
	return strings.Count(rel, string(os.PathSeparator)) + 1
}
