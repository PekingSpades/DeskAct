// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// See the COPYRIGHT file at the top-level directory of this distribution and at
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0>
//
// This file may not be copied, modified, or distributed
// except according to those terms.

#include "mouse.h"
#include "../base/deadbeef_rand.h"
#include "../base/microsleep.h"

#include <math.h> /* For floor() */

/* Some convenience macros for converting our enums to the system API types. */
DWORD MMMouseUpToMEventF(MMMouseButton button) {
	if (button == LEFT_BUTTON) { return MOUSEEVENTF_LEFTUP; }
	if (button == RIGHT_BUTTON) { return MOUSEEVENTF_RIGHTUP; }
	return MOUSEEVENTF_MIDDLEUP;
}

DWORD MMMouseDownToMEventF(MMMouseButton button) {
	if (button == LEFT_BUTTON) { return MOUSEEVENTF_LEFTDOWN; }
	if (button == RIGHT_BUTTON) { return MOUSEEVENTF_RIGHTDOWN; }
	return MOUSEEVENTF_MIDDLEDOWN;
}

DWORD MMMouseToMEventF(bool down, MMMouseButton button) {
	if (down) { return MMMouseDownToMEventF(button); }
	return MMMouseUpToMEventF(button);
}

/* Move the mouse to a specific point. */
void moveMouse(MMPointInt32 point){
	SetPhysicalCursorPos(point.x, point.y);
}

void dragMouse(MMPointInt32 point, const MMMouseButton button){
	moveMouse(point);
}

MMPointInt32 location() {
	POINT point;
	GetPhysicalCursorPos(&point);
	return MMPointInt32FromPOINT(point);
}

/* Press down a button, or release it. */
int toggleMouseErr(bool down, MMMouseButton button) {
	// mouse_event(MMMouseToMEventF(down, button), 0, 0, 0, 0);
	INPUT mouseInput;

	mouseInput.type = INPUT_MOUSE;
	mouseInput.mi.dx = 0;
	mouseInput.mi.dy = 0;
	mouseInput.mi.dwFlags = MMMouseToMEventF(down, button);
	mouseInput.mi.time = 0;
	mouseInput.mi.dwExtraInfo = 0;
	mouseInput.mi.mouseData = 0;
	UINT sent = SendInput(1, &mouseInput, sizeof(mouseInput));
	return sent == 1 ? 0 : (int)GetLastError();
}

/* Multi-click function supporting any click count (1=single, 2=double, 3=triple, etc.) */
int multiClickErr(MMMouseButton button, int clickCount){
	if (clickCount < 1) {
		return 0;
	}

	int i;
	for (i = 0; i < clickCount; i++) {
		int err = toggleMouseErr(true, button);
		if (err != 0) {
			return err;
		}
		microsleep(5.0);
		err = toggleMouseErr(false, button);
		if (err != 0) {
			return err;
		}
		if (i < clickCount - 1) {
			microsleep(200);
		}
	}
	return 0;
}

/* Function used to scroll the screen in the required direction. */
void scrollMouseXY(int x, int y) {
	// Fix for #97, C89 needs variables declared on top of functions (mouseScrollInput)
	INPUT mouseScrollInputH;
	INPUT mouseScrollInputV;

	mouseScrollInputH.type = INPUT_MOUSE;
	mouseScrollInputH.mi.dx = 0;
	mouseScrollInputH.mi.dy = 0;
	mouseScrollInputH.mi.dwFlags = MOUSEEVENTF_WHEEL;
	mouseScrollInputH.mi.time = 0;
	mouseScrollInputH.mi.dwExtraInfo = 0;
	mouseScrollInputH.mi.mouseData = WHEEL_DELTA * x;

	mouseScrollInputV.type = INPUT_MOUSE;
	mouseScrollInputV.mi.dx = 0;
	mouseScrollInputV.mi.dy = 0;
	mouseScrollInputV.mi.dwFlags = MOUSEEVENTF_WHEEL;
	mouseScrollInputV.mi.time = 0;
	mouseScrollInputV.mi.dwExtraInfo = 0;
	mouseScrollInputV.mi.mouseData = WHEEL_DELTA * y;

	SendInput(1, &mouseScrollInputH, sizeof(mouseScrollInputH));
	SendInput(1, &mouseScrollInputV, sizeof(mouseScrollInputV));
}

/* A crude, fast hypot() approximation to get around the fact that hypot() is not a standard ANSI C function. */
#if !defined(M_SQRT2)
	#define M_SQRT2 1.4142135623730950488016887 /* Fix for MSVC. */
#endif

static double crude_hypot(double x, double y){
	double big = fabs(x); /* max(|x|, |y|) */
	double small = fabs(y); /* min(|x|, |y|) */

	if (big > small) {
		double temp = big;
		big = small;
		small = temp;
	}

	return ((M_SQRT2 - 1.0) * small) + big;
}

bool smoothlyMoveMouse(MMPointInt32 endPoint, double lowSpeed, double highSpeed){
	MMPointInt32 pos = location();
	// MMSizeInt32 screenSize = getMainDisplaySize();
	double velo_x = 0.0, velo_y = 0.0;
	double distance;

	while ((distance =crude_hypot((double)pos.x - endPoint.x, (double)pos.y - endPoint.y)) > 1.0) {
		double gravity = DEADBEEF_UNIFORM(5.0, 500.0);
		// double gravity = DEADBEEF_UNIFORM(lowSpeed, highSpeed);
		double veloDistance;
		velo_x += (gravity * ((double)endPoint.x - pos.x)) / distance;
		velo_y += (gravity * ((double)endPoint.y - pos.y)) / distance;

		/* Normalize velocity to get a unit vector of length 1. */
		veloDistance = crude_hypot(velo_x, velo_y);
		velo_x /= veloDistance;
		velo_y /= veloDistance;

		pos.x += floor(velo_x + 0.5);
		pos.y += floor(velo_y + 0.5);

		/* Make sure we are in the screen boundaries! (Strange things will happen if we are not.) */
		// if (pos.x >= screenSize.w || pos.y >= screenSize.h) {
		// 	return false;
		// }
		moveMouse(pos);

		/* Wait 1 - 3 milliseconds. */
		microsleep(DEADBEEF_UNIFORM(lowSpeed, highSpeed));
		// microsleep(DEADBEEF_UNIFORM(1.0, 3.0));
	}

	return true;
}
