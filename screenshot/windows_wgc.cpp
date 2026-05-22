// Copyright (c) 2016-2025 AtomAI, All rights reserved.
//
// windows_wgc.cpp - Windows.Graphics.Capture wrapper stub.
//
// IMPORTANT: The full WGC integration is intentionally not implemented yet.
// mingw-w64's WinRT ABI headers (windows.graphics.capture.h /
// windows.graphics.capture.interop.h shipped with gcc-mingw-w64 13.x) have
// the following issues that block the standard cppwinrt-style implementation:
//
//   1. `__uuidof` in mingw is `__mingw_uuidof<T>()`, a single-template-argument
//      helper that does NOT accept multi-arg templates like
//      `ITypedEventHandler<Direct3D11CaptureFramePool*, IInspectable*>`.
//   2. `windows.foundation.h` has duplicate `IReference<boolean>` /
//      `IReference<BYTE>` definitions that fail compilation when included
//      via `windows.graphics.capture.h`.
//   3. `IDirect3DDxgiInterfaceAccess` is not declared inside
//      `ABI::Windows::Graphics::DirectX::Direct3D11` in mingw; the C ABI
//      macros must be used instead.
//
// Implementing WGC against the raw IUnknown / IInspectable C ABI through
// COBJMACROS is feasible but is a significant separate effort and warrants
// its own focused PR + cross-platform test pass.
//
// What ships today:
//   - wgc_available() returns 0 -> callers ALWAYS fall back to PrintWindow
//     (capture_window_printwindow in capture_window_windows.go).
//   - wgc_capture_window() returns -2 (runtime unavailable).
//   - The build infrastructure (mingw package list in
//     scripts/non-github-ci/bootstrap-windows.ps1, cgo LDFLAGS) is correct
//     and ready for the future WGC C ABI implementation to drop in.

#ifdef _WIN32

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <windows.h>
#include "windows_wgc.h"

#include <cstdlib>

extern "C" int wgc_available(void) {
	return 0;
}

extern "C" int wgc_capture_window(uintptr_t hwndPtr, wgc_image_t* out) {
	(void)hwndPtr;
	if (out != nullptr) {
		out->data = nullptr;
		out->width = 0;
		out->height = 0;
		out->stride = 0;
	}
	return -2;
}

extern "C" void wgc_free(wgc_image_t* img) {
	if (img && img->data) {
		free(img->data);
		img->data = nullptr;
		img->width = img->height = img->stride = 0;
	}
}

#endif /* _WIN32 */
