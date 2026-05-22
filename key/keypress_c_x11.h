// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// See the COPYRIGHT file at the top-level directory of this distribution and at
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0>
//
// This file may not be copied, modified, or distributed
// except according to those terms.

#include "../base/deadbeef_rand_c.h"
#include "../base/microsleep.h"
#include "../base/xdisplay_c.h"
#include "keypress.h"
#include "keycode_c.h"

#include <ctype.h> /* For isupper() */
#include <X11/extensions/XTest.h>

/*
 * keyTap - Atomic key tap (press + release) with modifiers
 *
 * Press order:  Modifiers -> Main key
 * Release order: Main key -> Modifiers (LIFO)
 */
int keyTap(MMKeyCode code, MMKeyFlags flags) {
	Display *display = XGetMainDisplay();
	if (display == NULL) {
		return MM_KEY_ERR_DISPLAY;
	}

	/* Press: modifiers -> main key */
	if (flags & MOD_META) {
		XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_META), True, CurrentTime);
	}
	if (flags & MOD_ALT) {
		XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_ALT), True, CurrentTime);
	}
	if (flags & MOD_CONTROL) {
		XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_CONTROL), True, CurrentTime);
	}
	if (flags & MOD_SHIFT) {
		XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_SHIFT), True, CurrentTime);
	}
	XTestFakeKeyEvent(display, XKeysymToKeycode(display, code), True, CurrentTime);

	/* Release: main key -> modifiers (LIFO) */
	XTestFakeKeyEvent(display, XKeysymToKeycode(display, code), False, CurrentTime);
	if (flags & MOD_SHIFT) {
		XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_SHIFT), False, CurrentTime);
	}
	if (flags & MOD_CONTROL) {
		XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_CONTROL), False, CurrentTime);
	}
	if (flags & MOD_ALT) {
		XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_ALT), False, CurrentTime);
	}
	if (flags & MOD_META) {
		XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_META), False, CurrentTime);
	}

	XSync(display, false);
	return MM_KEY_OK;
}

/*
 * keyToggle - Atomic key toggle (press or release) with modifiers
 *
 * down=true:  Modifiers -> Main key (press order)
 * down=false: Main key -> Modifiers (release order, LIFO)
 */
int keyToggle(MMKeyCode code, const bool down, MMKeyFlags flags) {
	Display *display = XGetMainDisplay();
	if (display == NULL) {
		return MM_KEY_ERR_DISPLAY;
	}
	const Bool is_press = down ? True : False;

	if (down) {
		/* Press: modifiers -> main key */
		if (flags & MOD_META) {
			XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_META), is_press, CurrentTime);
		}
		if (flags & MOD_ALT) {
			XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_ALT), is_press, CurrentTime);
		}
		if (flags & MOD_CONTROL) {
			XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_CONTROL), is_press, CurrentTime);
		}
		if (flags & MOD_SHIFT) {
			XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_SHIFT), is_press, CurrentTime);
		}
		XTestFakeKeyEvent(display, XKeysymToKeycode(display, code), is_press, CurrentTime);
	} else {
		/* Release: main key -> modifiers (LIFO) */
		XTestFakeKeyEvent(display, XKeysymToKeycode(display, code), is_press, CurrentTime);
		if (flags & MOD_SHIFT) {
			XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_SHIFT), is_press, CurrentTime);
		}
		if (flags & MOD_CONTROL) {
			XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_CONTROL), is_press, CurrentTime);
		}
		if (flags & MOD_ALT) {
			XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_ALT), is_press, CurrentTime);
		}
		if (flags & MOD_META) {
			XTestFakeKeyEvent(display, XKeysymToKeycode(display, K_META), is_press, CurrentTime);
		}
	}

	XSync(display, false);
	return MM_KEY_OK;
}

/*
 * keyTapXID - Send a synthetic KeyPress + KeyRelease to a specific X11 window.
 *
 * Uses XSendEvent so the user's actual focus is not disturbed. Modifier flags
 * are encoded in the event's `state` field rather than as separate fake key
 * events, because XSendEvent does not update the server-side modifier state.
 *
 * Caveat: applications may inspect the `send_event` field on the resulting
 * XKeyEvent and ignore synthetic input. This is a protocol-level limitation
 * that no client-side workaround can defeat.
 */
