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
} XCompShot;

static void xcompshot_free(XCompShot* s) {
    if (s && s->data) {
        free(s->data);
        s->data = NULL;
    }
}

// Long-lived display connection used so that the COMPOSITE redirects below
// stay in effect across consecutive captures of the same window. The
// Composite extension scopes redirects to the requesting client; closing
// the X connection unredirects every previously-redirected window.
// linux_capture_open() lazily opens the display on first use; it is OK to
// call repeatedly.
static Display *g_dpy = NULL;
static int      g_have_composite = 0;

// g_redirected is a small open-addressed hash set of XIDs we have already
// asked the server to redirect. The X server refcounts redirects per
// client, so calling XCompositeRedirectWindow more than once per (client,
// xid) pair leaks a refcount that's only released when the connection
// dies — wasting a bit of state and keeping the window redirected longer
// than expected after the caller stops using us. Track membership so the
// first capture of an xid sends the redirect and subsequent captures of
// the same xid just hit the existing offscreen pixmap.
#define REDIR_SET_CAP 4096
static unsigned long g_redirected[REDIR_SET_CAP];
static int           g_redirected_n = 0;

static int redirect_set_contains(unsigned long xid) {
    if (g_redirected_n == 0) return 0;
    unsigned long h = (xid * 2654435761UL) % REDIR_SET_CAP;
    for (int i = 0; i < REDIR_SET_CAP; i++) {
        unsigned long idx = (h + i) % REDIR_SET_CAP;
        unsigned long cur = g_redirected[idx];
        if (cur == 0) return 0;
        if (cur == xid) return 1;
    }
    return 0;
}

static void redirect_set_add(unsigned long xid) {
    if (g_redirected_n >= REDIR_SET_CAP - 1) {
        // Set is full — bail out and tolerate the extra refcount rather
        // than getting stuck in an infinite probe loop. Hitting this
        // means a session is tracking >4k distinct windows, well past
        // any realistic deskact usage.
        return;
    }
    unsigned long h = (xid * 2654435761UL) % REDIR_SET_CAP;
    for (int i = 0; i < REDIR_SET_CAP; i++) {
        unsigned long idx = (h + i) % REDIR_SET_CAP;
        if (g_redirected[idx] == 0) {
            g_redirected[idx] = xid;
            g_redirected_n++;
            return;
        }
        if (g_redirected[idx] == xid) return;
    }
}

static int linux_capture_open(void) {
    if (g_dpy != NULL) return 0;
    Display *d = XOpenDisplay(NULL);
    if (d == NULL) return 1;
    int event_base, error_base;
    if (!XCompositeQueryExtension(d, &event_base, &error_base)) {
        XCloseDisplay(d);
        return 2;
    }
    g_dpy = d;
    g_have_composite = 1;
    return 0;
}

// Capture the offscreen pixmap that the Composite extension keeps for any
// redirected window. Redirects are only requested once per window — once
// CompositeRedirectAutomatic has been set on the shared client, the
// X server keeps the offscreen backbuffer in sync until the connection
// drops (which we never do).
static XCompShot capture_window_xcomposite(unsigned long xid) {
    XCompShot out;
    memset(&out, 0, sizeof(out));
    out.status = 4;

    int rc = linux_capture_open();
    if (rc != 0) { out.status = rc; return out; }

    XWindowAttributes attrs;
    if (!XGetWindowAttributes(g_dpy, (Window)xid, &attrs)) {
        out.status = 3;
        return out;
    }
    if (attrs.width <= 0 || attrs.height <= 0) {
        out.status = 4;
        return out;
    }

    // Only request a redirect on first capture of this xid. Repeated
    // XCompositeRedirectWindow calls would each bump the server-side
    // refcount; once we've redirected once the offscreen pixmap stays
    // live for as long as our connection is open.
    if (!redirect_set_contains(xid)) {
        XCompositeRedirectWindow(g_dpy, (Window)xid, CompositeRedirectAutomatic);
        XSync(g_dpy, False);
        redirect_set_add(xid);
    }

    Pixmap pix = XCompositeNameWindowPixmap(g_dpy, (Window)xid);
    if (pix == 0) {
        out.status = 4;
        return out;
    }

    XImage* xi = XGetImage(g_dpy, pix, 0, 0, attrs.width, attrs.height, AllPlanes, ZPixmap);
    XFreePixmap(g_dpy, pix);
    if (xi == NULL) {
        out.status = 4;
        return out;
    }

    int w = xi->width;
    int h = xi->height;
    int stride = w * 4;
    uint8_t* buf = (uint8_t*)calloc((size_t)(stride * h), 1);
    if (buf == NULL) {
        XDestroyImage(xi);
        out.status = 4;
        return out;
    }

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
	"sync"
	"unsafe"

	cap "github.com/PekingSpades/DeskAct/capture"
)

