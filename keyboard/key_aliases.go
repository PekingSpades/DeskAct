package keyboard

var keyAliases = map[string]string{
	Escape:        Esc,
	Control:       Ctrl,
	Printscreen:   Print,
	"command":     Cmd,
	"right_shift": Rshift,
	"numpad_0":    Num0,
	"numpad_1":    Num1,
	"numpad_2":    Num2,
	"numpad_3":    Num3,
	"numpad_4":    Num4,
	"numpad_5":    Num5,
	"numpad_6":    Num6,
	"numpad_7":    Num7,
	"numpad_8":    Num8,
	"numpad_9":    Num9,
	"numpad_lock": NumLock,
}

func canonicalizeKeyName(key string) string {
	if replacement, ok := keyAliases[key]; ok {
		return replacement
	}
	return key
}
