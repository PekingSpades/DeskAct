//go:build darwin
// +build darwin

package window

func coordsArePhysical(options WindowOptions) bool {
	return false
}
