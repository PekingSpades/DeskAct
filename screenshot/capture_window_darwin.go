//go:build cgo && darwin
// +build cgo,darwin

package screenshot

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework Foundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdbool.h>
#include <stdint.h>

typedef struct {
    uint8_t* data;
    int width;
    int height;
    int stride;
    int status; // 0=ok, 1=window not found, 2=capture failed
} WinShot;

static void winshot_free(WinShot* s) {
    if (s && s->data) {
        free(s->data);
        s->data = NULL;
    }
}

// CGWindowListCreateImage with kCGWindowListOptionIncludingWindow returns the
// composited contents of the named window even if it is occluded by other
// windows, as long as the window exists in the window-server cache.
// kCGWindowImageBoundsIgnoreFraming trims OS-imposed shadows/decorations
// (window-manager artifacts) from the captured image.
static WinShot capture_window_cglist(uint32_t windowID) {
    WinShot out;
    out.data = NULL;
    out.width = 0;
    out.height = 0;
    out.stride = 0;
    out.status = 1;

    if (windowID == 0) return out;

    CGImageRef img = CGWindowListCreateImage(
        CGRectNull,
        kCGWindowListOptionIncludingWindow,
        (CGWindowID)windowID,
        kCGWindowImageBoundsIgnoreFraming | kCGWindowImageNominalResolution);
    if (img == NULL) {
        out.status = 2;
        return out;
    }

    size_t w = CGImageGetWidth(img);
    size_t h = CGImageGetHeight(img);
    if (w == 0 || h == 0) {
        CGImageRelease(img);
        out.status = 2;
        return out;
    }

    size_t stride = w * 4;
    uint8_t* buf = (uint8_t*)calloc(stride * h, 1);
    if (buf == NULL) {
        CGImageRelease(img);
        out.status = 2;
        return out;
    }

    CGColorSpaceRef cs = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
    CGContextRef ctx = CGBitmapContextCreate(buf, w, h, 8, stride, cs,
        kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
    CGColorSpaceRelease(cs);

    if (ctx == NULL) {
        free(buf);
        CGImageRelease(img);
        out.status = 2;
        return out;
    }

    CGContextDrawImage(ctx, CGRectMake(0, 0, (CGFloat)w, (CGFloat)h), img);
    CGContextRelease(ctx);
    CGImageRelease(img);

    out.data = buf;
    out.width = (int)w;
    out.height = (int)h;
    out.stride = (int)stride;
    out.status = 0;
    return out;
}
*/
import "C"

import (
	"fmt"
	"image"
	"unsafe"

	cap "github.com/PekingSpades/DeskAct/capture"
)

func captureWindowPlatform(req cap.WindowRequest) (*image.RGBA, error) {
	res := captureWindowPlatformEx(req)
	return res.Image, res.Err
}

func captureWindowPlatformEx(req cap.WindowRequest) CaptureWindowResult {
	backend := req.Options.Backend
	if backend == cap.CaptureBackendDefault {
		backend = cap.CaptureBackendCGWindowList
	}
	if backend != cap.CaptureBackendCGWindowList {
		return CaptureWindowResult{Err: fmt.Errorf("%w: backend %q is not supported for per-window capture on darwin", cap.ErrCaptureBackendUnavailable, backend)}
	}
	if req.WindowID == 0 {
		return CaptureWindowResult{Err: cap.ErrWindowNotFound}
	}

	shot := C.capture_window_cglist(C.uint32_t(req.WindowID))
	defer C.winshot_free(&shot)
	switch shot.status {
	case 1:
		return CaptureWindowResult{BackendUsed: cap.CaptureBackendCGWindowList, Err: fmt.Errorf("%w: cgwindow id=%d", cap.ErrWindowNotFound, req.WindowID)}
	case 2:
		return CaptureWindowResult{BackendUsed: cap.CaptureBackendCGWindowList, Err: fmt.Errorf("%w: CGWindowListCreateImage returned nil for id=%d", cap.ErrCaptureFailed, req.WindowID)}
	}
	if shot.data == nil || shot.width <= 0 || shot.height <= 0 {
		return CaptureWindowResult{BackendUsed: cap.CaptureBackendCGWindowList, Err: fmt.Errorf("%w: empty image for id=%d", cap.ErrCaptureFailed, req.WindowID)}
	}

	w := int(shot.width)
	h := int(shot.height)
	stride := int(shot.stride)
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	src := unsafe.Slice((*byte)(unsafe.Pointer(shot.data)), stride*h)
	copy(img.Pix, src)
	return CaptureWindowResult{Image: img, BackendUsed: cap.CaptureBackendCGWindowList}
}
