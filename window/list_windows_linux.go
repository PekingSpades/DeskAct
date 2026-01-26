//go:build linux
// +build linux

package window

func listWindows(options WindowOptions) ([]WindowInfo, error) {
	return nil, errUnsupported()
}
