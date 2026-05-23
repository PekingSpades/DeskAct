// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// See the COPYRIGHT file at the top-level directory of this distribution and at
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0>
//
// This file may not be copied, modified, or distributed
// except according to those terms.

#include "../base/deadbeef_rand_c.h"
#include "../base/microsleep.h"
#include "keypress.h"
#include "keycode_c.h"

#include <ctype.h> /* For isupper() */

/*
 * Platform-specific helper functions
 */
HWND GetHwndByPid(DWORD dwProcessId);

HWND getHwnd(uintptr pid, int8_t isPid) {
	HWND hwnd = (HWND) pid;
	if (isPid == 0) {
		hwnd = GetHwndByPid(pid);
	}
	return hwnd;
}

/* Helper: check if a key is an extended key */
static inline DWORD getExtendedKeyFlags(int key) {
	switch (key) {
		case VK_RCONTROL:
		case VK_SNAPSHOT: /* Print Screen */
		case VK_RMENU: /* Right Alt / Alt Gr */
		case VK_PAUSE: /* Pause / Break */
		case VK_HOME:
		case VK_UP:
		case VK_PRIOR: /* Page up */
		case VK_LEFT:
		case VK_RIGHT:
		case VK_END:
		case VK_DOWN:
		case VK_NEXT: /* Page Down */
		case VK_INSERT:
		case VK_DELETE:
		case VK_LWIN:
		case VK_RWIN:
		case VK_APPS: /* Application */
		case VK_VOLUME_MUTE:
		case VK_VOLUME_DOWN:
		case VK_VOLUME_UP:
		case VK_MEDIA_NEXT_TRACK:
		case VK_MEDIA_PREV_TRACK:
		case VK_MEDIA_STOP:
		case VK_MEDIA_PLAY_PAUSE:
		case VK_BROWSER_BACK:
		case VK_BROWSER_FORWARD:
		case VK_BROWSER_REFRESH:
		case VK_BROWSER_STOP:
		case VK_BROWSER_SEARCH:
		case VK_BROWSER_FAVORITES:
		case VK_BROWSER_HOME:
		case VK_LAUNCH_MAIL:
			return KEYEVENTF_EXTENDEDKEY;
		default:
			return 0;
	}
}

/* Helper: add a key input to an INPUT array */
static inline void addKeyInput(INPUT *input, int key, DWORD flags) {
	input->type = INPUT_KEYBOARD;
	input->ki.wVk = key;
	input->ki.wScan = MapVirtualKey(key & 0xff, MAPVK_VK_TO_VSC);
	input->ki.dwFlags = flags | getExtendedKeyFlags(key);
	input->ki.time = 0;
	input->ki.dwExtraInfo = 0;
}

/* Send key event to a specific window via PostMessage */
static int postMessageChecked(HWND hwnd, int msg, WPARAM wParam, LPARAM lParam) {
	if (!PostMessageW(hwnd, msg, wParam, lParam)) {
		return MM_KEY_ERR_POST;
	}
	return MM_KEY_OK;
}

/*
 * focusedChildOf: if `hwnd` has a focused descendant in the same UI thread
 * (e.g. an Edit control inside a top-level window), return that descendant.
 * Otherwise return `hwnd` unchanged. Non-preemptive: GetGUIThreadInfo
 * reads cross-thread state without an AttachThreadInput.
 *
 * Guards against the multi-window-per-thread case: if the thread also
 * owns an unrelated top-level (so hwndFocus may belong to it), only
 * accept the focus if it is the same HWND as `hwnd` or a descendant of
 * it (per IsChild). Falls back to `hwnd` when the focus is unrelated.
 */
static HWND focusedChildOf(HWND hwnd) {
	if (hwnd == NULL) return hwnd;
	DWORD tid = GetWindowThreadProcessId(hwnd, NULL);
	if (tid == 0) return hwnd;
	GUITHREADINFO gti;
	gti.cbSize = sizeof(gti);
	if (!GetGUIThreadInfo(tid, &gti)) return hwnd;
	if (gti.hwndFocus == NULL) return hwnd;
	if (gti.hwndFocus == hwnd) return hwnd;
	if (IsChild(hwnd, gti.hwndFocus)) return gti.hwndFocus;
	return hwnd;
}

static int keyEventToHwnd(HWND hwnd, int key, DWORD flags) {
	int msg = (flags & KEYEVENTF_KEYUP) ? WM_KEYUP : WM_KEYDOWN;
	return postMessageChecked(focusedChildOf(hwnd), msg, key, 0);
}

