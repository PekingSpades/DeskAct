package keyboard

const (
	DefaultKeySleep     = 10
	DefaultTypeDelay    = 0
	DefaultTypeUTFDelay = 7
)

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
