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
// #include </System/Library/Frameworks/ApplicationServices.framework/Headers/ApplicationServices.h>
#include <ApplicationServices/ApplicationServices.h>
// #include </System/Library/Frameworks/ApplicationServices.framework/Versions/A/Headers/ApplicationServices.h>

/* Some convenience macros for converting our enums to the system API types. */
CGEventType MMMouseDownToCGEventType(MMMouseButton button) {
	if (button == MM_BUTTON_LEFT) {
		return kCGEventLeftMouseDown;
	}
	if (button == MM_BUTTON_RIGHT) {
		return kCGEventRightMouseDown;
	}
	return kCGEventOtherMouseDown;
}

CGEventType MMMouseUpToCGEventType(MMMouseButton button) {
	if (button == MM_BUTTON_LEFT) { return kCGEventLeftMouseUp; }
	if (button == MM_BUTTON_RIGHT) { return kCGEventRightMouseUp; }
	return kCGEventOtherMouseUp;
}

CGEventType MMMouseDragToCGEventType(MMMouseButton button) {
	if (button == MM_BUTTON_LEFT) { return kCGEventLeftMouseDragged; }
	if (button == MM_BUTTON_RIGHT) { return kCGEventRightMouseDragged; }
	return kCGEventOtherMouseDragged;
}

CGEventType MMMouseToCGEventType(bool down, MMMouseButton button) {
	if (down) { return MMMouseDownToCGEventType(button); }
	return MMMouseUpToCGEventType(button);
}

/* Calculate the delta for a mouse move and add them to the event. */
void calculateDeltas(CGEventRef *event, MMPointInt32 point) {
	/* The next few lines are a workaround for games not detecting mouse moves. */
	CGEventRef get = CGEventCreate(NULL);
	CGPoint mouse = CGEventGetLocation(get);

	// Calculate the deltas.
	int64_t deltaX = point.x - mouse.x;
	int64_t deltaY = point.y - mouse.y;

	CGEventSetIntegerValueField(*event, kCGMouseEventDeltaX, deltaX);
	CGEventSetIntegerValueField(*event, kCGMouseEventDeltaY, deltaY);

	CFRelease(get);
}

/* Poll until the system reports the cursor at the expected position,
   or until maxWaitMs milliseconds have elapsed. */
static void waitForCursorSync(MMPointInt32 target, double maxWaitMs) {
	const double pollIntervalMs = 1.0;
	double elapsed = 0.0;
	while (elapsed < maxWaitMs) {
		MMPointInt32 cur = location();
		if (cur.x == target.x && cur.y == target.y) {
			return;
		}
		microsleep(pollIntervalMs);
		elapsed += pollIntervalMs;
	}
}

/* Move the mouse to a specific point. */
void moveMouse(MMPointInt32 point){
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	CGEventRef move = CGEventCreateMouseEvent(source, kCGEventMouseMoved,
							CGPointFromMMPointInt32(point), kCGMouseButtonLeft);

	calculateDeltas(&move, point);

	CGEventPost(kCGHIDEventTap, move);
	CFRelease(move);
	CFRelease(source);

	waitForCursorSync(point, 50.0);
}

void dragMouse(MMPointInt32 point, const MMMouseButton button){
	const CGEventType dragType = MMMouseDragToCGEventType(button);
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	CGEventRef drag = CGEventCreateMouseEvent(source, dragType,
							CGPointFromMMPointInt32(point), (CGMouseButton)button);

	calculateDeltas(&drag, point);

	CGEventPost(kCGHIDEventTap, drag);
	CFRelease(drag);
	CFRelease(source);

	waitForCursorSync(point, 50.0);
}

MMPointInt32 location() {
	CGEventRef event = CGEventCreate(NULL);
	CGPoint point = CGEventGetLocation(event);
	CFRelease(event);

	return MMPointInt32FromCGPoint(point);
}

