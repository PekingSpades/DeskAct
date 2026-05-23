//go:build linux
// +build linux

package keyboard

import (
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// lookupXIDByPID walks every visible top-level window and returns the first
// XID whose _NET_WM_PID matches the supplied pid. Best-effort: when no
// window is found (or no X server reachable) it returns 0 and the caller
// reports ErrKeyWindowNotFound.
func lookupXIDByPID(pid int) uint32 {
	if pid <= 0 {
		return 0
	}
	c, err := xgb.NewConn()
	if err != nil {
		return 0
	}
	defer c.Close()

	setup := xproto.Setup(c)
	if setup == nil || len(setup.Roots) == 0 {
		return 0
	}
	root := setup.DefaultScreen(c).Root

	pidAtom, err := xproto.InternAtom(c, false, uint16(len("_NET_WM_PID")), "_NET_WM_PID").Reply()
	if err != nil {
		return 0
	}
	clientListAtom, err := xproto.InternAtom(c, false, uint16(len("_NET_CLIENT_LIST")), "_NET_CLIENT_LIST").Reply()
	if err != nil {
		return 0
	}

	rep, err := xproto.GetProperty(c, false, root, clientListAtom.Atom,
		xproto.GetPropertyTypeAny, 0, 1<<20).Reply()
	if err != nil || rep == nil || rep.Format != 32 {
		// Fall back to QueryTree if _NET_CLIENT_LIST is missing.
		tree, err := xproto.QueryTree(c, root).Reply()
		if err != nil || tree == nil {
			return 0
		}
		for _, w := range tree.Children {
			if matched := matchesPID(c, w, pidAtom.Atom, uint32(pid)); matched {
				return uint32(w)
			}
		}
		return 0
	}

	for i := uint32(0); i < rep.ValueLen; i++ {
		b := i * 4
		w := xproto.Window(uint32(rep.Value[b]) | uint32(rep.Value[b+1])<<8 |
			uint32(rep.Value[b+2])<<16 | uint32(rep.Value[b+3])<<24)
		if matchesPID(c, w, pidAtom.Atom, uint32(pid)) {
			return uint32(w)
		}
	}
	return 0
}

func matchesPID(c *xgb.Conn, win xproto.Window, pidAtom xproto.Atom, want uint32) bool {
	r, err := xproto.GetProperty(c, false, win, pidAtom,
		xproto.GetPropertyTypeAny, 0, 1).Reply()
	if err != nil || r == nil || r.Format != 32 || r.ValueLen == 0 {
		return false
	}
	got := uint32(r.Value[0]) | uint32(r.Value[1])<<8 |
		uint32(r.Value[2])<<16 | uint32(r.Value[3])<<24
	return got == want
}
