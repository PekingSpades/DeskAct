package keyboardstate

import "errors"

var (
	// ErrDisplayUnavailable is returned when the X11 display cannot be opened.
	ErrDisplayUnavailable = errors.New("display not available")
)

type stateSnapshot struct {
	mask      uint32
	supported uint32
}

const (
	bitShift uint32 = 1 << iota
	bitCtrl
	bitAlt
	bitCmd
	bitCapsLock
	bitNumLock
	bitScrollLock
)

// PressState describes the current pressed state of a modifier key.
type PressState int

const (
	PressUnsupported PressState = iota
	PressUp
	PressDown
)

// ToggleState describes the current toggle state of a lock key.
type ToggleState int

const (
	ToggleUnsupported ToggleState = iota
	ToggleOff
	ToggleOn
)

// Snapshot reports the current modifier and lock key states.
type Snapshot struct {
	Shift      PressState
	Ctrl       PressState
	Alt        PressState
	Cmd        PressState
	CapsLock   ToggleState
	NumLock    ToggleState
	ScrollLock ToggleState
}

// Current returns a snapshot of modifier and lock key states.
func Current() (Snapshot, error) {
	state, err := currentState()
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Shift:      pressStateFrom(state, bitShift),
		Ctrl:       pressStateFrom(state, bitCtrl),
		Alt:        pressStateFrom(state, bitAlt),
		Cmd:        pressStateFrom(state, bitCmd),
		CapsLock:   toggleStateFrom(state, bitCapsLock),
		NumLock:    toggleStateFrom(state, bitNumLock),
		ScrollLock: toggleStateFrom(state, bitScrollLock),
	}, nil
}

// Shift returns the current state of the Shift modifier.
func Shift() (PressState, error) {
	return pressStateFor(bitShift)
}

// Ctrl returns the current state of the Ctrl modifier.
func Ctrl() (PressState, error) {
	return pressStateFor(bitCtrl)
}

// Alt returns the current state of the Alt modifier.
func Alt() (PressState, error) {
	return pressStateFor(bitAlt)
}

// Cmd returns the current state of the Command/Windows/Super modifier.
func Cmd() (PressState, error) {
	return pressStateFor(bitCmd)
}

// CapsLock returns the current state of Caps Lock.
func CapsLock() (ToggleState, error) {
	return toggleStateFor(bitCapsLock)
}

// NumLock returns the current state of Num Lock.
func NumLock() (ToggleState, error) {
	return toggleStateFor(bitNumLock)
}

// ScrollLock returns the current state of Scroll Lock.
func ScrollLock() (ToggleState, error) {
	return toggleStateFor(bitScrollLock)
}

func pressStateFor(bit uint32) (PressState, error) {
	state, err := currentState()
	if err != nil {
		return PressUnsupported, err
	}
	if state.supported&bit == 0 {
		return PressUnsupported, nil
	}
	if state.mask&bit != 0 {
		return PressDown, nil
	}
	return PressUp, nil
}

func toggleStateFor(bit uint32) (ToggleState, error) {
	state, err := currentState()
	if err != nil {
		return ToggleUnsupported, err
	}
	if state.supported&bit == 0 {
		return ToggleUnsupported, nil
	}
	if state.mask&bit != 0 {
		return ToggleOn, nil
	}
	return ToggleOff, nil
}

func pressStateFrom(state stateSnapshot, bit uint32) PressState {
	if state.supported&bit == 0 {
		return PressUnsupported
	}
	if state.mask&bit != 0 {
		return PressDown
	}
	return PressUp
}

func toggleStateFrom(state stateSnapshot, bit uint32) ToggleState {
	if state.supported&bit == 0 {
		return ToggleUnsupported
	}
	if state.mask&bit != 0 {
		return ToggleOn
	}
	return ToggleOff
}
