// Copyright 2023 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package directx

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type _DXGI_ALPHA_MODE uint32

const (
	_DXGI_ALPHA_MODE_UNSPECIFIED   _DXGI_ALPHA_MODE = 0
	_DXGI_ALPHA_MODE_PREMULTIPLIED _DXGI_ALPHA_MODE = 1
	_DXGI_ALPHA_MODE_STRAIGHT      _DXGI_ALPHA_MODE = 2
	_DXGI_ALPHA_MODE_IGNORE        _DXGI_ALPHA_MODE = 3
	_DXGI_ALPHA_MODE_FORCE_DWORD   _DXGI_ALPHA_MODE = 0xffffffff
)

type _DXGI_COLOR_SPACE_TYPE int32

type _DXGI_FEATURE int32

const (
	_DXGI_FEATURE_PRESENT_ALLOW_TEARING _DXGI_FEATURE = 0
)

type _DXGI_FORMAT int32

const (
	_DXGI_FORMAT_UNKNOWN            _DXGI_FORMAT = 0
	_DXGI_FORMAT_R32G32B32A32_FLOAT _DXGI_FORMAT = 2
	_DXGI_FORMAT_R32G32_FLOAT       _DXGI_FORMAT = 16
	_DXGI_FORMAT_R8G8B8A8_UNORM     _DXGI_FORMAT = 28
	_DXGI_FORMAT_R32_UINT           _DXGI_FORMAT = 42
	_DXGI_FORMAT_D24_UNORM_S8_UINT  _DXGI_FORMAT = 45
	_DXGI_FORMAT_B8G8R8A8_UNORM     _DXGI_FORMAT = 87
)

type _DXGI_MODE_SCANLINE_ORDER int32

type _DXGI_MODE_SCALING int32

type _DXGI_PRESENT uint32

const (
	_DXGI_PRESENT_TEST          _DXGI_PRESENT = 0x00000001
	_DXGI_PRESENT_ALLOW_TEARING _DXGI_PRESENT = 0x00000200
)

type _DXGI_SCALING int32

type _DXGI_SWAP_CHAIN_FLAG int32

const (
	_DXGI_SWAP_CHAIN_FLAG_ALLOW_TEARING _DXGI_SWAP_CHAIN_FLAG = 2048
)

type _DXGI_SWAP_EFFECT int32

const (
	_DXGI_SWAP_EFFECT_DISCARD         _DXGI_SWAP_EFFECT = 0
	_DXGI_SWAP_EFFECT_SEQUENTIAL      _DXGI_SWAP_EFFECT = 1
	_DXGI_SWAP_EFFECT_FLIP_SEQUENTIAL _DXGI_SWAP_EFFECT = 3
	_DXGI_SWAP_EFFECT_FLIP_DISCARD    _DXGI_SWAP_EFFECT = 4
)

type _DXGI_USAGE uint32

const (
	_DXGI_USAGE_RENDER_TARGET_OUTPUT _DXGI_USAGE = 1 << (1 + 4)
)

const (
	_DXGI_ADAPTER_FLAG_SOFTWARE = 2

	_DXGI_CREATE_FACTORY_DEBUG = 0x01

	_DXGI_ERROR_NOT_FOUND = handleError(0x887A0002)

	_DXGI_MWA_NO_ALT_ENTER      = 0x2
	_DXGI_MWA_NO_WINDOW_CHANGES = 0x1
)

var (
	_IID_IDXGIAdapter1   = windows.GUID{Data1: 0x29038f61, Data2: 0x3839, Data3: 0x4626, Data4: [...]byte{0x91, 0xfd, 0x08, 0x68, 0x79, 0x01, 0x1a, 0x05}}
	_IID_IDXGIDevice     = windows.GUID{Data1: 0x54ec77fa, Data2: 0x1377, Data3: 0x44e6, Data4: [...]byte{0x8c, 0x32, 0x88, 0xfd, 0x5f, 0x44, 0xc8, 0x4c}}
	_IID_IDXGIFactory    = windows.GUID{Data1: 0x7b7166ec, Data2: 0x21c7, Data3: 0x44ae, Data4: [...]byte{0xb2, 0x1a, 0xc9, 0xae, 0x32, 0x1a, 0xe3, 0x69}}
	_IID_IDXGIFactory4   = windows.GUID{Data1: 0x1bc6ea02, Data2: 0xef36, Data3: 0x464f, Data4: [...]byte{0xbf, 0x0c, 0x21, 0xca, 0x39, 0xe5, 0x16, 0x8a}}
	_IID_IDXGIFactory5   = windows.GUID{Data1: 0x7632e1f5, Data2: 0xee65, Data3: 0x4dca, Data4: [...]byte{0x87, 0xfd, 0x84, 0xcd, 0x75, 0xf8, 0x83, 0x8d}}
	_IID_IDXGISwapChain4 = windows.GUID{Data1: 0x3d585d5a, Data2: 0xbd4a, Data3: 0x489e, Data4: [...]byte{0xb1, 0xf4, 0x3d, 0xbc, 0xb6, 0x45, 0x2f, 0xfb}}
)

