//go:build windows

package keyboard

/*
#include <stdbool.h>
#include <stdint.h>
// keyTapHwnd / keyToggleHwnd are defined in key/keypress_c_windows.h (included
// by keyboard.go's cgo block). We only need the prototypes here so this Go
// file's cgo unit can reference them; the linker resolves to the one
// definition. Use plain primitive types here to avoid cgo's "inconsistent
// definitions for C.MMKeyCode" between cgo units in the same package.
int keyTapHwnd(int code, unsigned int flags, uintptr_t hwndVal);
int keyToggleHwnd(int code, bool down, unsigned int flags, uintptr_t hwndVal);
*/
import "C"

// keyTapForWindowTarget dispatches a per-window key tap on Windows. The
// caller-supplied `id` is the HWND value (per keyboardWindowTarget), so
// route through keyTapHwnd which skips the PID->HWND lookup that keyTapPid
// does. Otherwise the HWND would be treated as a PID and the lookup fails.
func keyTapForWindowTarget(key string, hwnd int, modifiers []Modifier, settings KeyboardSettings) error {
	key, modifiers = normalizeKeyAndModifiers(key, modifiers)

	keyCode, err := checkKeyCodes(key)
	if err != nil {
		return err
	}

	flags := modifiersToFlags(modifiers)
	ret := C.keyTapHwnd(C.int(keyCode), C.uint(flags), C.uintptr_t(hwnd))
	milliSleep(settings.Sleep)
	return keyActionError("keyTapHwnd", key, hwnd, ret)
}

func keyToggleForWindowTarget(key string, down bool, hwnd int, modifiers []Modifier, settings KeyboardSettings) error {
	key, modifiers = normalizeKeyAndModifiers(key, modifiers)

	keyCode, err := checkKeyCodes(key)
	if err != nil {
		return err
	}

	flags := modifiersToFlags(modifiers)
	ret := C.keyToggleHwnd(C.int(keyCode), C.bool(down), C.uint(flags), C.uintptr_t(hwnd))
	milliSleep(settings.Sleep)
	return keyActionError("keyToggleHwnd", key, hwnd, ret)
}
