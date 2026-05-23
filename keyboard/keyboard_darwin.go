//go:build darwin

package keyboard

// keyTapForWindowTarget on macOS: keyboardWindowTarget returns the macOS
// PID, and KeyTapWithPID is the correct entry point (calls into the
// CGEventPostToPid C implementation).
func keyTapForWindowTarget(key string, id int, modifiers []Modifier, settings KeyboardSettings) error {
	return KeyTapWithPID(key, id, modifiers, settings)
}

func keyToggleForWindowTarget(key string, down bool, id int, modifiers []Modifier, settings KeyboardSettings) error {
	return KeyToggleWithPID(key, down, id, modifiers, settings)
}

// unicodeTypeXIDPlatform is X11-only; on macOS UnicodeTypeWithWindow
// takes the darwin branch and never calls this. Provided so the
// shared keyboard.go compiles on all platforms.
func unicodeTypeXIDPlatform(value uint32, xid uint64) error {
	return ErrKeyWindowNotFound
}
