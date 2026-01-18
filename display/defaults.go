package display

const (
	DefaultDPIAware = false
)

// DisplayOptions defines platform-specific display options.
type DisplayOptions struct {
	DPIAware bool
}

// CaptureOptions defines platform-specific capture options.
type CaptureOptions struct {
	WaylandToken uint64
}

// DefaultCaptureOptions returns the default capture options.
func DefaultCaptureOptions() CaptureOptions {
	return CaptureOptions{}
}
