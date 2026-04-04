//go:build cgo && darwin

package screenshot

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework Foundation
#cgo LDFLAGS: -weak_framework ScreenCaptureKit
#include <CoreGraphics/CoreGraphics.h>
#include <Availability.h>
#include <Foundation/Foundation.h>
#include <dispatch/dispatch.h>
#if __has_include(<ScreenCaptureKit/ScreenCaptureKit.h>)
#include <ScreenCaptureKit/ScreenCaptureKit.h>
#define HAS_SCREENCAPTUREKIT 1
#endif

typedef enum {
    CaptureStatusOK = 0,
    CaptureStatusBackendUnavailable = 1,
    CaptureStatusWindowExclusionUnsupported = 2,
    CaptureStatusCaptureFailed = 3,
} CaptureStatus;

typedef struct {
    CGImageRef image;
    CaptureStatus status;
} CaptureResult;

static CaptureResult captureWithCGDisplay(CGDirectDisplayID id, CGRect diIntersectDisplayLocal, CGColorSpaceRef colorSpace) {
    CaptureResult result = {0};
    CGImageRef img = CGDisplayCreateImageForRect(id, diIntersectDisplayLocal);
    if (!img) {
        result.status = CaptureStatusCaptureFailed;
        return result;
    }
    result.image = CGImageCreateCopyWithColorSpace(img, colorSpace);
    CGImageRelease(img);
    result.status = result.image ? CaptureStatusOK : CaptureStatusCaptureFailed;
    return result;
}

static CaptureResult captureWithScreenCaptureKit(CGDirectDisplayID id,
                                                 CGRect diIntersectDisplayLocal,
                                                 CGColorSpaceRef colorSpace,
                                                 const uint64_t* excludedWindowIDs,
                                                 size_t excludedWindowCount) {
    CaptureResult result = {0};
#if defined(HAS_SCREENCAPTUREKIT)
    if (@available(macOS 14.4, *)) {
    } else {
        result.status = CaptureStatusBackendUnavailable;
        return result;
    }

    dispatch_semaphore_t semaphore = dispatch_semaphore_create(0);
    __block CaptureStatus status = CaptureStatusCaptureFailed;
    __block CGImageRef capturedImage = nil;

    [SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent* content, NSError* error) {
        @autoreleasepool {
            if (error || !content) {
                dispatch_semaphore_signal(semaphore);
                return;
            }

            SCDisplay* target = nil;
            for (SCDisplay* display in content.displays) {
                if (display.displayID == id) {
                    target = display;
                    break;
                }
            }
            if (!target) {
                dispatch_semaphore_signal(semaphore);
                return;
            }

            NSArray<SCWindow*>* excludedWindows = @[];
            if (excludedWindowCount > 0) {
                NSMutableArray<SCWindow*>* matches = [NSMutableArray array];
                for (SCWindow* window in content.windows) {
                    uint64_t windowID = (uint64_t)window.windowID;
                    for (size_t i = 0; i < excludedWindowCount; i++) {
                        if (windowID == excludedWindowIDs[i]) {
                            [matches addObject:window];
                            break;
                        }
                    }
                }
                excludedWindows = matches;
            }

            SCContentFilter* filter = [[[SCContentFilter alloc] initWithDisplay:target excludingWindows:excludedWindows] autorelease];
            SCStreamConfiguration* config = [[[SCStreamConfiguration alloc] init] autorelease];
            config.sourceRect = diIntersectDisplayLocal;
            config.width = (NSInteger)diIntersectDisplayLocal.size.width;
            config.height = (NSInteger)diIntersectDisplayLocal.size.height;
            config.showsCursor = NO;

            [SCScreenshotManager captureImageWithFilter:filter
                                          configuration:config
                                      completionHandler:^(CGImageRef image, NSError* error) {
                @autoreleasepool {
                    if (!error && image) {
                        capturedImage = CGImageCreateCopyWithColorSpace(image, colorSpace);
                        if (capturedImage) {
                            status = CaptureStatusOK;
                        }
                    }
                    dispatch_semaphore_signal(semaphore);
                }
            }];
        }
    }];

    dispatch_semaphore_wait(semaphore, DISPATCH_TIME_FOREVER);
    result.image = capturedImage;
    result.status = status;
    return result;
#else
    (void)id;
    (void)diIntersectDisplayLocal;
    (void)colorSpace;
    (void)excludedWindowIDs;
    (void)excludedWindowCount;
    result.status = CaptureStatusBackendUnavailable;
    return result;
#endif
}
*/
import "C"

import (
	"errors"
	"image"
	"unsafe"

	cap "github.com/PekingSpades/DeskAct/capture"
)

