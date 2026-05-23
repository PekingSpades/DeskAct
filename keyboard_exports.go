package deskact

import kbd "github.com/PekingSpades/DeskAct/keyboard"

type KeyboardSettings = kbd.KeyboardSettings
type Modifier = kbd.Modifier
type KeyError = kbd.KeyError
type KeyActionError = kbd.KeyActionError

var (
	ErrInvalidKey            = kbd.ErrInvalidKey
	ErrUnsupportedKey        = kbd.ErrUnsupportedKey
	ErrKeyActionFailed       = kbd.ErrKeyActionFailed
	ErrKeyEventFailed        = kbd.ErrKeyEventFailed
	ErrKeyDisplayUnavailable = kbd.ErrKeyDisplayUnavailable
	ErrKeyWindowNotFound     = kbd.ErrKeyWindowNotFound
	ErrKeyPostFailed         = kbd.ErrKeyPostFailed
)

const (
	DefaultKeySleep     = kbd.DefaultKeySleep
	DefaultTypeDelay    = kbd.DefaultTypeDelay
	DefaultTypeUTFDelay = kbd.DefaultTypeUTFDelay
	KeyA                = kbd.KeyA
	KeyB                = kbd.KeyB
	KeyC                = kbd.KeyC
	KeyD                = kbd.KeyD
	KeyE                = kbd.KeyE
	KeyF                = kbd.KeyF
	KeyG                = kbd.KeyG
	KeyH                = kbd.KeyH
	KeyI                = kbd.KeyI
	KeyJ                = kbd.KeyJ
	KeyK                = kbd.KeyK
	KeyL                = kbd.KeyL
	KeyM                = kbd.KeyM
	KeyN                = kbd.KeyN
	KeyO                = kbd.KeyO
	KeyP                = kbd.KeyP
	KeyQ                = kbd.KeyQ
	KeyR                = kbd.KeyR
	KeyS                = kbd.KeyS
	KeyT                = kbd.KeyT
	KeyU                = kbd.KeyU
	KeyV                = kbd.KeyV
	KeyW                = kbd.KeyW
	KeyX                = kbd.KeyX
	KeyY                = kbd.KeyY
	KeyZ                = kbd.KeyZ
	CapA                = kbd.CapA
	CapB                = kbd.CapB
	CapC                = kbd.CapC
	CapD                = kbd.CapD
	CapE                = kbd.CapE
	CapF                = kbd.CapF
	CapG                = kbd.CapG
	CapH                = kbd.CapH
	CapI                = kbd.CapI
	CapJ                = kbd.CapJ
	CapK                = kbd.CapK
	CapL                = kbd.CapL
	CapM                = kbd.CapM
	CapN                = kbd.CapN
	CapO                = kbd.CapO
	CapP                = kbd.CapP
	CapQ                = kbd.CapQ
	CapR                = kbd.CapR
	CapS                = kbd.CapS
	CapT                = kbd.CapT
	CapU                = kbd.CapU
	CapV                = kbd.CapV
	CapW                = kbd.CapW
	CapX                = kbd.CapX
	CapY                = kbd.CapY
	CapZ                = kbd.CapZ
	Key0                = kbd.Key0
	Key1                = kbd.Key1
	Key2                = kbd.Key2
	Key3                = kbd.Key3
	Key4                = kbd.Key4
	Key5                = kbd.Key5
	Key6                = kbd.Key6
	Key7                = kbd.Key7
	Key8                = kbd.Key8
	Key9                = kbd.Key9
	Backspace           = kbd.Backspace
	Delete              = kbd.Delete
	Enter               = kbd.Enter
	Tab                 = kbd.Tab
	Esc                 = kbd.Esc
	Up                  = kbd.Up
	Down                = kbd.Down
	Right               = kbd.Right
	Left                = kbd.Left
	Home                = kbd.Home
	End                 = kbd.End
	Pageup              = kbd.Pageup
	Pagedown            = kbd.Pagedown
	F1                  = kbd.F1
	F2                  = kbd.F2
	F3                  = kbd.F3
	F4                  = kbd.F4
	F5                  = kbd.F5
	F6                  = kbd.F6
	F7                  = kbd.F7
	F8                  = kbd.F8
	F9                  = kbd.F9
	F10                 = kbd.F10
	F11                 = kbd.F11
	F12                 = kbd.F12
	F13                 = kbd.F13
	F14                 = kbd.F14
	F15                 = kbd.F15
	F16                 = kbd.F16
	F17                 = kbd.F17
	F18                 = kbd.F18
	F19                 = kbd.F19
	F20                 = kbd.F20
	F21                 = kbd.F21
	F22                 = kbd.F22
	F23                 = kbd.F23
	F24                 = kbd.F24
	Cmd                 = kbd.Cmd
	Lcmd                = kbd.Lcmd
	Rcmd                = kbd.Rcmd
	Alt                 = kbd.Alt
	Lalt                = kbd.Lalt
	Ralt                = kbd.Ralt
	Ctrl                = kbd.Ctrl
	Lctrl               = kbd.Lctrl
	Rctrl               = kbd.Rctrl
	Shift               = kbd.Shift
	Lshift              = kbd.Lshift
	Rshift              = kbd.Rshift
	Capslock            = kbd.Capslock
	Space               = kbd.Space
	Print               = kbd.Print
	Insert              = kbd.Insert
	Menu                = kbd.Menu
	AudioMute           = kbd.AudioMute
	AudioVolDown        = kbd.AudioVolDown
	AudioVolUp          = kbd.AudioVolUp
	AudioPlay           = kbd.AudioPlay
	AudioStop           = kbd.AudioStop
	AudioPause          = kbd.AudioPause
	AudioPrev           = kbd.AudioPrev
	AudioNext           = kbd.AudioNext
	AudioRewind         = kbd.AudioRewind
	AudioForward        = kbd.AudioForward
	AudioRepeat         = kbd.AudioRepeat
	AudioRandom         = kbd.AudioRandom
	Num0                = kbd.Num0
	Num1                = kbd.Num1
	Num2                = kbd.Num2
	Num3                = kbd.Num3
	Num4                = kbd.Num4
	Num5                = kbd.Num5
	Num6                = kbd.Num6
	Num7                = kbd.Num7
	Num8                = kbd.Num8
	Num9                = kbd.Num9
	NumLock             = kbd.NumLock
	NumDecimal          = kbd.NumDecimal
	NumPlus             = kbd.NumPlus
	NumMinus            = kbd.NumMinus
	NumMul              = kbd.NumMul
	NumDiv              = kbd.NumDiv
	NumClear            = kbd.NumClear
	NumEnter            = kbd.NumEnter
	NumEqual            = kbd.NumEqual
	LightsMonUp         = kbd.LightsMonUp
	LightsMonDown       = kbd.LightsMonDown
	LightsKbdToggle     = kbd.LightsKbdToggle
	LightsKbdUp         = kbd.LightsKbdUp
	LightsKbdDown       = kbd.LightsKbdDown
	ModNone             = kbd.ModNone
	ModAlt              = kbd.ModAlt
	ModCtrl             = kbd.ModCtrl
	ModShift            = kbd.ModShift
	ModCmd              = kbd.ModCmd
)

