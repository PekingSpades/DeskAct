package deskact

import "time"

// MilliSleep sleep tm milli second.
func MilliSleep(tm int) {
	time.Sleep(time.Duration(tm) * time.Millisecond)
}

// Sleep time.Sleep tm second.
func Sleep(tm int) {
	time.Sleep(time.Duration(tm) * time.Second)
}
