//go:build !linux
// +build !linux

package keyboard

func lookupXIDByPID(pid int) uint32 { return 0 }