/* Press down a button, or release it. */
int toggleMouseErr(bool down, MMMouseButton button) {
	const CGPoint currentPos = CGPointFromMMPointInt32(location());
	const CGEventType mouseType = MMMouseToCGEventType(down, button);
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	CGEventRef event = CGEventCreateMouseEvent(source, mouseType, currentPos, (CGMouseButton)button);

	if (event == NULL) {
		CFRelease(source);
		return (int)kCGErrorCannotComplete;
	}

	CGEventPost(kCGHIDEventTap, event);
	CFRelease(event);
	CFRelease(source);

	return 0;
}

/* Multi-click function supporting any click count (1=single, 2=double, 3=triple, etc.) */
int multiClickErr(MMMouseButton button, int clickCount){
	if (clickCount < 1) {
		return 0;
	}

	const CGPoint currentPos = CGPointFromMMPointInt32(location());
	const CGEventType mouseTypeDown = MMMouseToCGEventType(true, button);
	const CGEventType mouseTypeUP = MMMouseToCGEventType(false, button);

	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	if (source == NULL) {
		return (int)kCGErrorCannotComplete;
	}

	int i;
	for (i = 0; i < clickCount; i++) {
		CGEventRef down = CGEventCreateMouseEvent(source, mouseTypeDown, currentPos, (CGMouseButton)button);
		if (down == NULL) {
			CFRelease(source);
			return (int)kCGErrorCannotComplete;
		}

		CGEventSetIntegerValueField(down, kCGMouseEventClickState, i + 1);
		CGEventPost(kCGHIDEventTap, down);
		CFRelease(down);

		microsleep(5.0);

		CGEventRef up = CGEventCreateMouseEvent(source, mouseTypeUP, currentPos, (CGMouseButton)button);
		if (up == NULL) {
			CFRelease(source);
			return (int)kCGErrorCannotComplete;
		}

		CGEventSetIntegerValueField(up, kCGMouseEventClickState, i + 1);
		CGEventPost(kCGHIDEventTap, up);
		CFRelease(up);

		if (i < clickCount - 1) {
			microsleep(200);
		}
	}

	CFRelease(source);

	return 0;
}

/* Function used to scroll the screen in the required direction. */
void scrollMouseXY(int x, int y, MMScrollUnit unit) {
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	CGScrollEventUnit cgUnit = unit == MM_SCROLL_UNIT_LINE ? kCGScrollEventUnitLine : kCGScrollEventUnitPixel;
	CGEventRef event = CGEventCreateScrollWheelEvent(source, cgUnit, 2, y, x);
	CGEventPost(kCGHIDEventTap, event);

	CFRelease(event);
	CFRelease(source);
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

bool smoothlyMoveMouse(MMPointInt32 startPoint, MMPointInt32 endPoint, double lowSpeed, double highSpeed){
	MMPointInt32 pos = startPoint;
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

bool smoothlyDragMouse(MMPointInt32 startPoint, MMPointInt32 endPoint, const MMMouseButton button, double lowSpeed, double highSpeed){
	MMPointInt32 pos = startPoint;
	double velo_x = 0.0, velo_y = 0.0;
	double distance;

	while ((distance =crude_hypot((double)pos.x - endPoint.x, (double)pos.y - endPoint.y)) > 1.0) {
		double gravity = DEADBEEF_UNIFORM(5.0, 500.0);
		double veloDistance;
		velo_x += (gravity * ((double)endPoint.x - pos.x)) / distance;
		velo_y += (gravity * ((double)endPoint.y - pos.y)) / distance;

		/* Normalize velocity to get a unit vector of length 1. */
		veloDistance = crude_hypot(velo_x, velo_y);
		velo_x /= veloDistance;
		velo_y /= veloDistance;

		pos.x += floor(velo_x + 0.5);
		pos.y += floor(velo_y + 0.5);

		dragMouse(pos, button);

		/* Wait 1 - 3 milliseconds. */
		microsleep(DEADBEEF_UNIFORM(lowSpeed, highSpeed));
	}

	return true;
}
