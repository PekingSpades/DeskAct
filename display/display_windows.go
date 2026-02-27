//go:build windows
// +build windows

package display

/*
#cgo windows LDFLAGS: -lgdi32 -luser32

#include "display_c.h"
*/
import "C"

import (
	"errors"
	"image"

	"github.com/PekingSpades/DeskAct/mouse"
	"github.com/PekingSpades/DeskAct/screenshot"
)

// WindowsPlatformInfo contains Windows-specific display information.
type WindowsPlatformInfo struct {
	// PhysicalOrigin is the top-left corner in physical pixel coordinates.
	PhysicalOrigin Point
}

// Platform returns the platform name.
func (w *WindowsPlatformInfo) Platform() string {
	return "windows"
}

// MainDisplay returns the main display.
func MainDisplay(options DisplayOptions) *Display {
	info := C.getMainDisplay()
	dpiAware := options.DPIAware
	scale := float64(info.scale)
	physW, physH := int(info.w), int(info.h)

	// Calculate logical size.
	logicalW, logicalH := int(info.vw), int(info.vh)
	if logicalW <= 0 || logicalH <= 0 {
		logicalW, logicalH = physW, physH
		if scale > 0 {
			logicalW = int(float64(physW) / scale)
			logicalH = int(float64(physH) / scale)
		}
	}

	// Always store physical origin for capture operations.
	physicalOrigin := Point{X: int(info.x), Y: int(info.y)}

	var origin Rect
	if dpiAware {
		// DPI aware: use physical coordinates and size.
		origin = Rect{
			Point: Point{X: int(info.x), Y: int(info.y)},
			Size:  Size{W: physW, H: physH},
		}
	} else {
		// DPI unaware: use virtual coordinates and logical size.
		origin = Rect{
			Point: Point{X: int(info.vx), Y: int(info.vy)},
			Size:  Size{W: logicalW, H: logicalH},
		}
	}

	return &Display{
		id:         int(info.handle),
		electronId: int64(info.electronId),
		index:      int(info.index),
		isMain:     info.isMain != 0,
		origin:     origin,
		size:       Size{W: physW, H: physH},
		scale:      scale,
		dpiAware:   dpiAware,
		platform: &WindowsPlatformInfo{
			PhysicalOrigin: physicalOrigin,
		},
	}
}

// AllDisplays returns all displays.
func AllDisplays(options DisplayOptions) []*Display {
	count := int(C.getDisplayCount())
	if count <= 0 {
		return nil
	}

	var cDisplays [32]C.DisplayInfoC
	actualCount := int(C.getAllDisplays(&cDisplays[0], C.int32_t(32)))

	displays := make([]*Display, actualCount)
	for i := 0; i < actualCount; i++ {
		info := cDisplays[i]
		dpiAware := options.DPIAware
		scale := float64(info.scale)
		physW, physH := int(info.w), int(info.h)

		logicalW, logicalH := int(info.vw), int(info.vh)
		if logicalW <= 0 || logicalH <= 0 {
			logicalW, logicalH = physW, physH
			if scale > 0 {
				logicalW = int(float64(physW) / scale)
				logicalH = int(float64(physH) / scale)
			}
		}

		// Always store physical origin for capture operations.
		physicalOrigin := Point{X: int(info.x), Y: int(info.y)}

		var origin Rect
		if dpiAware {
			origin = Rect{
				Point: Point{X: int(info.x), Y: int(info.y)},
				Size:  Size{W: physW, H: physH},
			}
		} else {
			origin = Rect{
				Point: Point{X: int(info.vx), Y: int(info.vy)},
				Size:  Size{W: logicalW, H: logicalH},
			}
		}

		displays[i] = &Display{
			id:         int(info.handle),
			electronId: int64(info.electronId),
			index:      int(info.index),
			isMain:     info.isMain != 0,
			origin:     origin,
			size:       Size{W: physW, H: physH},
			scale:      scale,
			dpiAware:   dpiAware,
			platform: &WindowsPlatformInfo{
				PhysicalOrigin: physicalOrigin,
			},
		}
	}

	return displays
}

// DisplayAt returns the display at the specified index.
// Returns nil if the index is invalid.
func DisplayAt(index int, options DisplayOptions) *Display {
	if index < 0 {
		return nil
	}

	info := C.getDisplayAt(C.int32_t(index))
	if info.w == 0 && info.h == 0 {
		return nil
	}

	dpiAware := options.DPIAware
	scale := float64(info.scale)
	physW, physH := int(info.w), int(info.h)

	logicalW, logicalH := int(info.vw), int(info.vh)
	if logicalW <= 0 || logicalH <= 0 {
		logicalW, logicalH = physW, physH
		if scale > 0 {
			logicalW = int(float64(physW) / scale)
			logicalH = int(float64(physH) / scale)
		}
	}

	// Always store physical origin for capture operations.
	physicalOrigin := Point{X: int(info.x), Y: int(info.y)}

	var origin Rect
	if dpiAware {
		origin = Rect{
			Point: Point{X: int(info.x), Y: int(info.y)},
			Size:  Size{W: physW, H: physH},
		}
	} else {
		origin = Rect{
			Point: Point{X: int(info.vx), Y: int(info.vy)},
			Size:  Size{W: logicalW, H: logicalH},
		}
	}

	return &Display{
		id:         int(info.handle),
		electronId: int64(info.electronId),
		index:      int(info.index),
		isMain:     info.isMain != 0,
		origin:     origin,
		size:       Size{W: physW, H: physH},
		scale:      scale,
		dpiAware:   dpiAware,
		platform: &WindowsPlatformInfo{
			PhysicalOrigin: physicalOrigin,
		},
	}
}

