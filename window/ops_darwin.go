//go:build darwin
// +build darwin

package window

/*
#cgo darwin LDFLAGS: -framework ApplicationServices -framework CoreFoundation -framework CoreGraphics
#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

#define OPS_OK 0
#define OPS_ERR_PERM 1
#define OPS_ERR_NOT_FOUND 2
#define OPS_ERR_AX 3
#define OPS_ERR_ARG 4

// Probe AX trust. options.kAXTrustedCheckOptionPrompt=true will pop the
// system permission dialog the first time. Returns 1 if trusted, 0 if not.
static int axTrusted(int prompt) {
	CFStringRef key = kAXTrustedCheckOptionPrompt;
	CFBooleanRef val = prompt ? kCFBooleanTrue : kCFBooleanFalse;
	CFDictionaryRef opts = CFDictionaryCreate(NULL, (const void **)&key, (const void **)&val, 1,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	Boolean trusted = AXIsProcessTrustedWithOptions(opts);
	CFRelease(opts);
	return trusted ? 1 : 0;
}

// Find the AXUIElementRef matching the supplied CGWindowID by walking the
// AX windows of the application identified by pid. We match on the
// _AXUIElementWindowID private attribute first, then fall back to the
// position/size pair from CGWindowList.
static AXUIElementRef axFindWindow(pid_t pid, uint32_t targetWindowID) {
	if (pid <= 0) return NULL;
	AXUIElementRef app = AXUIElementCreateApplication(pid);
	if (app == NULL) return NULL;

	CFTypeRef windowsRef = NULL;
	if (AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, &windowsRef) != kAXErrorSuccess || windowsRef == NULL) {
		CFRelease(app);
		return NULL;
	}
	CFArrayRef windows = (CFArrayRef)windowsRef;
	CFIndex count = CFArrayGetCount(windows);

	// AXUIElement private attribute that exposes the underlying CGWindowID.
	CFStringRef axWinIDAttr = CFStringCreateWithCString(NULL, "_AXUIElementWindowID", kCFStringEncodingUTF8);

	AXUIElementRef hit = NULL;
	for (CFIndex i = 0; i < count; i++) {
		AXUIElementRef w = (AXUIElementRef)CFArrayGetValueAtIndex(windows, i);
		CFTypeRef idRef = NULL;
		if (AXUIElementCopyAttributeValue(w, axWinIDAttr, &idRef) == kAXErrorSuccess && idRef) {
			uint32_t got = 0;
			CFNumberGetValue((CFNumberRef)idRef, kCFNumberSInt32Type, &got);
			CFRelease(idRef);
			if (got == targetWindowID) {
				hit = (AXUIElementRef)CFRetain(w);
				break;
			}
		}
	}

	CFRelease(axWinIDAttr);
	CFRelease(windows);
	CFRelease(app);
	return hit;
}

static int axSetPosition(AXUIElementRef win, int x, int y) {
	CGPoint pt = CGPointMake((CGFloat)x, (CGFloat)y);
	AXValueRef v = AXValueCreate(kAXValueCGPointType, &pt);
	if (v == NULL) return OPS_ERR_AX;
	AXError err = AXUIElementSetAttributeValue(win, kAXPositionAttribute, v);
	CFRelease(v);
	return (err == kAXErrorSuccess) ? OPS_OK : OPS_ERR_AX;
}

static int axSetSize(AXUIElementRef win, int w, int h) {
	CGSize sz = CGSizeMake((CGFloat)w, (CGFloat)h);
	AXValueRef v = AXValueCreate(kAXValueCGSizeType, &sz);
	if (v == NULL) return OPS_ERR_AX;
	AXError err = AXUIElementSetAttributeValue(win, kAXSizeAttribute, v);
	CFRelease(v);
	return (err == kAXErrorSuccess) ? OPS_OK : OPS_ERR_AX;
}

static int axMove(pid_t pid, uint32_t wid, int x, int y) {
	AXUIElementRef w = axFindWindow(pid, wid);
	if (!w) return OPS_ERR_NOT_FOUND;
	int rc = axSetPosition(w, x, y);
	CFRelease(w);
	return rc;
}

static int axResize(pid_t pid, uint32_t wid, int w, int h) {
	AXUIElementRef win = axFindWindow(pid, wid);
	if (!win) return OPS_ERR_NOT_FOUND;
	int rc = axSetSize(win, w, h);
	CFRelease(win);
	return rc;
}

static int axMoveResize(pid_t pid, uint32_t wid, int x, int y, int w, int h) {
	AXUIElementRef win = axFindWindow(pid, wid);
	if (!win) return OPS_ERR_NOT_FOUND;
	int rc = axSetPosition(win, x, y);
	if (rc == OPS_OK) {
		rc = axSetSize(win, w, h);
	}
	CFRelease(win);
	return rc;
}

static int axRaise(pid_t pid, uint32_t wid) {
	AXUIElementRef win = axFindWindow(pid, wid);
	if (!win) return OPS_ERR_NOT_FOUND;
	AXError err = AXUIElementPerformAction(win, kAXRaiseAction);
	CFRelease(win);
	return (err == kAXErrorSuccess) ? OPS_OK : OPS_ERR_AX;
}

static int axFocus(pid_t pid, uint32_t wid) {
	AXUIElementRef win = axFindWindow(pid, wid);
	if (!win) return OPS_ERR_NOT_FOUND;
	AXError err = AXUIElementSetAttributeValue(win, kAXMainAttribute, kCFBooleanTrue);
	if (err == kAXErrorSuccess) {
		AXUIElementSetAttributeValue(win, kAXFocusedAttribute, kCFBooleanTrue);
	}
	CFRelease(win);
	return (err == kAXErrorSuccess) ? OPS_OK : OPS_ERR_AX;
}

static int axMinimize(pid_t pid, uint32_t wid, int yes) {
	AXUIElementRef win = axFindWindow(pid, wid);
	if (!win) return OPS_ERR_NOT_FOUND;
	AXError err = AXUIElementSetAttributeValue(win, kAXMinimizedAttribute, yes ? kCFBooleanTrue : kCFBooleanFalse);
	CFRelease(win);
	return (err == kAXErrorSuccess) ? OPS_OK : OPS_ERR_AX;
}

static int axClose(pid_t pid, uint32_t wid) {
	AXUIElementRef win = axFindWindow(pid, wid);
	if (!win) return OPS_ERR_NOT_FOUND;
	CFTypeRef btnRef = NULL;
	AXError err = AXUIElementCopyAttributeValue(win, kAXCloseButtonAttribute, &btnRef);
	if (err != kAXErrorSuccess || btnRef == NULL) {
		CFRelease(win);
		return OPS_ERR_AX;
	}
	err = AXUIElementPerformAction((AXUIElementRef)btnRef, kAXPressAction);
	CFRelease(btnRef);
	CFRelease(win);
	return (err == kAXErrorSuccess) ? OPS_OK : OPS_ERR_AX;
}

static int axBounds(pid_t pid, uint32_t wid, int *x, int *y, int *w, int *h) {
	AXUIElementRef win = axFindWindow(pid, wid);
	if (!win) return OPS_ERR_NOT_FOUND;
	CFTypeRef posRef = NULL, sizeRef = NULL;
	AXError ep = AXUIElementCopyAttributeValue(win, kAXPositionAttribute, &posRef);
	AXError es = AXUIElementCopyAttributeValue(win, kAXSizeAttribute, &sizeRef);
	int rc = OPS_ERR_AX;
	if (ep == kAXErrorSuccess && es == kAXErrorSuccess && posRef && sizeRef) {
		CGPoint pt;
		CGSize sz;
		if (AXValueGetValue((AXValueRef)posRef, kAXValueCGPointType, &pt) &&
		    AXValueGetValue((AXValueRef)sizeRef, kAXValueCGSizeType, &sz)) {
			*x = (int)pt.x;
			*y = (int)pt.y;
			*w = (int)sz.width;
			*h = (int)sz.height;
			rc = OPS_OK;
		}
	}
	if (posRef) CFRelease(posRef);
	if (sizeRef) CFRelease(sizeRef);
	CFRelease(win);
	return rc;
}
*/
import "C"