var (
	dxgi = windows.NewLazySystemDLL("dxgi.dll")

	procCreateDXGIFactory = dxgi.NewProc("CreateDXGIFactory")
)

func _CreateDXGIFactory() (*_IDXGIFactory, error) {
	var factory *_IDXGIFactory
	r, _, _ := procCreateDXGIFactory.Call(uintptr(unsafe.Pointer(&_IID_IDXGIFactory)), uintptr(unsafe.Pointer(&factory)))
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: CreateDXGIFactory failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return factory, nil
}

type _DXGI_ADAPTER_DESC1 struct {
	Description           [128]uint16
	VendorId              uint32
	DeviceId              uint32
	SubSysId              uint32
	Revision              uint32
	DedicatedVideoMemory  uint
	DedicatedSystemMemory uint
	SharedSystemMemory    uint
	AdapterLuid           _LUID
	Flags                 uint32
}

type _DXGI_MODE_DESC struct {
	Width            uint32
	Height           uint32
	RefreshRate      _DXGI_RATIONAL
	Format           _DXGI_FORMAT
	ScanlineOrdering _DXGI_MODE_SCANLINE_ORDER
	Scaling          _DXGI_MODE_SCALING
}

type _DXGI_RATIONAL struct {
	Numerator   uint32
	Denominator uint32
}

type _DXGI_SWAP_CHAIN_FULLSCREEN_DESC struct {
	RefreshRate      _DXGI_RATIONAL
	ScanlineOrdering _DXGI_MODE_SCANLINE_ORDER
	Scaling          _DXGI_MODE_SCALING
	Windowed         _BOOL
}

type _DXGI_SAMPLE_DESC struct {
	Count   uint32
	Quality uint32
}

type _DXGI_SWAP_CHAIN_DESC struct {
	BufferDesc   _DXGI_MODE_DESC
	SampleDesc   _DXGI_SAMPLE_DESC
	BufferUsage  _DXGI_USAGE
	BufferCount  uint32
	OutputWindow windows.HWND
	Windowed     _BOOL
	SwapEffect   _DXGI_SWAP_EFFECT
	Flags        uint32
}

type _LUID struct {
	LowPart  uint32
	HighPart int32
}

type _IDXGIAdapter struct {
	vtbl *_IDXGIAdapter1_Vtbl
}

type _IDXGIAdapter_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr
	GetPrivateData          uintptr
	GetParent               uintptr
	EnumOutputs             uintptr
	GetDesc                 uintptr
	CheckInterfaceSupport   uintptr
}

func (i *_IDXGIAdapter) EnumOutputs(output uint32) (*_IDXGIOutput, error) {
	var pOutput *_IDXGIOutput
	r, _, _ := syscall.Syscall(i.vtbl.EnumOutputs, 3, uintptr(unsafe.Pointer(i)), uintptr(output), uintptr(unsafe.Pointer(&pOutput)))
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDXGIAdapter::EnumOutputs failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return pOutput, nil
}

func (i *_IDXGIAdapter) GetParent(riid *windows.GUID) (unsafe.Pointer, error) {
	var v unsafe.Pointer
	r, _, _ := syscall.Syscall(i.vtbl.GetParent, 3, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(riid)), uintptr(unsafe.Pointer(&v)))
	runtime.KeepAlive(riid)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDXGIAdapter::GetParent failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return v, nil
}

