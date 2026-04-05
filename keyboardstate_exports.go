package deskact

import ks "github.com/PekingSpades/DeskAct/keyboardstate"

type KeyboardPressState = ks.PressState
type KeyboardToggleState = ks.ToggleState
type KeyboardStateSnapshot = ks.Snapshot

const (
	KeyboardPressUnsupported = ks.PressUnsupported
	KeyboardPressUp          = ks.PressUp
	KeyboardPressDown        = ks.PressDown

	KeyboardToggleUnsupported = ks.ToggleUnsupported
	KeyboardToggleOff         = ks.ToggleOff
	KeyboardToggleOn          = ks.ToggleOn
)

func KeyboardStateCurrent() (KeyboardStateSnapshot, error) {
	return ks.Current()
}

func KeyboardStateShift() (KeyboardPressState, error) {
	return ks.Shift()
}

func KeyboardStateCtrl() (KeyboardPressState, error) {
	return ks.Ctrl()
}

func KeyboardStateAlt() (KeyboardPressState, error) {
	return ks.Alt()
}

func KeyboardStateCmd() (KeyboardPressState, error) {
	return ks.Cmd()
}

func KeyboardStateCapsLock() (KeyboardToggleState, error) {
	return ks.CapsLock()
}

func KeyboardStateNumLock() (KeyboardToggleState, error) {
	return ks.NumLock()
}

func KeyboardStateScrollLock() (KeyboardToggleState, error) {
	return ks.ScrollLock()
}
