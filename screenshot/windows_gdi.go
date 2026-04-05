//go:build windows

package screenshot

import (
	"errors"
	"image"
	"syscall"
	"unsafe"

	cap "github.com/PekingSpades/DeskAct/capture"
	"github.com/lxn/win"
)

func captureGDI(req cap.Request) (*image.RGBA, error) {
	rect := image.Rect(0, 0, req.Width, req.Height)
	img, err := createImage(rect)
	if err != nil {
		return nil, err
	}

	hwnd := getDesktopWindow()
	hdc := win.GetDC(hwnd)
	if hdc == 0 {
		return nil, errors.New("GetDC failed")
	}
	defer win.ReleaseDC(hwnd, hdc)

	memoryDevice := win.CreateCompatibleDC(hdc)
	if memoryDevice == 0 {
		return nil, errors.New("CreateCompatibleDC failed")
	}
	defer win.DeleteDC(memoryDevice)

	bitmap := win.CreateCompatibleBitmap(hdc, int32(req.Width), int32(req.Height))
	if bitmap == 0 {
		return nil, errors.New("CreateCompatibleBitmap failed")
	}
	defer win.DeleteObject(win.HGDIOBJ(bitmap))

	var header win.BITMAPINFOHEADER
	header.BiSize = uint32(unsafe.Sizeof(header))
	header.BiPlanes = 1
	header.BiBitCount = 32
	header.BiWidth = int32(req.Width)
	header.BiHeight = int32(-req.Height)
	header.BiCompression = win.BI_RGB
	header.BiSizeImage = 0

	// GetDIBits balks at using Go memory on some systems. The MSDN example uses
	// GlobalAlloc, so we'll do that too. See:
	// https://docs.microsoft.com/en-gb/windows/desktop/gdi/capturing-an-image
	bitmapDataSize := uintptr(((int64(req.Width)*int64(header.BiBitCount) + 31) / 32) * 4 * int64(req.Height))
	hmem := win.GlobalAlloc(win.GMEM_MOVEABLE, bitmapDataSize)
	defer win.GlobalFree(hmem)
	memptr := win.GlobalLock(hmem)
	defer win.GlobalUnlock(hmem)

	old := win.SelectObject(memoryDevice, win.HGDIOBJ(bitmap))
	if old == 0 {
		return nil, errors.New("SelectObject failed")
	}
	defer win.SelectObject(memoryDevice, old)

	if !win.BitBlt(memoryDevice, 0, 0, int32(req.Width), int32(req.Height), hdc, int32(req.X), int32(req.Y), win.SRCCOPY) {
		return nil, errors.New("BitBlt failed")
	}

	if win.GetDIBits(hdc, bitmap, 0, uint32(req.Height), (*uint8)(memptr), (*win.BITMAPINFO)(unsafe.Pointer(&header)), win.DIB_RGB_COLORS) == 0 {
		return nil, errors.New("GetDIBits failed")
	}

	i := 0
	src := uintptr(memptr)
	for y := 0; y < req.Height; y++ {
		for x := 0; x < req.Width; x++ {
			v0 := *(*uint8)(unsafe.Pointer(src))
			v1 := *(*uint8)(unsafe.Pointer(src + 1))
			v2 := *(*uint8)(unsafe.Pointer(src + 2))

			// BGRA => RGBA, and set A to 255
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v2, v1, v0, 255

			i += 4
			src += 4
		}
	}

	return img, nil
}

func getDesktopWindow() win.HWND {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("GetDesktopWindow")
	ret, _, _ := proc.Call()
	return win.HWND(ret)
}
