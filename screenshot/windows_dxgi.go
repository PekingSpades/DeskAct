//go:build windows

package screenshot

import (
	"errors"
	"fmt"
	"image"
	"syscall"
	"time"
	"unsafe"

	cap "github.com/PekingSpades/DeskAct/capture"
	"golang.org/x/sys/windows"
)

const (
	dxgiFactory1EnumAdapters1Method              = 12
	dxgiAdapterEnumOutputsMethod                 = 7
	dxgiOutputGetDescMethod                      = 7
	dxgiOutput1DuplicateOutputMethod             = 22
	dxgiOutputDuplicationAcquireNextFrame        = 8
	dxgiOutputDuplicationReleaseFrame            = 14
	d3d11DeviceCreateTexture2DMethod             = 5
	d3d11DeviceContextMapMethod                  = 10
	d3d11DeviceContextUnmapMethod                = 11
	d3d11DeviceContextCopyResourceMethod         = 43
	d3d11Texture2DGetDescMethod                  = 10
	dxgiAcquireFrameTimeoutMillis         uint   = 250
	d3dDriverTypeUnknown                  uint   = 0
	d3d11CreateDeviceBGRASupport          uint   = 0x20
	d3d11SDKVersion                       uint   = 7
	d3d11UsageStaging                     uint32 = 3
	d3d11CPUAccessRead                    uint32 = 0x20000
	d3d11MapRead                          uint32 = 1
	dxgiErrorNotFound                     uint32 = 0x887A0002
	dxgiErrorUnsupported                  uint32 = 0x887A0004
	dxgiErrorNotCurrentlyAvailable        uint32 = 0x887A0022
	dxgiErrorAccessLost                   uint32 = 0x887A0026
	dxgiErrorWaitTimeout                  uint32 = 0x887A0027
	dxgiErrorSessionDisconnected          uint32 = 0x887A0028
	eAccessDenied                         uint32 = 0x80070005
)

var (
	modDXGI                = windows.NewLazySystemDLL("dxgi.dll")
	procCreateDXGIFactory1 = modDXGI.NewProc("CreateDXGIFactory1")
	modD3D11               = windows.NewLazySystemDLL("d3d11.dll")
	procD3D11CreateDevice  = modD3D11.NewProc("D3D11CreateDevice")
	modUser32DXGI          = windows.NewLazySystemDLL("user32.dll")
	procOpenInputDesktop   = modUser32DXGI.NewProc("OpenInputDesktop")
	procSetThreadDesktop   = modUser32DXGI.NewProc("SetThreadDesktop")
	procCloseDesktop       = modUser32DXGI.NewProc("CloseDesktop")

	iidIDXGIFactory1   = windows.GUID{Data1: 0x770aae78, Data2: 0xf26f, Data3: 0x4dba, Data4: [8]byte{0xa8, 0x29, 0x25, 0x3c, 0x83, 0xd1, 0xb3, 0x87}}
	iidIDXGIOutput1    = windows.GUID{Data1: 0x00cddea8, Data2: 0x939b, Data3: 0x4b83, Data4: [8]byte{0xa3, 0x40, 0xa6, 0x85, 0x22, 0x66, 0x66, 0xcc}}
	iidID3D11Texture2D = windows.GUID{Data1: 0x6f15aaf2, Data2: 0xd208, Data3: 0x4e89, Data4: [8]byte{0x9a, 0xb4, 0x48, 0x95, 0x35, 0xd3, 0x4f, 0x9c}}
)

type dxgiFactory1 struct{ lpVtbl uintptr }
type dxgiAdapter1 struct{ lpVtbl uintptr }
type dxgiOutput struct{ lpVtbl uintptr }
type dxgiOutput1 struct{ lpVtbl uintptr }
type dxgiOutputDuplication struct{ lpVtbl uintptr }
type dxgiResource struct{ lpVtbl uintptr }
type d3d11Device struct{ lpVtbl uintptr }
type d3d11DeviceContext struct{ lpVtbl uintptr }
type d3d11Texture2D struct{ lpVtbl uintptr }

type dxgiRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type dxgiPoint struct {
	X int32
	Y int32
}

type dxgiOutputDesc struct {
	DeviceName         [32]uint16
	DesktopCoordinates dxgiRect
	AttachedToDesktop  int32
	Rotation           uint32
	Monitor            windows.Handle
}

