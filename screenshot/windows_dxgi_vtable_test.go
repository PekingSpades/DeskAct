//go:build windows

package screenshot

import "testing"

func TestD3D11DeviceContextMethodIndices(t *testing.T) {
	if d3d11DeviceContextMapMethod != 14 {
		t.Fatalf("expected ID3D11DeviceContext::Map to use vtable index 14, got %d", d3d11DeviceContextMapMethod)
	}
	if d3d11DeviceContextUnmapMethod != 15 {
		t.Fatalf("expected ID3D11DeviceContext::Unmap to use vtable index 15, got %d", d3d11DeviceContextUnmapMethod)
	}
	if d3d11DeviceContextCopyResourceMethod != 47 {
		t.Fatalf("expected ID3D11DeviceContext::CopyResource to use vtable index 47, got %d", d3d11DeviceContextCopyResourceMethod)
	}
}

func TestDXGIVtableMethodIndices(t *testing.T) {
	if dxgiFactory1EnumAdapters1Method != 12 {
		t.Fatalf("expected IDXGIFactory1::EnumAdapters1 to use vtable index 12, got %d", dxgiFactory1EnumAdapters1Method)
	}
	if dxgiAdapterEnumOutputsMethod != 7 {
		t.Fatalf("expected IDXGIAdapter::EnumOutputs to use vtable index 7, got %d", dxgiAdapterEnumOutputsMethod)
	}
	if dxgiOutputGetDescMethod != 7 {
		t.Fatalf("expected IDXGIOutput::GetDesc to use vtable index 7, got %d", dxgiOutputGetDescMethod)
	}
	if dxgiOutput1DuplicateOutputMethod != 22 {
		t.Fatalf("expected IDXGIOutput1::DuplicateOutput to use vtable index 22, got %d", dxgiOutput1DuplicateOutputMethod)
	}
	if dxgiOutputDuplicationAcquireNextFrame != 8 {
		t.Fatalf("expected IDXGIOutputDuplication::AcquireNextFrame to use vtable index 8, got %d", dxgiOutputDuplicationAcquireNextFrame)
	}
	if dxgiOutputDuplicationReleaseFrame != 14 {
		t.Fatalf("expected IDXGIOutputDuplication::ReleaseFrame to use vtable index 14, got %d", dxgiOutputDuplicationReleaseFrame)
	}
	if d3d11DeviceCreateTexture2DMethod != 5 {
		t.Fatalf("expected ID3D11Device::CreateTexture2D to use vtable index 5, got %d", d3d11DeviceCreateTexture2DMethod)
	}
	if d3d11Texture2DGetDescMethod != 10 {
		t.Fatalf("expected ID3D11Texture2D::GetDesc to use vtable index 10, got %d", d3d11Texture2DGetDescMethod)
	}
}
