package screenshot

import (
	"errors"
	"image"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cap "github.com/PekingSpades/DeskAct/capture"
)

func TestDXGIExpectedSurfaceSize(t *testing.T) {
	tests := []struct {
		name          string
		rotation      uint32
		desktopWidth  int
		desktopHeight int
		wantWidth     int
		wantHeight    int
		wantErr       bool
	}{
		{name: "identity", rotation: dxgiRotationIdentity, desktopWidth: 1920, desktopHeight: 1080, wantWidth: 1920, wantHeight: 1080},
		{name: "unspecified", rotation: dxgiRotationUnspecified, desktopWidth: 1920, desktopHeight: 1080, wantWidth: 1920, wantHeight: 1080},
		{name: "rotate90", rotation: dxgiRotationRotate90, desktopWidth: 768, desktopHeight: 1024, wantWidth: 1024, wantHeight: 768},
		{name: "rotate180", rotation: dxgiRotationRotate180, desktopWidth: 1920, desktopHeight: 1080, wantWidth: 1920, wantHeight: 1080},
		{name: "rotate270", rotation: dxgiRotationRotate270, desktopWidth: 768, desktopHeight: 1024, wantWidth: 1024, wantHeight: 768},
		{name: "unknown", rotation: 99, desktopWidth: 10, desktopHeight: 20, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWidth, gotHeight, err := dxgiExpectedSurfaceSize(tt.rotation, tt.desktopWidth, tt.desktopHeight)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error for unsupported rotation")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotWidth != tt.wantWidth || gotHeight != tt.wantHeight {
				t.Fatalf("expected %dx%d, got %dx%d", tt.wantWidth, tt.wantHeight, gotWidth, gotHeight)
			}
		})
	}
}

func TestMapDesktopPointToDXGISurface(t *testing.T) {
	tests := []struct {
		name          string
		rotation      uint32
		desktopWidth  int
		desktopHeight int
		x             int
		y             int
		want          image.Point
	}{
		{name: "identity top left", rotation: dxgiRotationIdentity, desktopWidth: 3, desktopHeight: 4, x: 0, y: 0, want: image.Pt(0, 0)},
		{name: "rotate90 top left", rotation: dxgiRotationRotate90, desktopWidth: 3, desktopHeight: 4, x: 0, y: 0, want: image.Pt(0, 2)},
		{name: "rotate90 bottom right", rotation: dxgiRotationRotate90, desktopWidth: 3, desktopHeight: 4, x: 2, y: 3, want: image.Pt(3, 0)},
		{name: "rotate180 middle", rotation: dxgiRotationRotate180, desktopWidth: 3, desktopHeight: 4, x: 1, y: 2, want: image.Pt(1, 1)},
		{name: "rotate270 top left", rotation: dxgiRotationRotate270, desktopWidth: 3, desktopHeight: 4, x: 0, y: 0, want: image.Pt(3, 0)},
		{name: "rotate270 bottom right", rotation: dxgiRotationRotate270, desktopWidth: 3, desktopHeight: 4, x: 2, y: 3, want: image.Pt(0, 2)},
		{name: "unspecified behaves like identity", rotation: dxgiRotationUnspecified, desktopWidth: 3, desktopHeight: 4, x: 2, y: 1, want: image.Pt(2, 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapDesktopPointToDXGISurface(tt.rotation, tt.desktopWidth, tt.desktopHeight, tt.x, tt.y); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestClassifyDXGIWaitTimeout(t *testing.T) {
	now := time.Now()
	deadline := now.Add(250 * time.Millisecond)

	if got := classifyDXGIWaitTimeout(true, now, deadline); got != dxgiWaitTimeoutUseCachedFrame {
		t.Fatalf("expected cached-frame action, got %v", got)
	}
	if got := classifyDXGIWaitTimeout(false, now, deadline); got != dxgiWaitTimeoutRetryAcquire {
		t.Fatalf("expected retry action before deadline, got %v", got)
	}
	if got := classifyDXGIWaitTimeout(false, deadline.Add(time.Millisecond), deadline); got != dxgiWaitTimeoutFail {
		t.Fatalf("expected fail action after deadline, got %v", got)
	}
}

func TestDXGIDuplicationManagerRecreatesOnAccessLost(t *testing.T) {
	var factoryCalls int32
	var firstClosed int32
	var secondCaptures int32

	manager := newDXGIDuplicationManager(func(displayID int) (dxgiCaptureSession, error) {
		call := atomic.AddInt32(&factoryCalls, 1)
		switch call {
		case 1:
			return &fakeDXGISession{
				captureFn: func(req cap.Request) (*image.RGBA, error) {
					return nil, errDXGIAccessLost
				},
				closeFn: func() {
					atomic.AddInt32(&firstClosed, 1)
				},
			}, nil
		case 2:
			return &fakeDXGISession{
				captureFn: func(req cap.Request) (*image.RGBA, error) {
					atomic.AddInt32(&secondCaptures, 1)
					return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
				},
			}, nil
		default:
			return nil, errors.New("unexpected extra factory call")
		}
	})

	img, err := manager.Capture(cap.Request{DisplayID: 7})
	if err != nil {
		t.Fatalf("expected recreate to succeed, got %v", err)
	}
	if img == nil {
		t.Fatal("expected an image after recreate")
	}
	if atomic.LoadInt32(&factoryCalls) != 2 {
		t.Fatalf("expected 2 factory calls, got %d", factoryCalls)
	}
	if atomic.LoadInt32(&firstClosed) != 1 {
		t.Fatalf("expected first session to be closed once, got %d", firstClosed)
	}
	if atomic.LoadInt32(&secondCaptures) != 1 {
		t.Fatalf("expected replacement session to capture once, got %d", secondCaptures)
	}
}

func TestDXGIDuplicationManagerSerializesSameDisplay(t *testing.T) {
	var factoryCalls int32
	var captureCalls int32

	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondEntered := make(chan struct{})

	session := &fakeDXGISession{
		captureFn: func(req cap.Request) (*image.RGBA, error) {
			call := atomic.AddInt32(&captureCalls, 1)
			if call == 1 {
				close(firstEntered)
				<-releaseFirst
			} else {
				close(secondEntered)
			}
			return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
		},
	}

	manager := newDXGIDuplicationManager(func(displayID int) (dxgiCaptureSession, error) {
		atomic.AddInt32(&factoryCalls, 1)
		return session, nil
	})

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	req := cap.Request{DisplayID: 9}

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := manager.Capture(req)
		errs <- err
	}()

	<-firstEntered

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := manager.Capture(req)
		errs <- err
	}()

	select {
	case <-secondEntered:
		t.Fatal("second capture entered the session before the first finished")
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFirst)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("unexpected capture error: %v", err)
		}
	}
	if atomic.LoadInt32(&factoryCalls) != 1 {
		t.Fatalf("expected one shared session for the same display, got %d", factoryCalls)
	}
	if atomic.LoadInt32(&captureCalls) != 2 {
		t.Fatalf("expected two capture calls, got %d", captureCalls)
	}
}