type dxgiOutduplPointerPosition struct {
	Position dxgiPoint
	Visible  int32
}

type dxgiOutduplFrameInfo struct {
	LastPresentTime           int64
	LastMouseUpdateTime       int64
	AccumulatedFrames         uint32
	RectsCoalesced            int32
	ProtectedContentMaskedOut int32
	PointerPosition           dxgiOutduplPointerPosition
	TotalMetadataBufferSize   uint32
	PointerShapeBufferSize    uint32
}

type d3d11SampleDesc struct {
	Count   uint32
	Quality uint32
}

type d3d11Texture2DDesc struct {
	Width          uint32
	Height         uint32
	MipLevels      uint32
	ArraySize      uint32
	Format         uint32
	SampleDesc     d3d11SampleDesc
	Usage          uint32
	BindFlags      uint32
	CPUAccessFlags uint32
	MiscFlags      uint32
}

type d3d11MappedSubresource struct {
	PData      unsafe.Pointer
	RowPitch   uint32
	DepthPitch uint32
}

type dxgiDuplicationSession struct {
	outputRect  dxgiRect
	rotation    uint32
	device      *d3d11Device
	context     *d3d11DeviceContext
	duplication *dxgiOutputDuplication
	staging     *d3d11Texture2D
	stagingDesc d3d11Texture2DDesc
	hasFrame    bool
}

func newDXGIDuplicationSession(displayID int) (*dxgiDuplicationSession, error) {
	factory, err := createDXGIFactory1()
	if err != nil {
		return nil, err
	}
	defer comRelease(factory)

	adapter, output, outputDesc, err := findDXGIOutput(factory, windows.Handle(uintptr(displayID)))
	if err != nil {
		return nil, err
	}
	defer comRelease(output)
	defer comRelease(adapter)

	output1, err := queryDXGIOutput1(output)
	if err != nil {
		return nil, err
	}
	defer comRelease(output1)

	device, context, err := createD3D11Device(adapter)
	if err != nil {
		return nil, err
	}

	duplication, err := duplicateOutput(output1, device)
	if err != nil {
		comRelease(context)
		comRelease(device)
		return nil, err
	}

	return &dxgiDuplicationSession{
		outputRect:  outputDesc.DesktopCoordinates,
		rotation:    outputDesc.Rotation,
		device:      device,
		context:     context,
		duplication: duplication,
	}, nil
}

func prepareDXGIThread() error {
	openInputDesktopAddr, err := procAddress(procOpenInputDesktop)
	if err != nil {
		return nil
	}
	setThreadDesktopAddr, err := procAddress(procSetThreadDesktop)
	if err != nil {
		return nil
	}
	closeDesktopAddr, err := procAddress(procCloseDesktop)
	if err != nil {
		return nil
	}

	desktop, _, callErr := syscall.SyscallN(
		openInputDesktopAddr,
		0,
		0,
		uintptr(windows.GENERIC_ALL),
	)
	if desktop == 0 {
		if callErr != 0 {
			return nil
		}
		return nil
	}
	defer syscall.SyscallN(closeDesktopAddr, desktop)

	result, _, callErr := syscall.SyscallN(setThreadDesktopAddr, desktop)
	if result != 0 {
		return nil
	}
	if callErr != 0 && callErr != windows.ERROR_BUSY {
		return fmt.Errorf("SetThreadDesktop failed: %w", callErr)
	}
	return nil
}

func (s *dxgiDuplicationSession) close() {
	comRelease(s.staging)
	s.staging = nil
	comRelease(s.duplication)
	s.duplication = nil
	comRelease(s.context)
	s.context = nil
	comRelease(s.device)
	s.device = nil
}

func (s *dxgiDuplicationSession) capture(req cap.Request) (*image.RGBA, error) {
	desktopWidth := int(s.outputRect.Right - s.outputRect.Left)
	desktopHeight := int(s.outputRect.Bottom - s.outputRect.Top)
	localX := req.X - int(s.outputRect.Left)
	localY := req.Y - int(s.outputRect.Top)
	if localX < 0 || localY < 0 || localX+req.Width > desktopWidth || localY+req.Height > desktopHeight {
		return nil, fmt.Errorf("capture rect %dx%d at (%d,%d) is outside display bounds", req.Width, req.Height, req.X, req.Y)
	}

	if err := s.refreshFrame(); err != nil {
		return nil, err
	}
	return s.readFrame(localX, localY, req.Width, req.Height, desktopWidth, desktopHeight)
}