import (
	"errors"
	"fmt"

	"github.com/PekingSpades/DeskAct/capture"
	"github.com/PekingSpades/DeskAct/display"
)

// AXTrusted returns true if the current process holds Accessibility
// permission. When prompt is true the OS shows the permission dialog on
// first call.
func AXTrusted(prompt bool) bool {
	p := C.int(0)
	if prompt {
		p = 1
	}
	return C.axTrusted(p) == 1
}

func ensureTrusted() error {
	if AXTrusted(true) {
		return nil
	}
	return fmt.Errorf("%w: accessibility permission required (System Settings -> Privacy & Security -> Accessibility)", capture.ErrPermissionDenied)
}

func mapAXErr(rc C.int) error {
	switch int(rc) {
	case 0:
		return nil
	case 2:
		return capture.ErrWindowNotFound
	case 3:
		return fmt.Errorf("%w: AX call failed", capture.ErrCaptureFailed)
	case 4:
		return errors.New("invalid argument")
	}
	return fmt.Errorf("ax rc=%d", int(rc))
}

func moveWindow(id uint64, pid int32, x, y int) error {
	if err := ensureTrusted(); err != nil {
		return err
	}
	return mapAXErr(C.axMove(C.pid_t(pid), C.uint32_t(id), C.int(x), C.int(y)))
}