static int keyTapXID(MMKeyCode code, MMKeyFlags flags, unsigned long xid) {
	Display *display = XGetMainDisplay();
	if (display == NULL) {
		return MM_KEY_ERR_DISPLAY;
	}
	if (xid == 0) {
		return MM_KEY_ERR_WINDOW;
	}

	KeyCode keycode = XKeysymToKeycode(display, code);
	if (keycode == 0) {
		return MM_KEY_ERR_EVENT;
	}

	Window root = DefaultRootWindow(display);

	XKeyEvent press;
	press.type = KeyPress;
	press.display = display;
	press.window = (Window)xid;
	press.root = root;
	press.subwindow = None;
	press.time = CurrentTime;
	press.x = 1;
	press.y = 1;
	press.x_root = 1;
	press.y_root = 1;
	press.state = (unsigned int)flags;
	press.keycode = keycode;
	press.same_screen = True;
	press.send_event = True;
	press.serial = 0;

	if (!XSendEvent(display, (Window)xid, True, KeyPressMask, (XEvent *)&press)) {
		return MM_KEY_ERR_POST;
	}

	XKeyEvent release = press;
	release.type = KeyRelease;
	if (!XSendEvent(display, (Window)xid, True, KeyReleaseMask, (XEvent *)&release)) {
		return MM_KEY_ERR_POST;
	}

	XSync(display, False);
	return MM_KEY_OK;
}

/*
 * keyToggleXID - Send a synthetic KeyPress OR KeyRelease (not both) to a
 * specific X11 window.
 */
static int keyToggleXID(MMKeyCode code, const bool down, MMKeyFlags flags, unsigned long xid) {
	Display *display = XGetMainDisplay();
	if (display == NULL) {
		return MM_KEY_ERR_DISPLAY;
	}
	if (xid == 0) {
		return MM_KEY_ERR_WINDOW;
	}
	KeyCode keycode = XKeysymToKeycode(display, code);
	if (keycode == 0) {
		return MM_KEY_ERR_EVENT;
	}

	XKeyEvent ev;
	ev.type = down ? KeyPress : KeyRelease;
	ev.display = display;
	ev.window = (Window)xid;
	ev.root = DefaultRootWindow(display);
	ev.subwindow = None;
	ev.time = CurrentTime;
	ev.x = 1;
	ev.y = 1;
	ev.x_root = 1;
	ev.y_root = 1;
	ev.state = (unsigned int)flags;
	ev.keycode = keycode;
	ev.same_screen = True;
	ev.send_event = True;
	ev.serial = 0;

	long mask = down ? KeyPressMask : KeyReleaseMask;
	if (!XSendEvent(display, (Window)xid, True, mask, (XEvent *)&ev)) {
		return MM_KEY_ERR_POST;
	}
	XSync(display, False);
	return MM_KEY_OK;
}

/*
 * keyTapPid - On X11 the historical "pid" parameter is reinterpreted as an
 * X11 Window XID (matching Windows where the same parameter is HWND). When
 * xid==0 we fall back to the global XTest path so legacy callers that pass
 * a real PID without first resolving it to a window still observe input.
 */
int keyTapPid(MMKeyCode code, MMKeyFlags flags, uintptr pid) {
	if (pid != 0) {
		return keyTapXID(code, flags, (unsigned long)pid);
	}
	return keyTap(code, flags);
}

/*
 * keyTogglePid - X11 reinterpretation: pid is treated as a Window XID.
 */
int keyTogglePid(MMKeyCode code, const bool down, MMKeyFlags flags, uintptr pid) {
	if (pid != 0) {
		return keyToggleXID(code, down, flags, (unsigned long)pid);
	}
	return keyToggle(code, down, flags);
}

/*
 * Legacy functions for compatibility
 */
bool toUpper(char c) {
	if (isupper(c)) {
		return true;
	}
	char *special = "~!@#$%^&*()_+{}|:\"<>?";
	while (*special) {
		if (*special == c) {
			return true;
		}
		special++;
	}
	return false;
}

void toggleKey(char c, const bool down, MMKeyFlags flags, uintptr pid) {
	MMKeyCode keyCode = keyCodeForChar(c);

	if (toUpper(c) && !(flags & MOD_SHIFT)) {
		flags |= MOD_SHIFT;
	}

	if (pid != 0) {
		keyTogglePid(keyCode, down, flags, pid);
	} else {
		keyToggle(keyCode, down, flags);
	}
}

#define toggleUniKey(c, down) toggleKey(c, down, MOD_NONE, 0)

void unicodeType(const unsigned value, uintptr pid, int8_t isPid) {
	toggleUniKey(value, true);
	microsleep(5.0);
	toggleUniKey(value, false);
}

int input_utf(const char *utf) {
	Display *dpy = XOpenDisplay(NULL);
	if (dpy == NULL) {
		return MM_KEY_ERR_DISPLAY;
	}
	KeySym sym = XStringToKeysym(utf);

	int min, max, numcodes;
	XDisplayKeycodes(dpy, &min, &max);
	KeySym *keysym;
	keysym = XGetKeyboardMapping(dpy, min, max-min+1, &numcodes);
	keysym[(max-min-1)*numcodes] = sym;
	XChangeKeyboardMapping(dpy, min, numcodes, keysym, (max-min));
	XFree(keysym);
	XFlush(dpy);

	KeyCode code = XKeysymToKeycode(dpy, sym);
	XTestFakeKeyEvent(dpy, code, True, 1);
	XTestFakeKeyEvent(dpy, code, False, 1);

	XFlush(dpy);
	XCloseDisplay(dpy);
	return MM_KEY_OK;
}