func (s *dxgiDuplicationSession) refreshFrame() error {
	deadline := time.Now().Add(dxgiInitialFrameMaxWait)
	for {
		err := s.refreshFrameOnce()
		if err == nil {
			return nil
		}
		if !errors.Is(err, errDXGIFrameTimeout) {
			return err
		}

		switch classifyDXGIWaitTimeout(s.hasFrame, time.Now(), deadline) {
		case dxgiWaitTimeoutUseCachedFrame:
			return nil
		case dxgiWaitTimeoutRetryAcquire:
			continue
		default:
			return fmt.Errorf("DXGI initial frame was not available within %s", dxgiInitialFrameMaxWait)
		}
	}
}

func (s *dxgiDuplicationSession) refreshFrameOnce() error {
	var frameInfo dxgiOutduplFrameInfo
	var resource *dxgiResource

	hr := dxgiOutputDuplicationAcquireFrame(s.duplication, uint32(dxgiAcquireFrameTimeoutMillis), &frameInfo, &resource)
	switch hresultCode(hr) {
	case 0:
	case dxgiErrorWaitTimeout:
		return errDXGIFrameTimeout
	case dxgiErrorAccessLost:
		return fmt.Errorf("%w: AcquireNextFrame returned %s", errDXGIAccessLost, formatHRESULT(hr))
	default:
		return fmt.Errorf("IDXGIOutputDuplication::AcquireNextFrame failed: %s", formatHRESULT(hr))
	}

	defer comRelease(resource)
	defer func() {
		releaseHR := dxgiOutputDuplicationReleaseCurrentFrame(s.duplication)
		if hresultCode(releaseHR) == dxgiErrorAccessLost {
			// Access loss is handled by the caller on the next capture attempt.
			s.hasFrame = false
		}
	}()

	texture, err := queryD3D11Texture(resource)
	if err != nil {
		return err
	}
	defer comRelease(texture)

	if err := s.ensureStagingTexture(texture); err != nil {
		return err
	}

	d3d11DeviceContextCopyResource(s.context, s.staging, texture)
	s.hasFrame = true
	return nil
}

func (s *dxgiDuplicationSession) ensureStagingTexture(source *d3d11Texture2D) error {
	sourceDesc := d3d11Texture2DDescription(source)
	if sourceDesc.Width == 0 || sourceDesc.Height == 0 {
		return errors.New("DXGI frame texture has invalid dimensions")
	}

	if s.staging != nil && s.stagingDesc.Width == sourceDesc.Width && s.stagingDesc.Height == sourceDesc.Height && s.stagingDesc.Format == sourceDesc.Format {
		return nil
	}

	comRelease(s.staging)
	s.staging = nil

	stagingDesc := sourceDesc
	stagingDesc.BindFlags = 0
	stagingDesc.CPUAccessFlags = d3d11CPUAccessRead
	stagingDesc.MiscFlags = 0
	stagingDesc.Usage = d3d11UsageStaging

	staging, err := d3d11DeviceCreateTexture2D(s.device, &stagingDesc)
	if err != nil {
		return err
	}

	s.staging = staging
	s.stagingDesc = stagingDesc
	return nil
}

