//go:build windows
// +build windows

package keyboardstate

import "github.com/lxn/win"

func currentState() (stateSnapshot, error) {
	var state stateSnapshot
	state.supported = bitShift | bitCtrl | bitAlt | bitCmd | bitCapsLock | bitNumLock | bitScrollLock

	if down, _ := keyState(win.VK_SHIFT); down {
		state.mask |= bitShift
	}
	if down, _ := keyState(win.VK_CONTROL); down {
		state.mask |= bitCtrl
	}
	if down, _ := keyState(win.VK_MENU); down {
		state.mask |= bitAlt
	}
	if down, _ := keyState(win.VK_LWIN); down {
		state.mask |= bitCmd
	}
	if down, _ := keyState(win.VK_RWIN); down {
		state.mask |= bitCmd
	}
	if _, toggled := keyState(win.VK_CAPITAL); toggled {
		state.mask |= bitCapsLock
	}
	if _, toggled := keyState(win.VK_NUMLOCK); toggled {
		state.mask |= bitNumLock
	}
	if _, toggled := keyState(win.VK_SCROLL); toggled {
		state.mask |= bitScrollLock
	}

	return state, nil
}

func keyState(vk int32) (down bool, toggled bool) {
	state := int32(win.GetKeyState(vk))
	return state&0x8000 != 0, state&0x0001 != 0
}
