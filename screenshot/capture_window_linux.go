//go:build cgo && linux
// +build cgo,linux

package screenshot

/*
#cgo LDFLAGS: -lX11 -lXcomposite -lXrender -lXfixes

#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/extensions/Xcomposite.h>
#include <X11/extensions/Xrender.h>
#include <X11/Xatom.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct {
    uint8_t* data;
    int width;
    int height;
    int stride;
    int status;  // 0=ok, 1=display unavailable, 2=no composite ext, 3=window not found, 4=capture failed
    int needsUnredirect;
    unsigned long pixmap;
} XCompShot;

static void xcompshot_free(XCompShot* s) {
    if (s && s->data) {
        free(s->data);
        s->data = NULL;
    }
}

// Capture the offscreen pixmap that the Composite extension keeps for any
// redirected window. We REDIRECT_AUTOMATIC the window if needed (some WMs
// already redirect to enable compositing; that's idempotent and observed by
// XCompositeRedirectWindow returning success).
//
// On older non-compositing setups this call will still succeed when the
// composite extension is loaded, because Composite v0.4+ supports
// per-window redirection regardless of root compositing state.
static XCompShot capture_window_xcomposite(unsigned long xid) {
    XCompShot out;
    memset(&out, 0, sizeof(out));
    out.status = 4;

    Display* dpy = XOpenDisplay(NULL);
    if (dpy == NULL) {
        out.status = 1;
        return out;
    }

    int event_base, error_base;
    if (!XCompositeQueryExtension(dpy, &event_base, &error_base)) {
        XCloseDisplay(dpy);
        out.status = 2;
        return out;
    }

    XWindowAttributes attrs;
    if (!XGetWindowAttributes(dpy, (Window)xid, &attrs)) {
        XCloseDisplay(dpy);
        out.status = 3;
        return out;
    }
    if (attrs.width <= 0 || attrs.height <= 0) {
        XCloseDisplay(dpy);
        out.status = 4;
        return out;
    }

    // Redirect to offscreen storage if not already.
    XCompositeRedirectWindow(dpy, (Window)xid, CompositeRedirectAutomatic);
    XSync(dpy, False);

    Pixmap pix = XCompositeNameWindowPixmap(dpy, (Window)xid);
    if (pix == 0) {
        XCompositeUnredirectWindow(dpy, (Window)xid, CompositeRedirectAutomatic);
        XCloseDisplay(dpy);
        out.status = 4;
        return out;
    }

    XImage* xi = XGetImage(dpy, pix, 0, 0, attrs.width, attrs.height, AllPlanes, ZPixmap);
    XFreePixmap(dpy, pix);
    if (xi == NULL) {
        XCloseDisplay(dpy);
        out.status = 4;
        return out;
    }

    int w = xi->width;
    int h = xi->height;
    int stride = w * 4;
    uint8_t* buf = (uint8_t*)calloc((size_t)(stride * h), 1);
    if (buf == NULL) {
        XDestroyImage(xi);
        XCloseDisplay(dpy);
        out.status = 4;
        return out;
    }

    // XGetImage with TrueColor/24bit visual returns BGRA. Convert to RGBA.
    int bpp = xi->bits_per_pixel / 8;
    if (bpp < 3) bpp = 4;
    for (int y = 0; y < h; y++) {
        uint8_t* src_row = (uint8_t*)xi->data + y * xi->bytes_per_line;
        uint8_t* dst_row = buf + y * stride;
        for (int x = 0; x < w; x++) {
            uint8_t b = src_row[x * bpp + 0];
            uint8_t g = src_row[x * bpp + 1];
            uint8_t r = src_row[x * bpp + 2];
            dst_row[x * 4 + 0] = r;
            dst_row[x * 4 + 1] = g;
            dst_row[x * 4 + 2] = b;
            dst_row[x * 4 + 3] = 0xFF;
        }
    }

    XDestroyImage(xi);
    XCloseDisplay(dpy);

    out.data = buf;
    out.width = w;
    out.height = h;
    out.stride = stride;
    out.status = 0;
    return out;
}
*/
import "C"

import (
	"fmt"
	"image"
	"os"
	"unsafe"

	cap "github.com/PekingSpades/DeskAct/capture"
)

func isWaylandSession() bool {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	return os.Getenv("XDG_SESSION_TYPE") == "wayland"
}

func captureWindowPlatform(req cap.WindowRequest) (*image.RGBA, error) {
	if isWaylandSession() {
		return nil, fmt.Errorf("%w: wayland session — per-window screenshot requires X11", cap.ErrUnsupported)
	}
	backend := req.Options.Backend
	if backend == cap.CaptureBackendDefault {
		backend = cap.CaptureBackendXComposite
	}
	if backend != cap.CaptureBackendXComposite {
		return nil, fmt.Errorf("%w: backend %q is not supported for per-window capture on linux", cap.ErrCaptureBackendUnavailable, backend)
	}
	if req.WindowID == 0 {
		return nil, cap.ErrWindowNotFound
	}

	shot := C.capture_window_xcomposite(C.ulong(req.WindowID))
	defer C.xcompshot_free(&shot)
	switch shot.status {
	case 1:
		return nil, fmt.Errorf("%w: X display unavailable", cap.ErrUnsupported)
	case 2:
		return nil, fmt.Errorf("%w: X Composite extension not present", cap.ErrUnsupported)
	case 3:
		return nil, fmt.Errorf("%w: xid=0x%x", cap.ErrWindowNotFound, req.WindowID)
	case 4:
		return nil, fmt.Errorf("%w: XComposite/XGetImage failed for xid=0x%x", cap.ErrCaptureFailed, req.WindowID)
	}
	if shot.data == nil || shot.width <= 0 || shot.height <= 0 {
		return nil, fmt.Errorf("%w: empty image for xid=0x%x", cap.ErrCaptureFailed, req.WindowID)
	}

	w := int(shot.width)
	h := int(shot.height)
	stride := int(shot.stride)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	src := unsafe.Slice((*byte)(unsafe.Pointer(shot.data)), stride*h)
	copy(img.Pix, src)
	return img, nil
}
