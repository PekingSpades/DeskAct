package screenshot

import (
	"errors"
	"fmt"
	"image"
	"runtime"
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
	worker *dxgiDuplicationWorker
}

type dxgiDuplicationWorker struct {
	requests chan dxgiCaptureRequest
}

type dxgiCaptureRequest struct {
	req  cap.Request
	resp chan dxgiCaptureResponse
}

type dxgiCaptureResponse struct {
	img *image.RGBA
	err error
}

func newDXGIDuplicationManager(factory dxgiSessionFactory) *dxgiDuplicationManager {
	return &dxgiDuplicationManager{
		entries:    make(map[int]*dxgiDuplicationEntry),
		newSession: factory,
	}
}

func (m *dxgiDuplicationManager) Capture(req cap.Request) (*image.RGBA, error) {
	entry := m.entry(req.DisplayID)
	resp := make(chan dxgiCaptureResponse, 1)
	entry.worker.requests <- dxgiCaptureRequest{
		req:  req,
		resp: resp,
	}
	result := <-resp
	return result.img, result.err
}

func (m *dxgiDuplicationManager) entry(displayID int) *dxgiDuplicationEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := m.entries[displayID]
	if entry == nil {
		entry = &dxgiDuplicationEntry{
			worker: newDXGIDuplicationWorker(displayID, m.newSession),
		}
		m.entries[displayID] = entry
	}
	return entry
}

func newDXGIDuplicationWorker(displayID int, factory dxgiSessionFactory) *dxgiDuplicationWorker {
	worker := &dxgiDuplicationWorker{
		requests: make(chan dxgiCaptureRequest),
	}
	go worker.run(displayID, factory)
	return worker
}

func (w *dxgiDuplicationWorker) run(displayID int, factory dxgiSessionFactory) {
	// Keep DXGI/Desktop Duplication work on a fixed OS thread. The reference
	// implementations are all single-threaded here, and this avoids Go runtime
	// thread migration across COM/D3D11/Desktop APIs.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var session dxgiCaptureSession
	defer func() {
		if session != nil {
			session.close()
		}
	}()

	for request := range w.requests {
		img, err := func() (img *image.RGBA, err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if session != nil {
						session.close()
						session = nil
					}
					err = fmt.Errorf("DXGI capture panicked: %v", recovered)
				}
			}()

			session, err = ensureDXGISession(session, displayID, factory)
			if err != nil {
				return nil, err
			}

			img, err = session.capture(request.req)
			if err == nil {
				return img, nil
			}
			if !errors.Is(err, errDXGIAccessLost) {
				return nil, err
			}

			session.close()
			session = nil

			session, err = ensureDXGISession(session, displayID, factory)
			if err != nil {
				return nil, err
			}
			return session.capture(request.req)
		}()

		request.resp <- dxgiCaptureResponse{
			img: img,
			err: err,
		}
	}
}

func ensureDXGISession(session dxgiCaptureSession, displayID int, factory dxgiSessionFactory) (dxgiCaptureSession, error) {
	if session != nil {
		return session, nil
	}

	session, err := factory(displayID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, errors.New("DXGI session factory returned nil session")
	}
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