func (s *dxgiDuplicationSession) readFrame(localX, localY, width, height, desktopWidth, desktopHeight int) (*image.RGBA, error) {
	rect := image.Rect(0, 0, width, height)
	img, err := createImage(rect)
	if err != nil {
		return nil, err
	}

	expectedSurfaceWidth, expectedSurfaceHeight, err := dxgiExpectedSurfaceSize(s.rotation, desktopWidth, desktopHeight)
	if err != nil {
		return nil, err
	}
	if int(s.stagingDesc.Width) != expectedSurfaceWidth || int(s.stagingDesc.Height) != expectedSurfaceHeight {
		return nil, fmt.Errorf(
			"DXGI staging texture dimensions %dx%d do not match expected %dx%d for rotation %d",
			s.stagingDesc.Width,
			s.stagingDesc.Height,
			expectedSurfaceWidth,
			expectedSurfaceHeight,
			s.rotation,
		)
	}

	var mapped d3d11MappedSubresource
	hr := d3d11DeviceContextMap(s.context, s.staging, &mapped)
	if hresultFailed(hr) {
		return nil, fmt.Errorf("ID3D11DeviceContext::Map failed: %s", formatHRESULT(hr))
	}
	defer d3d11DeviceContextUnmap(s.context, s.staging)

	rowPitch := int(mapped.RowPitch)
	for row := 0; row < height; row++ {
		dst := img.Pix[row*img.Stride:]
		desktopY := localY + row
		for col := 0; col < width; col++ {
			desktopX := localX + col
			srcPoint := mapDesktopPointToDXGISurface(s.rotation, desktopWidth, desktopHeight, desktopX, desktopY)
			src := uintptr(mapped.PData) + uintptr(srcPoint.Y*rowPitch+srcPoint.X*4)
			dstIndex := col * 4
			b := *(*uint8)(unsafe.Pointer(src))
			g := *(*uint8)(unsafe.Pointer(src + 1))
			r := *(*uint8)(unsafe.Pointer(src + 2))
			dst[dstIndex], dst[dstIndex+1], dst[dstIndex+2], dst[dstIndex+3] = r, g, b, 255
		}
	}

	return img, nil
}

func createDXGIFactory1() (*dxgiFactory1, error) {
	addr, err := procAddress(procCreateDXGIFactory1)
	if err != nil {
		return nil, backendUnavailableError(cap.CaptureBackendDXGI, "CreateDXGIFactory1 is unavailable: %v", err)
	}

	var factory *dxgiFactory1
	hr, _, _ := syscall.SyscallN(
		addr,
		uintptr(unsafe.Pointer(&iidIDXGIFactory1)),
		uintptr(unsafe.Pointer(&factory)),
	)
	if hresultFailed(hr) {
		return nil, wrapDXGIInitError("CreateDXGIFactory1", hr)
	}
	return factory, nil
}

func findDXGIOutput(factory *dxgiFactory1, monitor windows.Handle) (*dxgiAdapter1, *dxgiOutput, dxgiOutputDesc, error) {
	for adapterIndex := uint32(0); ; adapterIndex++ {
		var adapter *dxgiAdapter1
		hr := dxgiFactoryEnumAdapters1(factory, adapterIndex, &adapter)
		if hresultCode(hr) == dxgiErrorNotFound {
			break
		}
		if hresultFailed(hr) {
			return nil, nil, dxgiOutputDesc{}, fmt.Errorf("IDXGIFactory1::EnumAdapters1 failed: %s", formatHRESULT(hr))
		}

		for outputIndex := uint32(0); ; outputIndex++ {
			var output *dxgiOutput
			hr = dxgiAdapterEnumOutputs(adapter, outputIndex, &output)
			if hresultCode(hr) == dxgiErrorNotFound {
				break
			}
			if hresultFailed(hr) {
				comRelease(adapter)
				return nil, nil, dxgiOutputDesc{}, fmt.Errorf("IDXGIAdapter::EnumOutputs failed: %s", formatHRESULT(hr))
			}

			desc, err := dxgiOutputDescription(output)
			if err != nil {
				comRelease(output)
				comRelease(adapter)
				return nil, nil, dxgiOutputDesc{}, err
			}
			if desc.Monitor == monitor {
				return adapter, output, desc, nil
			}
			comRelease(output)
		}

		comRelease(adapter)
	}

	return nil, nil, dxgiOutputDesc{}, backendUnavailableError(cap.CaptureBackendDXGI, "no DXGI output matched display %d", uintptr(monitor))
}

func queryDXGIOutput1(output *dxgiOutput) (*dxgiOutput1, error) {
	var output1 *dxgiOutput1
	hr := comQueryInterface(unsafe.Pointer(output), &iidIDXGIOutput1, unsafe.Pointer(&output1))
	if hresultFailed(hr) {
		return nil, wrapDXGIInitError("IDXGIOutput::QueryInterface(IDXGIOutput1)", hr)
	}
	return output1, nil
}

