//go:build cgo && windows
// +build cgo,windows

package screenshot

/*
#cgo CFLAGS: -DWIN32_LEAN_AND_MEAN -DNOMINMAX
#cgo LDFLAGS: -ld3d11 -ldxgi -ldxguid -lwindowsapp -lruntimeobject -lole32 -loleaut32 -luuid -lgdi32 -luser32

#include "windows_wgc.h"

#include <windows.h>
#include <wingdi.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#ifndef PW_RENDERFULLCONTENT
#define PW_RENDERFULLCONTENT 0x00000002
#endif

typedef struct {
	uint8_t* data;
	int width;
	int height;
	int stride;
	int status;  // 0=ok, -1=invalid, -2=DC alloc, -3=PrintWindow returned 0
} pw_image_t;

static void pw_free(pw_image_t* img) {
	if (img && img->data) {
		free(img->data);
		img->data = NULL;
	}
}

// PrintWindow with PW_RENDERFULLCONTENT captures DWM-composited windows
// including those that are occluded. Returns a top-down BGRA buffer.
static pw_image_t capture_window_printwindow(uintptr_t hwndPtr) {
	pw_image_t out;
	memset(&out, 0, sizeof(out));
	out.status = -1;
	if (hwndPtr == 0) return out;
	HWND hwnd = (HWND)hwndPtr;

	RECT rect;
	if (!GetClientRect(hwnd, &rect)) return out;
	int w = rect.right - rect.left;
	int h = rect.bottom - rect.top;
	if (w <= 0 || h <= 0) {
		if (!GetWindowRect(hwnd, &rect)) return out;
		w = rect.right - rect.left;
		h = rect.bottom - rect.top;
		if (w <= 0 || h <= 0) return out;
	}

	HDC windowDC = GetDC(hwnd);
	if (!windowDC) { out.status = -2; return out; }
	HDC memDC = CreateCompatibleDC(windowDC);
	if (!memDC) { ReleaseDC(hwnd, windowDC); out.status = -2; return out; }

	BITMAPINFO bmi;
	memset(&bmi, 0, sizeof(bmi));
	bmi.bmiHeader.biSize = sizeof(BITMAPINFOHEADER);
	bmi.bmiHeader.biWidth = w;
	bmi.bmiHeader.biHeight = -h; // top-down
	bmi.bmiHeader.biPlanes = 1;
	bmi.bmiHeader.biBitCount = 32;
	bmi.bmiHeader.biCompression = BI_RGB;

	void* bits = NULL;
	HBITMAP dib = CreateDIBSection(windowDC, &bmi, DIB_RGB_COLORS, &bits, NULL, 0);
	if (!dib || !bits) {
		if (dib) DeleteObject(dib);
		DeleteDC(memDC); ReleaseDC(hwnd, windowDC); out.status = -2; return out;
	}
	HGDIOBJ old = SelectObject(memDC, dib);

	BOOL ok = PrintWindow(hwnd, memDC, PW_RENDERFULLCONTENT);
	if (!ok) {
		ok = PrintWindow(hwnd, memDC, 0);
	}

	int stride = w * 4;
	if (ok) {
		uint8_t* buf = (uint8_t*)malloc((size_t)stride * (size_t)h);
		if (buf) {
			memcpy(buf, bits, (size_t)stride * (size_t)h);
			out.data = buf;
			out.width = w;
			out.height = h;
			out.stride = stride;
			out.status = 0;
		} else {
			out.status = -2;
		}
	} else {
		out.status = -3;
	}

	SelectObject(memDC, old);
	DeleteObject(dib);
	DeleteDC(memDC);
	ReleaseDC(hwnd, windowDC);
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
	if req.WindowID == 0 {
		return CaptureWindowResult{Err: cap.ErrWindowNotFound}
	}

	requested := req.Options.Backend
	backend := requested
	if backend == cap.CaptureBackendDefault {
		backend = cap.CaptureBackendWGC
	}

	switch backend {
	case cap.CaptureBackendWGC:
		img, wgcErr := tryWGC(req.WindowID)
		if wgcErr == nil {
			return CaptureWindowResult{Image: img, BackendUsed: cap.CaptureBackendWGC}
		}
		// Strict mode: when the caller explicitly asked for WGC, do NOT
		// silently fall back to PrintWindow — report the WGC failure
		// truthfully so "WGC verified" claims can be checked.
		if requested == cap.CaptureBackendWGC {
			return CaptureWindowResult{
				BackendUsed: cap.CaptureBackendWGC,
				Err:         fmt.Errorf("WGC capture failed (strict mode, no fallback): %w", wgcErr),
			}
		}
		// Default mode: fall back to PrintWindow.
		img, pwErr := capturePrintWindow(req.WindowID)
		res := CaptureWindowResult{Image: img, BackendUsed: cap.CaptureBackendPrintWindow}
		if pwErr != nil {
			// capturePrintWindow may return img + err for blank frames;
			// surface that as partial.
			if img != nil {
				res.Partial = true
				res.Err = pwErr
			} else {
				res.Err = pwErr
			}
		}
		return res
	case cap.CaptureBackendPrintWindow:
		img, err := capturePrintWindow(req.WindowID)
		res := CaptureWindowResult{Image: img, BackendUsed: cap.CaptureBackendPrintWindow}
		if err != nil {
			if img != nil {
				res.Partial = true
				res.Err = err
			} else {
				res.Err = err
			}
		}
		return res
	default:
		return CaptureWindowResult{
			Err: fmt.Errorf("%w: backend %q is not supported on windows", cap.ErrCaptureBackendUnavailable, backend),
		}
	}
}

func tryWGC(windowID uint64) (*image.RGBA, error) {
	if C.wgc_available() == 0 {
		return nil, fmt.Errorf("%w: WGC runtime unavailable", cap.ErrCaptureBackendUnavailable)
	}
	var shot C.wgc_image_t
	rc := C.wgc_capture_window(C.uintptr_t(windowID), &shot)
	defer C.wgc_free(&shot)
	if int(rc) != 0 {
		return nil, fmt.Errorf("%w: WGC rc=%d", cap.ErrCaptureFailed, int(rc))
	}
	w := int(shot.width)
	h := int(shot.height)
	stride := int(shot.stride)
	if shot.data == nil || w <= 0 || h <= 0 {
		return nil, fmt.Errorf("%w: empty WGC frame", cap.ErrCaptureFailed)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	src := unsafe.Slice((*byte)(unsafe.Pointer(shot.data)), stride*h)
	bgraToRGBA(src, img.Pix, w, h, stride)
	return img, nil
}

func capturePrintWindow(windowID uint64) (*image.RGBA, error) {
	shot := C.capture_window_printwindow(C.uintptr_t(windowID))
	defer C.pw_free(&shot)
	if shot.status != 0 || shot.data == nil {
		return nil, fmt.Errorf("%w: PrintWindow status=%d", cap.ErrCaptureFailed, int(shot.status))
	}
	w := int(shot.width)
	h := int(shot.height)
	stride := int(shot.stride)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	src := unsafe.Slice((*byte)(unsafe.Pointer(shot.data)), stride*h)
	bgraToRGBA(src, img.Pix, w, h, stride)

	if isBlankFrame(img.Pix, w, h) {
		return img, fmt.Errorf("%w: PrintWindow returned blank frame (GPU-accelerated window?)", cap.ErrCaptureFailed)
	}
	return img, nil
}

func bgraToRGBA(src, dst []byte, w, h, stride int) {
	for y := 0; y < h; y++ {
		soff := y * stride
		doff := y * w * 4
		for x := 0; x < w; x++ {
			b := src[soff+x*4+0]
			g := src[soff+x*4+1]
			r := src[soff+x*4+2]
			dst[doff+x*4+0] = r
			dst[doff+x*4+1] = g
			dst[doff+x*4+2] = b
			dst[doff+x*4+3] = 0xFF
		}
	}
}

// isBlankFrame samples 16 evenly-spaced pixels; if every one is fully black
// the capture is treated as a blank frame.
func isBlankFrame(pix []byte, w, h int) bool {
	if w == 0 || h == 0 {
		return true
	}
	for sy := 0; sy < 4; sy++ {
		for sx := 0; sx < 4; sx++ {
			x := (w * (sx*2 + 1)) / 8
			y := (h * (sy*2 + 1)) / 8
			off := (y*w + x) * 4
			if pix[off+0] != 0 || pix[off+1] != 0 || pix[off+2] != 0 {
				return false
			}
		}
	}
	return true
}