func TestDXGIDuplicationManagerAllowsDifferentDisplays(t *testing.T) {
	displayOneEntered := make(chan struct{})
	releaseDisplayOne := make(chan struct{})
	displayTwoEntered := make(chan struct{})

	manager := newDXGIDuplicationManager(func(displayID int) (dxgiCaptureSession, error) {
		switch displayID {
		case 1:
			return &fakeDXGISession{
				captureFn: func(req cap.Request) (*image.RGBA, error) {
					close(displayOneEntered)
					<-releaseDisplayOne
					return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
				},
			}, nil
		case 2:
			return &fakeDXGISession{
				captureFn: func(req cap.Request) (*image.RGBA, error) {
					close(displayTwoEntered)
					return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
				},
			}, nil
		default:
			return nil, errors.New("unexpected display ID")
		}
	})

	doneOne := make(chan error, 1)
	go func() {
		_, err := manager.Capture(cap.Request{DisplayID: 1})
		doneOne <- err
	}()

	<-displayOneEntered

	doneTwo := make(chan error, 1)
	go func() {
		_, err := manager.Capture(cap.Request{DisplayID: 2})
		doneTwo <- err
	}()

	select {
	case <-displayTwoEntered:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("second display capture did not proceed while the first display was blocked")
	}

	close(releaseDisplayOne)

	if err := <-doneOne; err != nil {
		t.Fatalf("unexpected error for display 1: %v", err)
	}
	if err := <-doneTwo; err != nil {
		t.Fatalf("unexpected error for display 2: %v", err)
	}
}

type fakeDXGISession struct {
	captureFn func(req cap.Request) (*image.RGBA, error)
	closeFn   func()
}

func (s *fakeDXGISession) capture(req cap.Request) (*image.RGBA, error) {
	if s.captureFn == nil {
		return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
	}
	return s.captureFn(req)
}

func (s *fakeDXGISession) close() {
	if s.closeFn != nil {
		s.closeFn()
	}
}
