//go:build darwin
// +build darwin

package display

/*
#cgo darwin CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo darwin LDFLAGS: -framework Cocoa -framework CoreFoundation -framework IOKit
#cgo darwin LDFLAGS: -framework Carbon -framework OpenGL
#cgo darwin LDFLAGS: -weak_framework ScreenCaptureKit

#include "display_c.h"
*/
import "C"

import (
	"image"

	"github.com/PekingSpades/DeskAct/mouse"
	"github.com/PekingSpades/DeskAct/screenshot"
)

// MainDisplay returns the main display.
func MainDisplay(options DisplayOptions) *Display {
	info := C.getMainDisplay()
	scale := float64(info.scale)
	physW, physH := int(info.w), int(info.h)

	// Calculate logical size for origin.
	logicalW, logicalH := physW, physH
	if scale > 0 {
		logicalW = int(float64(physW) / scale)
		logicalH = int(float64(physH) / scale)
	}

	return &Display{
		id:       int(info.handle),
		index:    int(info.index),
		isMain:   info.isMain != 0,
		origin:   Rect{Point: Point{X: int(info.x), Y: int(info.y)}, Size: Size{W: logicalW, H: logicalH}},
		size:     Size{W: physW, H: physH},
		scale:    scale,
		dpiAware: options.DPIAware,
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
		scale := float64(info.scale)
		physW, physH := int(info.w), int(info.h)

		logicalW, logicalH := physW, physH
		if scale > 0 {
			logicalW = int(float64(physW) / scale)
			logicalH = int(float64(physH) / scale)
		}

		displays[i] = &Display{
			id:       int(info.handle),
			index:    int(info.index),
			isMain:   info.isMain != 0,
			origin:   Rect{Point: Point{X: int(info.x), Y: int(info.y)}, Size: Size{W: logicalW, H: logicalH}},
			size:     Size{W: physW, H: physH},
			scale:    scale,
			dpiAware: options.DPIAware,
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

	scale := float64(info.scale)
	physW, physH := int(info.w), int(info.h)

	logicalW, logicalH := physW, physH
	if scale > 0 {
		logicalW = int(float64(physW) / scale)
		logicalH = int(float64(physH) / scale)
	}

	return &Display{
		id:       int(info.handle),
		index:    int(info.index),
		isMain:   info.isMain != 0,
		origin:   Rect{Point: Point{X: int(info.x), Y: int(info.y)}, Size: Size{W: logicalW, H: logicalH}},
		size:     Size{W: physW, H: physH},
		scale:    scale,
		dpiAware: options.DPIAware,
	}
}

// DisplayCount returns the number of displays.
func DisplayCount() int {
	return int(C.getDisplayCount())
}

// ToAbsolute converts physical pixel coordinates relative to this display
// to virtual absolute coordinates (for use with macOS APIs).
func (d *Display) ToAbsolute(physX, physY int) (virtAbsX, virtAbsY int) {
	if d.scale > 0 {
		// Convert physical to virtual relative, then add virtual origin.
		virtAbsX = d.origin.X + int(float64(physX)/d.scale)
		virtAbsY = d.origin.Y + int(float64(physY)/d.scale)
	} else {
		virtAbsX = d.origin.X + physX
		virtAbsY = d.origin.Y + physY
	}
	return
}

// ToRelative converts virtual absolute coordinates to physical pixel coordinates
// relative to this display.
func (d *Display) ToRelative(virtAbsX, virtAbsY int) (physX, physY int, ok bool) {
	if !d.Contains(virtAbsX, virtAbsY) {
		return 0, 0, false
	}
	// Calculate virtual relative coordinates.
	virtRelX := virtAbsX - d.origin.X
	virtRelY := virtAbsY - d.origin.Y
	// Convert to physical coordinates.
	if d.scale > 0 {
		physX = int(float64(virtRelX) * d.scale)
		physY = int(float64(virtRelY) * d.scale)
	} else {
		physX = virtRelX
		physY = virtRelY
	}
	return physX, physY, true
}

// Contains checks if the specified virtual absolute coordinate is within this display.
func (d *Display) Contains(virtAbsX, virtAbsY int) bool {
	return virtAbsX >= d.origin.X && virtAbsX < d.origin.X+d.origin.W &&
		virtAbsY >= d.origin.Y && virtAbsY < d.origin.Y+d.origin.H
}

// Move moves the mouse to the specified physical pixel coordinates relative to this display.
func (d *Display) Move(physX, physY int, settings MouseSettings) error {
	virtAbsX, virtAbsY := d.ToAbsolute(physX, physY)
	return mouse.Move(virtAbsX, virtAbsY, settings)
}

// MoveSmooth smoothly moves the mouse to the specified physical pixel coordinates
// relative to this display.
func (d *Display) MoveSmooth(physX, physY int, settings MouseSettings) error {
	virtAbsX, virtAbsY := d.ToAbsolute(physX, physY)
	return mouse.MoveSmooth(virtAbsX, virtAbsY, settings)
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
func (d *Display) DragTo(physX, physY int, button MouseButton, settings MouseSettings) error {
	if err := mouse.Toggle(button, true, false, settings); err != nil {
		return err
	}
	mouse.MilliSleep(50)
	if err := d.MoveSmooth(physX, physY, settings); err != nil {
		_ = mouse.Toggle(button, false, false, settings)
		return err
	}
	return mouse.Toggle(button, false, false, settings)
}

// CaptureRect captures a rectangular region of this display.
func (d *Display) CaptureRect(physX, physY, w, h int, options CaptureOptions) (*image.RGBA, error) {
	virtAbsX, virtAbsY := d.ToAbsolute(physX, physY)
	// Convert size from physical to virtual for the capture API.
	virtW := w
	virtH := h
	if d.scale > 0 {
		virtW = int(float64(w) / d.scale)
		virtH = int(float64(h) / d.scale)
	}
	return screenshot.Capture(virtAbsX, virtAbsY, virtW, virtH, options.WaylandToken)
}

// MouseLocation gets the mouse location in physical pixels relative to this display.
func (d *Display) MouseLocation() (physX, physY int, ok bool) {
	virtAbsX, virtAbsY := mouse.Location()
	return d.ToRelative(virtAbsX, virtAbsY)
}

// ContainsMouse checks if the mouse is on this display.
func (d *Display) ContainsMouse() bool {
	virtAbsX, virtAbsY := mouse.Location()
	return d.Contains(virtAbsX, virtAbsY)
}
