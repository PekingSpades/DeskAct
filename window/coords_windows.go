//go:build windows
// +build windows

package window

func coordsArePhysical(options WindowOptions) bool {
	return options.DPIAware
}