func DefaultKeyboardSettings() KeyboardSettings {
	return kbd.DefaultKeyboardSettings()
}

func KeyNames() []string {
	return kbd.KeyNames()
}

func SupportedKeyNames() []string {
	return kbd.SupportedKeyNames()
}

func ModifierNames() []Modifier {
	return kbd.ModifierNames()
}

func DefaultSpecialKeys() map[string]string {
	return kbd.DefaultSpecialKeys()
}

func CmdCtrl() string {
	return kbd.CmdCtrl()
}

func KeyTap(key string, modifiers []Modifier, settings KeyboardSettings) error {
	return kbd.KeyTap(key, modifiers, settings)
}

func KeyTapWithPID(key string, pid int, modifiers []Modifier, settings KeyboardSettings) error {
	return kbd.KeyTapWithPID(key, pid, modifiers, settings)
}

func KeyToggle(key string, down bool, modifiers []Modifier, settings KeyboardSettings) error {
	return kbd.KeyToggle(key, down, modifiers, settings)
}

func KeyToggleWithPID(key string, down bool, pid int, modifiers []Modifier, settings KeyboardSettings) error {
	return kbd.KeyToggleWithPID(key, down, pid, modifiers, settings)
}

// KeyTapWithWindow taps a key targeting a specific window, picking the
// platform-correct identifier from the supplied (windowID, pid) pair:
// HWND on Windows, X11 Window XID on Linux, PID on macOS. See the
// keyboard package godoc for details.
func KeyTapWithWindow(key string, windowID uint64, pid int, modifiers []Modifier, settings KeyboardSettings) error {
	return kbd.KeyTapWithWindow(key, windowID, pid, modifiers, settings)
}

// KeyToggleWithWindow is the per-window analogue of KeyToggleWithPID.
func KeyToggleWithWindow(key string, down bool, windowID uint64, pid int, modifiers []Modifier, settings KeyboardSettings) error {
	return kbd.KeyToggleWithWindow(key, down, windowID, pid, modifiers, settings)
}

func KeyPress(key string, modifiers []Modifier, settings KeyboardSettings) error {
	return kbd.KeyPress(key, modifiers, settings)
}

func CharCodeAt(s string, n int) rune {
	return kbd.CharCodeAt(s, n)
}

func UnicodeType(str uint32, pid int, isPid bool) {
	kbd.UnicodeType(str, pid, isPid)
}

// UnicodeTypeWithWindow types a single unicode codepoint into the given
// window, using the platform-correct mechanism (WM_CHAR on Windows,
// CGEventPostToPid on macOS, XSendEvent on Linux X11). For shortcut keys
// (Ctrl+S, arrows, F1) use KeyTapWithWindow instead.
func UnicodeTypeWithWindow(r rune, windowID uint64, pid int) error {
	return kbd.UnicodeTypeWithWindow(r, windowID, pid)
}

func ToUC(text string) []string {
	return kbd.ToUC(text)
}

func Type(str string, pid int, settings KeyboardSettings) {
	kbd.Type(str, pid, settings)
}

func TypeDelay(str string, pid int, delay int, settings KeyboardSettings) {
	kbd.TypeDelay(str, pid, delay, settings)
}
