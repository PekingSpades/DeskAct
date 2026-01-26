// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// See the COPYRIGHT file at the top-level directory of this distribution and at
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0>
//
// This file may not be copied, modified, or distributed
// except according to those terms.

#ifndef DISPLAY_C_WINDOWS_H
#define DISPLAY_C_WINDOWS_H

#include "../base/types.h"
#include <windows.h>

// MDT_EFFECTIVE_DPI for GetDpiForMonitor
#ifndef MDT_EFFECTIVE_DPI
#define MDT_EFFECTIVE_DPI 0
#endif

// Function pointer type for GetDpiForMonitor
typedef HRESULT (WINAPI *GetDpiForMonitorFunc)(HMONITOR, int, UINT*, UINT*);

// Get DPI for a monitor using dynamic loading (Windows 8.1+)
static double getMonitorScale(HMONITOR hMonitor) {
    static GetDpiForMonitorFunc pGetDpiForMonitor = NULL;
    static BOOL initialized = FALSE;

    if (!initialized) {
        initialized = TRUE;
        HMODULE hShcore = LoadLibraryW(L"Shcore.dll");
        if (hShcore) {
            pGetDpiForMonitor = (GetDpiForMonitorFunc)GetProcAddress(hShcore, "GetDpiForMonitor");
        }
    }

    if (pGetDpiForMonitor) {
        UINT dpiX = 96, dpiY = 96;
        HRESULT hr = pGetDpiForMonitor(hMonitor, MDT_EFFECTIVE_DPI, &dpiX, &dpiY);
        if (SUCCEEDED(hr) && dpiX > 0) {
            return (double)dpiX / 96.0;
        }
    }
    return 1.0;
}

// DisplayInfoC contains display information in C struct
typedef struct {
    uintptr handle;     // HMONITOR handle
    int32_t index;      // Display index
    int8_t  isMain;     // Is main display
    int32_t x, y, w, h; // Physical coordinates and size
    int32_t vx, vy;     // Virtual (logical) coordinates origin
    int32_t vw, vh;     // Virtual (logical) size
    double  scale;      // Scale factor (physical/logical)
} DisplayInfoC;

// EnumDisplayContext is the enumeration context
typedef struct {
    int32_t       targetIndex;   // Target index (-1 means enumerate all)
    int32_t       currentIndex;  // Current index
    DisplayInfoC* displays;      // Output array
    int32_t       maxCount;      // Max count
    int32_t       foundCount;    // Found count
} EnumDisplayContext;

// Get monitor real physical size (bypassing DPI virtualization)
// Reference: screenshot library's getMonitorRealSize implementation
static int getMonitorRealSize(HMONITOR hMonitor, RECT* outRect) {
    // Step 1: Get device name via GetMonitorInfoW
    MONITORINFOEXW info = {0};
    info.cbSize = sizeof(info);

    if (!GetMonitorInfoW(hMonitor, (MONITORINFO*)&info)) {
        return 0;  // Failed, caller should use logical coordinates as fallback
    }

    // Step 2: Get physical resolution via EnumDisplaySettingsW
    DEVMODEW devMode = {0};
    devMode.dmSize = sizeof(devMode);

    if (!EnumDisplaySettingsW(info.szDevice, ENUM_CURRENT_SETTINGS, &devMode)) {
        return 0;  // Failed
    }

    // Step 3: Build physical coordinate rect
    outRect->left   = devMode.dmPosition.x;
    outRect->top    = devMode.dmPosition.y;
    outRect->right  = devMode.dmPosition.x + (LONG)devMode.dmPelsWidth;
    outRect->bottom = devMode.dmPosition.y + (LONG)devMode.dmPelsHeight;

    return 1;  // Success
}

// Monitor enumeration callback for counting
static BOOL CALLBACK countMonitorCallback(HMONITOR hMonitor, HDC hdcMonitor, LPRECT lprcMonitor, LPARAM dwData) {
    int32_t *count = (int32_t*)dwData;
    (*count)++;
    return TRUE;
}

// Monitor enumeration callback for getting info
static BOOL CALLBACK MonitorInfoEnumProc(HMONITOR hMonitor, HDC hdcMonitor, LPRECT lprcMonitor, LPARAM dwData) {
    EnumDisplayContext* ctx = (EnumDisplayContext*)dwData;

    if (ctx->currentIndex >= ctx->maxCount) {
        return FALSE; // Stop enumeration
    }

    // Get physical size
    RECT physRect;
    int hasPhysical = getMonitorRealSize(hMonitor, &physRect);

    // Get monitor info (to check if main display)
    MONITORINFO mi = {0};
    mi.cbSize = sizeof(mi);
    GetMonitorInfoW(hMonitor, &mi);

    DisplayInfoC* info = &ctx->displays[ctx->currentIndex];
    info->handle = (uintptr)hMonitor;
    info->index = ctx->currentIndex;
    info->isMain = (mi.dwFlags & MONITORINFOF_PRIMARY) ? 1 : 0;

    // Always store virtual (logical) coordinates
    info->vx = lprcMonitor->left;
    info->vy = lprcMonitor->top;
    int32_t logicalW = lprcMonitor->right - lprcMonitor->left;
    int32_t logicalH = lprcMonitor->bottom - lprcMonitor->top;
    info->vw = logicalW;
    info->vh = logicalH;

    if (hasPhysical) {
        // Use physical coordinates
        info->x = physRect.left;
        info->y = physRect.top;
        info->w = physRect.right - physRect.left;
        info->h = physRect.bottom - physRect.top;

        // Calculate scale based on DPI awareness mode
        if (logicalW > 0 && info->w != logicalW) {
            // Physical and logical sizes differ (DPI unaware mode)
            // Windows virtualizes coordinates, use ratio to calculate scale
            info->scale = (double)info->w / (double)logicalW;
        } else {
            // Physical and logical sizes are same (DPI aware mode)
            // Use GetDpiForMonitor to get actual scale
            info->scale = getMonitorScale(hMonitor);
        }
    } else {
        // Fallback: use logical coordinates
        info->x = lprcMonitor->left;
        info->y = lprcMonitor->top;
        info->w = logicalW;
        info->h = logicalH;
        info->scale = 1.0;
    }

    ctx->currentIndex++;
    ctx->foundCount++;
    return TRUE;  // Continue enumeration
}

// Get display count
static int32_t getDisplayCount() {
    int32_t count = 0;
    EnumDisplayMonitors(NULL, NULL, countMonitorCallback, (LPARAM)&count);
    return count;
}

// Get all displays info
static int32_t getAllDisplays(DisplayInfoC* displays, int32_t maxCount) {
    EnumDisplayContext ctx = {0};
    ctx.targetIndex = -1;
    ctx.currentIndex = 0;
    ctx.displays = displays;
    ctx.maxCount = maxCount;
    ctx.foundCount = 0;

    EnumDisplayMonitors(NULL, NULL, MonitorInfoEnumProc, (LPARAM)&ctx);
    return ctx.foundCount;
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
    return empty;
}

#endif /* DISPLAY_C_WINDOWS_H */
