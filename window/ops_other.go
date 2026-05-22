//go:build !windows && !darwin && !linux
// +build !windows,!darwin,!linux

package window

import "github.com/PekingSpades/DeskAct/display"

func moveWindow(id uint64, pid int32, x, y int) error          { return errOpUnsupported() }
func resizeWindow(id uint64, pid int32, w, h int) error        { return errOpUnsupported() }
func moveResizeWindow(id uint64, pid int32, x, y, w, h int) error {
	return errOpUnsupported()
}
func raiseWindow(id uint64, pid int32) error                    { return errOpUnsupported() }
func focusWindow(id uint64, pid int32) error                    { return errOpUnsupported() }
func minimizeWindow(id uint64, pid int32) error                 { return errOpUnsupported() }
func restoreWindow(id uint64, pid int32) error                  { return errOpUnsupported() }
func closeWindow(id uint64, pid int32) error                    { return errOpUnsupported() }
func boundsWindow(id uint64, pid int32) (display.Rect, error)   { return display.Rect{}, errOpUnsupported() }
