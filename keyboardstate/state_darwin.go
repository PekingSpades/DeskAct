//go:build darwin
// +build darwin

package keyboardstate

/*
#cgo darwin CFLAGS: -Wno-deprecated-declarations
#cgo darwin LDFLAGS: -framework ApplicationServices -framework IOKit -framework CoreFoundation

#include <ApplicationServices/ApplicationServices.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/hidsystem/IOHIDLib.h>
#include <IOKit/hidsystem/IOHIDParameter.h>
#include <IOKit/hidsystem/IOHIDShared.h>
#include <IOKit/hidsystem/IOHIDSystem.h>

static uint64_t deskactKeyFlagsState(void) {
	return (uint64_t)CGEventSourceFlagsState(kCGEventSourceStateCombinedSessionState);
}

static int deskactGetLockStates(bool *caps, bool *num, bool *capsOk, bool *numOk) {
	*caps = false;
	*num = false;
	*capsOk = false;
	*numOk = false;

	io_service_t service = IOServiceGetMatchingService(kIOMasterPortDefault,
		IOServiceMatching(kIOHIDSystemClass));
	if (service == 0) {
		return -1;
	}

	io_connect_t connect = IO_OBJECT_NULL;
	kern_return_t kr = IOServiceOpen(service, mach_task_self(), kIOHIDParamConnectType, &connect);
	IOObjectRelease(service);
	if (kr != KERN_SUCCESS) {
		return -2;
	}

	kr = IOHIDGetModifierLockState(connect, kIOHIDCapsLockState, caps);
	if (kr == KERN_SUCCESS) {
		*capsOk = true;
	}

	kr = IOHIDGetModifierLockState(connect, kIOHIDNumLockState, num);
	if (kr == KERN_SUCCESS) {
		*numOk = true;
	}

	IOServiceClose(connect);
	return 0;
}
*/
import "C"

func currentState() (stateSnapshot, error) {
	var state stateSnapshot
	flags := uint64(C.deskactKeyFlagsState())

	state.supported = bitShift | bitCtrl | bitAlt | bitCmd | bitCapsLock

	if flags&uint64(C.kCGEventFlagMaskShift) != 0 {
		state.mask |= bitShift
	}
	if flags&uint64(C.kCGEventFlagMaskControl) != 0 {
		state.mask |= bitCtrl
	}
	if flags&uint64(C.kCGEventFlagMaskAlternate) != 0 {
		state.mask |= bitAlt
	}
	if flags&uint64(C.kCGEventFlagMaskCommand) != 0 {
		state.mask |= bitCmd
	}
	if flags&uint64(C.kCGEventFlagMaskAlphaShift) != 0 {
		state.mask |= bitCapsLock
	}

	var caps C.bool
	var num C.bool
	var capsOk C.bool
	var numOk C.bool
	if C.deskactGetLockStates(&caps, &num, &capsOk, &numOk) == 0 {
		if capsOk != 0 {
			state.mask &^= bitCapsLock
			if caps != 0 {
				state.mask |= bitCapsLock
			}
		}
		if numOk != 0 {
			state.supported |= bitNumLock
			if num != 0 {
				state.mask |= bitNumLock
			}
		}
	}

	return state, nil
}

