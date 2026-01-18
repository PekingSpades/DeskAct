package deskact

const (
	DefaultMouseSleep           = 0
	DefaultKeySleep             = 10
	DefaultTypeDelay            = 0
	DefaultTypeUTFDelay         = 7
	DefaultMoveSmoothLow        = 1.0
	DefaultMoveSmoothHigh       = 3.0
	DefaultMoveSmoothDelay      = 1
	DefaultScrollDelay          = 10
	DefaultScrollSmoothCount    = 5
	DefaultScrollSmoothInterval = 100
	DefaultScrollSmoothX        = 0
	DefaultDPIAware             = false
)

// MouseSettings defines timing and smoothing parameters for mouse actions.
// Use DefaultMouseSettings() to start from the built-in defaults.
type MouseSettings struct {
	Sleep                int
	MoveSmoothLow        float64
	MoveSmoothHigh       float64
	MoveSmoothDelay      int
	ScrollDelay          int
	ScrollSmoothCount    int
	ScrollSmoothInterval int
	ScrollSmoothX        int
}

// DefaultMouseSettings returns the default mouse settings.
func DefaultMouseSettings() MouseSettings {
	return MouseSettings{
		Sleep:                DefaultMouseSleep,
		MoveSmoothLow:        DefaultMoveSmoothLow,
		MoveSmoothHigh:       DefaultMoveSmoothHigh,
		MoveSmoothDelay:      DefaultMoveSmoothDelay,
		ScrollDelay:          DefaultScrollDelay,
		ScrollSmoothCount:    DefaultScrollSmoothCount,
		ScrollSmoothInterval: DefaultScrollSmoothInterval,
		ScrollSmoothX:        DefaultScrollSmoothX,
	}
}

// KeyboardSettings defines timing parameters for keyboard actions.
// Use DefaultKeyboardSettings() to start from the built-in defaults.
type KeyboardSettings struct {
	Sleep        int
	TypeDelay    int
	TypeUTFDelay int
}

// DefaultKeyboardSettings returns the default keyboard settings.
func DefaultKeyboardSettings() KeyboardSettings {
	return KeyboardSettings{
		Sleep:        DefaultKeySleep,
		TypeDelay:    DefaultTypeDelay,
		TypeUTFDelay: DefaultTypeUTFDelay,
	}
}

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
