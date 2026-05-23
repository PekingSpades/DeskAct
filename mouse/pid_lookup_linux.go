//go:build linux

package mouse

import (
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// xidByPIDPlatform walks every visible top-level X11 window and returns the
// first XID whose _NET_WM_PID matches the supplied pid. Mirrors the
// keyboard package's lookupXIDByPID (copied here to avoid mouse importing
// keyboard). Best-effort: returns 0 when no window is found or no X
// server is reachable.
func xidByPIDPlatform(pid int) uint64 {
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
		tree, err := xproto.QueryTree(c, root).Reply()
		if err != nil || tree == nil {
			return 0
		}
		for _, w := range tree.Children {
			if matchesPID(c, w, pidAtom.Atom, uint32(pid)) {
				return uint64(w)
			}
		}
		return 0
	}

	for i := uint32(0); i < rep.ValueLen; i++ {
		b := i * 4
		w := xproto.Window(uint32(rep.Value[b]) | uint32(rep.Value[b+1])<<8 |
			uint32(rep.Value[b+2])<<16 | uint32(rep.Value[b+3])<<24)
		if matchesPID(c, w, pidAtom.Atom, uint32(pid)) {
			return uint64(w)
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

// hwndByPIDPlatform is Windows-only; on Linux MoveWithPID never calls it.
func hwndByPIDPlatform(pid int) uint64 { return 0 }

// frontWindowIDByPIDPlatform is macOS-only.
func frontWindowIDByPIDPlatform(pid int) uint64 { return 0 }
