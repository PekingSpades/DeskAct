// Copyright (c) 2016-2025 AtomAI, All rights reserved.

#ifndef MOUSE_C_PID_H
#define MOUSE_C_PID_H

#include "../base/os.h"
#include "mouse.h"

#if defined(IS_MACOSX)
	#include "mouse_c_macos_pid.h"
#elif defined(USE_X11)
	#include "mouse_c_x11_pid.h"
#elif defined(IS_WINDOWS)
	#include "mouse_c_windows_pid.h"
#endif

#endif /* MOUSE_C_PID_H */
