// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// See the COPYRIGHT file at the top-level directory of this distribution and at
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0>
//
// This file may not be copied, modified, or distributed
// except according to those terms.

#ifndef DISPLAY_C_X11_H
#define DISPLAY_C_X11_H

#include "../base/types.h"
#include "../base/xdisplay_c.h"
#include <X11/Xlib.h>
#include <X11/Xresource.h>
#include <X11/Xatom.h>
#include <X11/extensions/Xinerama.h>
#include <X11/extensions/Xrandr.h>
#include <stdlib.h>
#include <string.h>

// DisplayInfoC contains display information in C struct
typedef struct {
    uintptr handle;     // Xinerama screen index
    int32_t index;      // Display index
    int8_t  isMain;     // Is main display
    int32_t x, y, w, h; // Physical coordinates and size
    double  scale;      // Scale factor (from Xft.dpi)
    // EDID data for electron ID computation (populated by XRandR)
    uint16_t edidManufacturerId;
    char     edidDisplayName[14];
    int32_t  edidDisplayNameLen;
    uint32_t edidOutputId;
    int8_t   edidValid;   // 1 if matched via XRandR, 0 otherwise
} DisplayInfoC;

// Populate EDID data for electron ID computation using XRandR.
// Parses EDID for manufacturer_id and display_name, gets output_id,
// and matches XRandR outputs to Xinerama screens via CRTC geometry.
// The actual hash and ID computation is done in Go.
static void populateEdidData(Display* dpy, DisplayInfoC* displays, int32_t count) {
    int major, minor;
    if (!XRRQueryVersion(dpy, &major, &minor)) {
        return;
    }

    Window root = DefaultRootWindow(dpy);
    XRRScreenResources* res = XRRGetScreenResourcesCurrent(dpy, root);
    if (!res) {
        return;
    }

    int8_t* matched = (int8_t*)calloc(count, sizeof(int8_t));
    if (!matched) {
        XRRFreeScreenResources(res);
        return;
    }

    for (int o = 0; o < res->noutput; o++) {
        XRROutputInfo* outInfo = XRRGetOutputInfo(dpy, res, res->outputs[o]);
        if (!outInfo) continue;

        if (outInfo->connection != RR_Connected || outInfo->crtc == None) {
            XRRFreeOutputInfo(outInfo);
            continue;
        }

        XRRCrtcInfo* crtcInfo = XRRGetCrtcInfo(dpy, res, outInfo->crtc);
        if (!crtcInfo) {
            XRRFreeOutputInfo(outInfo);
            continue;
        }

        // Parse EDID
        uint16_t manufacturer_id = 0;
        char display_name[14];
        int display_name_len = 0;
        memset(display_name, 0, sizeof(display_name));

        Atom edidAtom = XInternAtom(dpy, "EDID", True);
        if (edidAtom != None) {
            Atom actualType;
            int actualFormat;
            unsigned long nitems, bytesAfter;
            unsigned char* edid = NULL;

            if (XRRGetOutputProperty(dpy, res->outputs[o], edidAtom,
                    0, 128, False, False, AnyPropertyType,
                    &actualType, &actualFormat, &nitems, &bytesAfter, &edid) == Success
                && edid && nitems >= 128) {

                manufacturer_id = ((uint16_t)edid[8] << 8) | (uint16_t)edid[9];

                for (int d = 0; d < 4; d++) {
                    int offset = 54 + d * 18;
                    if (offset + 18 > (int)nitems) break;
                    if (edid[offset] == 0 && edid[offset+1] == 0 &&
                        edid[offset+2] == 0 && edid[offset+3] == 0xFC) {
                        for (int k = 0; k < 13; k++) {
                            char c = (char)edid[offset + 5 + k];
                            if (c == '\n' || c == '\0') break;
                            display_name[display_name_len++] = c;
                        }
                        break;
                    }
                }
            }
            if (edid) XFree(edid);
        }

        uint32_t output_id = (uint32_t)res->outputs[o];

        // Match CRTC geometry to Xinerama screen
        for (int32_t s = 0; s < count; s++) {
            if (matched[s]) continue;
            if ((int)crtcInfo->x == displays[s].x &&
                (int)crtcInfo->y == displays[s].y &&
                (int)crtcInfo->width == displays[s].w &&
                (int)crtcInfo->height == displays[s].h) {
                displays[s].edidManufacturerId = manufacturer_id;
                memcpy(displays[s].edidDisplayName, display_name, display_name_len);
                displays[s].edidDisplayNameLen = (int32_t)display_name_len;
                displays[s].edidOutputId = output_id;
                displays[s].edidValid = 1;
                matched[s] = 1;
                break;
            }
        }

        XRRFreeCrtcInfo(crtcInfo);
        XRRFreeOutputInfo(outInfo);
    }

    free(matched);
    XRRFreeScreenResources(res);
}

