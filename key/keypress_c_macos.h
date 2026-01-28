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
#include <ApplicationServices/ApplicationServices.h>
#import <IOKit/hidsystem/IOHIDLib.h>
#import <IOKit/hidsystem/ev_keymap.h>

/*
 * Platform-specific helper functions
 */
static int SendTo(uintptr pid, CGEventRef event) {
	if (pid != 0) {
		CGEventPostToPid(pid, event);
	} else {
		CGEventPost(kCGHIDEventTap, event);
	}
	CFRelease(event);
	return 0;
}

static io_connect_t _getAuxiliaryKeyDriver(void) {
	static mach_port_t sEventDrvrRef = 0;
	mach_port_t masterPort, service, iter;
	kern_return_t kr;

	if (!sEventDrvrRef) {
		kr = IOMasterPort(bootstrap_port, &masterPort);
		assert(KERN_SUCCESS == kr);
		kr = IOServiceGetMatchingServices(masterPort, IOServiceMatching(kIOHIDSystemClass), &iter);
		assert(KERN_SUCCESS == kr);

		service = IOIteratorNext(iter);
		assert(service);

		kr = IOServiceOpen(service, mach_task_self(), kIOHIDParamConnectType, &sEventDrvrRef);
		assert(KERN_SUCCESS == kr);

		IOObjectRelease(service);
		IOObjectRelease(iter);
	}
	return sEventDrvrRef;
}

/* Helper: post media key event via IOKit HID */
static int postMediaKeyEvent(MMKeyCode code, bool down) {
	NXEventData event;
	kern_return_t kr;
	IOGPoint loc = { 0, 0 };
	UInt32 evtInfo = code << 16 | (down ? NX_KEYDOWN : NX_KEYUP) << 8;

	bzero(&event, sizeof(NXEventData));
	event.compound.subType = NX_SUBTYPE_AUX_CONTROL_BUTTONS;
	event.compound.misc.L[0] = evtInfo;

	kr = IOHIDPostEvent(_getAuxiliaryKeyDriver(),
		NX_SYSDEFINED, loc, &event, kNXEventDataVersion, 0, FALSE);
	return (kr == KERN_SUCCESS) ? MM_KEY_OK : MM_KEY_ERR_EVENT;
}

/*
 * keyTap - Atomic key tap (press + release) with modifiers
 *
 * Press order:  Modifiers -> Main key
 * Release order: Main key -> Modifiers (LIFO)
 */
int keyTap(MMKeyCode code, MMKeyFlags flags) {
	/* The media keys all have 1000 added to them to help us detect them. */
	if (code >= 1000) {
		code = code - 1000; /* Get the real keycode. */
		int err = postMediaKeyEvent(code, true);
		if (err != MM_KEY_OK) { return err; }
		microsleep(5.0);
		return postMediaKeyEvent(code, false);
	}

	/* macOS: CGEventFlags makes it atomic - modifiers are set on the event itself */
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	if (source == NULL) {
		return MM_KEY_ERR_EVENT;
	}

	/* Press */
	CGEventRef keyDown = CGEventCreateKeyboardEvent(source, (CGKeyCode)code, true);
	if (keyDown == NULL) {
		CFRelease(source);
		return MM_KEY_ERR_EVENT;
	}
	if (flags != 0) {
		CGEventSetFlags(keyDown, (CGEventFlags)flags);
	}
	CGEventPost(kCGHIDEventTap, keyDown);
	CFRelease(keyDown);

	/* Release */
	CGEventRef keyUp = CGEventCreateKeyboardEvent(source, (CGKeyCode)code, false);
	if (keyUp == NULL) {
		CFRelease(source);
		return MM_KEY_ERR_EVENT;
	}
	if (flags != 0) {
		CGEventSetFlags(keyUp, (CGEventFlags)flags);
	}
	CGEventPost(kCGHIDEventTap, keyUp);
	CFRelease(keyUp);

	CFRelease(source);
	return MM_KEY_OK;
}

/*
 * keyToggle - Atomic key toggle (press or release) with modifiers
 *
 * down=true:  Modifiers -> Main key (press order)
 * down=false: Main key -> Modifiers (release order, LIFO)
 */
