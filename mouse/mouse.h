#pragma once
#ifndef MOUSE_H
#define MOUSE_H

#include "../base/os.h"
#include "../base/types.h"
#include <stdbool.h>
#include <stdint.h>

typedef int32_t MMMouseButton;

typedef enum {
	MM_SCROLL_UNIT_LINE = 0,
	MM_SCROLL_UNIT_PIXEL = 1,
} MMScrollUnit;

#if defined(IS_MACOSX)
	#include <ApplicationServices/ApplicationServices.h>

	#define MM_BUTTON_LEFT kCGMouseButtonLeft
	#define MM_BUTTON_RIGHT kCGMouseButtonRight
	#define MM_BUTTON_MIDDLE kCGMouseButtonCenter
	#define MM_BUTTON_BACK 3
	#define MM_BUTTON_FORWARD 4
#elif defined(USE_X11)
	#define MM_BUTTON_LEFT 1
	#define MM_BUTTON_MIDDLE 2
	#define MM_BUTTON_RIGHT 3
	#define MM_BUTTON_BACK 8
	#define MM_BUTTON_FORWARD 9
#elif defined(IS_WINDOWS)
	#define MM_BUTTON_LEFT 1
	#define MM_BUTTON_MIDDLE 2
	#define MM_BUTTON_RIGHT 3
	#define MM_BUTTON_BACK 4
	#define MM_BUTTON_FORWARD 5
#else
	#error "No mouse button constants set for platform"
#endif

#endif /* MOUSE_H */
