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
	return 0;
}

/*
 * keyToggle - Atomic key toggle (press or release) with modifiers
 *
 * down=true:  Modifiers -> Main key (press order)
 * down=false: Main key -> Modifiers (release order, LIFO)
 */
int keyToggle(MMKeyCode code, const bool down, MMKeyFlags flags) {
	Display *display = XGetMainDisplay();
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
	return 0;
}

/*
 * keyTapPid - Key tap to a specific process (non-atomic, uses PostMessage on Windows)
 */
int keyTapPid(MMKeyCode code, MMKeyFlags flags, uintptr pid) {
	/* X11: no direct PID support, fall back to global */
	return keyTap(code, flags);
}

/*
 * keyTogglePid - Key toggle to a specific process
 */
int keyTogglePid(MMKeyCode code, const bool down, MMKeyFlags flags, uintptr pid) {
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
	return 0;
}
