//go:build cgo && darwin

package screenshot

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#cgo LDFLAGS: -weak_framework ScreenCaptureKit
#include <CoreGraphics/CoreGraphics.h>
#include <Availability.h>
#if __has_include(<ScreenCaptureKit/ScreenCaptureKit.h>)
#include <ScreenCaptureKit/ScreenCaptureKit.h>
#define HAS_SCREENCAPTUREKIT 1
#endif

static CGImageRef capture(CGDirectDisplayID id, CGRect diIntersectDisplayLocal, CGColorSpaceRef colorSpace) {
#if defined(HAS_SCREENCAPTUREKIT)
    if (@available(macOS 14.4, *)) {
        dispatch_semaphore_t semaphore = dispatch_semaphore_create(0);
        __block CGImageRef result = nil;
        [SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent* content, NSError* error) {
            @autoreleasepool {
                if (error) {
                    dispatch_semaphore_signal(semaphore);
                    return;
                }
                SCDisplay* target = nil;
                for (SCDisplay *display in content.displays) {
                    if (display.displayID == id) {
                        target = display;
                        break;
                    }
                }
                if (!target) {
                    dispatch_semaphore_signal(semaphore);
                    return;
                }
                SCContentFilter* filter = [[SCContentFilter alloc] initWithDisplay:target excludingWindows:@[]];
                SCStreamConfiguration* config = [[SCStreamConfiguration alloc] init];
                config.sourceRect = diIntersectDisplayLocal;
                config.width = diIntersectDisplayLocal.size.width;
                config.height = diIntersectDisplayLocal.size.height;
                config.showsCursor = NO;
                [SCScreenshotManager captureImageWithFilter:filter
                                              configuration:config
                                          completionHandler:^(CGImageRef img, NSError* error) {
                    if (!error) {
                        result = CGImageCreateCopyWithColorSpace(img, colorSpace);
                    }
                    dispatch_semaphore_signal(semaphore);
                }];
            }
        }];
        dispatch_semaphore_wait(semaphore, DISPATCH_TIME_FOREVER);
        dispatch_release(semaphore);
        return result;
    }
#if __MAC_OS_X_VERSION_MIN_REQUIRED < 150000
    CGImageRef img = CGDisplayCreateImageForRect(id, diIntersectDisplayLocal);
    if (!img) {
        return nil;
    }
    CGImageRef copy = CGImageCreateCopyWithColorSpace(img, colorSpace);
    CGImageRelease(img);
    if (!copy) {
        return nil;
    }
    return copy;
#endif
#else
    CGImageRef img = CGDisplayCreateImageForRect(id, diIntersectDisplayLocal);
    if (!img) {
        return nil;
    }
    CGImageRef copy = CGImageCreateCopyWithColorSpace(img, colorSpace);
    CGImageRelease(img);
    if (!copy) {
        return nil;
    }
    return copy;
#endif
    return nil;
}
*/
import "C"

import (
	"errors"
	"image"
	"unsafe"
)

