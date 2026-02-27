//go:build linux
// +build linux

package display

/*
#cgo linux CFLAGS: -I/usr/src
#cgo linux LDFLAGS: -L/usr/src -lm -lX11 -lXtst -lXinerama -lXrandr

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
	physW, physH := int(info.w), int(info.h)
	return &Display{
		id:         int(info.handle),
		electronId: int64(info.electronId),
		index:      int(info.index),
		isMain:     info.isMain != 0,
		origin:     Rect{Point: Point{X: int(info.x), Y: int(info.y)}, Size: Size{W: physW, H: physH}},
		size:       Size{W: physW, H: physH},
		scale:      float64(info.scale),
		dpiAware:   options.DPIAware,
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
		physW, physH := int(info.w), int(info.h)
		displays[i] = &Display{
			id:         int(info.handle),
			electronId: int64(info.electronId),
			index:      int(info.index),
			isMain:     info.isMain != 0,
			origin:     Rect{Point: Point{X: int(info.x), Y: int(info.y)}, Size: Size{W: physW, H: physH}},
			size:       Size{W: physW, H: physH},
			scale:      float64(info.scale),
			dpiAware:   options.DPIAware,
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

	physW, physH := int(info.w), int(info.h)
	return &Display{
		id:         int(info.handle),
		electronId: int64(info.electronId),
		index:      int(info.index),
		isMain:     info.isMain != 0,
		origin:     Rect{Point: Point{X: int(info.x), Y: int(info.y)}, Size: Size{W: physW, H: physH}},
		size:       Size{W: physW, H: physH},
		scale:      float64(info.scale),
		dpiAware:   options.DPIAware,
	}
}

// DisplayCount returns the number of displays.
func DisplayCount() int {
	return int(C.getDisplayCount())
}

// ToAbsolute converts coordinates relative to this display to absolute coordinates.
func (d *Display) ToAbsolute(x, y int) (absX, absY int) {
	return d.origin.X + x, d.origin.Y + y
}

// ToRelative converts absolute coordinates to coordinates relative to this display.
func (d *Display) ToRelative(absX, absY int) (x, y int, ok bool) {
	if !d.Contains(absX, absY) {
		return 0, 0, false
	}
	return absX - d.origin.X, absY - d.origin.Y, true
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
func (d *Display) CaptureRect(x, y, w, h int, options CaptureOptions) (*image.RGBA, error) {
	absX, absY := d.ToAbsolute(x, y)
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
