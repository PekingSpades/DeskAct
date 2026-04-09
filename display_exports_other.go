//go:build !windows

package deskact

// IsDPIAware reports whether the process is DPI aware.
// DPI awareness is a Windows-specific concept, so this returns false on other platforms.
func IsDPIAware() bool {
	return false
}

// InitDPIAwareness is a no-op on non-Windows platforms and returns false.
func InitDPIAwareness() bool {
	return false
}
