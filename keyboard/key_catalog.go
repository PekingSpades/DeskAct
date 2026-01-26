package keyboard

var keyNames = []string{
	KeyA, KeyB, KeyC, KeyD, KeyE, KeyF, KeyG, KeyH, KeyI, KeyJ, KeyK, KeyL, KeyM,
	KeyN, KeyO, KeyP, KeyQ, KeyR, KeyS, KeyT, KeyU, KeyV, KeyW, KeyX, KeyY, KeyZ,
	CapA, CapB, CapC, CapD, CapE, CapF, CapG, CapH, CapI, CapJ, CapK, CapL, CapM,
	CapN, CapO, CapP, CapQ, CapR, CapS, CapT, CapU, CapV, CapW, CapX, CapY, CapZ,
	Key0, Key1, Key2, Key3, Key4, Key5, Key6, Key7, Key8, Key9,
	Backspace, Delete, Enter, Tab, Esc, Up, Down, Right, Left, Home, End,
	Pageup, Pagedown,
	F1, F2, F3, F4, F5, F6, F7, F8, F9, F10, F11, F12, F13, F14, F15, F16, F17,
	F18, F19, F20, F21, F22, F23, F24,
	Cmd, Lcmd, Rcmd, Alt, Lalt, Ralt, Ctrl, Lctrl, Rctrl, Shift, Lshift, Rshift,
	Capslock, Space, Print, Insert, Menu,
	AudioMute, AudioVolDown, AudioVolUp, AudioPlay, AudioStop, AudioPause, AudioPrev,
	AudioNext, AudioRewind, AudioForward, AudioRepeat, AudioRandom,
	Num0, Num1, Num2, Num3, Num4, Num5, Num6, Num7, Num8, Num9, NumLock,
	NumDecimal, NumPlus, NumMinus, NumMul, NumDiv, NumClear, NumEnter, NumEqual,
	LightsMonUp, LightsMonDown, LightsKbdToggle, LightsKbdUp, LightsKbdDown,
}

var modifierNames = []Modifier{ModNone, ModAlt, ModCtrl, ModShift, ModCmd}

// KeyNames returns a copy of the exported key name constants.
func KeyNames() []string {
	names := make([]string, len(keyNames))
	copy(names, keyNames)
	return names
}

// SupportedKeyNames returns the key names supported on the current platform.
func SupportedKeyNames() []string {
	names := make([]string, 0, len(keyNames))
	for _, name := range keyNames {
		normalized, _ := normalizeKeyAndModifiers(name, nil)
		if _, err := checkKeyCodes(normalized); err == nil {
			names = append(names, name)
		}
	}
	return names
}

// ModifierNames returns a copy of the supported modifier names.
func ModifierNames() []Modifier {
	names := make([]Modifier, len(modifierNames))
	copy(names, modifierNames)
	return names
}
