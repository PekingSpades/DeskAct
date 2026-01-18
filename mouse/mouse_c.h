// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// See the COPYRIGHT file at the top-level directory of this distribution and at
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0>
//
// This file may not be copied, modified, or distributed
// except according to those terms.

#if defined(IS_MACOSX)
	#include "mouse_c_macos.h"
#elif defined(USE_X11)
	#include "mouse_c_x11.h"
#elif defined(IS_WINDOWS)
	#include "mouse_c_windows.h"
#endif
