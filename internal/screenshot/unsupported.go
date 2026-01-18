//go:build s390x || ppc64le || (!(cgo && darwin) && !windows && !linux && !freebsd && !openbsd && !netbsd)

package screenshot

import "image"

// Capture returns screen capture of specified desktop region.
// x and y represent distance from the upper-left corner of primary display.
// Y-axis is downward direction. This means coordinates system is similar to Windows OS.
func Capture(x, y, width, height int, waylandToken uint64) (*image.RGBA, error) {
	_ = waylandToken
	return nil, errUnsupported()
}
