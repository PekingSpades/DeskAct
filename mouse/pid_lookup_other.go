//go:build !windows && !linux && !darwin

package mouse

// hwndByPIDPlatform / xidByPIDPlatform / frontWindowIDByPIDPlatform:
// stubs for niche/non-supported platforms.
func hwndByPIDPlatform(pid int) uint64            { return 0 }
func xidByPIDPlatform(pid int) uint64             { return 0 }
func frontWindowIDByPIDPlatform(pid int) uint64   { return 0 }
