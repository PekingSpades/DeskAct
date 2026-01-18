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
#include "../base/xdisplay_c.h"

#include <math.h> /* For floor() */
#include <X11/Xlib.h>
#include <X11/extensions/XTest.h>
#include <stdlib.h>

/* Move the mouse to a specific point. */
void moveMouse(MMPointInt32 point){
	Display *display = XGetMainDisplay();
	XWarpPointer(display, None, DefaultRootWindow(display), 0, 0, 0, 0, point.x, point.y);

	XSync(display, false);
}

void dragMouse(MMPointInt32 point, const MMMouseButton button){
	moveMouse(point);
}

MMPointInt32 location() {
	int x, y; 	/* This is all we care about. Seriously. */
	Window garb1, garb2; 	/* Why you can't specify NULL as a parameter */
	int garb_x, garb_y;  	/* is beyond me. */
	unsigned int more_garbage;

	Display *display = XGetMainDisplay();
	XQueryPointer(display, XDefaultRootWindow(display), &garb1, &garb2, &x, &y,
					&garb_x, &garb_y, &more_garbage);

	return MMPointInt32Make(x, y);
}

/* Press down a button, or release it. */
int toggleMouseErr(bool down, MMMouseButton button) {
	Display *display = XGetMainDisplay();
	Status status = XTestFakeButtonEvent(display, button, down ? True : False, CurrentTime);
	XSync(display, false);

	return status ? 0 : 1;
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
void scrollMouseXY(int x, int y, MMScrollUnit unit) {
	(void)unit;
	int ydir = 4; /* Button 4 is up, 5 is down. */
	int xdir = 6;
	Display *display = XGetMainDisplay();

	if (y < 0) { ydir = 5; }
	if (x < 0) { xdir = 7; }

	int xi; int yi;
	for (xi = 0; xi < abs(x); xi++) {
		XTestFakeButtonEvent(display, xdir, 1, CurrentTime);
		XTestFakeButtonEvent(display, xdir, 0, CurrentTime);
	}
	for (yi = 0; yi < abs(y); yi++) {
		XTestFakeButtonEvent(display, ydir, 1, CurrentTime);
		XTestFakeButtonEvent(display, ydir, 0, CurrentTime);
	}

	XSync(display, false);
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