func createD3D11Device(adapter *dxgiAdapter1) (*d3d11Device, *d3d11DeviceContext, error) {
	addr, err := procAddress(procD3D11CreateDevice)
	if err != nil {
		return nil, nil, backendUnavailableError(cap.CaptureBackendDXGI, "D3D11CreateDevice is unavailable: %v", err)
	}

	var device *d3d11Device
	var context *d3d11DeviceContext
	hr, _, _ := syscall.SyscallN(
		addr,
		uintptr(unsafe.Pointer(adapter)),
		uintptr(d3dDriverTypeUnknown),
		0,
		uintptr(d3d11CreateDeviceBGRASupport),
		0,
		0,
		uintptr(d3d11SDKVersion),
		uintptr(unsafe.Pointer(&device)),
		0,
		uintptr(unsafe.Pointer(&context)),
	)
	if hresultFailed(hr) {
		return nil, nil, wrapDXGIInitError("D3D11CreateDevice", hr)
	}
	return device, context, nil
}

func duplicateOutput(output *dxgiOutput1, device *d3d11Device) (*dxgiOutputDuplication, error) {
	var duplication *dxgiOutputDuplication
	hr := comCall(
		unsafe.Pointer(output),
		dxgiOutput1DuplicateOutputMethod,
		uintptr(unsafe.Pointer(device)),
		uintptr(unsafe.Pointer(&duplication)),
	)
	if hresultFailed(hr) {
		return nil, wrapDXGIInitError("IDXGIOutput1::DuplicateOutput", hr)
	}
	return duplication, nil
}

func dxgiFactoryEnumAdapters1(factory *dxgiFactory1, index uint32, adapter **dxgiAdapter1) uintptr {
	return comCall(unsafe.Pointer(factory), dxgiFactory1EnumAdapters1Method, uintptr(index), uintptr(unsafe.Pointer(adapter)))
}

func dxgiAdapterEnumOutputs(adapter *dxgiAdapter1, index uint32, output **dxgiOutput) uintptr {
	return comCall(unsafe.Pointer(adapter), dxgiAdapterEnumOutputsMethod, uintptr(index), uintptr(unsafe.Pointer(output)))
}

func dxgiOutputDescription(output *dxgiOutput) (dxgiOutputDesc, error) {
	var desc dxgiOutputDesc
	hr := comCall(unsafe.Pointer(output), dxgiOutputGetDescMethod, uintptr(unsafe.Pointer(&desc)))
	if hresultFailed(hr) {
		return dxgiOutputDesc{}, fmt.Errorf("IDXGIOutput::GetDesc failed: %s", formatHRESULT(hr))
	}
	if desc.Monitor == 0 {
		return dxgiOutputDesc{}, errors.New("IDXGIOutput::GetDesc returned an invalid monitor handle")
	}
	return desc, nil
}

func dxgiOutputDuplicationAcquireFrame(duplication *dxgiOutputDuplication, timeoutMillis uint32, frameInfo *dxgiOutduplFrameInfo, resource **dxgiResource) uintptr {
	return comCall(
		unsafe.Pointer(duplication),
		dxgiOutputDuplicationAcquireNextFrame,
		uintptr(timeoutMillis),
		uintptr(unsafe.Pointer(frameInfo)),
		uintptr(unsafe.Pointer(resource)),
	)
}

func dxgiOutputDuplicationReleaseCurrentFrame(duplication *dxgiOutputDuplication) uintptr {
	return comCall(unsafe.Pointer(duplication), dxgiOutputDuplicationReleaseFrame)
}

func queryD3D11Texture(resource *dxgiResource) (*d3d11Texture2D, error) {
	var texture *d3d11Texture2D
	hr := comQueryInterface(unsafe.Pointer(resource), &iidID3D11Texture2D, unsafe.Pointer(&texture))
	if hresultFailed(hr) {
		return nil, fmt.Errorf("IDXGIResource::QueryInterface(ID3D11Texture2D) failed: %s", formatHRESULT(hr))
	}
	return texture, nil
}

func d3d11Texture2DDescription(texture *d3d11Texture2D) d3d11Texture2DDesc {
	var desc d3d11Texture2DDesc
	comCall(unsafe.Pointer(texture), d3d11Texture2DGetDescMethod, uintptr(unsafe.Pointer(&desc)))
	return desc
}