int keyToggle(MMKeyCode code, const bool down, MMKeyFlags flags) {
	/* The media keys all have 1000 added to them to help us detect them. */
	if (code >= 1000) {
		code = code - 1000; /* Get the real keycode. */
		return postMediaKeyEvent(code, down);
	}

	/* macOS: CGEventFlags makes it atomic */
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	if (source == NULL) {
		return MM_KEY_ERR_EVENT;
	}
	CGEventRef keyEvent = CGEventCreateKeyboardEvent(source, (CGKeyCode)code, down);

	if (keyEvent == NULL) {
		CFRelease(source);
		return MM_KEY_ERR_EVENT;
	}

	CGEventSetType(keyEvent, down ? kCGEventKeyDown : kCGEventKeyUp);
	if (flags != 0) {
		CGEventSetFlags(keyEvent, (CGEventFlags)flags);
	}

	CGEventPost(kCGHIDEventTap, keyEvent);
	CFRelease(keyEvent);
	CFRelease(source);
	return MM_KEY_OK;
}

/*
 * keyTapPid - Key tap to a specific process (non-atomic, uses PostMessage on Windows)
 */
int keyTapPid(MMKeyCode code, MMKeyFlags flags, uintptr pid) {
	/* macOS: supports PID natively */
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	if (source == NULL) {
		return MM_KEY_ERR_EVENT;
	}

	CGEventRef keyDown = CGEventCreateKeyboardEvent(source, (CGKeyCode)code, true);
	if (keyDown == NULL) {
		CFRelease(source);
		return MM_KEY_ERR_EVENT;
	}
	if (flags != 0) {
		CGEventSetFlags(keyDown, (CGEventFlags)flags);
	}
	CGEventPostToPid(pid, keyDown);
	CFRelease(keyDown);

	CGEventRef keyUp = CGEventCreateKeyboardEvent(source, (CGKeyCode)code, false);
	if (keyUp == NULL) {
		CFRelease(source);
		return MM_KEY_ERR_EVENT;
	}
	if (flags != 0) {
		CGEventSetFlags(keyUp, (CGEventFlags)flags);
	}
	CGEventPostToPid(pid, keyUp);
	CFRelease(keyUp);

	CFRelease(source);
	return MM_KEY_OK;
}

/*
 * keyTogglePid - Key toggle to a specific process
 */
int keyTogglePid(MMKeyCode code, const bool down, MMKeyFlags flags, uintptr pid) {
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	if (source == NULL) {
		return MM_KEY_ERR_EVENT;
	}
	CGEventRef keyEvent = CGEventCreateKeyboardEvent(source, (CGKeyCode)code, down);

	if (keyEvent == NULL) {
		CFRelease(source);
		return MM_KEY_ERR_EVENT;
	}

	CGEventSetType(keyEvent, down ? kCGEventKeyDown : kCGEventKeyUp);
	if (flags != 0) {
		CGEventSetFlags(keyEvent, (CGEventFlags)flags);
	}

	CGEventPostToPid(pid, keyEvent);
	CFRelease(keyEvent);
	CFRelease(source);
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

	if (pid != 0) {
		keyTogglePid(keyCode, down, flags, pid);
	} else {
		keyToggle(keyCode, down, flags);
	}
}

void toggleUnicode(const UniChar *chars, size_t len, const bool down, uintptr pid) {
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	CGEventRef keyEvent = CGEventCreateKeyboardEvent(source, 0, down);
	if (keyEvent == NULL) {
		fputs("Could not create keyboard event.\n", stderr);
		CFRelease(source);
		return;
	}

	CGEventKeyboardSetUnicodeString(keyEvent, (UniCharCount)len, chars);
	SendTo(pid, keyEvent);
	CFRelease(source);
}

void unicodeType(const unsigned value, uintptr pid, int8_t isPid) {
	UniChar chars[2];
	size_t len = 1;

	if (value > 0xFFFF) {
		uint32_t v = (uint32_t)value - 0x10000;
		chars[0] = (UniChar)(0xD800 + (v >> 10));
		chars[1] = (UniChar)(0xDC00 + (v & 0x3FF));
		len = 2;
	} else {
		chars[0] = (UniChar)value;
	}

	toggleUnicode(chars, len, true, pid);
	microsleep(5.0);
	toggleUnicode(chars, len, false, pid);
}

int input_utf(const char *utf) {
	return 0;
}
