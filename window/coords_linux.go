//go:build linux
// +build linux

package window

func coordsArePhysical(options WindowOptions) bool {
	return false
}
