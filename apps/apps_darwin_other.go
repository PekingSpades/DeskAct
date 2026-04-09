//go:build darwin && !cgo

package apps

func DesktopApps() ([]AppInfo, error) {
	return nil, errUnsupported
}

func InstalledApps() ([]AppInfo, error) {
	return nil, errUnsupported
}
