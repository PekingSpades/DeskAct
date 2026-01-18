package deskact

/*
#include "base/types.h"
#include "base/pubs.h"
#include "key/keypress_c.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unicode"
	"unsafe"
)

// Defining a bunch of constants.
const (
	// KeyA define key "a"
	KeyA = "a"
	KeyB = "b"
	KeyC = "c"
	KeyD = "d"
	KeyE = "e"
	KeyF = "f"
	KeyG = "g"
	KeyH = "h"
	KeyI = "i"
	KeyJ = "j"
	KeyK = "k"
	KeyL = "l"
	KeyM = "m"
	KeyN = "n"
	KeyO = "o"
	KeyP = "p"
	KeyQ = "q"
	KeyR = "r"
	KeyS = "s"
	KeyT = "t"
	KeyU = "u"
	KeyV = "v"
	KeyW = "w"
	KeyX = "x"
	KeyY = "y"
	KeyZ = "z"
	//
	CapA = "A"
	CapB = "B"
	CapC = "C"
	CapD = "D"
	CapE = "E"
	CapF = "F"
	CapG = "G"
	CapH = "H"
	CapI = "I"
	CapJ = "J"
	CapK = "K"
	CapL = "L"
	CapM = "M"
	CapN = "N"
	CapO = "O"
	CapP = "P"
	CapQ = "Q"
	CapR = "R"
	CapS = "S"
	CapT = "T"
	CapU = "U"
	CapV = "V"
	CapW = "W"
	CapX = "X"
	CapY = "Y"
	CapZ = "Z"
	//
	Key0 = "0"
	Key1 = "1"
	Key2 = "2"
	Key3 = "3"
	Key4 = "4"
	Key5 = "5"
	Key6 = "6"
	Key7 = "7"
	Key8 = "8"
	Key9 = "9"

	// Backspace backspace key string
	Backspace = "backspace"
	Delete    = "delete"
	Enter     = "enter"
	Tab       = "tab"
	Esc       = "esc"
	Escape    = "escape"
	Up        = "up"    // Up arrow key
	Down      = "down"  // Down arrow key
	Right     = "right" // Right arrow key
	Left      = "left"  // Left arrow key
	Home      = "home"
	End       = "end"
	Pageup    = "pageup"
	Pagedown  = "pagedown"

	F1  = "f1"
	F2  = "f2"
	F3  = "f3"
	F4  = "f4"
	F5  = "f5"
	F6  = "f6"
	F7  = "f7"
	F8  = "f8"
	F9  = "f9"
	F10 = "f10"
	F11 = "f11"
	F12 = "f12"
	F13 = "f13"
	F14 = "f14"
	F15 = "f15"
	F16 = "f16"
	F17 = "f17"
	F18 = "f18"
	F19 = "f19"
	F20 = "f20"
	F21 = "f21"
	F22 = "f22"
	F23 = "f23"
	F24 = "f24"

	Cmd  = "cmd"  // is the "win" key for windows
	Lcmd = "lcmd" // left command
	Rcmd = "rcmd" // right command
	// "command"
	Alt     = "alt"
	Lalt    = "lalt" // left alt
	Ralt    = "ralt" // right alt
	Ctrl    = "ctrl"
	Lctrl   = "lctrl" // left ctrl
	Rctrl   = "rctrl" // right ctrl
	Control = "control"
	Shift   = "shift"
	Lshift  = "lshift" // left shift
	Rshift  = "rshift" // right shift
	// "right_shift"
	Capslock    = "capslock"
	Space       = "space"
	Print       = "print"
	Printscreen = "printscreen" // No Mac support
	Insert      = "insert"
	Menu        = "menu" // Windows only

	AudioMute    = "audio_mute"     // Mute the volume
	AudioVolDown = "audio_vol_down" // Lower the volume
	AudioVolUp   = "audio_vol_up"   // Increase the volume
	AudioPlay    = "audio_play"
	AudioStop    = "audio_stop"
	AudioPause   = "audio_pause"
	AudioPrev    = "audio_prev"    // Previous Track
	AudioNext    = "audio_next"    // Next Track
	AudioRewind  = "audio_rewind"  // Linux only
	AudioForward = "audio_forward" // Linux only
	AudioRepeat  = "audio_repeat"  // Linux only
	AudioRandom  = "audio_random"  // Linux only

	Num0    = "num0" // numpad 0
	Num1    = "num1"
	Num2    = "num2"
	Num3    = "num3"
	Num4    = "num4"
	Num5    = "num5"
	Num6    = "num6"
	Num7    = "num7"
	Num8    = "num8"
	Num9    = "num9"
	NumLock = "num_lock"

	NumDecimal = "num."
	NumPlus    = "num+"
	NumMinus   = "num-"
	NumMul     = "num*"
	NumDiv     = "num/"
	NumClear   = "num_clear"
	NumEnter   = "num_enter"
	NumEqual   = "num_equal"

	LightsMonUp     = "lights_mon_up"     // Turn up monitor brightness
	LightsMonDown   = "lights_mon_down"   // Turn down monitor brightness
	LightsKbdToggle = "lights_kbd_toggle" // Toggle keyboard backlight on/off
	LightsKbdUp     = "lights_kbd_up"     // Turn up keyboard backlight brightness
	LightsKbdDown   = "lights_kbd_down"
)

// Modifier represents a keyboard modifier key (ctrl, alt, shift, cmd).
type Modifier string

// Modifier key constants.
const (
	ModNone  Modifier = ""
	ModAlt   Modifier = "alt"
	ModCtrl  Modifier = "ctrl"
	ModShift Modifier = "shift"
	ModCmd   Modifier = "cmd" // macOS Command / Windows Win key
)

const keyErrMessage = "Invalid key flag specified."
const keyActionErrMessage = "key action failed"

var ErrKeyActionFailed = errors.New(keyActionErrMessage)

func keyActionError(op, key string, pid int, code C.int) error {
	if code == 0 {
		return nil
	}

	detail := cErrorDetail(code)
	if pid > 0 {
		return fmt.Errorf("%w: %s(%q) pid=%d: %s", ErrKeyActionFailed, op, key, pid, detail)
	}
	return fmt.Errorf("%w: %s(%q): %s", ErrKeyActionFailed, op, key, detail)
}

func cErrorDetail(code C.int) string {
	if code > 0 {
		return fmt.Sprintf("%s (code=%d)", syscall.Errno(code), int(code))
	}
	return fmt.Sprintf("code=%d", int(code))
}

func keyNameMap() map[string]C.MMKeyCode {
	return map[string]C.MMKeyCode{
		"backspace": C.K_BACKSPACE,
		"delete":    C.K_DELETE,
		"enter":     C.K_RETURN,
		"tab":       C.K_TAB,
		"esc":       C.K_ESCAPE,
		"escape":    C.K_ESCAPE,
		"up":        C.K_UP,
		"down":      C.K_DOWN,
		"right":     C.K_RIGHT,
		"left":      C.K_LEFT,
		"home":      C.K_HOME,
		"end":       C.K_END,
		"pageup":    C.K_PAGEUP,
		"pagedown":  C.K_PAGEDOWN,
		//
		"f1":  C.K_F1,
		"f2":  C.K_F2,
		"f3":  C.K_F3,
		"f4":  C.K_F4,
		"f5":  C.K_F5,
		"f6":  C.K_F6,
		"f7":  C.K_F7,
		"f8":  C.K_F8,
		"f9":  C.K_F9,
		"f10": C.K_F10,
		"f11": C.K_F11,
		"f12": C.K_F12,
		"f13": C.K_F13,
		"f14": C.K_F14,
		"f15": C.K_F15,
		"f16": C.K_F16,
		"f17": C.K_F17,
		"f18": C.K_F18,
		"f19": C.K_F19,
		"f20": C.K_F20,
		"f21": C.K_F21,
		"f22": C.K_F22,
		"f23": C.K_F23,
		"f24": C.K_F24,
		//
		"cmd":         C.K_META,
		"lcmd":        C.K_LMETA,
		"rcmd":        C.K_RMETA,
		"command":     C.K_META,
		"alt":         C.K_ALT,
		"lalt":        C.K_LALT,
		"ralt":        C.K_RALT,
		"ctrl":        C.K_CONTROL,
		"lctrl":       C.K_LCONTROL,
		"rctrl":       C.K_RCONTROL,
		"control":     C.K_CONTROL,
		"shift":       C.K_SHIFT,
		"lshift":      C.K_LSHIFT,
		"rshift":      C.K_RSHIFT,
		"right_shift": C.K_RSHIFT,
		"capslock":    C.K_CAPSLOCK,
		"space":       C.K_SPACE,
		"print":       C.K_PRINTSCREEN,
		"printscreen": C.K_PRINTSCREEN,
		"insert":      C.K_INSERT,
		"menu":        C.K_MENU,

		"audio_mute":     C.K_AUDIO_VOLUME_MUTE,
		"audio_vol_down": C.K_AUDIO_VOLUME_DOWN,
		"audio_vol_up":   C.K_AUDIO_VOLUME_UP,
		"audio_play":     C.K_AUDIO_PLAY,
		"audio_stop":     C.K_AUDIO_STOP,
		"audio_pause":    C.K_AUDIO_PAUSE,
		"audio_prev":     C.K_AUDIO_PREV,
		"audio_next":     C.K_AUDIO_NEXT,
		"audio_rewind":   C.K_AUDIO_REWIND,
		"audio_forward":  C.K_AUDIO_FORWARD,
		"audio_repeat":   C.K_AUDIO_REPEAT,
		"audio_random":   C.K_AUDIO_RANDOM,

		"num0":     C.K_NUMPAD_0,
		"num1":     C.K_NUMPAD_1,
		"num2":     C.K_NUMPAD_2,
		"num3":     C.K_NUMPAD_3,
		"num4":     C.K_NUMPAD_4,
		"num5":     C.K_NUMPAD_5,
		"num6":     C.K_NUMPAD_6,
		"num7":     C.K_NUMPAD_7,
		"num8":     C.K_NUMPAD_8,
		"num9":     C.K_NUMPAD_9,
		"num_lock": C.K_NUMPAD_LOCK,

		// todo: removed
		"numpad_0":    C.K_NUMPAD_0,
		"numpad_1":    C.K_NUMPAD_1,
		"numpad_2":    C.K_NUMPAD_2,
		"numpad_3":    C.K_NUMPAD_3,
		"numpad_4":    C.K_NUMPAD_4,
		"numpad_5":    C.K_NUMPAD_5,
		"numpad_6":    C.K_NUMPAD_6,
		"numpad_7":    C.K_NUMPAD_7,
		"numpad_8":    C.K_NUMPAD_8,
		"numpad_9":    C.K_NUMPAD_9,
		"numpad_lock": C.K_NUMPAD_LOCK,

		"num.":      C.K_NUMPAD_DECIMAL,
		"num+":      C.K_NUMPAD_PLUS,
		"num-":      C.K_NUMPAD_MINUS,
		"num*":      C.K_NUMPAD_MUL,
		"num/":      C.K_NUMPAD_DIV,
		"num_clear": C.K_NUMPAD_CLEAR,
		"num_enter": C.K_NUMPAD_ENTER,
		"num_equal": C.K_NUMPAD_EQUAL,

		"lights_mon_up":     C.K_LIGHTS_MON_UP,
		"lights_mon_down":   C.K_LIGHTS_MON_DOWN,
		"lights_kbd_toggle": C.K_LIGHTS_KBD_TOGGLE,
		"lights_kbd_up":     C.K_LIGHTS_KBD_UP,
		"lights_kbd_down":   C.K_LIGHTS_KBD_DOWN,
	}
}

// CmdCtrl If the operating system is macOS, return the key string "cmd",
// otherwise return the key string "ctrl".
func CmdCtrl() string {
	if runtime.GOOS == "darwin" {
		return "cmd"
	}
	return "ctrl"
}

func checkKeyCodes(k string) (key C.MMKeyCode, err error) {
	if k == "" {
		return
	}

	if len(k) == 1 {
		val1 := C.CString(k)
		defer C.free(unsafe.Pointer(val1))

		key = C.keyCodeForChar(*val1)
		if key == C.K_NOT_A_KEY {
			err = errors.New(keyErrMessage)
			return
		}
		return
	}

	if v, ok := keyNameMap()[k]; ok {
		key = v
		if key == C.K_NOT_A_KEY {
			err = errors.New(keyErrMessage)
			return
		}
	}
	return
}

// modifiersToFlags converts Modifier slice to C.MMKeyFlags.
func modifiersToFlags(modifiers []Modifier) C.MMKeyFlags {
	var flags C.MMKeyFlags = C.MOD_NONE
	for _, m := range modifiers {
		switch m {
		case ModAlt:
			flags |= C.MOD_ALT
		case ModCtrl:
			flags |= C.MOD_CONTROL
		case ModShift:
			flags |= C.MOD_SHIFT
		case ModCmd:
			flags |= C.MOD_META
		}
	}
	return flags
}

func appendModifier(modifiers []Modifier, modifier Modifier) []Modifier {
	for _, m := range modifiers {
		if m == modifier {
			return modifiers
		}
	}
	return append(modifiers, modifier)
}

func normalizeKeyAndModifiers(key string, modifiers []Modifier) (string, []Modifier) {
	if len(key) == 1 && unicode.IsUpper([]rune(key)[0]) {
		modifiers = appendModifier(modifiers, ModShift)
		key = strings.ToLower(key)
	}

	if replacement, ok := lookupSpecialKey(key); ok {
		key = replacement
		modifiers = appendModifier(modifiers, ModShift)
	}

	return key, modifiers
}

// KeyTap taps the keyboard code with optional modifier keys (atomic operation).
func KeyTap(key string, modifiers []Modifier, settings KeyboardSettings) error {
	key, modifiers = normalizeKeyAndModifiers(key, modifiers)

	keyCode, err := checkKeyCodes(key)
	if err != nil {
		return err
	}

	flags := modifiersToFlags(modifiers)

	// Atomic operation - C layer handles platform differences.
	ret := C.keyTap(keyCode, flags)

	MilliSleep(settings.Sleep)
	return keyActionError("keyTap", key, 0, ret)
}

// KeyTapWithPID taps the keyboard code on a specific process.
func KeyTapWithPID(key string, pid int, modifiers []Modifier, settings KeyboardSettings) error {
	key, modifiers = normalizeKeyAndModifiers(key, modifiers)

	keyCode, err := checkKeyCodes(key)
	if err != nil {
		return err
	}

	flags := modifiersToFlags(modifiers)

	// PID-specific operation.
	ret := C.keyTapPid(keyCode, flags, C.uintptr(pid))

	MilliSleep(settings.Sleep)
	return keyActionError("keyTapPid", key, pid, ret)
}

// KeyToggle toggles a key up or down with optional modifier keys (atomic operation).
func KeyToggle(key string, down bool, modifiers []Modifier, settings KeyboardSettings) error {
	key, modifiers = normalizeKeyAndModifiers(key, modifiers)

	keyCode, err := checkKeyCodes(key)
	if err != nil {
		return err
	}

	flags := modifiersToFlags(modifiers)

	// Atomic operation - C layer handles platform differences.
	ret := C.keyToggle(keyCode, C.bool(down), flags)

	MilliSleep(settings.Sleep)
	return keyActionError("keyToggle", key, 0, ret)
}

// KeyToggleWithPID toggles a key on a specific process.
func KeyToggleWithPID(key string, down bool, pid int, modifiers []Modifier, settings KeyboardSettings) error {
	key, modifiers = normalizeKeyAndModifiers(key, modifiers)

	keyCode, err := checkKeyCodes(key)
	if err != nil {
		return err
	}

	flags := modifiersToFlags(modifiers)

	// PID-specific operation.
	ret := C.keyTogglePid(keyCode, C.bool(down), flags, C.uintptr(pid))

	MilliSleep(settings.Sleep)
	return keyActionError("keyTogglePid", key, pid, ret)
}

// KeyPress press and release a key with random delay (more human-like).
func KeyPress(key string, modifiers []Modifier, settings KeyboardSettings) error {
	err := KeyToggle(key, true, modifiers, settings)
	if err != nil {
		return err
	}

	MilliSleep(1 + rand.Intn(3))
	return KeyToggle(key, false, modifiers, settings)
}

// CharCodeAt char code at utf-8.
func CharCodeAt(s string, n int) rune {
	i := 0
	for _, r := range s {
		if i == n {
			return r
		}
		i++
	}

	return 0
}

// UnicodeType tap the uint32 unicode.
func UnicodeType(str uint32, pid int, isPid bool) {
	cstr := C.uint(str)
	isPidFlag := C.int8_t(0)
	if isPid {
		isPidFlag = C.int8_t(1)
	}
	C.unicodeType(cstr, C.uintptr(pid), isPidFlag)
}

// ToUC trans string to unicode []string.
func ToUC(text string) []string {
	var uc []string

	for _, r := range text {
		textQ := strconv.QuoteToASCII(string(r))
		textUnQ := textQ[1 : len(textQ)-1]

		st := strings.Replace(textUnQ, "\\u", "U", -1)
		if st == "\\\\" {
			st = "\\"
		}
		if st == `\"` {
			st = `"`
		}
		uc = append(uc, st)
	}

	return uc
}

func inputUTF(str string) {
	cstr := C.CString(str)
	C.input_utf(cstr)

	C.free(unsafe.Pointer(cstr))
}

// Type type a string (supported UTF-8).
func Type(str string, pid int, settings KeyboardSettings) {
	tm := settings.TypeDelay
	tm1 := settings.TypeUTFDelay

	if runtime.GOOS == "linux" {
		strUc := ToUC(str)
		for i := 0; i < len(strUc); i++ {
			ru := []rune(strUc[i])
			if len(ru) <= 1 {
				ustr := uint32(CharCodeAt(strUc[i], 0))
				UnicodeType(ustr, pid, false)
			} else {
				inputUTF(strUc[i])
				MilliSleep(tm1)
			}

			MilliSleep(tm)
		}
		return
	}

	for i := 0; i < len([]rune(str)); i++ {
		ustr := uint32(CharCodeAt(str, i))
		UnicodeType(ustr, pid, false)
		MilliSleep(tm)
	}
	MilliSleep(settings.Sleep)
}

// TypeDelay type string with delayed.
func TypeDelay(str string, pid int, delay int, settings KeyboardSettings) {
	Type(str, pid, settings)
	MilliSleep(delay)
}