/*
 * keyTap - Atomic key tap (press + release) with modifiers
 *
 * Press order:  Modifiers -> Main key
 * Release order: Main key -> Modifiers (LIFO)
 */
int keyTap(MMKeyCode code, MMKeyFlags flags) {
	INPUT inputs[10];
	int count = 0;

	/* Press: modifiers -> main key */
	if (flags & MOD_META) { addKeyInput(&inputs[count++], K_META, 0); }
	if (flags & MOD_ALT) { addKeyInput(&inputs[count++], K_ALT, 0); }
	if (flags & MOD_CONTROL) { addKeyInput(&inputs[count++], K_CONTROL, 0); }
	if (flags & MOD_SHIFT) { addKeyInput(&inputs[count++], K_SHIFT, 0); }
	addKeyInput(&inputs[count++], code, 0);

	/* Release: main key -> modifiers (LIFO) */
	addKeyInput(&inputs[count++], code, KEYEVENTF_KEYUP);
	if (flags & MOD_SHIFT) { addKeyInput(&inputs[count++], K_SHIFT, KEYEVENTF_KEYUP); }
	if (flags & MOD_CONTROL) { addKeyInput(&inputs[count++], K_CONTROL, KEYEVENTF_KEYUP); }
	if (flags & MOD_ALT) { addKeyInput(&inputs[count++], K_ALT, KEYEVENTF_KEYUP); }
	if (flags & MOD_META) { addKeyInput(&inputs[count++], K_META, KEYEVENTF_KEYUP); }

	return SendInput(count, inputs, sizeof(INPUT)) == count ? MM_KEY_OK : GetLastError();
}

/*
 * keyToggle - Atomic key toggle (press or release) with modifiers
 *
 * down=true:  Modifiers -> Main key (press order)
 * down=false: Main key -> Modifiers (release order, LIFO)
 */
int keyToggle(MMKeyCode code, const bool down, MMKeyFlags flags) {
	INPUT inputs[5];
	int count = 0;
	const DWORD dwFlags = down ? 0 : KEYEVENTF_KEYUP;

	if (down) {
		/* Press: modifiers -> main key */
		if (flags & MOD_META) { addKeyInput(&inputs[count++], K_META, dwFlags); }
		if (flags & MOD_ALT) { addKeyInput(&inputs[count++], K_ALT, dwFlags); }
		if (flags & MOD_CONTROL) { addKeyInput(&inputs[count++], K_CONTROL, dwFlags); }
		if (flags & MOD_SHIFT) { addKeyInput(&inputs[count++], K_SHIFT, dwFlags); }
		addKeyInput(&inputs[count++], code, dwFlags);
	} else {
		/* Release: main key -> modifiers (LIFO) */
		addKeyInput(&inputs[count++], code, dwFlags);
		if (flags & MOD_SHIFT) { addKeyInput(&inputs[count++], K_SHIFT, dwFlags); }
		if (flags & MOD_CONTROL) { addKeyInput(&inputs[count++], K_CONTROL, dwFlags); }
		if (flags & MOD_ALT) { addKeyInput(&inputs[count++], K_ALT, dwFlags); }
		if (flags & MOD_META) { addKeyInput(&inputs[count++], K_META, dwFlags); }
	}

	return SendInput(count, inputs, sizeof(INPUT)) == count ? MM_KEY_OK : GetLastError();
}

/*
 * keyTapPid - Key tap to a specific process (non-atomic, uses PostMessage on Windows)
 */
int keyTapPid(MMKeyCode code, MMKeyFlags flags, uintptr pid) {
	/* Windows: use PostMessage (non-atomic) */
	HWND hwnd = getHwnd(pid, 0);
	int err = MM_KEY_OK;

	if (hwnd == NULL) {
		return MM_KEY_ERR_WINDOW;
	}

	if (flags & MOD_META) { err = keyEventToHwnd(hwnd, K_META, 0); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_ALT) { err = keyEventToHwnd(hwnd, K_ALT, 0); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_CONTROL) { err = keyEventToHwnd(hwnd, K_CONTROL, 0); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_SHIFT) { err = keyEventToHwnd(hwnd, K_SHIFT, 0); if (err != MM_KEY_OK) return err; }
	err = keyEventToHwnd(hwnd, code, 0);
	if (err != MM_KEY_OK) return err;

	err = keyEventToHwnd(hwnd, code, KEYEVENTF_KEYUP);
	if (err != MM_KEY_OK) return err;
	if (flags & MOD_SHIFT) { err = keyEventToHwnd(hwnd, K_SHIFT, KEYEVENTF_KEYUP); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_CONTROL) { err = keyEventToHwnd(hwnd, K_CONTROL, KEYEVENTF_KEYUP); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_ALT) { err = keyEventToHwnd(hwnd, K_ALT, KEYEVENTF_KEYUP); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_META) { err = keyEventToHwnd(hwnd, K_META, KEYEVENTF_KEYUP); if (err != MM_KEY_OK) return err; }
	return MM_KEY_OK;
}

