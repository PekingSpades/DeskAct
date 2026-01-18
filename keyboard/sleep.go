package keyboard

import "time"

func milliSleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
