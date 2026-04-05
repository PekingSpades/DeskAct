//go:build !windows && !darwin
// +build !windows,!darwin

package keyboardstate

/*
#cgo linux CFLAGS: -I/usr/src
#cgo linux LDFLAGS: -L/usr/src -lm -lX11 -lXtst

#include "../base/xdisplay_c.h"
#include <stdint.h>
#include <X11/XKBlib.h>
#include <X11/keysym.h>

static int deskactKeycodeExists(Display *display, KeySym sym) {
	return XKeysymToKeycode(display, sym) != 0;
}

static int deskactKeyDown(Display *display, const char *keys, KeySym sym) {
	KeyCode code = XKeysymToKeycode(display, sym);
	if (code == 0) {
		return 0;
	}
	return (keys[code / 8] & (1 << (code % 8))) != 0;
}

static int deskactIndicatorState(Display *display, const char *name, Bool *state) {
	Atom atom = XInternAtom(display, name, False);
	if (atom == None) {
		return 0;
	}
	if (XkbGetNamedIndicator(display, atom, NULL, state, NULL, NULL) == True) {
		return 1;
	}
	return 0;
}

static int deskactIndicatorStateAny(Display *display, const char *primary, const char *fallback, Bool *state) {
	if (deskactIndicatorState(display, primary, state)) {
		return 1;
	}
	if (fallback != NULL && deskactIndicatorState(display, fallback, state)) {
		return 1;
	}
	return 0;
}

static int deskactGetModifierState(uint32_t *stateOut, uint32_t *supportedOut) {
	Display *display = XGetMainDisplay();
	if (display == NULL) {
		return -1;
	}

	char keys[32];
	XQueryKeymap(display, keys);

	uint32_t state = 0;
	uint32_t supported = 0;

	if (deskactKeycodeExists(display, XK_Shift_L) || deskactKeycodeExists(display, XK_Shift_R)) {
		supported |= (1u << 0);
		if (deskactKeyDown(display, keys, XK_Shift_L) || deskactKeyDown(display, keys, XK_Shift_R)) {
			state |= (1u << 0);
		}
	}

	if (deskactKeycodeExists(display, XK_Control_L) || deskactKeycodeExists(display, XK_Control_R)) {
		supported |= (1u << 1);
		if (deskactKeyDown(display, keys, XK_Control_L) || deskactKeyDown(display, keys, XK_Control_R)) {
			state |= (1u << 1);
		}
	}

	if (deskactKeycodeExists(display, XK_Alt_L) || deskactKeycodeExists(display, XK_Alt_R)) {
		supported |= (1u << 2);
		if (deskactKeyDown(display, keys, XK_Alt_L) || deskactKeyDown(display, keys, XK_Alt_R)) {
			state |= (1u << 2);
		}
	}

	if (deskactKeycodeExists(display, XK_Super_L) || deskactKeycodeExists(display, XK_Super_R)) {
		supported |= (1u << 3);
		if (deskactKeyDown(display, keys, XK_Super_L) || deskactKeyDown(display, keys, XK_Super_R)) {
			state |= (1u << 3);
		}
	}

	int xkbOpcode, xkbEvent, xkbError;
	int xkbMajor = XkbMajorVersion;
	int xkbMinor = XkbMinorVersion;
	if (XkbQueryExtension(display, &xkbOpcode, &xkbEvent, &xkbError, &xkbMajor, &xkbMinor)) {
		Bool indicator = False;
		if (deskactIndicatorStateAny(display, "Caps Lock", "CapsLock", &indicator)) {
			supported |= (1u << 4);
			if (indicator) {
				state |= (1u << 4);
			}
		}
		if (deskactIndicatorStateAny(display, "Num Lock", "NumLock", &indicator)) {
			supported |= (1u << 5);
			if (indicator) {
				state |= (1u << 5);
			}
		}
		if (deskactIndicatorStateAny(display, "Scroll Lock", "ScrollLock", &indicator)) {
			supported |= (1u << 6);
			if (indicator) {
				state |= (1u << 6);
			}
		}
	}

	*stateOut = state;
	*supportedOut = supported;
	return 0;
}
*/
import "C"

func currentState() (stateSnapshot, error) {
	var state stateSnapshot
	var mask C.uint32_t
	var supported C.uint32_t
	if C.deskactGetModifierState(&mask, &supported) != 0 {
		return stateSnapshot{}, ErrDisplayUnavailable
	}
	state.mask = uint32(mask)
	state.supported = uint32(supported)
	return state, nil
}