func Capture(x, y, width, height int, waylandToken uint64) (*image.RGBA, error) {
	_ = waylandToken
	if width <= 0 || height <= 0 {
		return nil, errors.New("width or height should be > 0")
	}

	rect := image.Rect(0, 0, width, height)
	img, err := createImage(rect)
	if err != nil {
		return nil, err
	}

	// cg: CoreGraphics coordinate (origin: lower-left corner of primary display, x-axis: rightward, y-axis: upward)
	// win: Windows coordinate (origin: upper-left corner of primary display, x-axis: rightward, y-axis: downward)
	// di: Display local coordinate (origin: upper-left corner of the display, x-axis: rightward, y-axis: downward)

	cgMainDisplayBounds := getCoreGraphicsCoordinateOfDisplay(C.CGMainDisplayID())

	winBottomLeft := C.CGPointMake(C.CGFloat(x), C.CGFloat(y+height))
	cgBottomLeft := getCoreGraphicsCoordinateFromWindowsCoordinate(winBottomLeft, cgMainDisplayBounds)
	cgCaptureBounds := C.CGRectMake(cgBottomLeft.x, cgBottomLeft.y, C.CGFloat(width), C.CGFloat(height))

	ids := activeDisplayList()

	ctx := createBitmapContext(width, height, (*C.uint32_t)(unsafe.Pointer(&img.Pix[0])), img.Stride)
	if ctx == 0 {
		return nil, errors.New("cannot create bitmap context")
	}

	colorSpace := createColorspace()
	if colorSpace == 0 {
		return nil, errors.New("cannot create colorspace")
	}
	defer C.CGColorSpaceRelease(colorSpace)

	for _, id := range ids {
		cgBounds := getCoreGraphicsCoordinateOfDisplay(id)
		cgIntersect := C.CGRectIntersection(cgBounds, cgCaptureBounds)
		if C.CGRectIsNull(cgIntersect) {
			continue
		}
		if cgIntersect.size.width <= 0 || cgIntersect.size.height <= 0 {
			continue
		}

		// CGDisplayCreateImageForRect potentially fail in case width/height is odd number.
		if int(cgIntersect.size.width)%2 != 0 {
			cgIntersect.size.width = C.CGFloat(int(cgIntersect.size.width) + 1)
		}
		if int(cgIntersect.size.height)%2 != 0 {
			cgIntersect.size.height = C.CGFloat(int(cgIntersect.size.height) + 1)
		}

		diIntersectDisplayLocal := C.CGRectMake(cgIntersect.origin.x-cgBounds.origin.x,
			cgBounds.origin.y+cgBounds.size.height-(cgIntersect.origin.y+cgIntersect.size.height),
			cgIntersect.size.width, cgIntersect.size.height)

		image := C.capture(id, diIntersectDisplayLocal, colorSpace)
		if unsafe.Pointer(image) == nil {
			return nil, errors.New("cannot capture display")
		}
		defer C.CGImageRelease(image)

		cgDrawRect := C.CGRectMake(cgIntersect.origin.x-cgCaptureBounds.origin.x, cgIntersect.origin.y-cgCaptureBounds.origin.y,
			cgIntersect.size.width, cgIntersect.size.height)
		C.CGContextDrawImage(ctx, cgDrawRect, image)
	}

	i := 0
	for iy := 0; iy < height; iy++ {
		j := i
		for ix := 0; ix < width; ix++ {
			// ARGB => RGBA, and set A to 255
			img.Pix[j], img.Pix[j+1], img.Pix[j+2], img.Pix[j+3] = img.Pix[j+1], img.Pix[j+2], img.Pix[j+3], 255
			j += 4
		}
		i += img.Stride
	}

	return img, nil
}

func NumActiveDisplays() int {
	var count C.uint32_t = 0
	if C.CGGetActiveDisplayList(0, nil, &count) == C.kCGErrorSuccess {
		return int(count)
	} else {
		return 0
	}
}

func getCoreGraphicsCoordinateOfDisplay(id C.CGDirectDisplayID) C.CGRect {
	main := C.CGDisplayBounds(C.CGMainDisplayID())
	r := C.CGDisplayBounds(id)
	return C.CGRectMake(r.origin.x, -r.origin.y-r.size.height+main.size.height,
		r.size.width, r.size.height)
}

func getCoreGraphicsCoordinateFromWindowsCoordinate(p C.CGPoint, mainDisplayBounds C.CGRect) C.CGPoint {
	return C.CGPointMake(p.x, mainDisplayBounds.size.height-p.y)
}

func createBitmapContext(width int, height int, data *C.uint32_t, bytesPerRow int) C.CGContextRef {
	colorSpace := createColorspace()
	if colorSpace == 0 {
		return 0
	}
	defer C.CGColorSpaceRelease(colorSpace)

	return C.CGBitmapContextCreate(unsafe.Pointer(data),
		C.size_t(width),
		C.size_t(height),
		8, // bits per component
		C.size_t(bytesPerRow),
		colorSpace,
		C.kCGImageAlphaNoneSkipFirst)
}

func createColorspace() C.CGColorSpaceRef {
	return C.CGColorSpaceCreateWithName(C.kCGColorSpaceSRGB)
}

func activeDisplayList() []C.CGDirectDisplayID {
	count := C.uint32_t(NumActiveDisplays())
	ret := make([]C.CGDirectDisplayID, count)
	if count > 0 && C.CGGetActiveDisplayList(count, (*C.CGDirectDisplayID)(unsafe.Pointer(&ret[0])), nil) == C.kCGErrorSuccess {
		return ret
	} else {
		return make([]C.CGDirectDisplayID, 0)
	}
}
