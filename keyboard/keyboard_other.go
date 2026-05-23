//go:build !windows

package keyboard

// keyTapForWindowTarget on non-Windows just calls KeyTapWithPID, since:
//   - macOS: the resolved `id` is the macOS PID, and keyTapPid is the
//     correct entry point.
//   - Linux X11: keyTapPid already reinterprets its argument as an XID,
//     which is what keyboardWindowTarget returns on Linux.
func keyTapForWindowTarget(key string, id int, modifiers []Modifier, settings KeyboardSettings) error {
	return KeyTapWithPID(key, id, modifiers, settings)
}

func keyToggleForWindowTarget(key string, down bool, id int, modifiers []Modifier, settings KeyboardSettings) error {
	return KeyToggleWithPID(key, down, id, modifiers, settings)
}
