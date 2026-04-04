package screenshot

import (
	"errors"
	"fmt"
	"image"
	"sync"
	"time"

	cap "github.com/PekingSpades/DeskAct/capture"
)

const (
	dxgiRotationUnspecified uint32 = 0
	dxgiRotationIdentity    uint32 = 1
	dxgiRotationRotate90    uint32 = 2
	dxgiRotationRotate180   uint32 = 3
	dxgiRotationRotate270   uint32 = 4

	dxgiInitialFrameMaxWait = 2 * time.Second
)

var (
	errDXGIAccessLost   = errors.New("dxgi duplication access lost")
	errDXGIFrameTimeout = errors.New("dxgi frame wait timeout")
)

type dxgiWaitTimeoutAction int

const (
	dxgiWaitTimeoutUseCachedFrame dxgiWaitTimeoutAction = iota
	dxgiWaitTimeoutRetryAcquire
	dxgiWaitTimeoutFail
)

type dxgiCaptureSession interface {
	capture(req cap.Request) (*image.RGBA, error)
	close()
}

type dxgiSessionFactory func(displayID int) (dxgiCaptureSession, error)

type dxgiDuplicationManager struct {
	mu         sync.Mutex
	entries    map[int]*dxgiDuplicationEntry
	newSession dxgiSessionFactory
}

type dxgiDuplicationEntry struct {
	mu      sync.Mutex
	session dxgiCaptureSession
}

func newDXGIDuplicationManager(factory dxgiSessionFactory) *dxgiDuplicationManager {
	return &dxgiDuplicationManager{
		entries:    make(map[int]*dxgiDuplicationEntry),
		newSession: factory,
	}
}

func (m *dxgiDuplicationManager) Capture(req cap.Request) (*image.RGBA, error) {
	entry := m.entry(req.DisplayID)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	session, err := m.ensureSessionLocked(entry, req.DisplayID)
	if err != nil {
		return nil, err
	}

	img, err := session.capture(req)
	if err == nil {
		return img, nil
	}
	if !errors.Is(err, errDXGIAccessLost) {
		return nil, err
	}

	session.close()
	entry.session = nil

	session, err = m.ensureSessionLocked(entry, req.DisplayID)
	if err != nil {
		return nil, err
	}
	return session.capture(req)
}

func (m *dxgiDuplicationManager) entry(displayID int) *dxgiDuplicationEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := m.entries[displayID]
	if entry == nil {
		entry = &dxgiDuplicationEntry{}
		m.entries[displayID] = entry
	}
	return entry
}

func (m *dxgiDuplicationManager) ensureSessionLocked(entry *dxgiDuplicationEntry, displayID int) (dxgiCaptureSession, error) {
	if entry.session != nil {
		return entry.session, nil
	}

	session, err := m.newSession(displayID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, errors.New("DXGI session factory returned nil session")
	}
	entry.session = session
	return session, nil
}

func classifyDXGIWaitTimeout(hasFrame bool, now, deadline time.Time) dxgiWaitTimeoutAction {
	if hasFrame {
		return dxgiWaitTimeoutUseCachedFrame
	}
	if !now.After(deadline) {
		return dxgiWaitTimeoutRetryAcquire
	}
	return dxgiWaitTimeoutFail
}

func normalizeDXGIRotation(rotation uint32) uint32 {
	if rotation == dxgiRotationUnspecified {
		return dxgiRotationIdentity
	}
	return rotation
}

func dxgiExpectedSurfaceSize(rotation uint32, desktopWidth, desktopHeight int) (int, int, error) {
	switch normalizeDXGIRotation(rotation) {
	case dxgiRotationIdentity, dxgiRotationRotate180:
		return desktopWidth, desktopHeight, nil
	case dxgiRotationRotate90, dxgiRotationRotate270:
		return desktopHeight, desktopWidth, nil
	default:
		return 0, 0, fmt.Errorf("unsupported DXGI rotation value %d", rotation)
	}
}

func mapDesktopPointToDXGISurface(rotation uint32, desktopWidth, desktopHeight, x, y int) image.Point {
	switch normalizeDXGIRotation(rotation) {
	case dxgiRotationRotate90:
		return image.Pt(y, desktopWidth-1-x)
	case dxgiRotationRotate180:
		return image.Pt(desktopWidth-1-x, desktopHeight-1-y)
	case dxgiRotationRotate270:
		return image.Pt(desktopHeight-1-y, x)
	default:
		return image.Pt(x, y)
	}
}
