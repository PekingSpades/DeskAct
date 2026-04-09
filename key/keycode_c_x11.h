#include "keycode.h"

MMKeyCode keyCodeForChar(const char c) {
	char buf[2];
	buf[0] = c;
	buf[1] = '\0';

	MMKeyCode code = XStringToKeysym(buf);
	if (code == NoSymbol) {
		/* Some special keys are apparently not handled properly */
		struct XSpecialCharacterMapping* xs = XSpecialCharacterTable;
		while (xs->name) {
			if (c == xs->name) {
				code = xs->code;
				//
				break;
			}
			xs++;
		}
	}

	if (code == NoSymbol) {
		return K_NOT_A_KEY;
	}

	// x11 key bug
	if (c == 60) {
		code = 44;
	}
	return code;
}