/*
 * keyTapHwnd - Key tap directly to an HWND (skips the PID->HWND lookup).
 * Used by the per-window keyboard path where the caller already resolved
 * the window. Behavior is otherwise identical to keyTapPid.
 */
int keyTapHwnd(MMKeyCode code, MMKeyFlags flags, uintptr hwndVal) {
	HWND hwnd = getHwnd(hwndVal, 1);
	int err = MM_KEY_OK;

	if (hwnd == NULL) {
		return MM_KEY_ERR_WINDOW;
	}

	if (flags & MOD_META) { err = keyEventToHwnd(hwnd, K_META, 0); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_ALT) { err = keyEventToHwnd(hwnd, K_ALT, 0); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_CONTROL) { err = keyEventToHwnd(hwnd, K_CONTROL, 0); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_SHIFT) { err = keyEventToHwnd(hwnd, K_SHIFT, 0); if (err != MM_KEY_OK) return err; }
	err = keyEventToHwnd(hwnd, code, 0);
	if (err != MM_KEY_OK) return err;

	err = keyEventToHwnd(hwnd, code, KEYEVENTF_KEYUP);
	if (err != MM_KEY_OK) return err;
	if (flags & MOD_SHIFT) { err = keyEventToHwnd(hwnd, K_SHIFT, KEYEVENTF_KEYUP); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_CONTROL) { err = keyEventToHwnd(hwnd, K_CONTROL, KEYEVENTF_KEYUP); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_ALT) { err = keyEventToHwnd(hwnd, K_ALT, KEYEVENTF_KEYUP); if (err != MM_KEY_OK) return err; }
	if (flags & MOD_META) { err = keyEventToHwnd(hwnd, K_META, KEYEVENTF_KEYUP); if (err != MM_KEY_OK) return err; }
	return MM_KEY_OK;
}

/*
 * keyToggleHwnd - keyTogglePid's HWND-direct counterpart. See keyTapHwnd.
 */
