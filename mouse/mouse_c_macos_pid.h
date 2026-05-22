// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// Non-preemptive per-PID mouse injection on macOS via CGEventPostToPid.
// Coordinates are in global screen space (the caller is expected to map
// window-local coords to screen coords before invoking — see mouse_pid.go).

#ifndef MOUSE_C_MACOS_PID_H
#define MOUSE_C_MACOS_PID_H

#include "mouse.h"
#include <ApplicationServices/ApplicationServices.h>
#include <stdint.h>

#define MM_MOUSE_PID_OK 0
#define MM_MOUSE_PID_ERR_WINDOW -1
#define MM_MOUSE_PID_ERR_BUTTON -2
#define MM_MOUSE_PID_ERR_POST -3
#define MM_MOUSE_PID_ERR_EVENT -4

static CGEventType mouseEventType(MMMouseButton button, int down) {
	if (button == kCGMouseButtonLeft) {
		return down ? kCGEventLeftMouseDown : kCGEventLeftMouseUp;
	}
	if (button == kCGMouseButtonRight) {
		return down ? kCGEventRightMouseDown : kCGEventRightMouseUp;
	}
	return down ? kCGEventOtherMouseDown : kCGEventOtherMouseUp;
}

static int mouseMovePidGo(uintptr_t pid, int screenX, int screenY) {
	if (pid == 0) {
		return MM_MOUSE_PID_ERR_WINDOW;
	}
	CGPoint pt = CGPointMake((CGFloat)screenX, (CGFloat)screenY);
	CGEventRef ev = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, pt, kCGMouseButtonLeft);
	if (ev == NULL) {
		return MM_MOUSE_PID_ERR_EVENT;
	}
	CGEventPostToPid((pid_t)pid, ev);
	CFRelease(ev);
	return MM_MOUSE_PID_OK;
}

static int mouseTogglePidGo(uintptr_t pid, int screenX, int screenY, MMMouseButton button, int down) {
	if (pid == 0) {
		return MM_MOUSE_PID_ERR_WINDOW;
	}
	CGPoint pt = CGPointMake((CGFloat)screenX, (CGFloat)screenY);
	CGEventType type = mouseEventType(button, down);
	CGEventRef ev = CGEventCreateMouseEvent(NULL, type, pt, (CGMouseButton)button);
	if (ev == NULL) {
		return MM_MOUSE_PID_ERR_EVENT;
	}
	CGEventPostToPid((pid_t)pid, ev);
	CFRelease(ev);
	return MM_MOUSE_PID_OK;
}

static int mouseClickPidGo(uintptr_t pid, int screenX, int screenY, MMMouseButton button) {
	int rc = mouseTogglePidGo(pid, screenX, screenY, button, 1);
	if (rc != MM_MOUSE_PID_OK) {
		return rc;
	}
	return mouseTogglePidGo(pid, screenX, screenY, button, 0);
}

/* macOS scroll wheel events accept line-unit values; pixel-unit is emulated
 * by CGEventCreateScrollWheelEvent2 with kCGScrollEventUnitPixel. x and y
 * arguments are ignored — the wheel event is posted to whichever window the
 * targeted process currently owns under the system cursor. */
static int mouseScrollPidGo(uintptr_t pid, int x, int y, int dx, int dy, MMScrollUnit unit) {
	(void)x;
	(void)y;
	if (pid == 0) {
		return MM_MOUSE_PID_ERR_WINDOW;
	}
	CGScrollEventUnit u = (unit == MM_SCROLL_UNIT_LINE)
		? kCGScrollEventUnitLine
		: kCGScrollEventUnitPixel;
	CGEventRef ev = CGEventCreateScrollWheelEvent(NULL, u, 2, dy, dx);
	if (ev == NULL) {
		return MM_MOUSE_PID_ERR_EVENT;
	}
	CGEventPostToPid((pid_t)pid, ev);
	CFRelease(ev);
	return MM_MOUSE_PID_OK;
}

#endif /* MOUSE_C_MACOS_PID_H */
