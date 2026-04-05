package mouse

const (
	DefaultMouseSleep           = 0
	DefaultMoveSmoothLow        = 1.0
	DefaultMoveSmoothHigh       = 3.0
	DefaultMoveSmoothDelay      = 1
	DefaultScrollDelay          = 10
	DefaultScrollSmoothCount    = 5
	DefaultScrollSmoothInterval = 100
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
	}
}
