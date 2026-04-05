#include "../base/os.h"

#if defined(IS_MACOSX)
	#include "keycode_c_macos.h"
#elif defined(IS_WINDOWS)
	#include "keycode_c_windows.h"
#elif defined(USE_X11)
	#include "keycode_c_x11.h"
#endif