func (i *_IDXGIAdapter) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _IDXGIAdapter1 struct {
	vtbl *_IDXGIAdapter1_Vtbl
}

type _IDXGIAdapter1_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr
	GetPrivateData          uintptr
	GetParent               uintptr
	EnumOutputs             uintptr
	GetDesc                 uintptr
	CheckInterfaceSupport   uintptr
	GetDesc1                uintptr
}

func (i *_IDXGIAdapter1) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

func (i *_IDXGIAdapter1) GetDesc1() (*_DXGI_ADAPTER_DESC1, error) {
	var desc _DXGI_ADAPTER_DESC1
	r, _, _ := syscall.Syscall(i.vtbl.GetDesc1, 2, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(&desc)), 0)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDXGIAdapter1::GetDesc1 failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return &desc, nil
}

type _IDXGIDevice struct {
	vtbl *_IDXGIDevice_Vtbl
}

type _IDXGIDevice_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr
	GetPrivateData          uintptr
	GetParent               uintptr
	GetAdapter              uintptr
	CreateSurface           uintptr
	QueryResourceResidency  uintptr
	SetGPUThreadPriority    uintptr
	GetGPUThreadPriority    uintptr
}

func (i *_IDXGIDevice) GetAdapter() (*_IDXGIAdapter, error) {
	var adapter *_IDXGIAdapter
	r, _, _ := syscall.Syscall(i.vtbl.GetAdapter, 2, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(&adapter)), 0)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDXGIDevice::GetAdapter failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return adapter, nil
}

func (i *_IDXGIDevice) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _IDXGIFactory struct {
	vtbl *_IDXGIFactory_Vtbl
}

type _IDXGIFactory_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr
	GetPrivateData          uintptr
	GetParent               uintptr
	EnumAdapters            uintptr
	MakeWindowAssociation   uintptr
	GetWindowAssociation    uintptr
	CreateSwapChain         uintptr
	CreateSoftwareAdapter   uintptr
}

func (i *_IDXGIFactory) CreateSwapChain(pDevice unsafe.Pointer, pDesc *_DXGI_SWAP_CHAIN_DESC) (*_IDXGISwapChain, error) {
	var swapChain *_IDXGISwapChain
	r, _, _ := syscall.Syscall6(i.vtbl.CreateSwapChain, 4, uintptr(unsafe.Pointer(i)),
		uintptr(pDevice), uintptr(unsafe.Pointer(pDesc)), uintptr(unsafe.Pointer(&swapChain)),
		0, 0)
	runtime.KeepAlive(pDevice)
	runtime.KeepAlive(pDesc)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDXGIFactory::CreateSwapChain failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return swapChain, nil
}

func (i *_IDXGIFactory) MakeWindowAssociation(windowHandle windows.HWND, flags uint32) error {
	r, _, _ := syscall.Syscall(i.vtbl.MakeWindowAssociation, 3, uintptr(unsafe.Pointer(i)), uintptr(windowHandle), uintptr(flags))
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("directx: IDXGIFactory::MakeWIndowAssociation failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (i *_IDXGIFactory) QueryInterface(riid *windows.GUID) (unsafe.Pointer, error) {
	var v unsafe.Pointer
	r, _, _ := syscall.Syscall(i.vtbl.QueryInterface, 3, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(riid)), uintptr(unsafe.Pointer(&v)))
	runtime.KeepAlive(riid)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDXGIFactory::QueryInterface failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return v, nil
}

func (i *_IDXGIFactory) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _IDXGIFactory4 struct {
	vtbl *_IDXGIFactory4_Vtbl
}

type _IDXGIFactory4_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetPrivateData                uintptr
	SetPrivateDataInterface       uintptr
	GetPrivateData                uintptr
	GetParent                     uintptr
	EnumAdapters                  uintptr
	MakeWindowAssociation         uintptr
	GetWindowAssociation          uintptr
	CreateSwapChain               uintptr
	CreateSoftwareAdapter         uintptr
	EnumAdapters1                 uintptr
	IsCurrent                     uintptr
	IsWindowedStereoEnabled       uintptr
	CreateSwapChainForHwnd        uintptr
	CreateSwapChainForCoreWindow  uintptr
	GetSharedResourceAdapterLuid  uintptr
	RegisterStereoStatusWindow    uintptr
	RegisterStereoStatusEvent     uintptr
	UnregisterStereoStatus        uintptr
	RegisterOcclusionStatusWindow uintptr
	RegisterOcclusionStatusEvent  uintptr
	UnregisterOcclusionStatus     uintptr
	CreateSwapChainForComposition uintptr
	GetCreationFlags              uintptr
	EnumAdapterByLuid             uintptr
	EnumWarpAdapter               uintptr
}