// DisplayCount returns the number of displays.
func DisplayCount() int {
	return int(C.getDisplayCount())
}

// ToAbsolute converts physical pixel coordinates relative to this display
// to absolute coordinates suitable for mouse APIs.
func (d *Display) ToAbsolute(physX, physY int) (absX, absY int) {
	if d.dpiAware {
		// DPI aware: origin is physical, coordinates are physical.
		return d.origin.X + physX, d.origin.Y + physY
	}
	// DPI unaware: origin is virtual, convert physical to virtual.
	if d.scale > 0 {
		absX = d.origin.X + int(float64(physX)/d.scale)
		absY = d.origin.Y + int(float64(physY)/d.scale)
	} else {
		absX = d.origin.X + physX
		absY = d.origin.Y + physY
	}
	return
}

// ToRelative converts absolute coordinates from mouse APIs to physical pixel
// coordinates relative to this display.
func (d *Display) ToRelative(absX, absY int) (physX, physY int, ok bool) {
	if !d.Contains(absX, absY) {
		return 0, 0, false
	}
	if d.dpiAware {
		// DPI aware: coordinates are physical.
		return absX - d.origin.X, absY - d.origin.Y, true
	}
	// DPI unaware: coordinates are virtual, convert to physical.
	virtRelX := absX - d.origin.X
	virtRelY := absY - d.origin.Y
	if d.scale > 0 {
		physX = int(float64(virtRelX) * d.scale)
		physY = int(float64(virtRelY) * d.scale)
	} else {
		physX = virtRelX
		physY = virtRelY
	}
	return physX, physY, true
}

// Contains checks if the specified absolute coordinate is within this display.
func (d *Display) Contains(absX, absY int) bool {
	return absX >= d.origin.X && absX < d.origin.X+d.origin.W &&
		absY >= d.origin.Y && absY < d.origin.Y+d.origin.H
}

// Move moves the mouse to the specified coordinates relative to this display.
func (d *Display) Move(x, y int, settings MouseSettings) error {
	absX, absY := d.ToAbsolute(x, y)
	return mouse.Move(absX, absY, settings)
}

// MoveSmooth smoothly moves the mouse to the specified coordinates relative to this display.
func (d *Display) MoveSmooth(x, y int, settings MouseSettings) error {
	absX, absY := d.ToAbsolute(x, y)
	return mouse.MoveSmooth(absX, absY, settings)
}

// Drag drags the mouse from one position to another on this display.
func (d *Display) Drag(fromX, fromY, toX, toY int, button MouseButton, settings MouseSettings) error {
	if err := d.Move(fromX, fromY, settings); err != nil {
		return err
	}
	if err := mouse.Toggle(button, true, false, settings); err != nil {
		return err
	}
	mouse.MilliSleep(50)
	if err := d.MoveSmooth(toX, toY, settings); err != nil {
		_ = mouse.Toggle(button, false, false, settings)
		return err
	}
	return mouse.Toggle(button, false, false, settings)
}

// DragTo drags the mouse from the current position to the specified position on this display.
func (d *Display) DragTo(x, y int, button MouseButton, settings MouseSettings) error {
	if err := mouse.Toggle(button, true, false, settings); err != nil {
		return err
	}
	mouse.MilliSleep(50)
	if err := d.MoveSmooth(x, y, settings); err != nil {
		_ = mouse.Toggle(button, false, false, settings)
		return err
	}
	return mouse.Toggle(button, false, false, settings)
}

// CaptureRect captures a rectangular region of this display.
func (d *Display) CaptureRect(physX, physY, w, h int, options CaptureOptions) (*image.RGBA, error) {
	pi, ok := d.platform.(*WindowsPlatformInfo)
	if !ok {
		return nil, errors.New("invalid platform info")
	}
	absX := pi.PhysicalOrigin.X + physX
	absY := pi.PhysicalOrigin.Y + physY
	return screenshot.Capture(absX, absY, w, h, options.WaylandToken)
}

// MouseLocation gets the mouse location relative to this display.
func (d *Display) MouseLocation() (x, y int, ok bool) {
	absX, absY := mouse.Location()
	return d.ToRelative(absX, absY)
}

// ContainsMouse checks if the mouse is on this display.
func (d *Display) ContainsMouse() bool {
	absX, absY := mouse.Location()
	return d.Contains(absX, absY)
}
