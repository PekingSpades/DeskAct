package apps

import (
	"errors"
	"image"
	"image/draw"
	"sort"
	"strings"
)

// AppInfo describes an application and its icon.
type AppInfo struct {
	Name string
	Path string
	Icon *image.RGBA
}

var errUnsupported = errors.New("apps does not support your platform")

// ErrIconNotFound is returned when an app icon cannot be resolved.
var ErrIconNotFound = errors.New("icon source not found")

func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func toRGBA(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

func sortApps(apps []AppInfo) {
	sort.SliceStable(apps, func(i, j int) bool {
		ai := strings.ToLower(apps[i].Name)
		aj := strings.ToLower(apps[j].Name)
		if ai == aj {
			return strings.ToLower(apps[i].Path) < strings.ToLower(apps[j].Path)
		}
		return ai < aj
	})
}
