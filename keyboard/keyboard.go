package keyboard

/*
#include "../base/types.h"
#include "../base/pubs.h"
#include "../key/keypress_c.h"
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

	cap "github.com/PekingSpades/DeskAct/capture"
)

const keyErrMessage = "Invalid key flag specified."
const keyActionErrMessage = "key action failed"

var (
	ErrInvalidKey            = errors.New(keyErrMessage)
	ErrUnsupportedKey        = errors.New("key not supported on this platform")
	ErrKeyActionFailed       = errors.New(keyActionErrMessage)
	ErrKeyEventFailed        = errors.New("keyboard event creation failed")
	ErrKeyDisplayUnavailable = errors.New("display not available")
	ErrKeyWindowNotFound     = errors.New("window not found")
	ErrKeyPostFailed         = errors.New("key event post failed")
)

type mmKeyCode = C.MMKeyCode

const (
	mmKeyBackspace  mmKeyCode = C.K_BACKSPACE
	mmKeyDelete     mmKeyCode = C.K_DELETE
	mmKeyReturn     mmKeyCode = C.K_RETURN
	mmKeyTab        mmKeyCode = C.K_TAB
	mmKeyEscape     mmKeyCode = C.K_ESCAPE
	mmKeyUp         mmKeyCode = C.K_UP
	mmKeyDown       mmKeyCode = C.K_DOWN
	mmKeyRight      mmKeyCode = C.K_RIGHT
	mmKeyLeft       mmKeyCode = C.K_LEFT
	mmKeyHome       mmKeyCode = C.K_HOME
	mmKeyEnd        mmKeyCode = C.K_END
	mmKeyPageUp     mmKeyCode = C.K_PAGEUP
	mmKeyPageDown   mmKeyCode = C.K_PAGEDOWN
	mmKeyF1         mmKeyCode = C.K_F1
	mmKeyF2         mmKeyCode = C.K_F2
	mmKeyF3         mmKeyCode = C.K_F3
	mmKeyF4         mmKeyCode = C.K_F4
	mmKeyF5         mmKeyCode = C.K_F5
	mmKeyF6         mmKeyCode = C.K_F6
	mmKeyF7         mmKeyCode = C.K_F7
	mmKeyF8         mmKeyCode = C.K_F8
	mmKeyF9         mmKeyCode = C.K_F9
	mmKeyF10        mmKeyCode = C.K_F10
	mmKeyF11        mmKeyCode = C.K_F11
	mmKeyF12        mmKeyCode = C.K_F12
	mmKeyF13        mmKeyCode = C.K_F13
	mmKeyF14        mmKeyCode = C.K_F14
	mmKeyF15        mmKeyCode = C.K_F15
	mmKeyF16        mmKeyCode = C.K_F16
	mmKeyF17        mmKeyCode = C.K_F17
	mmKeyF18        mmKeyCode = C.K_F18
	mmKeyF19        mmKeyCode = C.K_F19
	mmKeyF20        mmKeyCode = C.K_F20
	mmKeyF21        mmKeyCode = C.K_F21
	mmKeyF22        mmKeyCode = C.K_F22
	mmKeyF23        mmKeyCode = C.K_F23
	mmKeyF24        mmKeyCode = C.K_F24
	mmKeyMeta       mmKeyCode = C.K_META
	mmKeyLMeta      mmKeyCode = C.K_LMETA
	mmKeyRMeta      mmKeyCode = C.K_RMETA
	mmKeyAlt        mmKeyCode = C.K_ALT
	mmKeyLAlt       mmKeyCode = C.K_LALT
	mmKeyRAlt       mmKeyCode = C.K_RALT
	mmKeyControl    mmKeyCode = C.K_CONTROL
	mmKeyLControl   mmKeyCode = C.K_LCONTROL
	mmKeyRControl   mmKeyCode = C.K_RCONTROL
	mmKeyShift      mmKeyCode = C.K_SHIFT
	mmKeyLShift     mmKeyCode = C.K_LSHIFT
	mmKeyRShift     mmKeyCode = C.K_RSHIFT
	mmKeyCapsLock   mmKeyCode = C.K_CAPSLOCK
	mmKeySpace      mmKeyCode = C.K_SPACE
	mmKeyPrint      mmKeyCode = C.K_PRINTSCREEN
	mmKeyInsert     mmKeyCode = C.K_INSERT
	mmKeyMenu       mmKeyCode = C.K_MENU
	mmKeyAudioMute  mmKeyCode = C.K_AUDIO_VOLUME_MUTE
	mmKeyAudioDown  mmKeyCode = C.K_AUDIO_VOLUME_DOWN
	mmKeyAudioUp    mmKeyCode = C.K_AUDIO_VOLUME_UP
	mmKeyAudioPlay  mmKeyCode = C.K_AUDIO_PLAY
	mmKeyAudioStop  mmKeyCode = C.K_AUDIO_STOP
	mmKeyAudioPause mmKeyCode = C.K_AUDIO_PAUSE
	mmKeyAudioPrev  mmKeyCode = C.K_AUDIO_PREV
	mmKeyAudioNext  mmKeyCode = C.K_AUDIO_NEXT
	mmKeyAudioRew   mmKeyCode = C.K_AUDIO_REWIND
	mmKeyAudioFwd   mmKeyCode = C.K_AUDIO_FORWARD
	mmKeyAudioRep   mmKeyCode = C.K_AUDIO_REPEAT
	mmKeyAudioRand  mmKeyCode = C.K_AUDIO_RANDOM
	mmKeyNum0       mmKeyCode = C.K_NUMPAD_0
	mmKeyNum1       mmKeyCode = C.K_NUMPAD_1
	mmKeyNum2       mmKeyCode = C.K_NUMPAD_2
	mmKeyNum3       mmKeyCode = C.K_NUMPAD_3
	mmKeyNum4       mmKeyCode = C.K_NUMPAD_4
	mmKeyNum5       mmKeyCode = C.K_NUMPAD_5
	mmKeyNum6       mmKeyCode = C.K_NUMPAD_6
	mmKeyNum7       mmKeyCode = C.K_NUMPAD_7
	mmKeyNum8       mmKeyCode = C.K_NUMPAD_8
	mmKeyNum9       mmKeyCode = C.K_NUMPAD_9
	mmKeyNumLock    mmKeyCode = C.K_NUMPAD_LOCK
	mmKeyNumDecimal mmKeyCode = C.K_NUMPAD_DECIMAL
	mmKeyNumPlus    mmKeyCode = C.K_NUMPAD_PLUS
	mmKeyNumMinus   mmKeyCode = C.K_NUMPAD_MINUS
	mmKeyNumMul     mmKeyCode = C.K_NUMPAD_MUL
	mmKeyNumDiv     mmKeyCode = C.K_NUMPAD_DIV
	mmKeyNumClear   mmKeyCode = C.K_NUMPAD_CLEAR
	mmKeyNumEnter   mmKeyCode = C.K_NUMPAD_ENTER
	mmKeyNumEqual   mmKeyCode = C.K_NUMPAD_EQUAL
	mmKeyMonUp      mmKeyCode = C.K_LIGHTS_MON_UP
	mmKeyMonDown    mmKeyCode = C.K_LIGHTS_MON_DOWN
	mmKeyKbdToggle  mmKeyCode = C.K_LIGHTS_KBD_TOGGLE
	mmKeyKbdUp      mmKeyCode = C.K_LIGHTS_KBD_UP
	mmKeyKbdDown    mmKeyCode = C.K_LIGHTS_KBD_DOWN
)

type KeyError struct {
	Key  string
	OS   string
	Kind error
}

func (e *KeyError) Error() string {
	if e.OS != "" {
		return fmt.Sprintf("%s: %q (os=%s)", e.Kind, e.Key, e.OS)
	}
	return fmt.Sprintf("%s: %q", e.Kind, e.Key)
}

func (e *KeyError) Unwrap() error {
	return e.Kind
}

func keyLookupError(kind error, key string) error {
	return &KeyError{
		Key:  key,
		OS:   runtime.GOOS,
		Kind: kind,
	}
}

type KeyActionError struct {
	Op   string
	Key  string
	PID  int
	Code int
	Kind error
}

func (e *KeyActionError) Error() string {
	detail := cErrorDetail(C.int(e.Code))
	if e.PID > 0 {
		return fmt.Sprintf("%s(%q) pid=%d: %s", e.Op, e.Key, e.PID, detail)
	}
	return fmt.Sprintf("%s(%q): %s", e.Op, e.Key, detail)
}

func (e *KeyActionError) Unwrap() error {
	return e.Kind
}

func keyActionError(op, key string, pid int, code C.int) error {
	if code == 0 {
		return nil
	}

	kind := ErrKeyActionFailed
	if specific := cErrorKind(code); specific != nil {
		kind = errors.Join(ErrKeyActionFailed, specific)
	}

	return &KeyActionError{
		Op:   op,
		Key:  key,
		PID:  pid,
		Code: int(code),
		Kind: kind,
	}
}

func cErrorDetail(code C.int) string {
	if code < 0 {
		if kind := cErrorKind(code); kind != nil {
			return kind.Error()
		}
		return fmt.Sprintf("code=%d", int(code))
	}
	if code > 0 {
		return fmt.Sprintf("%s (code=%d)", syscall.Errno(code), int(code))
	}
	return fmt.Sprintf("code=%d", int(code))
}

func cErrorKind(code C.int) error {
	switch code {
	case C.MM_KEY_ERR_EVENT:
		return ErrKeyEventFailed
	case C.MM_KEY_ERR_DISPLAY:
		return ErrKeyDisplayUnavailable
	case C.MM_KEY_ERR_WINDOW:
		return ErrKeyWindowNotFound
	case C.MM_KEY_ERR_POST:
		return ErrKeyPostFailed
	default:
		return nil
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
		return 0, keyLookupError(ErrInvalidKey, k)
	}

	if len(k) == 1 {
		val1 := C.CString(k)
		defer C.free(unsafe.Pointer(val1))

		key = C.keyCodeForChar(*val1)
		if key == C.K_NOT_A_KEY {
			return 0, keyLookupError(ErrInvalidKey, k)
		}
		return
	}

	if v, ok := keyNameMap[k]; ok {
		key = v
		if key == C.K_NOT_A_KEY {
			return 0, keyLookupError(ErrUnsupportedKey, k)
		}
		return
	}
	return 0, keyLookupError(ErrInvalidKey, k)
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

	milliSleep(settings.Sleep)
	return keyActionError("keyTap", key, 0, ret)
}

// KeyTapWithPID taps the keyboard code on a specific process. On Linux X11
// the underlying C entry expects an X11 Window XID, not a PID; this Go
// wrapper resolves PID -> XID via _NET_WM_PID before dispatch, so callers
// of KeyTapWithPID get true per-process behavior on every platform.
// Returns ErrKeyWindowNotFound if no window can be resolved.
func KeyTapWithPID(key string, pid int, modifiers []Modifier, settings KeyboardSettings) error {
	if runtime.GOOS == "linux" && isWaylandSession() {
		return fmt.Errorf("%w: wayland session — per-window keyboard injection requires X11", cap.ErrUnsupported)
	}
	key, modifiers = normalizeKeyAndModifiers(key, modifiers)

	keyCode, err := checkKeyCodes(key)
	if err != nil {
		return err
	}

	flags := modifiersToFlags(modifiers)

	target := pid
	if runtime.GOOS == "linux" && pid > 0 {
		xid := lookupXIDByPID(pid)
		if xid == 0 {
			return ErrKeyWindowNotFound
		}
		target = int(xid)
	}

	// PID-specific operation. On Linux `target` is the resolved XID; on
	// macOS it is the actual PID; on Windows the C side treats this as
	// a PID and runs GetHwndByPid() — use KeyTapWithWindow when you have
	// an HWND.
	ret := C.keyTapPid(keyCode, flags, C.uintptr(target))

	milliSleep(settings.Sleep)
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

	milliSleep(settings.Sleep)
	return keyActionError("keyToggle", key, 0, ret)
}

// KeyToggleWithPID toggles a key on a specific process. Like KeyTapWithPID,
// resolves PID -> XID on Linux X11 before dispatch.
func KeyToggleWithPID(key string, down bool, pid int, modifiers []Modifier, settings KeyboardSettings) error {
	if runtime.GOOS == "linux" && isWaylandSession() {
		return fmt.Errorf("%w: wayland session — per-window keyboard injection requires X11", cap.ErrUnsupported)
	}
	key, modifiers = normalizeKeyAndModifiers(key, modifiers)

	keyCode, err := checkKeyCodes(key)
	if err != nil {
		return err
	}

	flags := modifiersToFlags(modifiers)

	target := pid
	if runtime.GOOS == "linux" && pid > 0 {
		xid := lookupXIDByPID(pid)
		if xid == 0 {
			return ErrKeyWindowNotFound
		}
		target = int(xid)
	}

	ret := C.keyTogglePid(keyCode, C.bool(down), flags, C.uintptr(target))

	milliSleep(settings.Sleep)
	return keyActionError("keyTogglePid", key, pid, ret)
}

// KeyTapWithWindow taps a key targeting a specific window without disturbing
// the user's keyboard focus. It is the cross-platform-correct counterpart to
// KeyTapWithPID: callers always supply both windowID (HWND on Windows, X11
// XID on Linux) and pid (used by macOS CGEventPostToPid), and the right one
// is dispatched per platform.
//
//   - Windows: uses PostMessageW(WM_KEYDOWN/UP) against windowID (HWND).
//   - macOS:   uses CGEventPostToPid(pid, ev).
//   - Linux X11: uses XSendEvent against windowID (XID). When windowID is
//                zero and pid is non-zero, attempts a best-effort PID->XID
//                lookup via _NET_WM_PID on every visible window.
func KeyTapWithWindow(key string, windowID uint64, pid int, modifiers []Modifier, settings KeyboardSettings) error {
	if runtime.GOOS == "linux" && isWaylandSession() {
		return fmt.Errorf("%w: wayland session — per-window keyboard injection requires X11", cap.ErrUnsupported)
	}
	id, err := keyboardWindowTarget(windowID, pid)
	if err != nil {
		return err
	}
	return keyTapForWindowTarget(key, id, modifiers, settings)
}

// KeyToggleWithWindow is the per-window analogue of KeyToggleWithPID. See
// KeyTapWithWindow for the platform-specific dispatch behavior.
func KeyToggleWithWindow(key string, down bool, windowID uint64, pid int, modifiers []Modifier, settings KeyboardSettings) error {
	if runtime.GOOS == "linux" && isWaylandSession() {
		return fmt.Errorf("%w: wayland session — per-window keyboard injection requires X11", cap.ErrUnsupported)
	}
	id, err := keyboardWindowTarget(windowID, pid)
	if err != nil {
		return err
	}
	return keyToggleForWindowTarget(key, down, id, modifiers, settings)
}

// keyboardWindowTarget reduces (windowID, pid) to the single int the
// platform's C entry point expects — HWND on Windows, XID on Linux, PID on
// macOS. Returns ErrKeyWindowNotFound when neither identifier is usable.
func keyboardWindowTarget(windowID uint64, pid int) (int, error) {
	switch runtime.GOOS {
	case "darwin":
		if pid != 0 {
			return pid, nil
		}
	case "windows":
		if windowID != 0 {
			return int(windowID), nil
		}
	case "linux":
		if windowID != 0 {
			return int(windowID), nil
		}
		if pid != 0 {
			if xid := lookupXIDByPID(pid); xid != 0 {
				return int(xid), nil
			}
		}
	}
	return 0, ErrKeyWindowNotFound
}

// KeyPress press and release a key with random delay (more human-like).
func KeyPress(key string, modifiers []Modifier, settings KeyboardSettings) error {
	err := KeyToggle(key, true, modifiers, settings)
	if err != nil {
		return err
	}

	milliSleep(1 + rand.Intn(3))
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

// UnicodeTypeWithWindow types a unicode codepoint into a specific window,
// using the per-platform mechanism that actually delivers text input:
//   - Windows: PostMessage(WM_CHAR) routed to the focused descendant of windowID.
//   - macOS:   CGEventPostToPid with a synthetic unicode keyboard event.
//   - Linux X11: XSendEvent against windowID. Returns ErrKeyWindowNotFound
//                if no identifier is supplied (we no longer silently fall
//                back to global focus on Linux).
//
// This is the right call for "type this text" scenarios; for individual
// virtual-key events (shortcuts, arrow keys) use KeyTapWithWindow.
func UnicodeTypeWithWindow(r rune, windowID uint64, pid int) error {
	if runtime.GOOS == "linux" && isWaylandSession() {
		return fmt.Errorf("%w: wayland session — per-window text input requires X11", cap.ErrUnsupported)
	}
	switch runtime.GOOS {
	case "windows":
		if windowID == 0 {
			return ErrKeyWindowNotFound
		}
		UnicodeType(uint32(r), int(windowID), true)
	case "darwin":
		if pid == 0 {
			return ErrKeyWindowNotFound
		}
		UnicodeType(uint32(r), pid, false)
	case "linux":
		xid := windowID
		if xid == 0 && pid > 0 {
			xid = uint64(lookupXIDByPID(pid))
		}
		if xid == 0 {
			return ErrKeyWindowNotFound
		}
		return unicodeTypeXIDPlatform(uint32(r), xid)
	}
	return nil
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
				milliSleep(tm1)
			}

			milliSleep(tm)
		}
		return
	}

	for i := 0; i < len([]rune(str)); i++ {
		ustr := uint32(CharCodeAt(str, i))
		UnicodeType(ustr, pid, false)
		milliSleep(tm)
	}
	milliSleep(settings.Sleep)
}

// TypeDelay type string with delayed.
func TypeDelay(str string, pid int, delay int, settings KeyboardSettings) {
	Type(str, pid, settings)
	milliSleep(delay)
}
