#ifndef XDISPLAY_C_H
#define XDISPLAY_C_H

#include <stdio.h> /* For fputs() */
#include <stdlib.h> /* For atexit() */
#include <string.h> /* For strdup() */
#include <X11/Xlib.h>

static Display *mainDisplay = NULL;
static int registered = 0;

static char *displayName = NULL;
static int hasDisplayNameChanged = 0;

static void XCloseMainDisplay(void) {
	if (mainDisplay != NULL) {
		XCloseDisplay(mainDisplay);
		mainDisplay = NULL;
	}
}

static Display *XGetMainDisplay(void) {
	/* Close the display if displayName has changed */
	if (hasDisplayNameChanged) {
		XCloseMainDisplay();
		hasDisplayNameChanged = 0;
	}

	if (mainDisplay == NULL) {
		/* First try the user set displayName */
		mainDisplay = XOpenDisplay(displayName);

		/* Then try using environment variable DISPLAY */
		if (mainDisplay == NULL && displayName != NULL) {
			mainDisplay = XOpenDisplay(NULL);
		}

		/* Fall back to the most likely :0.0*/
		if (mainDisplay == NULL) {
			mainDisplay = XOpenDisplay(":0.0");
		}

		if (mainDisplay == NULL) {
			fputs("Could not open main display\n", stderr);
		} else if (!registered) {
			atexit(&XCloseMainDisplay);
			registered = 1;
		}
	}

	return mainDisplay;
}

static void setXDisplay(char *name) {
	displayName = strdup(name);
	hasDisplayNameChanged = 1;
}

static char *getXDisplay(void) {
	return displayName;
}

#endif /* XDISPLAY_C_H */
