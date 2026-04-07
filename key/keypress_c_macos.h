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

static CGEventSourceRef CreateKeyboardEventSource(void) {
	return CGEventSourceCreate(kCGEventSourceStateCombinedSessionState);
}

static CGEventFlags modifierFlagForKeyCode(MMKeyCode code) {
	if (code == K_META || code == K_LMETA || code == K_RMETA) {
		return kCGEventFlagMaskCommand;
	}
	if (code == K_ALT || code == K_LALT || code == K_RALT) {
		return kCGEventFlagMaskAlternate;
	}
	if (code == K_CONTROL || code == K_LCONTROL || code == K_RCONTROL) {
		return kCGEventFlagMaskControl;
	}
	if (code == K_SHIFT || code == K_LSHIFT || code == K_RSHIFT) {
		return kCGEventFlagMaskShift;
	}
	if (code == K_CAPSLOCK) {
		return kCGEventFlagMaskAlphaShift;
	}
	return 0;
}

static int postKeyboardEvent(MMKeyCode code, bool down, CGEventFlags flags, uintptr pid) {
	CGEventSourceRef source = CreateKeyboardEventSource();
	if (source == NULL) {
		return MM_KEY_ERR_EVENT;
	}
	CGEventRef keyEvent = CGEventCreateKeyboardEvent(source, (CGKeyCode)code, down);
	CGEventFlags modifierFlag = modifierFlagForKeyCode(code);
	CGEventType eventType = down ? kCGEventKeyDown : kCGEventKeyUp;
	if (keyEvent == NULL) {
		CFRelease(source);
		return MM_KEY_ERR_EVENT;
	}

	if (modifierFlag != 0) {
		eventType = kCGEventFlagsChanged;
		if (down) {
			flags |= modifierFlag;
		} else {
			flags &= ~modifierFlag;
		}
	}

	CGEventSetType(keyEvent, eventType);
	CGEventSetFlags(keyEvent, flags);

	SendTo(pid, keyEvent);
	CFRelease(source);
	return MM_KEY_OK;
}

static int pressModifierFlags(CGEventFlags flags, uintptr pid) {
	CGEventFlags active = 0;

	if (flags & kCGEventFlagMaskCommand) {
		active |= kCGEventFlagMaskCommand;
		int err = postKeyboardEvent(K_META, true, active, pid);
		if (err != MM_KEY_OK) { return err; }
		microsleep(1.0);
	}
	if (flags & kCGEventFlagMaskAlternate) {
		active |= kCGEventFlagMaskAlternate;
		int err = postKeyboardEvent(K_ALT, true, active, pid);
		if (err != MM_KEY_OK) { return err; }
		microsleep(1.0);
	}
	if (flags & kCGEventFlagMaskControl) {
		active |= kCGEventFlagMaskControl;
		int err = postKeyboardEvent(K_CONTROL, true, active, pid);
		if (err != MM_KEY_OK) { return err; }
		microsleep(1.0);
	}
	if (flags & kCGEventFlagMaskShift) {
		active |= kCGEventFlagMaskShift;
		int err = postKeyboardEvent(K_SHIFT, true, active, pid);
		if (err != MM_KEY_OK) { return err; }
		microsleep(1.0);
	}

	return MM_KEY_OK;
}

static int releaseModifierFlags(CGEventFlags flags, uintptr pid) {
	CGEventFlags active = flags;

	if (flags & kCGEventFlagMaskShift) {
		active &= ~kCGEventFlagMaskShift;
		int err = postKeyboardEvent(K_SHIFT, false, active, pid);
		if (err != MM_KEY_OK) { return err; }
		microsleep(1.0);
	}
	if (flags & kCGEventFlagMaskControl) {
		active &= ~kCGEventFlagMaskControl;
		int err = postKeyboardEvent(K_CONTROL, false, active, pid);
		if (err != MM_KEY_OK) { return err; }
		microsleep(1.0);
	}
	if (flags & kCGEventFlagMaskAlternate) {
		active &= ~kCGEventFlagMaskAlternate;
		int err = postKeyboardEvent(K_ALT, false, active, pid);
		if (err != MM_KEY_OK) { return err; }
		microsleep(1.0);
	}
	if (flags & kCGEventFlagMaskCommand) {
		active &= ~kCGEventFlagMaskCommand;
		int err = postKeyboardEvent(K_META, false, active, pid);
		if (err != MM_KEY_OK) { return err; }
		microsleep(1.0);
	}

	return MM_KEY_OK;
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

	CGEventFlags eventFlags = (CGEventFlags)flags;
	int err = pressModifierFlags(eventFlags, 0);
	if (err != MM_KEY_OK) { return err; }

	err = postKeyboardEvent(code, true, eventFlags, 0);
	if (err != MM_KEY_OK) {
		releaseModifierFlags(eventFlags, 0);
		return err;
	}

	microsleep(5.0);

	err = postKeyboardEvent(code, false, eventFlags, 0);
	if (err != MM_KEY_OK) {
		releaseModifierFlags(eventFlags, 0);
		return err;
	}

	return releaseModifierFlags(eventFlags, 0);
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

	CGEventFlags eventFlags = (CGEventFlags)flags;
	if (down) {
		int err = pressModifierFlags(eventFlags, 0);
		if (err != MM_KEY_OK) { return err; }
		return postKeyboardEvent(code, true, eventFlags, 0);
	}

	int err = postKeyboardEvent(code, false, eventFlags, 0);
	if (err != MM_KEY_OK) { return err; }
	return releaseModifierFlags(eventFlags, 0);
}

/*
 * keyTapPid - Key tap to a specific process (non-atomic, uses PostMessage on Windows)
 */
int keyTapPid(MMKeyCode code, MMKeyFlags flags, uintptr pid) {
	CGEventFlags eventFlags = (CGEventFlags)flags;
	int err = pressModifierFlags(eventFlags, pid);
	if (err != MM_KEY_OK) { return err; }

	err = postKeyboardEvent(code, true, eventFlags, pid);
	if (err != MM_KEY_OK) {
		releaseModifierFlags(eventFlags, pid);
		return err;
	}

	microsleep(5.0);

	err = postKeyboardEvent(code, false, eventFlags, pid);
	if (err != MM_KEY_OK) {
		releaseModifierFlags(eventFlags, pid);
		return err;
	}

	return releaseModifierFlags(eventFlags, pid);
}

/*
 * keyTogglePid - Key toggle to a specific process
 */
int keyTogglePid(MMKeyCode code, const bool down, MMKeyFlags flags, uintptr pid) {
	CGEventFlags eventFlags = (CGEventFlags)flags;
	if (down) {
		int err = pressModifierFlags(eventFlags, pid);
		if (err != MM_KEY_OK) { return err; }
		return postKeyboardEvent(code, true, eventFlags, pid);
	}

	int err = postKeyboardEvent(code, false, eventFlags, pid);
	if (err != MM_KEY_OK) { return err; }
	return releaseModifierFlags(eventFlags, pid);
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
	CGEventSourceRef source = CreateKeyboardEventSource();
	if (source == NULL) {
		fputs("Could not create event source.\n", stderr);
		return;
	}
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
