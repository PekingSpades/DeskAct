//go:build windows

package deskact

// DefaultDisplayOptions returns the default display options.
func DefaultDisplayOptions() DisplayOptions {
	return DisplayOptions{DPIAware: IsDPIAware()}
}