func (i *_IDXGIFactory4) EnumAdapters1(adapter uint32) (*_IDXGIAdapter1, error) {
	var ptr *_IDXGIAdapter1
	r, _, _ := syscall.Syscall(i.vtbl.EnumAdapters1, 3, uintptr(unsafe.Pointer(i)), uintptr(adapter), uintptr(unsafe.Pointer(&ptr)))
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDXGIFactory4::EnumAdapters1 failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return ptr, nil
}

func (i *_IDXGIFactory4) EnumWarpAdapter() (*_IDXGIAdapter1, error) {
	var ptr *_IDXGIAdapter1
	r, _, _ := syscall.Syscall(i.vtbl.EnumWarpAdapter, 3, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(&_IID_IDXGIAdapter1)), uintptr(unsafe.Pointer(&ptr)))
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDXGIFactory4::EnumWarpAdapter failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return ptr, nil
}

func (i *_IDXGIFactory4) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _IDXGIFactory5 struct {
	vtbl *_IDXGIFactory5_Vtbl
}

type _IDXGIFactory5_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetPrivateData                uintptr
	SetPrivateDataInterface       uintptr
	GetPrivateData                uintptr
	GetParent                     uintptr
	EnumAdapters                  uintptr
	MakeWindowAssociation         uintptr
	GetWindowAssociation          uintptr
	CreateSwapChain               uintptr
	CreateSoftwareAdapter         uintptr
	EnumAdapters1                 uintptr
	IsCurrent                     uintptr
	IsWindowedStereoEnabled       uintptr
	CreateSwapChainForHwnd        uintptr
	CreateSwapChainForCoreWindow  uintptr
	GetSharedResourceAdapterLuid  uintptr
	RegisterStereoStatusWindow    uintptr
	RegisterStereoStatusEvent     uintptr
	UnregisterStereoStatus        uintptr
	RegisterOcclusionStatusWindow uintptr
	RegisterOcclusionStatusEvent  uintptr
	UnregisterOcclusionStatus     uintptr
	CreateSwapChainForComposition uintptr
	GetCreationFlags              uintptr
	EnumAdapterByLuid             uintptr
	EnumWarpAdapter               uintptr
	CheckFeatureSupport           uintptr
}

func (i *_IDXGIFactory5) CheckFeatureSupport(feature _DXGI_FEATURE, pFeatureSupportData unsafe.Pointer, featureSupportDataSize uint32) error {
	r, _, _ := syscall.Syscall6(i.vtbl.CheckFeatureSupport, 4, uintptr(unsafe.Pointer(i)),
		uintptr(feature), uintptr(pFeatureSupportData), uintptr(featureSupportDataSize),
		0, 0)
	runtime.KeepAlive(pFeatureSupportData)
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("directx: IDXGIFactory5::CheckFeatureSupport failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (i *_IDXGIFactory5) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _IDXGIOutput struct {
	vtbl *_IDXGIOutput_Vtbl
}

type _IDXGIOutput_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetPrivateData              uintptr
	SetPrivateDataInterface     uintptr
	GetPrivateData              uintptr
	GetParent                   uintptr
	GetDesc                     uintptr
	GetDisplayModeList          uintptr
	FindClosestMatchingMode     uintptr
	WaitForVBlank               uintptr
	TakeOwnership               uintptr
	ReleaseOwnership            uintptr
	GetGammaControlCapabilities uintptr
	SetGammaControl             uintptr
	GetGammaControl             uintptr
	SetDisplaySurface           uintptr
	GetDisplaySurfaceData       uintptr
	GetFrameStatistics          uintptr
}

func (i *_IDXGIOutput) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _IDXGISwapChain struct {
	vtbl *_IDXGISwapChain_Vtbl
}

