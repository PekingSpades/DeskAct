/* Copyright (c) 2016-2025 AtomAI, All rights reserved.
 *
 * windows_wgc.c - Windows.Graphics.Capture per-window capture, implemented
 * in C against the COBJMACROS / WIDL-friendly-name C ABI exposed by
 * mingw-w64's WinRT headers. Compiled with the cgo C compiler (not C++)
 * to dodge the cppwinrt-style template issues that block windows_wgc.cpp
 * on mingw 13.x.
 *
 * Capture loop:
 *   1. RoInitialize (single-threaded apartment).
 *   2. RoGetActivationFactory("Windows.Graphics.Capture.GraphicsCaptureItem")
 *      -> IGraphicsCaptureItemInterop, then CreateForWindow(hwnd, ...).
 *   3. D3D11CreateDevice + CreateDirect3D11DeviceFromDXGIDevice to wrap as
 *      IDirect3DDevice for WGC.
 *   4. IDirect3D11CaptureFramePoolStatics::Create -> pool, then
 *      CreateCaptureSession -> session, then StartCapture.
 *   5. POLL TryGetNextFrame() up to ~1.5 s instead of subscribing to the
 *      FrameArrived event (the WinRT TypedEventHandler<T,U> template is
 *      not usable through mingw's C ABI).
 *   6. Frame -> Surface -> IDirect3DDxgiInterfaceAccess::GetInterface ->
 *      ID3D11Texture2D. Copy into a staging texture, Map for CPU read,
 *      memcpy into the caller's malloc'd buffer.
 *
 * All WinRT entry points (RoInitialize, RoGetActivationFactory,
 * WindowsCreateString, etc., plus CreateDirect3D11DeviceFromDXGIDevice)
 * are resolved with LoadLibrary + GetProcAddress so the resulting binary
 * still loads on Windows 8 and older, where wgc_available() returns 0.
 */

#ifdef _WIN32

/* INITGUID before any windows headers makes DEFINE_GUID() emit storage for
 * the IID symbols rather than mere declarations. mingw-w64 ships the IID
 * declarations in the WGC headers but has no .lib that defines them, so
 * this file is the single translation unit that materialises them. */
#define INITGUID

#define COBJMACROS
#define WIDL_using_Windows_Graphics
#define WIDL_using_Windows_Graphics_Capture
#define WIDL_using_Windows_Graphics_DirectX
#define WIDL_using_Windows_Graphics_DirectX_Direct3D11
#define WIDL_using_Windows_Foundation

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#ifndef NOMINMAX
#define NOMINMAX
#endif

#include <windows.h>
#include <initguid.h>
#include <objbase.h>
#include <inspectable.h>
#include <roapi.h>
#include <winstring.h>

#include <d3d11.h>
#include <dxgi.h>

#include <windows.graphics.capture.h>
#include <windows.graphics.capture.interop.h>
#include <windows.graphics.directx.direct3d11.h>

#include "windows_wgc.h"

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

/* Manually declare IDirect3DDxgiInterfaceAccess — mingw 13.x does not ship
 * windows.graphics.directx.direct3d11.interop.h. The IID and one-method
 * vtable are stable since Windows 8.1. */
typedef struct IDirect3DDxgiInterfaceAccess IDirect3DDxgiInterfaceAccess;
typedef struct IDirect3DDxgiInterfaceAccessVtbl {
    HRESULT (STDMETHODCALLTYPE *QueryInterface)(IDirect3DDxgiInterfaceAccess*, REFIID, void**);
    ULONG   (STDMETHODCALLTYPE *AddRef)(IDirect3DDxgiInterfaceAccess*);
    ULONG   (STDMETHODCALLTYPE *Release)(IDirect3DDxgiInterfaceAccess*);
    HRESULT (STDMETHODCALLTYPE *GetInterface)(IDirect3DDxgiInterfaceAccess*, REFIID, void**);
} IDirect3DDxgiInterfaceAccessVtbl;
struct IDirect3DDxgiInterfaceAccess {
    const IDirect3DDxgiInterfaceAccessVtbl *lpVtbl;
};
static const GUID IID_IDirect3DDxgiInterfaceAccess = {
    0xA9B3D012, 0x3DF2, 0x4EE3, {0xB8, 0xD1, 0x86, 0x95, 0xF4, 0x57, 0xD3, 0xC1}
};

/* Lazy-loaded WinRT entry points. */
typedef HRESULT (WINAPI *PFN_RoInitialize)(RO_INIT_TYPE);
typedef HRESULT (WINAPI *PFN_RoGetActivationFactory)(HSTRING, REFIID, void**);
typedef HRESULT (WINAPI *PFN_WindowsCreateString)(PCNZWCH, UINT32, HSTRING*);
typedef HRESULT (WINAPI *PFN_WindowsDeleteString)(HSTRING);
typedef HRESULT (WINAPI *PFN_CreateDirect3D11DeviceFromDXGIDevice)(IDXGIDevice*, IInspectable**);