int keyToggleHwnd(MMKeyCode code, const bool down, MMKeyFlags flags, uintptr hwndVal) {
	DWORD dwFlags = down ? 0 : KEYEVENTF_KEYUP;
	HWND hwnd = getHwnd(hwndVal, 1);
	int err = MM_KEY_OK;

	if (hwnd == NULL) {
		return MM_KEY_ERR_WINDOW;
	}

	if (down) {
		if (flags & MOD_META) { err = keyEventToHwnd(hwnd, K_META, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_ALT) { err = keyEventToHwnd(hwnd, K_ALT, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_CONTROL) { err = keyEventToHwnd(hwnd, K_CONTROL, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_SHIFT) { err = keyEventToHwnd(hwnd, K_SHIFT, dwFlags); if (err != MM_KEY_OK) return err; }
		err = keyEventToHwnd(hwnd, code, dwFlags);
		if (err != MM_KEY_OK) return err;
	} else {
		err = keyEventToHwnd(hwnd, code, dwFlags);
		if (err != MM_KEY_OK) return err;
		if (flags & MOD_SHIFT) { err = keyEventToHwnd(hwnd, K_SHIFT, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_CONTROL) { err = keyEventToHwnd(hwnd, K_CONTROL, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_ALT) { err = keyEventToHwnd(hwnd, K_ALT, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_META) { err = keyEventToHwnd(hwnd, K_META, dwFlags); if (err != MM_KEY_OK) return err; }
	}
	return MM_KEY_OK;
}

/*
 * keyTogglePid - Key toggle to a specific process
 */
int keyTogglePid(MMKeyCode code, const bool down, MMKeyFlags flags, uintptr pid) {
	DWORD dwFlags = down ? 0 : KEYEVENTF_KEYUP;
	HWND hwnd = getHwnd(pid, 0);
	int err = MM_KEY_OK;

	if (hwnd == NULL) {
		return MM_KEY_ERR_WINDOW;
	}

	if (down) {
		if (flags & MOD_META) { err = keyEventToHwnd(hwnd, K_META, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_ALT) { err = keyEventToHwnd(hwnd, K_ALT, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_CONTROL) { err = keyEventToHwnd(hwnd, K_CONTROL, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_SHIFT) { err = keyEventToHwnd(hwnd, K_SHIFT, dwFlags); if (err != MM_KEY_OK) return err; }
		err = keyEventToHwnd(hwnd, code, dwFlags);
		if (err != MM_KEY_OK) return err;
	} else {
		err = keyEventToHwnd(hwnd, code, dwFlags);
		if (err != MM_KEY_OK) return err;
		if (flags & MOD_SHIFT) { err = keyEventToHwnd(hwnd, K_SHIFT, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_CONTROL) { err = keyEventToHwnd(hwnd, K_CONTROL, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_ALT) { err = keyEventToHwnd(hwnd, K_ALT, dwFlags); if (err != MM_KEY_OK) return err; }
		if (flags & MOD_META) { err = keyEventToHwnd(hwnd, K_META, dwFlags); if (err != MM_KEY_OK) return err; }
	}
	return MM_KEY_OK;
}

/*
 * Legacy functions for compatibility
 */
void toggleKey(char c, const bool down, MMKeyFlags flags, uintptr pid) {
	MMKeyCode keyCode = keyCodeForChar(c);

	if (isupper(c) && !(flags & MOD_SHIFT)) {
		flags |= MOD_SHIFT;
	}

	int modifiers = keyCode >> 8;
	if ((modifiers & 1) != 0) { flags |= MOD_SHIFT; }
	if ((modifiers & 2) != 0) { flags |= MOD_CONTROL; }
	if ((modifiers & 4) != 0) { flags |= MOD_ALT; }
	keyCode = keyCode & 0xff;

	if (pid != 0) {
		keyTogglePid(keyCode, down, flags, pid);
	} else {
		keyToggle(keyCode, down, flags);
	}
}

void unicodeType(const unsigned value, uintptr pid, int8_t isPid) {
	if (pid != 0) {
		HWND hwnd = focusedChildOf(getHwnd(pid, isPid));
		if (value > 0xFFFF) {
			uint32_t v = (uint32_t)value - 0x10000;
			WCHAR hi = (WCHAR)(0xD800 + (v >> 10));
			WCHAR lo = (WCHAR)(0xDC00 + (v & 0x3FF));
			PostMessageW(hwnd, WM_CHAR, hi, 0);
			PostMessageW(hwnd, WM_CHAR, lo, 0);
			return;
		}
		PostMessageW(hwnd, WM_CHAR, value, 0);
		return;
	}

	if (value > 0xFFFF) {
		uint32_t v = (uint32_t)value - 0x10000;
		WORD hi = (WORD)(0xD800 + (v >> 10));
		WORD lo = (WORD)(0xDC00 + (v & 0x3FF));

		INPUT input[4];
		memset(input, 0, sizeof(input));

		input[0].type = INPUT_KEYBOARD;
		input[0].ki.wVk = 0;
		input[0].ki.wScan = hi;
		input[0].ki.dwFlags = 0x4; // KEYEVENTF_UNICODE

		input[1].type = INPUT_KEYBOARD;
		input[1].ki.wVk = 0;
		input[1].ki.wScan = hi;
		input[1].ki.dwFlags = KEYEVENTF_KEYUP | 0x4;

		input[2].type = INPUT_KEYBOARD;
		input[2].ki.wVk = 0;
		input[2].ki.wScan = lo;
		input[2].ki.dwFlags = 0x4; // KEYEVENTF_UNICODE

		input[3].type = INPUT_KEYBOARD;
		input[3].ki.wVk = 0;
		input[3].ki.wScan = lo;
		input[3].ki.dwFlags = KEYEVENTF_KEYUP | 0x4;

		SendInput(4, input, sizeof(INPUT));
		return;
	}

	INPUT input[2];
	memset(input, 0, sizeof(input));

	input[0].type = INPUT_KEYBOARD;
	input[0].ki.wVk = 0;
	input[0].ki.wScan = value;
	input[0].ki.dwFlags = 0x4; // KEYEVENTF_UNICODE

	input[1].type = INPUT_KEYBOARD;
	input[1].ki.wVk = 0;
	input[1].ki.wScan = value;
	input[1].ki.dwFlags = KEYEVENTF_KEYUP | 0x4;

	SendInput(2, input, sizeof(INPUT));
}

int input_utf(const char *utf) {
	return 0;
}
