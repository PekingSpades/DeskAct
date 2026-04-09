#include "keycode.h"

MMKeyCode keyCodeForChar(const char c) {
	MMKeyCode code;
	code = VkKeyScan(c);
	if (code == 0xFFFF) {
		return K_NOT_A_KEY;
	}

	return code;
}
