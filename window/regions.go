package window

import (
	"math"

	"github.com/PekingSpades/DeskAct/display"
)

func windowRegions(bounds display.Rect, displays []*display.Display, coordsPhysical bool) []DisplayRegion {
	regions := make([]DisplayRegion, 0, len(displays))
	for _, d := range displays {
		intersect, ok := intersectRect(bounds, d.Origin())
		if !ok {
			continue
		}
		phys := physicalRect(intersect, d, coordsPhysical)
		if phys.W <= 0 || phys.H <= 0 {
			continue
		}
		regions = append(regions, DisplayRegion{
			DisplayIndex: d.Index(),
			DisplayID:    d.ID(),
			PhysicalRect: phys,
		})
	}
	return regions
}

func intersectRect(a, b display.Rect) (display.Rect, bool) {
	x0 := maxInt(a.X, b.X)
	y0 := maxInt(a.Y, b.Y)
	x1 := minInt(a.X+a.W, b.X+b.W)
	y1 := minInt(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return display.Rect{}, false
	}
	return makeRect(x0, y0, x1-x0, y1-y0), true
}

func physicalRect(virt display.Rect, d *display.Display, coordsPhysical bool) display.Rect {
	origin := d.Origin()
	relX := virt.X - origin.X
	relY := virt.Y - origin.Y
	relW := virt.W
	relH := virt.H

	scale := d.Scale()
	if scale <= 0 {
		scale = 1
	}

	physX, physY, physW, physH := relX, relY, relW, relH
	if !coordsPhysical {
		physX0 := scaleRound(relX, scale)
		physY0 := scaleRound(relY, scale)
		physX1 := scaleRound(relX+relW, scale)
		physY1 := scaleRound(relY+relH, scale)
		physX = physX0
		physY = physY0
		physW = physX1 - physX0
		physH = physY1 - physY0
	}

	size := d.Size()
	if physX < 0 {
		physW += physX
		physX = 0
	}
	if physY < 0 {
		physH += physY
		physY = 0
	}
	if physW <= 0 || physH <= 0 {
		return display.Rect{}
	}
	if physX+physW > size.W {
		physW = size.W - physX
	}
	if physY+physH > size.H {
		physH = size.H - physY
	}
	if physW <= 0 || physH <= 0 {
		return display.Rect{}
	}
	return makeRect(physX, physY, physW, physH)
}

func scaleRound(value int, scale float64) int {
	if scale == 1 {
		return value
	}
	return int(math.Round(float64(value) * scale))
}

func makeRect(x, y, w, h int) display.Rect {
	return display.Rect{
		Point: display.Point{X: x, Y: y},
		Size:  display.Size{W: w, H: h},
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
