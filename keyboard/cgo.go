package keyboard

/*
#cgo darwin CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo darwin LDFLAGS: -framework Cocoa -framework CoreFoundation -framework IOKit
#cgo darwin LDFLAGS: -framework Carbon -framework OpenGL
#cgo darwin LDFLAGS: -weak_framework ScreenCaptureKit

#cgo linux CFLAGS: -I/usr/src
#cgo linux LDFLAGS: -L/usr/src -lm -lX11 -lXtst

#cgo windows LDFLAGS: -lgdi32 -luser32
*/
import "C"
