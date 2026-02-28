//go:build windows

package deskact

import "github.com/PekingSpades/DeskAct/display"

type WindowsPlatformInfo = display.WindowsPlatformInfo

func IsDPIAware() bool {
	return display.IsDPIAware()
}

func InitDPIAwareness() bool {
	return display.InitDPIAwareness()
}
