// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// Non-preemptive per-window mouse injection on X11 via XSendEvent. The
// resulting events carry send_event=True, which some Qt/GTK applications
// detect and ignore; this is a protocol-level limitation.

#ifndef MOUSE_C_X11_PID_H
#define MOUSE_C_X11_PID_H

#include "mouse.h"
#include "../base/xdisplay_c.h"

#include <X11/Xlib.h>
#include <stdint.h>

#define MM_MOUSE_PID_OK 0
#define MM_MOUSE_PID_ERR_WINDOW -1
#define MM_MOUSE_PID_ERR_BUTTON -2
#define MM_MOUSE_PID_ERR_POST -3
#define MM_MOUSE_PID_ERR_DISPLAY -4

static int mouseMovePidGo(uintptr_t xid, int x, int y) {
	Display *dpy = XGetMainDisplay();
	if (dpy == NULL) {
		return MM_MOUSE_PID_ERR_DISPLAY;
	}
	if (xid == 0) {
		return MM_MOUSE_PID_ERR_WINDOW;
	}
	XMotionEvent ev;
	ev.type = MotionNotify;
	ev.display = dpy;
	ev.window = (Window)xid;
	ev.root = DefaultRootWindow(dpy);
	ev.subwindow = None;
	ev.time = CurrentTime;
	ev.x = x;
	ev.y = y;
	ev.x_root = x;
	ev.y_root = y;
	ev.state = 0;
	ev.is_hint = NotifyNormal;
	ev.same_screen = True;
	ev.send_event = True;
	ev.serial = 0;
	if (!XSendEvent(dpy, (Window)xid, True, PointerMotionMask, (XEvent *)&ev)) {
		return MM_MOUSE_PID_ERR_POST;
	}
	XSync(dpy, False);
	return MM_MOUSE_PID_OK;
}

static int mouseTogglePidGo(uintptr_t xid, int x, int y, MMMouseButton button, int down) {
	Display *dpy = XGetMainDisplay();
	if (dpy == NULL) {
		return MM_MOUSE_PID_ERR_DISPLAY;
	}
	if (xid == 0) {
		return MM_MOUSE_PID_ERR_WINDOW;
	}
	if (button < 1 || button > 9) {
		return MM_MOUSE_PID_ERR_BUTTON;
	}
	XButtonEvent ev;
	ev.type = down ? ButtonPress : ButtonRelease;
	ev.display = dpy;
	ev.window = (Window)xid;
	ev.root = DefaultRootWindow(dpy);
	ev.subwindow = None;
	ev.time = CurrentTime;
	ev.x = x;
	ev.y = y;
	ev.x_root = x;
	ev.y_root = y;
	ev.state = 0;
	ev.button = (unsigned int)button;
	ev.same_screen = True;
	ev.send_event = True;
	ev.serial = 0;
	long mask = down ? ButtonPressMask : ButtonReleaseMask;
	if (!XSendEvent(dpy, (Window)xid, True, mask, (XEvent *)&ev)) {
		return MM_MOUSE_PID_ERR_POST;
	}
	XSync(dpy, False);
	return MM_MOUSE_PID_OK;
}

static int mouseClickPidGo(uintptr_t xid, int x, int y, MMMouseButton button) {
	int rc = mouseTogglePidGo(xid, x, y, button, 1);
	if (rc != MM_MOUSE_PID_OK) {
		return rc;
	}
	return mouseTogglePidGo(xid, x, y, button, 0);
}

/*
 * X11 scroll wheel uses synthetic ButtonPress / ButtonRelease events on
 * buttons 4 (wheel up) / 5 (wheel down) / 6 (wheel left) / 7 (wheel right).
 * dy>0 scrolls up. dx>0 scrolls right. Each unit produces one press+release.
 */
static int mouseScrollPidGo(uintptr_t xid, int x, int y, int dx, int dy, MMScrollUnit unit) {
	(void)unit; /* X11 has no native pixel-unit scroll; treat all as lines. */
	int verticalButton = (dy > 0) ? 4 : 5;
	int horizontalButton = (dx > 0) ? 7 : 6;
	int ticks;

	for (ticks = 0; ticks < (dy > 0 ? dy : -dy); ticks++) {
		int rc = mouseClickPidGo(xid, x, y, (MMMouseButton)verticalButton);
		if (rc != MM_MOUSE_PID_OK) {
			return rc;
		}
	}
	for (ticks = 0; ticks < (dx > 0 ? dx : -dx); ticks++) {
		int rc = mouseClickPidGo(xid, x, y, (MMMouseButton)horizontalButton);
		if (rc != MM_MOUSE_PID_OK) {
			return rc;
		}
	}
	return MM_MOUSE_PID_OK;
}

#endif /* MOUSE_C_X11_PID_H */