func d3d11DeviceCreateTexture2D(device *d3d11Device, desc *d3d11Texture2DDesc) (*d3d11Texture2D, error) {
	var texture *d3d11Texture2D
	hr := comCall(
		unsafe.Pointer(device),
		d3d11DeviceCreateTexture2DMethod,
		uintptr(unsafe.Pointer(desc)),
		0,
		uintptr(unsafe.Pointer(&texture)),
	)
	if hresultFailed(hr) {
		return nil, fmt.Errorf("ID3D11Device::CreateTexture2D failed: %s", formatHRESULT(hr))
	}
	return texture, nil
}

func d3d11DeviceContextCopyResource(context *d3d11DeviceContext, dst, src *d3d11Texture2D) {
	comCall(
		unsafe.Pointer(context),
		d3d11DeviceContextCopyResourceMethod,
		uintptr(unsafe.Pointer(dst)),
		uintptr(unsafe.Pointer(src)),
	)
}

func d3d11DeviceContextMap(context *d3d11DeviceContext, resource *d3d11Texture2D, mapped *d3d11MappedSubresource) uintptr {
	return comCall(
		unsafe.Pointer(context),
		d3d11DeviceContextMapMethod,
		uintptr(unsafe.Pointer(resource)),
		0,
		uintptr(d3d11MapRead),
		0,
		uintptr(unsafe.Pointer(mapped)),
	)
}

func d3d11DeviceContextUnmap(context *d3d11DeviceContext, resource *d3d11Texture2D) {
	comCall(
		unsafe.Pointer(context),
		d3d11DeviceContextUnmapMethod,
		uintptr(unsafe.Pointer(resource)),
		0,
	)
}

func comQueryInterface(obj unsafe.Pointer, iid *windows.GUID, out unsafe.Pointer) uintptr {
	return comCall(obj, 0, uintptr(unsafe.Pointer(iid)), uintptr(out))
}

func comRelease(obj any) {
	switch v := obj.(type) {
	case nil:
		return
	case *dxgiFactory1:
		if v != nil {
			comCall(unsafe.Pointer(v), 2)
		}
	case *dxgiAdapter1:
		if v != nil {
			comCall(unsafe.Pointer(v), 2)
		}
	case *dxgiOutput:
		if v != nil {
			comCall(unsafe.Pointer(v), 2)
		}
	case *dxgiOutput1:
		if v != nil {
			comCall(unsafe.Pointer(v), 2)
		}
	case *dxgiOutputDuplication:
		if v != nil {
			comCall(unsafe.Pointer(v), 2)
		}
	case *dxgiResource:
		if v != nil {
			comCall(unsafe.Pointer(v), 2)
		}
	case *d3d11Device:
		if v != nil {
			comCall(unsafe.Pointer(v), 2)
		}
	case *d3d11DeviceContext:
		if v != nil {
			comCall(unsafe.Pointer(v), 2)
		}
	case *d3d11Texture2D:
		if v != nil {
			comCall(unsafe.Pointer(v), 2)
		}
	}
}

func comCall(obj unsafe.Pointer, methodIndex uintptr, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(obj)
	method := *(*uintptr)(unsafe.Pointer(vtbl + methodIndex*unsafe.Sizeof(uintptr(0))))
	callArgs := make([]uintptr, 1+len(args))
	callArgs[0] = uintptr(obj)
	copy(callArgs[1:], args)
	r1, _, _ := syscall.SyscallN(method, callArgs...)
	return r1
}

func procAddress(proc *windows.LazyProc) (uintptr, error) {
	if err := proc.Find(); err != nil {
		return 0, err
	}
	return proc.Addr(), nil
}

func wrapDXGIInitError(operation string, hr uintptr) error {
	if isDXGIBackendUnavailableHRESULT(hresultCode(hr)) {
		return backendUnavailableError(cap.CaptureBackendDXGI, "%s failed: %s", operation, formatHRESULT(hr))
	}
	return fmt.Errorf("%s failed: %s", operation, formatHRESULT(hr))
}

func isDXGIBackendUnavailableHRESULT(code uint32) bool {
	switch code {
	case dxgiErrorUnsupported, dxgiErrorNotCurrentlyAvailable, dxgiErrorSessionDisconnected, eAccessDenied:
		return true
	default:
		return false
	}
}

func hresultFailed(hr uintptr) bool {
	return int32(hr) < 0
}

func hresultCode(hr uintptr) uint32 {
	return uint32(hr)
}

func formatHRESULT(hr uintptr) string {
	return fmt.Sprintf("HRESULT 0x%08x", hresultCode(hr))
}
