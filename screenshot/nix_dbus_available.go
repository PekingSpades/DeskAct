//go:build !s390x && !ppc64le && !darwin && !windows && (linux || openbsd || netbsd)

package screenshot

import (
	"image"
	"os"
)

// Capture returns screen capture of specified desktop region.
// x and y represent distance from the upper-left corner of primary display.
// Y-axis is downward direction. This means coordinates system is similar to Windows OS.
func Capture(x, y, width, height int, waylandToken uint64) (img *image.RGBA, e error) {
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType == "wayland" {
		return captureDbus(x, y, width, height, waylandToken)
	} else {
		return captureXinerama(x, y, width, height)
	}
}
