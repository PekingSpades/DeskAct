//go:build linux
// +build linux

package window

import (
	"fmt"

	"github.com/PekingSpades/DeskAct/capture"
	"github.com/PekingSpades/DeskAct/display"
	"github.com/jezek/xgb/xproto"
)

// EWMH client message sources.
const (
	ewmhSourceApplication = 1

	// _NET_WM_STATE actions.
	netWmStateRemove = 0
	netWmStateAdd    = 1
)

// sendEWMH sends a 32-bit ClientMessage to the root window. data should be
// exactly 5 32-bit values (will be padded if shorter).
func sendEWMH(s *xConnState, win xproto.Window, atomName string, data []uint32) error {
	a, err := s.atom(atomName)
	if err != nil {
		return err
	}
	var raw [5]uint32
	for i := 0; i < len(data) && i < 5; i++ {
		raw[i] = data[i]
	}
	cm := xproto.ClientMessageEvent{
		Format: 32,
		Window: win,
		Type:   a,
		Data: xproto.ClientMessageDataUnionData32New([]uint32{
			raw[0], raw[1], raw[2], raw[3], raw[4],
		}),
	}
	return xproto.SendEventChecked(s.conn, false, s.root,
		uint32(xproto.EventMaskSubstructureNotify|xproto.EventMaskSubstructureRedirect),
		string(cm.Bytes())).Check()
}

func moveWindow(id uint64, pid int32, x, y int) error {
	s, err := linuxXConn()
	if err != nil {
		return err
	}
	win := xproto.Window(id)
	// _NET_MOVERESIZE_WINDOW data: gravity|flags, x, y, w, h.
	// flags bit 8 = x, bit 9 = y, gravity 0 = keep WM default.
	const flags = 0 | (1 << 8) | (1 << 9)
	return sendEWMH(s, win, "_NET_MOVERESIZE_WINDOW", []uint32{
		uint32(ewmhSourceApplication<<12) | uint32(flags),
		uint32(int32(x)),
		uint32(int32(y)),
		0, 0,
	})
}

func resizeWindow(id uint64, pid int32, w, h int) error {
	s, err := linuxXConn()
	if err != nil {
		return err
	}
	win := xproto.Window(id)
	const flags = 0 | (1 << 10) | (1 << 11)
	return sendEWMH(s, win, "_NET_MOVERESIZE_WINDOW", []uint32{
		uint32(ewmhSourceApplication<<12) | uint32(flags),
		0, 0,
		uint32(int32(w)),
		uint32(int32(h)),
	})
}

func moveResizeWindow(id uint64, pid int32, x, y, w, h int) error {
	s, err := linuxXConn()
	if err != nil {
		return err
	}
	win := xproto.Window(id)
	const flags = 0 | (1 << 8) | (1 << 9) | (1 << 10) | (1 << 11)
	return sendEWMH(s, win, "_NET_MOVERESIZE_WINDOW", []uint32{
		uint32(ewmhSourceApplication<<12) | uint32(flags),
		uint32(int32(x)), uint32(int32(y)),
		uint32(int32(w)), uint32(int32(h)),
	})
}

func raiseWindow(id uint64, pid int32) error {
	s, err := linuxXConn()
	if err != nil {
		return err
	}
	return xproto.ConfigureWindowChecked(s.conn, xproto.Window(id),
		xproto.ConfigWindowStackMode,
		[]uint32{xproto.StackModeAbove}).Check()
}

func focusWindow(id uint64, pid int32) error {
	s, err := linuxXConn()
	if err != nil {
		return err
	}
	win := xproto.Window(id)
	// _NET_ACTIVE_WINDOW: source, timestamp, requestor's active window.
	return sendEWMH(s, win, "_NET_ACTIVE_WINDOW", []uint32{
		uint32(ewmhSourceApplication),
		0, 0,
	})
}

func minimizeWindow(id uint64, pid int32) error {
	s, err := linuxXConn()
	if err != nil {
		return err
	}
	win := xproto.Window(id)
	// ICCCM WM_CHANGE_STATE with IconicState=3.
	return sendEWMH(s, win, "WM_CHANGE_STATE", []uint32{3})
}

func restoreWindow(id uint64, pid int32) error {
	s, err := linuxXConn()
	if err != nil {
		return err
	}
	if err := xproto.MapWindowChecked(s.conn, xproto.Window(id)).Check(); err != nil {
		return fmt.Errorf("%w: MapWindow: %v", capture.ErrCaptureFailed, err)
	}
	return focusWindow(id, pid)
}

func closeWindow(id uint64, pid int32) error {
	s, err := linuxXConn()
	if err != nil {
		return err
	}
	return sendEWMH(s, xproto.Window(id), "_NET_CLOSE_WINDOW", []uint32{
		0, uint32(ewmhSourceApplication),
	})
}

func boundsWindow(id uint64, pid int32) (display.Rect, error) {
	s, err := linuxXConn()
	if err != nil {
		return display.Rect{}, err
	}
	win := xproto.Window(id)
	geo, err := xproto.GetGeometry(s.conn, xproto.Drawable(win)).Reply()
	if err != nil || geo == nil {
		return display.Rect{}, fmt.Errorf("%w: xid=0x%x", capture.ErrWindowNotFound, id)
	}
	tx, err := xproto.TranslateCoordinates(s.conn, win, s.root, 0, 0).Reply()
	if err != nil || tx == nil {
		return display.Rect{}, fmt.Errorf("%w: xid=0x%x", capture.ErrWindowNotFound, id)
	}
	return makeRect(int(tx.DstX), int(tx.DstY), int(geo.Width), int(geo.Height)), nil
}
