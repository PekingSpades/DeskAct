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
void keyEventToWindow(int key, DWORD flags, uintptr pid, int8_t isPid) {
	HWND hwnd = getHwnd(pid, isPid);
	int msg = (flags & KEYEVENTF_KEYUP) ? WM_KEYUP : WM_KEYDOWN;
	PostMessageW(hwnd, msg, key, 0);
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

	return SendInput(count, inputs, sizeof(INPUT)) == count ? 0 : GetLastError();
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

	return SendInput(count, inputs, sizeof(INPUT)) == count ? 0 : GetLastError();
}

/*
 * keyTapPid - Key tap to a specific process (non-atomic, uses PostMessage on Windows)
 */
int keyTapPid(MMKeyCode code, MMKeyFlags flags, uintptr pid) {
	/* Windows: use PostMessage (non-atomic) */
	if (flags & MOD_META) { keyEventToWindow(K_META, 0, pid, 0); }
	if (flags & MOD_ALT) { keyEventToWindow(K_ALT, 0, pid, 0); }
	if (flags & MOD_CONTROL) { keyEventToWindow(K_CONTROL, 0, pid, 0); }
	if (flags & MOD_SHIFT) { keyEventToWindow(K_SHIFT, 0, pid, 0); }
	keyEventToWindow(code, 0, pid, 0);

	keyEventToWindow(code, KEYEVENTF_KEYUP, pid, 0);
	if (flags & MOD_SHIFT) { keyEventToWindow(K_SHIFT, KEYEVENTF_KEYUP, pid, 0); }
	if (flags & MOD_CONTROL) { keyEventToWindow(K_CONTROL, KEYEVENTF_KEYUP, pid, 0); }
	if (flags & MOD_ALT) { keyEventToWindow(K_ALT, KEYEVENTF_KEYUP, pid, 0); }
	if (flags & MOD_META) { keyEventToWindow(K_META, KEYEVENTF_KEYUP, pid, 0); }
	return 0;
}

/*
 * keyTogglePid - Key toggle to a specific process
 */
int keyTogglePid(MMKeyCode code, const bool down, MMKeyFlags flags, uintptr pid) {
	DWORD dwFlags = down ? 0 : KEYEVENTF_KEYUP;

	if (down) {
		if (flags & MOD_META) { keyEventToWindow(K_META, dwFlags, pid, 0); }
		if (flags & MOD_ALT) { keyEventToWindow(K_ALT, dwFlags, pid, 0); }
		if (flags & MOD_CONTROL) { keyEventToWindow(K_CONTROL, dwFlags, pid, 0); }
		if (flags & MOD_SHIFT) { keyEventToWindow(K_SHIFT, dwFlags, pid, 0); }
		keyEventToWindow(code, dwFlags, pid, 0);
	} else {
		keyEventToWindow(code, dwFlags, pid, 0);
		if (flags & MOD_SHIFT) { keyEventToWindow(K_SHIFT, dwFlags, pid, 0); }
		if (flags & MOD_CONTROL) { keyEventToWindow(K_CONTROL, dwFlags, pid, 0); }
		if (flags & MOD_ALT) { keyEventToWindow(K_ALT, dwFlags, pid, 0); }
		if (flags & MOD_META) { keyEventToWindow(K_META, dwFlags, pid, 0); }
	}
	return 0;
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
		HWND hwnd = getHwnd(pid, isPid);
		PostMessageW(hwnd, WM_CHAR, value, 0);
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
