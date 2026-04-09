//go:build !windows
// +build !windows

package window

import "github.com/PekingSpades/DeskAct/display"

func attachDisplayRegions(windows []WindowInfo, displays []*display.Display, options WindowOptions) {
	coordsPhysical := coordsArePhysical(options)
	for i := range windows {
		windows[i].DisplayRegions = windowRegions(windows[i].Bounds, displays, coordsPhysical)
	}
}
