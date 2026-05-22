// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// Non-preemptive per-window mouse injection on Windows via PostMessageW.
// These deliver the event to a specific HWND without touching the user's
// real cursor or focus, mirroring the keyboard PostMessage path already in
// key/keypress_c_windows.h.

#ifndef MOUSE_C_WINDOWS_PID_H
#define MOUSE_C_WINDOWS_PID_H

#include "mouse.h"

#include <windows.h>
#include <stdint.h>

#define MM_MOUSE_PID_OK 0
#define MM_MOUSE_PID_ERR_WINDOW -1
#define MM_MOUSE_PID_ERR_BUTTON -2
#define MM_MOUSE_PID_ERR_POST -3

static int mouseWindowButtonMessages(MMMouseButton button, int down,
                                     UINT *outMsg, WPARAM *outButtonFlag,
                                     WPARAM *outWParamExtra) {
	*outWParamExtra = 0;
	if (button == MM_BUTTON_LEFT) {
		*outMsg = down ? WM_LBUTTONDOWN : WM_LBUTTONUP;
		*outButtonFlag = down ? MK_LBUTTON : 0;
	} else if (button == MM_BUTTON_RIGHT) {
		*outMsg = down ? WM_RBUTTONDOWN : WM_RBUTTONUP;
		*outButtonFlag = down ? MK_RBUTTON : 0;
	} else if (button == MM_BUTTON_MIDDLE) {
		*outMsg = down ? WM_MBUTTONDOWN : WM_MBUTTONUP;
		*outButtonFlag = down ? MK_MBUTTON : 0;
	} else if (button == MM_BUTTON_BACK) {
		*outMsg = down ? WM_XBUTTONDOWN : WM_XBUTTONUP;
		*outButtonFlag = down ? MK_XBUTTON1 : 0;
		*outWParamExtra = (XBUTTON1 << 16);
	} else if (button == MM_BUTTON_FORWARD) {
		*outMsg = down ? WM_XBUTTONDOWN : WM_XBUTTONUP;
		*outButtonFlag = down ? MK_XBUTTON2 : 0;
		*outWParamExtra = (XBUTTON2 << 16);
	} else {
		return MM_MOUSE_PID_ERR_BUTTON;
	}
	return MM_MOUSE_PID_OK;
}

/* Move the mouse cursor (virtual to this window only) by sending WM_MOUSEMOVE. */
static int mouseMovePidGo(uintptr_t hwnd, int x, int y) {
	if (hwnd == 0) {
		return MM_MOUSE_PID_ERR_WINDOW;
	}
	LPARAM lparam = MAKELPARAM((SHORT)x, (SHORT)y);
	if (!PostMessageW((HWND)hwnd, WM_MOUSEMOVE, 0, lparam)) {
		return MM_MOUSE_PID_ERR_POST;
	}
	return MM_MOUSE_PID_OK;
}

/* Press or release a mouse button targeting a specific window. */
static int mouseTogglePidGo(uintptr_t hwnd, int x, int y, MMMouseButton button, int down) {
	if (hwnd == 0) {
		return MM_MOUSE_PID_ERR_WINDOW;
	}
	UINT msg = 0;
	WPARAM buttonFlag = 0;
	WPARAM extra = 0;
	int rc = mouseWindowButtonMessages(button, down, &msg, &buttonFlag, &extra);
	if (rc != MM_MOUSE_PID_OK) {
		return rc;
	}
	WPARAM wparam = buttonFlag | extra;
	LPARAM lparam = MAKELPARAM((SHORT)x, (SHORT)y);
	if (!PostMessageW((HWND)hwnd, msg, wparam, lparam)) {
		return MM_MOUSE_PID_ERR_POST;
	}
	return MM_MOUSE_PID_OK;
}

/* Single click = down + up at the same coordinates. */
static int mouseClickPidGo(uintptr_t hwnd, int x, int y, MMMouseButton button) {
	int rc = mouseTogglePidGo(hwnd, x, y, button, 1);
	if (rc != MM_MOUSE_PID_OK) {
		return rc;
	}
	return mouseTogglePidGo(hwnd, x, y, button, 0);
}

/*
 * Scroll a specific window. WM_MOUSEWHEEL / WM_MOUSEHWHEEL expect lParam to
 * carry SCREEN coordinates, not client coordinates, so we ClientToScreen
 * the supplied (x, y) before posting. unit==0 means LINE; we use WHEEL_DELTA
 * per unit. unit==1 means PIXEL; raw delta is passed as-is (apps that honor
 * fractional scrolling will see it).
 */
static int mouseScrollPidGo(uintptr_t hwnd, int x, int y, int dx, int dy, MMScrollUnit unit) {
	if (hwnd == 0) {
		return MM_MOUSE_PID_ERR_WINDOW;
	}
	POINT pt = { (LONG)x, (LONG)y };
	if (!ClientToScreen((HWND)hwnd, &pt)) {
		pt.x = (LONG)x;
		pt.y = (LONG)y;
	}
	LPARAM lparam = MAKELPARAM((SHORT)pt.x, (SHORT)pt.y);
	int scale = (unit == MM_SCROLL_UNIT_LINE) ? WHEEL_DELTA : 1;

	if (dy != 0) {
		WPARAM w = (WPARAM)(((DWORD)(dy * scale)) << 16);
		if (!PostMessageW((HWND)hwnd, WM_MOUSEWHEEL, w, lparam)) {
			return MM_MOUSE_PID_ERR_POST;
		}
	}
	if (dx != 0) {
		WPARAM w = (WPARAM)(((DWORD)(dx * scale)) << 16);
		if (!PostMessageW((HWND)hwnd, WM_MOUSEHWHEEL, w, lparam)) {
			return MM_MOUSE_PID_ERR_POST;
		}
	}
	return MM_MOUSE_PID_OK;
}

#endif /* MOUSE_C_WINDOWS_PID_H */
