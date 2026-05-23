//go:build cgo && darwin
// +build cgo,darwin

package mouse

/*
#include <ApplicationServices/ApplicationServices.h>
#cgo darwin LDFLAGS: -framework ApplicationServices
static void mouse_warp(int x, int y) {
    CGPoint pt = CGPointMake((CGFloat)x, (CGFloat)y);
    CGWarpMouseCursorPosition(pt);
    CGAssociateMouseAndMouseCursorPosition(1);
}
*/
import "C"

func warpSystemCursor(x, y int) {
	C.mouse_warp(C.int(x), C.int(y))
}
