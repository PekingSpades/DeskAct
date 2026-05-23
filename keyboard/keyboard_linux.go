//go:build linux

package keyboard

/*
#include <stdbool.h>
#include <stdint.h>
// Linux X11 keyTapPid / keyTogglePid / unicodeTypeXID: the C definitions
// live in keypress_c_x11.h (included by keyboard.go's cgo block). Declaring
// just prototypes here lets this Go file's cgo unit link to the one
// definition without duplicating it. Use the same primitive types as
// keypress_c_x11.h: MMKeyCode is `KeySym` (typedef'd to unsigned long on
// X11), MMKeyFlags is `unsigned int`, and uintptr is uintptr_t.
int keyTapPid(unsigned long code, unsigned int flags, uintptr_t xid);
int keyTogglePid(unsigned long code, bool down, unsigned int flags, uintptr_t xid);
void unicodeTypeXID(unsigned value, uintptr_t xid);
*/
import "C"

import (
	"fmt"

	cap "github.com/PekingSpades/DeskAct/capture"
)

func waylandUnsupportedErr() error {
	return fmt.Errorf("%w: wayland session — per-window keyboard injection requires X11", cap.ErrUnsupported)
}

// keyTapForWindowTarget on Linux X11: keyboardWindowTarget returns the
// X11 Window XID. The C keyTapPid entry point on X11 already reinterprets
// its argument as an XID (see key/keypress_c_x11.h: keyTapPid), so call
// it directly without going through KeyTapWithPID — KeyTapWithPID now
// resolves PID -> XID itself and would treat our already-XID input as a
// PID and re-resolve to a different window.
func keyTapForWindowTarget(key string, xid int, modifiers []Modifier, settings KeyboardSettings) error {
	if isWaylandSession() {
		return waylandUnsupportedErr()
	}
	key, modifiers = normalizeKeyAndModifiers(key, modifiers)
	keyCode, err := checkKeyCodes(key)
	if err != nil {
		return err
	}
	flags := modifiersToFlags(modifiers)
	ret := C.keyTapPid(C.ulong(keyCode), C.uint(flags), C.uintptr_t(xid))
	milliSleep(settings.Sleep)
	return keyActionError("keyTapPid", key, xid, ret)
}

func keyToggleForWindowTarget(key string, down bool, xid int, modifiers []Modifier, settings KeyboardSettings) error {
	if isWaylandSession() {
		return waylandUnsupportedErr()
	}
	key, modifiers = normalizeKeyAndModifiers(key, modifiers)
	keyCode, err := checkKeyCodes(key)
	if err != nil {
		return err
	}
	flags := modifiersToFlags(modifiers)
	ret := C.keyTogglePid(C.ulong(keyCode), C.bool(down), C.uint(flags), C.uintptr_t(xid))
	milliSleep(settings.Sleep)
	return keyActionError("keyTogglePid", key, xid, ret)
}

// unicodeTypeXIDPlatform delivers a unicode codepoint to the given X11
// Window XID via XSendEvent (no global focus disturbance).
func unicodeTypeXIDPlatform(value uint32, xid uint64) error {
	C.unicodeTypeXID(C.uint(value), C.uintptr_t(xid))
	return nil
}
