//go:build linux
// +build linux

package window

import (
	"github.com/PekingSpades/DeskAct/display"
	"github.com/jezek/xgb/xproto"
)

func listWindows(options WindowOptions) ([]WindowInfo, error) {
	s, err := linuxXConn()
	if err != nil {
		return nil, err
	}

	// Prefer _NET_CLIENT_LIST_STACKING (bottom-to-top); fall back to
	// _NET_CLIENT_LIST then QueryTree.
	var wins []xproto.Window
	if rep, _ := s.getProperty(s.root, "_NET_CLIENT_LIST_STACKING"); rep != nil {
		wins = decodeWindowList(rep)
	}
	if len(wins) == 0 {
		if rep, _ := s.getProperty(s.root, "_NET_CLIENT_LIST"); rep != nil {
			wins = decodeWindowList(rep)
		}
	}
	if len(wins) == 0 {
		tree, err := xproto.QueryTree(s.conn, s.root).Reply()
		if err != nil {
			return nil, err
		}
		wins = tree.Children
	}

	out := make([]WindowInfo, 0, len(wins))
	for idx, w := range wins {
		info, ok := buildLinuxWindowInfo(s, w, idx, options)
		if !ok {
			continue
		}
		out = append(out, info)
	}
	return out, nil
}

func buildLinuxWindowInfo(s *xConnState, win xproto.Window, stacking int, options WindowOptions) (WindowInfo, bool) {
	// Geometry + absolute coordinates via TranslateCoordinates(win, root, 0, 0).
	geo, err := xproto.GetGeometry(s.conn, xproto.Drawable(win)).Reply()
	if err != nil || geo == nil {
		return WindowInfo{}, false
	}
	tx, err := xproto.TranslateCoordinates(s.conn, win, s.root, 0, 0).Reply()
	if err != nil || tx == nil {
		return WindowInfo{}, false
	}

	bounds := makeRect(int(tx.DstX), int(tx.DstY), int(geo.Width), int(geo.Height))

	attrs, err := xproto.GetWindowAttributes(s.conn, win).Reply()
	isVisible := err == nil && attrs != nil && attrs.MapState == xproto.MapStateViewable

	// Properties.
	nameRep, _ := s.getProperty(win, "_NET_WM_NAME")
	name := decodeUTF8String(nameRep)
	if name == "" {
		if rep, _ := s.getProperty(win, "WM_NAME"); rep != nil {
			name = decodeUTF8String(rep)
		}
	}

	pidRep, _ := s.getProperty(win, "_NET_WM_PID")
	pid32, _ := decodeCARDINAL(pidRep)

	classRep, _ := s.getProperty(win, "WM_CLASS")
	instance, class := decodeWMClass(classRep)

	frameRep, _ := s.getProperty(win, "_NET_FRAME_EXTENTS")
	var frame [4]int
	if vals, ok := decodeCARDINALs(frameRep, 4); ok {
		for i := range frame {
			frame[i] = int(vals[i])
		}
	}

	stateRep, _ := s.getProperty(win, "_NET_WM_STATE")
	stateAtoms := decodeAtomList(stateRep)
	states := s.atomNames(stateAtoms)

	parentXID := uint32(0)
	if tree, err := xproto.QueryTree(s.conn, win).Reply(); err == nil && tree != nil {
		parentXID = uint32(tree.Parent)
	}

	platform := &LinuxPlatformInfo{
		XID:          uint32(win),
		ParentXID:    parentXID,
		Name:         name,
		Class:        class,
		Instance:     instance,
		FrameExtents: frame,
		States:       states,
		Stacking:     stacking,
	}

	isMinimized := platform.IsMinimized()

	if !options.IncludeMinimized && isMinimized {
		return WindowInfo{}, false
	}
	if !options.IncludeInvisible && !isVisible {
		return WindowInfo{}, false
	}
	if bounds.W <= 0 || bounds.H <= 0 {
		return WindowInfo{}, false
	}

	return WindowInfo{
		ID:          uint64(win),
		PID:         int(pid32),
		Title:       name,
		Bounds:      bounds,
		IsVisible:   isVisible,
		IsMinimized: isMinimized,
		platform:    platform,
	}, true
}

// adjustForFrameExtents is a helper exposed for ops_linux.go: returns the
// adjusted (outer) bounds that EWMH _NET_MOVERESIZE_WINDOW expects when given
// the inner client rect.
func (l *LinuxPlatformInfo) adjustForFrame(r display.Rect) display.Rect {
	return display.Rect{
		Point: display.Point{X: r.X - l.FrameExtents[0], Y: r.Y - l.FrameExtents[2]},
		Size:  display.Size{W: r.W + l.FrameExtents[0] + l.FrameExtents[1], H: r.H + l.FrameExtents[2] + l.FrameExtents[3]},
	}
}
