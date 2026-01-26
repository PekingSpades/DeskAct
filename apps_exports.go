package deskact

import "github.com/PekingSpades/DeskAct/apps"

type AppInfo = apps.AppInfo

var ErrIconNotFound = apps.ErrIconNotFound

func DesktopApps() ([]AppInfo, error) {
	return apps.DesktopApps()
}

func InstalledApps() ([]AppInfo, error) {
	return apps.InstalledApps()
}
