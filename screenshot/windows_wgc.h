// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// windows_wgc.h - flat C ABI for the Windows.Graphics.Capture C++ wrapper.
// The implementation lives in windows_wgc.cpp.

#ifndef WINDOWS_WGC_H
#define WINDOWS_WGC_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
	uint8_t* data;
	int      width;
	int      height;
	int      stride;
} wgc_image_t;

/* 1 if the Windows.Graphics.Capture runtime is available on this system,
 * 0 otherwise. Uses LoadLibraryW("Windows.Graphics.Capture.dll") + LoadLibrary
 * of combase.dll to feature-detect at runtime. Safe to call from any thread. */
int wgc_available(void);

/* Capture a single frame of the supplied HWND. On success returns 0 and
 * fills *out with a malloc'd BGRA buffer the caller must release with
 * wgc_free. Non-zero return = capture failed; *out is zeroed.
 *
 * Status codes:
 *   0 : success
 *  -1 : invalid argument (hwnd == 0 or out == NULL)
 *  -2 : runtime unavailable (LoadLibrary failed)
 *  -3 : COM init failed
 *  -4 : capture item creation failed
 *  -5 : D3D11 device creation failed
 *  -6 : frame pool / session setup failed
 *  -7 : timed out waiting for first frame
 *  -8 : texture readback failed
 */
int wgc_capture_window(uintptr_t hwnd, wgc_image_t* out);

void wgc_free(wgc_image_t* img);

#ifdef __cplusplus
}
#endif

#endif /* WINDOWS_WGC_H */
