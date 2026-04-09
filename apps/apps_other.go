//go:build !windows && !darwin

package apps

func DesktopApps() ([]AppInfo, error) {
	return nil, errUnsupported
}

func InstalledApps() ([]AppInfo, error) {
	return nil, errUnsupported
}
