//go:build windows

package display

// DefaultDisplayOptions returns the default display options.
func DefaultDisplayOptions() DisplayOptions {
	return DisplayOptions{DPIAware: IsDPIAware()}
}
