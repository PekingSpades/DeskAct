//go:build !windows && !linux

package mouse

// hwndByPIDPlatform / xidByPIDPlatform: macOS and other platforms route
// PID-based mouse calls through the macOS / fallback paths where the
// PID itself is the relevant identifier, so these are stubs.
func hwndByPIDPlatform(pid int) uint64 { return 0 }
func xidByPIDPlatform(pid int) uint64  { return 0 }
