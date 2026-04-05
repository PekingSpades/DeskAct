// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// See the COPYRIGHT file at the top-level directory of this distribution and at
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0>
//
// This file may not be copied, modified, or distributed
// except according to those terms.

#include "../base/os.h"

#if defined(IS_WINDOWS)
	#include "keypress_c_windows.h"
#elif defined(USE_X11)
	#include "keypress_c_x11.h"
#elif defined(IS_MACOSX)
	#include "keypress_c_macos.h"
#endif
