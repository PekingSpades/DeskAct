package keyboard

import "os"

func isWaylandSession() bool {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	return os.Getenv("XDG_SESSION_TYPE") == "wayland"
}
