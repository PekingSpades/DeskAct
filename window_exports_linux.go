//go:build linux
// +build linux

package deskact

import "github.com/PekingSpades/DeskAct/window"

// LinuxPlatformInfo aliases window.LinuxPlatformInfo so callers using only
// the top-level deskact package can type-assert WindowInfo.GetPlatformInfo()
// to *LinuxPlatformInfo without importing the sub-package.
//
// (WindowsPlatformInfo / DarwinPlatformInfo are already exported by
// display_exports_windows.go / display_exports_other.go for the display
// package's per-OS metadata; the window package's same-named types are
// reachable through window.WindowsPlatformInfo / window.DarwinPlatformInfo.)
type LinuxPlatformInfo = window.LinuxPlatformInfo