func Capture(req cap.Request) (*image.RGBA, error) {
	if req.Width <= 0 || req.Height <= 0 {
		return nil, errors.New("width or height should be > 0")
	}

	backend := normalizeRequestedBackend(req.Options.Backend, cap.CaptureBackendScreenCaptureKit)
	if hasExcludedWindowIDs(req.Options) && backend != cap.CaptureBackendScreenCaptureKit {
		return nil, windowExclusionUnsupportedError(backend)
	}

	switch backend {
	case cap.CaptureBackendScreenCaptureKit, cap.CaptureBackendCGDisplay:
	default:
		return nil, backendUnavailableError(backend, "backend %q is not supported on macOS", backend)
	}

	rect := image.Rect(0, 0, req.Width, req.Height)
	img, err := createImage(rect)
	if err != nil {
		return nil, err
	}

	// cg: CoreGraphics coordinate (origin: lower-left corner of primary display, x-axis: rightward, y-axis: upward)
	// win: Windows coordinate (origin: upper-left corner of primary display, x-axis: rightward, y-axis: downward)
	// di: Display local coordinate (origin: upper-left corner of the display, x-axis: rightward, y-axis: downward)

	cgMainDisplayBounds := getCoreGraphicsCoordinateOfDisplay(C.CGMainDisplayID())

	winBottomLeft := C.CGPointMake(C.CGFloat(req.X), C.CGFloat(req.Y+req.Height))
	cgBottomLeft := getCoreGraphicsCoordinateFromWindowsCoordinate(winBottomLeft, cgMainDisplayBounds)
	cgCaptureBounds := C.CGRectMake(cgBottomLeft.x, cgBottomLeft.y, C.CGFloat(req.Width), C.CGFloat(req.Height))

	ids := activeDisplayList()

	ctx := createBitmapContext(req.Width, req.Height, (*C.uint32_t)(unsafe.Pointer(&img.Pix[0])), img.Stride)
	if ctx == 0 {
		return nil, errors.New("cannot create bitmap context")
	}

	colorSpace := createColorspace()
	if colorSpace == 0 {
		return nil, errors.New("cannot create colorspace")
	}
	defer C.CGColorSpaceRelease(colorSpace)

	var excludedWindowIDs []C.uint64_t
	if backend == cap.CaptureBackendScreenCaptureKit && len(req.Options.ExcludedWindowIDs) > 0 {
		excludedWindowIDs = make([]C.uint64_t, len(req.Options.ExcludedWindowIDs))
		for i, windowID := range req.Options.ExcludedWindowIDs {
			excludedWindowIDs[i] = C.uint64_t(windowID)
		}
	}

	for _, id := range ids {
		cgBounds := getCoreGraphicsCoordinateOfDisplay(id)
		cgIntersect := C.CGRectIntersection(cgBounds, cgCaptureBounds)
		if C.CGRectIsNull(cgIntersect) {
			continue
		}
		if cgIntersect.size.width <= 0 || cgIntersect.size.height <= 0 {
			continue
		}

		// CGDisplayCreateImageForRect potentially fails in case width/height is odd number.
		if int(cgIntersect.size.width)%2 != 0 {
			cgIntersect.size.width = C.CGFloat(int(cgIntersect.size.width) + 1)
		}
		if int(cgIntersect.size.height)%2 != 0 {
			cgIntersect.size.height = C.CGFloat(int(cgIntersect.size.height) + 1)
		}

		diIntersectDisplayLocal := C.CGRectMake(
			cgIntersect.origin.x-cgBounds.origin.x,
			cgBounds.origin.y+cgBounds.size.height-(cgIntersect.origin.y+cgIntersect.size.height),
			cgIntersect.size.width,
			cgIntersect.size.height,
		)

		imageRef, err := captureDarwinImage(id, diIntersectDisplayLocal, colorSpace, backend, excludedWindowIDs)
		if err != nil {
			return nil, err
		}
		defer C.CGImageRelease(imageRef)

		cgDrawRect := C.CGRectMake(
			cgIntersect.origin.x-cgCaptureBounds.origin.x,
			cgIntersect.origin.y-cgCaptureBounds.origin.y,
			cgIntersect.size.width,
			cgIntersect.size.height,
		)
		C.CGContextDrawImage(ctx, cgDrawRect, imageRef)
	}

	i := 0
	for iy := 0; iy < req.Height; iy++ {
		j := i
		for ix := 0; ix < req.Width; ix++ {
			// ARGB => RGBA, and set A to 255
			img.Pix[j], img.Pix[j+1], img.Pix[j+2], img.Pix[j+3] = img.Pix[j+1], img.Pix[j+2], img.Pix[j+3], 255
			j += 4
		}
		i += img.Stride
	}

	return img, nil
}

func captureDarwinImage(id C.CGDirectDisplayID, rect C.CGRect, colorSpace C.CGColorSpaceRef, backend cap.CaptureBackend, excludedWindowIDs []C.uint64_t) (C.CGImageRef, error) {
	switch backend {
	case cap.CaptureBackendScreenCaptureKit:
		var ptr *C.uint64_t
		if len(excludedWindowIDs) > 0 {
			ptr = &excludedWindowIDs[0]
		}
		result := C.captureWithScreenCaptureKit(id, rect, colorSpace, ptr, C.size_t(len(excludedWindowIDs)))
		return darwinCaptureResult(result, backend)
	case cap.CaptureBackendCGDisplay:
		result := C.captureWithCGDisplay(id, rect, colorSpace)
		return darwinCaptureResult(result, backend)
	default:
		return nil, backendUnavailableError(backend, "backend %q is not supported on macOS", backend)
	}
}

func darwinCaptureResult(result C.CaptureResult, backend cap.CaptureBackend) (C.CGImageRef, error) {
	switch result.status {
	case C.CaptureStatusOK:
		if unsafe.Pointer(result.image) == nil {
			return nil, errors.New("cannot capture display")
		}
		return result.image, nil
	case C.CaptureStatusBackendUnavailable:
		return nil, backendUnavailableError(backend, "backend %q is unavailable on this macOS version", backend)
	case C.CaptureStatusWindowExclusionUnsupported:
		return nil, windowExclusionUnsupportedError(backend)
	default:
		return nil, errors.New("cannot capture display")
	}
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