type _IDXGISwapChain_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr
	GetPrivateData          uintptr
	GetParent               uintptr
	GetDevice               uintptr
	Present                 uintptr
	GetBuffer               uintptr
	SetFullscreenState      uintptr
	GetFullscreenState      uintptr
	GetDesc                 uintptr
	ResizeBuffers           uintptr
	ResizeTarget            uintptr
	GetContainingOutput     uintptr
	GetFrameStatistics      uintptr
	GetLastPresentCount     uintptr
}

func (i *_IDXGISwapChain) GetBuffer(buffer uint32, riid *windows.GUID) (unsafe.Pointer, error) {
	var resource unsafe.Pointer
	r, _, _ := syscall.Syscall6(i.vtbl.GetBuffer, 4, uintptr(unsafe.Pointer(i)),
		uintptr(buffer), uintptr(unsafe.Pointer(riid)), uintptr(unsafe.Pointer(&resource)),
		0, 0)
	runtime.KeepAlive(riid)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDXGISwapChain::GetBuffer failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return resource, nil
}

func (i *_IDXGISwapChain) ResizeBuffers(bufferCount uint32, width uint32, height uint32, newFormat _DXGI_FORMAT, swapChainFlags uint32) error {
	r, _, _ := syscall.Syscall6(i.vtbl.ResizeBuffers, 6,
		uintptr(unsafe.Pointer(i)), uintptr(bufferCount), uintptr(width),
		uintptr(height), uintptr(newFormat), uintptr(swapChainFlags))
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("directx: IDXGISwapChain::ResizeBuffers failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (i *_IDXGISwapChain) Present(syncInterval uint32, flags uint32) (occluded bool, err error) {
	r, _, _ := syscall.Syscall(i.vtbl.Present, 3, uintptr(unsafe.Pointer(i)), uintptr(syncInterval), uintptr(flags))
	if uint32(r) != uint32(windows.S_OK) {
		// During a screen lock, Present fails (#2179).
		if uint32(r) == uint32(windows.DXGI_STATUS_OCCLUDED) {
			return true, nil
		}
		return false, fmt.Errorf("directx: IDXGISwapChain::Present failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return false, nil
}

func (i *_IDXGISwapChain) QueryInterface(riid *windows.GUID) (unsafe.Pointer, error) {
	var v unsafe.Pointer
	r, _, _ := syscall.Syscall(i.vtbl.QueryInterface, 3, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(riid)), uintptr(unsafe.Pointer(&v)))
	runtime.KeepAlive(riid)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDXGISwapChain::QueryInterface failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return v, nil
}

func (i *_IDXGISwapChain) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _IDXGISwapChain4 struct {
	vtbl *_IDXGISwapChain4_Vtbl
}

type _IDXGISwapChain4_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetPrivateData           uintptr
	SetPrivateDataInterface  uintptr
	GetPrivateData           uintptr
	GetParent                uintptr
	GetDevice                uintptr
	Present                  uintptr
	GetBuffer                uintptr
	SetFullscreenState       uintptr
	GetFullscreenState       uintptr
	GetDesc                  uintptr
	ResizeBuffers            uintptr
	ResizeTarget             uintptr
	GetContainingOutput      uintptr
	GetFrameStatistics       uintptr
	GetLastPresentCount      uintptr
	GetDesc1                 uintptr
	GetFullscreenDesc        uintptr
	GetHwnd                  uintptr
	GetCoreWindow            uintptr
	Present1                 uintptr
	IsTemporaryMonoSupported uintptr
	GetRestrictToOutput      uintptr
	SetBackgroundColor       uintptr
	GetBackgroundColor       uintptr
	SetRotation              uintptr
	GetRotation              uintptr

	SetSourceSize                 uintptr
	GetSourceSize                 uintptr
	SetMaximumFrameLatency        uintptr
	GetMaximumFrameLatency        uintptr
	GetFrameLatencyWaitableObject uintptr
	SetMatrixTransform            uintptr
	GetMatrixTransform            uintptr
	GetCurrentBackBufferIndex     uintptr
	CheckColorSpaceSupport        uintptr
	SetColorSpace1                uintptr
	ResizeBuffers1                uintptr
	SetHDRMetaData                uintptr
}

func (i *_IDXGISwapChain4) GetCurrentBackBufferIndex() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.GetCurrentBackBufferIndex, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

func (i *_IDXGISwapChain4) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}
