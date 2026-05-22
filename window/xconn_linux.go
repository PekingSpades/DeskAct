//go:build linux
// +build linux

package window

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// errWaylandSession is returned when this process is running under a Wayland
// session. Per-window listing and per-window injection both require X11.
var errWaylandSession = fmt.Errorf("%w: wayland session detected", errUnsupported())

func isWaylandSession() bool {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return os.Getenv("DISPLAY") == ""
	}
	if t := os.Getenv("XDG_SESSION_TYPE"); t == "wayland" {
		return true
	}
	return false
}

type xConnState struct {
	conn    *xgb.Conn
	root    xproto.Window
	atoms   map[string]xproto.Atom
	atomsMu sync.Mutex
	err     error
}

var (
	xConnOnce sync.Once
	xConn     *xConnState
)

// linuxXConn returns a shared X server connection or an error if X11 is not
// available (Wayland-only sessions, no DISPLAY, etc.). The connection is
// opened on first use and reused for both window listing and per-window ops.
func linuxXConn() (*xConnState, error) {
	xConnOnce.Do(func() {
		if isWaylandSession() {
			xConn = &xConnState{err: errWaylandSession}
			return
		}
		if os.Getenv("DISPLAY") == "" {
			xConn = &xConnState{err: fmt.Errorf("%w: DISPLAY is unset", errUnsupported())}
			return
		}
		c, err := xgb.NewConn()
		if err != nil {
			xConn = &xConnState{err: fmt.Errorf("%w: %v", errUnsupported(), err)}
			return
		}
		setup := xproto.Setup(c)
		if setup == nil || len(setup.Roots) == 0 {
			c.Close()
			xConn = &xConnState{err: fmt.Errorf("%w: no X11 roots", errUnsupported())}
			return
		}
		xConn = &xConnState{
			conn:  c,
			root:  setup.DefaultScreen(c).Root,
			atoms: make(map[string]xproto.Atom),
		}
	})
	if xConn != nil && xConn.err != nil {
		return nil, xConn.err
	}
	return xConn, nil
}

func (s *xConnState) atom(name string) (xproto.Atom, error) {
	s.atomsMu.Lock()
	defer s.atomsMu.Unlock()
	if a, ok := s.atoms[name]; ok {
		return a, nil
	}
	reply, err := xproto.InternAtom(s.conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, err
	}
	s.atoms[name] = reply.Atom
	return reply.Atom, nil
}

// getProperty fetches a window property by atom name. It returns the raw bytes
// alongside the reply so callers can inspect Type/Format. Errors that simply
// mean "property not set" become (nil, nil).
func (s *xConnState) getProperty(win xproto.Window, name string) (*xproto.GetPropertyReply, error) {
	a, err := s.atom(name)
	if err != nil {
		return nil, err
	}
	reply, err := xproto.GetProperty(s.conn, false, win, a, xproto.GetPropertyTypeAny, 0, 1<<20).Reply()
	if err != nil {
		// Window may have been destroyed between enumeration and property fetch.
		return nil, nil
	}
	return reply, nil
}

func decodeAtomList(reply *xproto.GetPropertyReply) []xproto.Atom {
	if reply == nil || reply.Format != 32 || reply.ValueLen == 0 {
		return nil
	}
	out := make([]xproto.Atom, reply.ValueLen)
	for i := uint32(0); i < reply.ValueLen; i++ {
		base := i * 4
		out[i] = xproto.Atom(uint32(reply.Value[base]) | uint32(reply.Value[base+1])<<8 |
			uint32(reply.Value[base+2])<<16 | uint32(reply.Value[base+3])<<24)
	}
	return out
}

func decodeWindowList(reply *xproto.GetPropertyReply) []xproto.Window {
	atoms := decodeAtomList(reply)
	if atoms == nil {
		return nil
	}
	out := make([]xproto.Window, len(atoms))
	for i, a := range atoms {
		out[i] = xproto.Window(a)
	}
	return out
}

func decodeCARDINAL(reply *xproto.GetPropertyReply) (uint32, bool) {
	if reply == nil || reply.Format != 32 || reply.ValueLen == 0 {
		return 0, false
	}
	v := uint32(reply.Value[0]) | uint32(reply.Value[1])<<8 |
		uint32(reply.Value[2])<<16 | uint32(reply.Value[3])<<24
	return v, true
}

func decodeCARDINALs(reply *xproto.GetPropertyReply, n int) ([]int32, bool) {
	if reply == nil || reply.Format != 32 || int(reply.ValueLen) < n {
		return nil, false
	}
	out := make([]int32, n)
	for i := 0; i < n; i++ {
		base := i * 4
		out[i] = int32(uint32(reply.Value[base]) | uint32(reply.Value[base+1])<<8 |
			uint32(reply.Value[base+2])<<16 | uint32(reply.Value[base+3])<<24)
	}
	return out, true
}

func decodeUTF8String(reply *xproto.GetPropertyReply) string {
	if reply == nil || reply.ValueLen == 0 {
		return ""
	}
	return string(reply.Value)
}

func decodeWMClass(reply *xproto.GetPropertyReply) (instance, class string) {
	if reply == nil || reply.ValueLen == 0 {
		return "", ""
	}
	// Two NUL-terminated strings: instance\0class\0
	data := reply.Value
	first := 0
	for first < len(data) && data[first] != 0 {
		first++
	}
	instance = string(data[:first])
	second := first + 1
	if second >= len(data) {
		return instance, ""
	}
	end := second
	for end < len(data) && data[end] != 0 {
		end++
	}
	class = string(data[second:end])
	return instance, class
}

// atomNames resolves a slice of atom IDs to their string names, caching as it
// goes. Failures degrade gracefully (empty string in place).
func (s *xConnState) atomNames(atoms []xproto.Atom) []string {
	out := make([]string, len(atoms))
	for i, a := range atoms {
		if a == 0 {
			continue
		}
		reply, err := xproto.GetAtomName(s.conn, a).Reply()
		if err != nil || reply == nil {
			continue
		}
		out[i] = string(reply.Name)
	}
	return out
}

// sliceErrors records non-fatal errors during a List call.
var errBatch = errors.New("partial errors during window enumeration")