func resizeWindow(id uint64, pid int32, w, h int) error {
	if err := ensureTrusted(); err != nil {
		return err
	}
	return mapAXErr(C.axResize(C.pid_t(pid), C.uint32_t(id), C.int(w), C.int(h)))
}

func moveResizeWindow(id uint64, pid int32, x, y, w, h int) error {
	if err := ensureTrusted(); err != nil {
		return err
	}
	return mapAXErr(C.axMoveResize(C.pid_t(pid), C.uint32_t(id), C.int(x), C.int(y), C.int(w), C.int(h)))
}

func raiseWindow(id uint64, pid int32) error {
	if err := ensureTrusted(); err != nil {
		return err
	}
	return mapAXErr(C.axRaise(C.pid_t(pid), C.uint32_t(id)))
}

func focusWindow(id uint64, pid int32) error {
	if err := ensureTrusted(); err != nil {
		return err
	}
	return mapAXErr(C.axFocus(C.pid_t(pid), C.uint32_t(id)))
}

func minimizeWindow(id uint64, pid int32) error {
	if err := ensureTrusted(); err != nil {
		return err
	}
	return mapAXErr(C.axMinimize(C.pid_t(pid), C.uint32_t(id), 1))
}

func restoreWindow(id uint64, pid int32) error {
	if err := ensureTrusted(); err != nil {
		return err
	}
	return mapAXErr(C.axMinimize(C.pid_t(pid), C.uint32_t(id), 0))
}

func closeWindow(id uint64, pid int32) error {
	if err := ensureTrusted(); err != nil {
		return err
	}
	return mapAXErr(C.axClose(C.pid_t(pid), C.uint32_t(id)))
}

func boundsWindow(id uint64, pid int32) (display.Rect, error) {
	if err := ensureTrusted(); err != nil {
		return display.Rect{}, err
	}
	var x, y, w, h C.int
	rc := C.axBounds(C.pid_t(pid), C.uint32_t(id), &x, &y, &w, &h)
	if err := mapAXErr(rc); err != nil {
		return display.Rect{}, err
	}
	return makeRect(int(x), int(y), int(w), int(h)), nil
}