func isWaylandSession() bool {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	return os.Getenv("XDG_SESSION_TYPE") == "wayland"
}

// captureMu serialises access to the shared X display g_dpy in the C
// translation unit. Xlib is not thread-safe by default unless XInitThreads
// is called, and several goroutines could otherwise race on the same
// connection.
var captureMu sync.Mutex

func captureWindowPlatform(req cap.WindowRequest) (*image.RGBA, error) {
	res := captureWindowPlatformEx(req)
	return res.Image, res.Err
}

func captureWindowPlatformEx(req cap.WindowRequest) CaptureWindowResult {
	if isWaylandSession() {
		return CaptureWindowResult{Err: fmt.Errorf("%w: wayland session — per-window screenshot requires X11", cap.ErrUnsupported)}
	}
	backend := req.Options.Backend
	if backend == cap.CaptureBackendDefault {
		backend = cap.CaptureBackendXComposite
	}
	if backend != cap.CaptureBackendXComposite {
		return CaptureWindowResult{Err: fmt.Errorf("%w: backend %q is not supported for per-window capture on linux", cap.ErrCaptureBackendUnavailable, backend)}
	}
	if req.WindowID == 0 {
		return CaptureWindowResult{Err: cap.ErrWindowNotFound}
	}

	captureMu.Lock()
	shot := C.capture_window_xcomposite(C.ulong(req.WindowID))
	captureMu.Unlock()
	defer C.xcompshot_free(&shot)
	switch shot.status {
	case 1:
		return CaptureWindowResult{BackendUsed: cap.CaptureBackendXComposite, Err: fmt.Errorf("%w: X display unavailable", cap.ErrUnsupported)}
	case 2:
		return CaptureWindowResult{BackendUsed: cap.CaptureBackendXComposite, Err: fmt.Errorf("%w: X Composite extension not present", cap.ErrUnsupported)}
	case 3:
		return CaptureWindowResult{BackendUsed: cap.CaptureBackendXComposite, Err: fmt.Errorf("%w: xid=0x%x", cap.ErrWindowNotFound, req.WindowID)}
	case 4:
		return CaptureWindowResult{BackendUsed: cap.CaptureBackendXComposite, Err: fmt.Errorf("%w: XComposite/XGetImage failed for xid=0x%x", cap.ErrCaptureFailed, req.WindowID)}
	}
	if shot.data == nil || shot.width <= 0 || shot.height <= 0 {
		return CaptureWindowResult{BackendUsed: cap.CaptureBackendXComposite, Err: fmt.Errorf("%w: empty image for xid=0x%x", cap.ErrCaptureFailed, req.WindowID)}
	}

	w := int(shot.width)
	h := int(shot.height)
	stride := int(shot.stride)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	src := unsafe.Slice((*byte)(unsafe.Pointer(shot.data)), stride*h)
	copy(img.Pix, src)
	return CaptureWindowResult{Image: img, BackendUsed: cap.CaptureBackendXComposite}
}