// Get scale factor from Xft.dpi
static double getX11Scale() {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) {
        return 1.0;
    }

    double scale = 1.0;
    char *rms = XResourceManagerString(dpy);
    if (rms) {
        XrmDatabase db = XrmGetStringDatabase(rms);
        if (db) {
            XrmValue value;
            char *type = NULL;

            if (XrmGetResource(db, "Xft.dpi", "String", &type, &value)) {
                if (value.addr) {
                    double dpi = atof(value.addr);
                    scale = dpi / 96.0;
                }
            }

            XrmDestroyDatabase(db);
        }
    }
    XCloseDisplay(dpy);

    return scale;
}

// Get display count
static int32_t getDisplayCount() {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) {
        return 1;
    }

    int event_base, error_base;
    if (XineramaQueryExtension(dpy, &event_base, &error_base) &&
        XineramaIsActive(dpy)) {
        int count;
        XineramaScreenInfo *info = XineramaQueryScreens(dpy, &count);
        if (info) {
            XFree(info);
            XCloseDisplay(dpy);
            return (int32_t)count;
        }
    }

    XCloseDisplay(dpy);
    return 1;  // Single display fallback
}

// Get all displays info
static int32_t getAllDisplays(DisplayInfoC* displays, int32_t maxCount) {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) {
        // Fallback: return single display with default screen size
        if (maxCount > 0) {
            Display *mainDpy = XGetMainDisplay();
            if (mainDpy) {
                int screen = DefaultScreen(mainDpy);
                displays[0].handle = 0;
                displays[0].index = 0;
                displays[0].isMain = 1;
                displays[0].x = 0;
                displays[0].y = 0;
                displays[0].w = DisplayWidth(mainDpy, screen);
                displays[0].h = DisplayHeight(mainDpy, screen);
                displays[0].scale = 1.0;
                return 1;
            }
        }
        return 0;
    }

    double scale = getX11Scale();

    int event_base, error_base;
    if (XineramaQueryExtension(dpy, &event_base, &error_base) &&
        XineramaIsActive(dpy)) {
        int count;
        XineramaScreenInfo *screens = XineramaQueryScreens(dpy, &count);
        if (screens) {
            int32_t resultCount = (count < maxCount) ? count : maxCount;

            for (int32_t i = 0; i < resultCount; i++) {
                displays[i].handle = (uintptr)screens[i].screen_number;
                displays[i].index = i;
                displays[i].isMain = (i == 0) ? 1 : 0;  // First screen is main
                displays[i].x = screens[i].x_org;
                displays[i].y = screens[i].y_org;
                displays[i].w = screens[i].width;
                displays[i].h = screens[i].height;
                displays[i].scale = scale;
            }

            XFree(screens);
            populateEdidData(dpy, displays, resultCount);
            XCloseDisplay(dpy);
            return resultCount;
        }
    }

    // Fallback: single display
    if (maxCount > 0) {
        int screen = DefaultScreen(dpy);
        displays[0].handle = 0;
        displays[0].index = 0;
        displays[0].isMain = 1;
        displays[0].x = 0;
        displays[0].y = 0;
        displays[0].w = DisplayWidth(dpy, screen);
        displays[0].h = DisplayHeight(dpy, screen);
        displays[0].scale = scale;
        XCloseDisplay(dpy);
        return 1;
    }

    XCloseDisplay(dpy);
    return 0;
}

// Get main display info
static DisplayInfoC getMainDisplay() {
    DisplayInfoC displays[32];
    int32_t count = getAllDisplays(displays, 32);

    for (int32_t i = 0; i < count; i++) {
        if (displays[i].isMain) {
            return displays[i];
        }
    }

    // Fallback: return first display
    if (count > 0) {
        return displays[0];
    }

    // No display found
    DisplayInfoC empty = {0};
    empty.scale = 1.0;
    return empty;
}

// Get display at index
static DisplayInfoC getDisplayAt(int32_t index) {
    DisplayInfoC displays[32];
    int32_t count = getAllDisplays(displays, 32);

    if (index >= 0 && index < count) {
        return displays[index];
    }

    // Index out of range
    DisplayInfoC empty = {0};
    empty.scale = 1.0;
    return empty;
}

#endif /* DISPLAY_C_X11_H */