static struct {
    HMODULE combase;
    HMODULE d3d11;
    HMODULE capdll;
    PFN_RoInitialize RoInitialize_;
    PFN_RoGetActivationFactory RoGetActivationFactory_;
    PFN_WindowsCreateString WindowsCreateString_;
    PFN_WindowsDeleteString WindowsDeleteString_;
    PFN_CreateDirect3D11DeviceFromDXGIDevice CreateDirect3D11DeviceFromDXGIDevice_;
    int loaded;       /* 0 = not tried, 1 = ok, -1 = failed */
    CRITICAL_SECTION lock;
    int lockInit;
} g_rt;

static void init_rt_lock_once(void) {
    static LONG inited = 0;
    if (InterlockedCompareExchange(&inited, 1, 0) == 0) {
        InitializeCriticalSection(&g_rt.lock);
        g_rt.lockInit = 1;
    }
    /* spin-wait until the initializer has actually created the section */
    while (!g_rt.lockInit) {
        SwitchToThread();
    }
}

static int load_runtime(void) {
    init_rt_lock_once();
    EnterCriticalSection(&g_rt.lock);
    if (g_rt.loaded != 0) {
        int rc = g_rt.loaded;
        LeaveCriticalSection(&g_rt.lock);
        return rc == 1 ? 0 : -2;
    }
    g_rt.combase = LoadLibraryW(L"combase.dll");
    g_rt.d3d11   = LoadLibraryW(L"d3d11.dll");
    g_rt.capdll  = LoadLibraryW(L"Windows.Graphics.Capture.dll");

    if (g_rt.combase) {
        g_rt.RoInitialize_           = (PFN_RoInitialize)(void*)GetProcAddress(g_rt.combase, "RoInitialize");
        g_rt.RoGetActivationFactory_ = (PFN_RoGetActivationFactory)(void*)GetProcAddress(g_rt.combase, "RoGetActivationFactory");
        g_rt.WindowsCreateString_    = (PFN_WindowsCreateString)(void*)GetProcAddress(g_rt.combase, "WindowsCreateString");
        g_rt.WindowsDeleteString_    = (PFN_WindowsDeleteString)(void*)GetProcAddress(g_rt.combase, "WindowsDeleteString");
    }
    if (g_rt.d3d11) {
        g_rt.CreateDirect3D11DeviceFromDXGIDevice_ =
            (PFN_CreateDirect3D11DeviceFromDXGIDevice)(void*)GetProcAddress(g_rt.d3d11, "CreateDirect3D11DeviceFromDXGIDevice");
    }
    int ok = (g_rt.combase && g_rt.d3d11 && g_rt.capdll
        && g_rt.RoInitialize_ && g_rt.RoGetActivationFactory_
        && g_rt.WindowsCreateString_ && g_rt.WindowsDeleteString_
        && g_rt.CreateDirect3D11DeviceFromDXGIDevice_) ? 1 : 0;
    g_rt.loaded = ok ? 1 : -1;
    LeaveCriticalSection(&g_rt.lock);
    return ok ? 0 : -2;
}

int wgc_available(void) {
    return load_runtime() == 0 ? 1 : 0;
}

void wgc_free(wgc_image_t* img) {
    if (img && img->data) {
        free(img->data);
        img->data = NULL;
        img->width = img->height = img->stride = 0;
    }
}

static HSTRING make_hs(const wchar_t* s) {
    HSTRING h = NULL;
    g_rt.WindowsCreateString_(s, (UINT32)wcslen(s), &h);
    return h;
}

int wgc_capture_window(uintptr_t hwndPtr, wgc_image_t* out) {
    if (!out || hwndPtr == 0) return -1;
    out->data = NULL; out->width = out->height = out->stride = 0;

    int rc = load_runtime();
    if (rc != 0) return rc;

    HRESULT hr = g_rt.RoInitialize_(RO_INIT_SINGLETHREADED);
    if (FAILED(hr) && hr != (HRESULT)0x80010106L /* RPC_E_CHANGED_MODE */)
        return -3;

    HWND hwnd = (HWND)hwndPtr;

    /* --- Capture item via interop ---------------------------------------- */
    HSTRING ciName = make_hs(L"Windows.Graphics.Capture.GraphicsCaptureItem");
    IGraphicsCaptureItemInterop* interop = NULL;
    hr = g_rt.RoGetActivationFactory_(ciName, &IID_IGraphicsCaptureItemInterop, (void**)&interop);
    g_rt.WindowsDeleteString_(ciName);
    if (FAILED(hr) || !interop) return -4;

    __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem* item = NULL;
    hr = interop->lpVtbl->CreateForWindow(interop, hwnd,
        &IID___x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem,
        (void**)&item);
    interop->lpVtbl->Release(interop);
    if (FAILED(hr) || !item) return -4;

    __x_ABI_CWindows_CGraphics_CSizeInt32 itemSize = {0};
    __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_get_Size(item, &itemSize);

    /* --- D3D11 device ---------------------------------------------------- */
    ID3D11Device* d3dDevice = NULL;
    ID3D11DeviceContext* d3dCtx = NULL;
    D3D_FEATURE_LEVEL flOut;
    hr = D3D11CreateDevice(NULL, D3D_DRIVER_TYPE_HARDWARE, NULL,
        D3D11_CREATE_DEVICE_BGRA_SUPPORT, NULL, 0, D3D11_SDK_VERSION,
        &d3dDevice, &flOut, &d3dCtx);
    if (FAILED(hr) || !d3dDevice) {
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -5;
    }

    IDXGIDevice* dxgiDev = NULL;
    ID3D11Device_QueryInterface(d3dDevice, &IID_IDXGIDevice, (void**)&dxgiDev);
    if (!dxgiDev) {
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -5;
    }

    IInspectable* d3dInspect = NULL;
    hr = g_rt.CreateDirect3D11DeviceFromDXGIDevice_(dxgiDev, &d3dInspect);
    IDXGIDevice_Release(dxgiDev);
    if (FAILED(hr) || !d3dInspect) {
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -5;
    }
    __x_ABI_CWindows_CGraphics_CDirectX_CDirect3D11_CIDirect3DDevice* iDevice = NULL;
    d3dInspect->lpVtbl->QueryInterface(d3dInspect,
        &IID___x_ABI_CWindows_CGraphics_CDirectX_CDirect3D11_CIDirect3DDevice,
        (void**)&iDevice);
    d3dInspect->lpVtbl->Release(d3dInspect);
    if (!iDevice) {
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -5;
    }

    /* --- Frame pool + session ------------------------------------------- */
    HSTRING poolName = make_hs(L"Windows.Graphics.Capture.Direct3D11CaptureFramePool");
    __x_ABI_CWindows_CGraphics_CCapture_CIDirect3D11CaptureFramePoolStatics* poolStat = NULL;
    hr = g_rt.RoGetActivationFactory_(poolName,
        &IID___x_ABI_CWindows_CGraphics_CCapture_CIDirect3D11CaptureFramePoolStatics,
        (void**)&poolStat);
    g_rt.WindowsDeleteString_(poolName);
    if (FAILED(hr) || !poolStat) {
        iDevice->lpVtbl->Release(iDevice);
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -6;
    }

    __x_ABI_CWindows_CGraphics_CCapture_CIDirect3D11CaptureFramePool* pool = NULL;
    hr = poolStat->lpVtbl->Create(poolStat, iDevice,
        DirectXPixelFormat_B8G8R8A8UIntNormalized,
        1, itemSize, &pool);
    poolStat->lpVtbl->Release(poolStat);
    if (FAILED(hr) || !pool) {
        iDevice->lpVtbl->Release(iDevice);
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -6;
    }

    __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureSession* session = NULL;
    hr = pool->lpVtbl->CreateCaptureSession(pool, item, &session);
    if (FAILED(hr) || !session) {
        pool->lpVtbl->Release(pool);
        iDevice->lpVtbl->Release(iDevice);
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -6;
    }

    session->lpVtbl->StartCapture(session);

    /* --- Poll for first frame (skip ITypedEventHandler entirely) -------- */
    __x_ABI_CWindows_CGraphics_CCapture_CIDirect3D11CaptureFrame* frame = NULL;
    int waited_ms = 0;
    while (waited_ms < 1500) {
        Sleep(20);
        waited_ms += 20;
        hr = pool->lpVtbl->TryGetNextFrame(pool, &frame);
        if (SUCCEEDED(hr) && frame) break;
        frame = NULL;
    }
    if (!frame) {
        session->lpVtbl->Release(session); pool->lpVtbl->Release(pool);
        iDevice->lpVtbl->Release(iDevice);
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -7;
    }

    /* --- Frame -> Surface -> ID3D11Texture2D ----------------------------- */
    __x_ABI_CWindows_CGraphics_CDirectX_CDirect3D11_CIDirect3DSurface* surface = NULL;
    frame->lpVtbl->get_Surface(frame, &surface);
    if (!surface) {
        frame->lpVtbl->Release(frame);
        session->lpVtbl->Release(session); pool->lpVtbl->Release(pool);
        iDevice->lpVtbl->Release(iDevice);
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -8;
    }

    IDirect3DDxgiInterfaceAccess* access = NULL;
    surface->lpVtbl->QueryInterface(surface,
        &IID_IDirect3DDxgiInterfaceAccess, (void**)&access);
    ID3D11Texture2D* srcTex = NULL;
    if (access) {
        access->lpVtbl->GetInterface(access, &IID_ID3D11Texture2D, (void**)&srcTex);
        access->lpVtbl->Release(access);
    }
    surface->lpVtbl->Release(surface);
    if (!srcTex) {
        frame->lpVtbl->Release(frame);
        session->lpVtbl->Release(session); pool->lpVtbl->Release(pool);
        iDevice->lpVtbl->Release(iDevice);
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -8;
    }

    /* --- Staging texture + Map ------------------------------------------ */
    D3D11_TEXTURE2D_DESC desc;
    ID3D11Texture2D_GetDesc(srcTex, &desc);
    desc.Usage = D3D11_USAGE_STAGING;
    desc.BindFlags = 0;
    desc.CPUAccessFlags = D3D11_CPU_ACCESS_READ;
    desc.MiscFlags = 0;

    ID3D11Texture2D* stage = NULL;
    hr = ID3D11Device_CreateTexture2D(d3dDevice, &desc, NULL, &stage);
    if (FAILED(hr) || !stage) {
        ID3D11Texture2D_Release(srcTex); frame->lpVtbl->Release(frame);
        session->lpVtbl->Release(session); pool->lpVtbl->Release(pool);
        iDevice->lpVtbl->Release(iDevice);
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -8;
    }

    ID3D11DeviceContext_CopyResource(d3dCtx, (ID3D11Resource*)stage, (ID3D11Resource*)srcTex);

    D3D11_MAPPED_SUBRESOURCE mapped;
    hr = ID3D11DeviceContext_Map(d3dCtx, (ID3D11Resource*)stage, 0, D3D11_MAP_READ, 0, &mapped);
    if (FAILED(hr)) {
        ID3D11Texture2D_Release(stage); ID3D11Texture2D_Release(srcTex);
        frame->lpVtbl->Release(frame); session->lpVtbl->Release(session);
        pool->lpVtbl->Release(pool); iDevice->lpVtbl->Release(iDevice);
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -8;
    }

    int w = (int)desc.Width;
    int h = (int)desc.Height;
    int outStride = w * 4;
    uint8_t* buf = (uint8_t*)malloc((size_t)outStride * (size_t)h);
    if (!buf) {
        ID3D11DeviceContext_Unmap(d3dCtx, (ID3D11Resource*)stage, 0);
        ID3D11Texture2D_Release(stage); ID3D11Texture2D_Release(srcTex);
        frame->lpVtbl->Release(frame); session->lpVtbl->Release(session);
        pool->lpVtbl->Release(pool); iDevice->lpVtbl->Release(iDevice);
        ID3D11DeviceContext_Release(d3dCtx); ID3D11Device_Release(d3dDevice);
        __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);
        return -8;
    }

    for (int y = 0; y < h; y++) {
        uint8_t* src = (uint8_t*)mapped.pData + (size_t)y * mapped.RowPitch;
        uint8_t* dst = buf + (size_t)y * outStride;
        memcpy(dst, src, (size_t)outStride);
    }

    ID3D11DeviceContext_Unmap(d3dCtx, (ID3D11Resource*)stage, 0);
    ID3D11Texture2D_Release(stage); ID3D11Texture2D_Release(srcTex);
    frame->lpVtbl->Release(frame);
    session->lpVtbl->Release(session);
    pool->lpVtbl->Release(pool);
    iDevice->lpVtbl->Release(iDevice);
    ID3D11DeviceContext_Release(d3dCtx);
    ID3D11Device_Release(d3dDevice);
    __x_ABI_CWindows_CGraphics_CCapture_CIGraphicsCaptureItem_Release(item);

    out->data = buf;
    out->width = w;
    out->height = h;
    out->stride = outStride;
    return 0;
}

#else /* not _WIN32: provide stubs so the file still compiles on cross-targets */

#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include "windows_wgc.h"

int wgc_available(void) { return 0; }
int wgc_capture_window(uintptr_t hwnd, wgc_image_t* out) {
    (void)hwnd;
    if (out) { out->data = NULL; out->width = out->height = out->stride = 0; }
    return -2;
}
void wgc_free(wgc_image_t* img) {
    if (img && img->data) { free(img->data); img->data = NULL; }
}

#endif /* _WIN32 */
