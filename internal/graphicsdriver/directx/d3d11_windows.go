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
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Reference:
// * https://github.com/wine-mirror/wine/blob/master/include/d3d11.idl

const (
	_D3D11_APPEND_ALIGNED_ELEMENT     = 0xffffffff
	_D3D11_DEFAULT_STENCIL_READ_MASK  = 0xff
	_D3D11_DEFAULT_STENCIL_WRITE_MASK = 0xff
	_D3D11_SDK_VERSION                = 7
)

type _D3D11_BIND_FLAG int32

const (
	_D3D11_BIND_VERTEX_BUFFER    _D3D11_BIND_FLAG = 0x1
	_D3D11_BIND_INDEX_BUFFER     _D3D11_BIND_FLAG = 0x2
	_D3D11_BIND_CONSTANT_BUFFER  _D3D11_BIND_FLAG = 0x4
	_D3D11_BIND_SHADER_RESOURCE  _D3D11_BIND_FLAG = 0x8
	_D3D11_BIND_STREAM_OUTPUT    _D3D11_BIND_FLAG = 0x10
	_D3D11_BIND_RENDER_TARGET    _D3D11_BIND_FLAG = 0x20
	_D3D11_BIND_DEPTH_STENCIL    _D3D11_BIND_FLAG = 0x40
	_D3D11_BIND_UNORDERED_ACCESS _D3D11_BIND_FLAG = 0x80
	_D3D11_BIND_DECODER          _D3D11_BIND_FLAG = 0x200
	_D3D11_BIND_VIDEO_ENCODER    _D3D11_BIND_FLAG = 0x400
)

type _D3D11_BLEND int32

const (
	_D3D11_BLEND_ZERO             _D3D11_BLEND = 1
	_D3D11_BLEND_ONE              _D3D11_BLEND = 2
	_D3D11_BLEND_SRC_COLOR        _D3D11_BLEND = 3
	_D3D11_BLEND_INV_SRC_COLOR    _D3D11_BLEND = 4
	_D3D11_BLEND_SRC_ALPHA        _D3D11_BLEND = 5
	_D3D11_BLEND_INV_SRC_ALPHA    _D3D11_BLEND = 6
	_D3D11_BLEND_DEST_ALPHA       _D3D11_BLEND = 7
	_D3D11_BLEND_INV_DEST_ALPHA   _D3D11_BLEND = 8
	_D3D11_BLEND_DEST_COLOR       _D3D11_BLEND = 9
	_D3D11_BLEND_INV_DEST_COLOR   _D3D11_BLEND = 10
	_D3D11_BLEND_SRC_ALPHA_SAT    _D3D11_BLEND = 11
	_D3D11_BLEND_BLEND_FACTOR     _D3D11_BLEND = 14
	_D3D11_BLEND_INV_BLEND_FACTOR _D3D11_BLEND = 15
	_D3D11_BLEND_SRC1_COLOR       _D3D11_BLEND = 16
	_D3D11_BLEND_INV_SRC1_COLOR   _D3D11_BLEND = 17
	_D3D11_BLEND_SRC1_ALPHA       _D3D11_BLEND = 18
	_D3D11_BLEND_INV_SRC1_ALPHA   _D3D11_BLEND = 19
)

type _D3D11_BLEND_OP int32

const (
	_D3D11_BLEND_OP_ADD          _D3D11_BLEND_OP = 1
	_D3D11_BLEND_OP_SUBTRACT     _D3D11_BLEND_OP = 2
	_D3D11_BLEND_OP_REV_SUBTRACT _D3D11_BLEND_OP = 3
	_D3D11_BLEND_OP_MIN          _D3D11_BLEND_OP = 4
	_D3D11_BLEND_OP_MAX          _D3D11_BLEND_OP = 5
)

type _D3D11_CLEAR_FLAG int32

const (
	_D3D11_CLEAR_DEPTH   _D3D11_CLEAR_FLAG = 0x1
	_D3D11_CLEAR_STENCIL _D3D11_CLEAR_FLAG = 0x2
)

type _D3D11_COLOR_WRITE_ENABLE int32

const (
	_D3D11_COLOR_WRITE_ENABLE_RED   _D3D11_COLOR_WRITE_ENABLE = 1
	_D3D11_COLOR_WRITE_ENABLE_GREEN _D3D11_COLOR_WRITE_ENABLE = 2
	_D3D11_COLOR_WRITE_ENABLE_BLUE  _D3D11_COLOR_WRITE_ENABLE = 4
	_D3D11_COLOR_WRITE_ENABLE_ALPHA _D3D11_COLOR_WRITE_ENABLE = 8
	_D3D11_COLOR_WRITE_ENABLE_ALL   _D3D11_COLOR_WRITE_ENABLE = _D3D11_COLOR_WRITE_ENABLE_RED | _D3D11_COLOR_WRITE_ENABLE_GREEN | _D3D11_COLOR_WRITE_ENABLE_BLUE | _D3D11_COLOR_WRITE_ENABLE_ALPHA
)

type _D3D11_COMPARISON_FUNC int32

const (
	_D3D11_COMPARISON_NEVER         _D3D11_COMPARISON_FUNC = 1
	_D3D11_COMPARISON_LESS          _D3D11_COMPARISON_FUNC = 2
	_D3D11_COMPARISON_EQUAL         _D3D11_COMPARISON_FUNC = 3
	_D3D11_COMPARISON_LESS_EQUAL    _D3D11_COMPARISON_FUNC = 4
	_D3D11_COMPARISON_GREATER       _D3D11_COMPARISON_FUNC = 5
	_D3D11_COMPARISON_NOT_EQUAL     _D3D11_COMPARISON_FUNC = 6
	_D3D11_COMPARISON_GREATER_EQUAL _D3D11_COMPARISON_FUNC = 7
	_D3D11_COMPARISON_ALWAYS        _D3D11_COMPARISON_FUNC = 8
)

type _D3D11_CPU_ACCESS_FLAG int32

const (
	_D3D11_CPU_ACCESS_WRITE _D3D11_CPU_ACCESS_FLAG = 0x10000
	_D3D11_CPU_ACCESS_READ  _D3D11_CPU_ACCESS_FLAG = 0x20000
)

type _D3D11_CREATE_DEVICE_FLAG int32

const (
	_D3D11_CREATE_DEVICE_SINGLETHREADED                                _D3D11_CREATE_DEVICE_FLAG = 0x1
	_D3D11_CREATE_DEVICE_DEBUG                                         _D3D11_CREATE_DEVICE_FLAG = 0x2
	_D3D11_CREATE_DEVICE_SWITCH_TO_REF                                 _D3D11_CREATE_DEVICE_FLAG = 0x4
	_D3D11_CREATE_DEVICE_PREVENT_INTERNAL_THREADING_OPTIMIZATIONS      _D3D11_CREATE_DEVICE_FLAG = 0x8
	_D3D11_CREATE_DEVICE_BGRA_SUPPORT                                  _D3D11_CREATE_DEVICE_FLAG = 0x20
	_D3D11_CREATE_DEVICE_DEBUGGABLE                                    _D3D11_CREATE_DEVICE_FLAG = 0x40
	_D3D11_CREATE_DEVICE_PREVENT_ALTERING_LAYER_SETTINGS_FROM_REGISTRY _D3D11_CREATE_DEVICE_FLAG = 0x80
	_D3D11_CREATE_DEVICE_DISABLE_GPU_TIMEOUT                           _D3D11_CREATE_DEVICE_FLAG = 0x100
	_D3D11_CREATE_DEVICE_VIDEO_SUPPORT                                 _D3D11_CREATE_DEVICE_FLAG = 0x800
)

type _D3D11_CULL_MODE int32

const (
	_D3D11_CULL_NONE  _D3D11_CULL_MODE = 1
	_D3D11_CULL_FRONT _D3D11_CULL_MODE = 2
	_D3D11_CULL_BACK  _D3D11_CULL_MODE = 3
)

type _D3D11_DSV_FLAG int32

const (
	_D3D11_DSV_READ_ONLY_DEPTH   _D3D11_DSV_FLAG = 0x1
	_D3D11_DSV_READ_ONLY_STENCIL _D3D11_DSV_FLAG = 0x2
)

type _D3D11_DEPTH_WRITE_MASK int32

const (
	_D3D11_DEPTH_WRITE_MASK_ZERO _D3D11_DEPTH_WRITE_MASK = 0
	_D3D11_DEPTH_WRITE_MASK_ALL  _D3D11_DEPTH_WRITE_MASK = 1
)

type _D3D11_DSV_DIMENSION int32

const (
	_D3D11_DSV_DIMENSION_UNKNOWN          _D3D11_DSV_DIMENSION = 0
	_D3D11_DSV_DIMENSION_TEXTURE1D        _D3D11_DSV_DIMENSION = 1
	_D3D11_DSV_DIMENSION_TEXTURE1DARRAY   _D3D11_DSV_DIMENSION = 2
	_D3D11_DSV_DIMENSION_TEXTURE2D        _D3D11_DSV_DIMENSION = 3
	_D3D11_DSV_DIMENSION_TEXTURE2DARRAY   _D3D11_DSV_DIMENSION = 4
	_D3D11_DSV_DIMENSION_TEXTURE2DMS      _D3D11_DSV_DIMENSION = 5
	_D3D11_DSV_DIMENSION_TEXTURE2DMSARRAY _D3D11_DSV_DIMENSION = 6
)

type _D3D11_FILL_MODE int32

const (
	_D3D11_FILL_WIREFRAME _D3D11_FILL_MODE = 2
	_D3D11_FILL_SOLID     _D3D11_FILL_MODE = 3
)

type _D3D11_FILTER int32

const (
	_D3D11_FILTER_MIN_MAG_MIP_POINT _D3D11_FILTER = 0
)

type _D3D11_INPUT_CLASSIFICATION int32

const (
	_D3D11_INPUT_PER_VERTEX_DATA   _D3D11_INPUT_CLASSIFICATION = 0
	_D3D11_INPUT_PER_INSTANCE_DATA _D3D11_INPUT_CLASSIFICATION = 1
)

type _D3D11_MAP int32

const (
	_D3D11_MAP_READ               _D3D11_MAP = 1
	_D3D11_MAP_WRITE              _D3D11_MAP = 2
	_D3D11_MAP_READ_WRITE         _D3D11_MAP = 3
	_D3D11_MAP_WRITE_DISCARD      _D3D11_MAP = 4
	_D3D11_MAP_WRITE_NO_OVERWRITE _D3D11_MAP = 5
)

type _D3D11_MAP_FLAG int32

const (
	_D3D11_MAP_FLAG_DO_NOT_WAIT _D3D11_MAP_FLAG = 0x100000
)

type _D3D11_PRIMITIVE_TOPOLOGY int32

const (
	_D3D11_PRIMITIVE_TOPOLOGY_TRIANGLELIST _D3D11_PRIMITIVE_TOPOLOGY = 4
)

type _D3D11_RTV_DIMENSION int32

const (
	_D3D11_RTV_DIMENSION_UNKNOWN          _D3D11_RTV_DIMENSION = 0
	_D3D11_RTV_DIMENSION_BUFFER           _D3D11_RTV_DIMENSION = 1
	_D3D11_RTV_DIMENSION_TEXTURE1D        _D3D11_RTV_DIMENSION = 2
	_D3D11_RTV_DIMENSION_TEXTURE1DARRAY   _D3D11_RTV_DIMENSION = 3
	_D3D11_RTV_DIMENSION_TEXTURE2D        _D3D11_RTV_DIMENSION = 4
	_D3D11_RTV_DIMENSION_TEXTURE2DARRAY   _D3D11_RTV_DIMENSION = 5
	_D3D11_RTV_DIMENSION_TEXTURE2DMS      _D3D11_RTV_DIMENSION = 6
	_D3D11_RTV_DIMENSION_TEXTURE2DMSARRAY _D3D11_RTV_DIMENSION = 7
	_D3D11_RTV_DIMENSION_TEXTURE3D        _D3D11_RTV_DIMENSION = 8
)

type _D3D11_RESOURCE_MISC_FLAG int32

const (
	_D3D11_RESOURCE_MISC_GENERATE_MIPS                   _D3D11_RESOURCE_MISC_FLAG = 0x1
	_D3D11_RESOURCE_MISC_SHARED                          _D3D11_RESOURCE_MISC_FLAG = 0x2
	_D3D11_RESOURCE_MISC_TEXTURECUBE                     _D3D11_RESOURCE_MISC_FLAG = 0x4
	_D3D11_RESOURCE_MISC_DRAWINDIRECT_ARGS               _D3D11_RESOURCE_MISC_FLAG = 0x10
	_D3D11_RESOURCE_MISC_BUFFER_ALLOW_RAW_VIEWS          _D3D11_RESOURCE_MISC_FLAG = 0x20
	_D3D11_RESOURCE_MISC_BUFFER_STRUCTURED               _D3D11_RESOURCE_MISC_FLAG = 0x40
	_D3D11_RESOURCE_MISC_RESOURCE_CLAMP                  _D3D11_RESOURCE_MISC_FLAG = 0x80
	_D3D11_RESOURCE_MISC_SHARED_KEYEDMUTEX               _D3D11_RESOURCE_MISC_FLAG = 0x100
	_D3D11_RESOURCE_MISC_GDI_COMPATIBLE                  _D3D11_RESOURCE_MISC_FLAG = 0x200
	_D3D11_RESOURCE_MISC_SHARED_NTHANDLE                 _D3D11_RESOURCE_MISC_FLAG = 0x800
	_D3D11_RESOURCE_MISC_RESTRICTED_CONTENT              _D3D11_RESOURCE_MISC_FLAG = 0x1000
	_D3D11_RESOURCE_MISC_RESTRICT_SHARED_RESOURCE        _D3D11_RESOURCE_MISC_FLAG = 0x2000
	_D3D11_RESOURCE_MISC_RESTRICT_SHARED_RESOURCE_DRIVER _D3D11_RESOURCE_MISC_FLAG = 0x4000
	_D3D11_RESOURCE_MISC_GUARDED                         _D3D11_RESOURCE_MISC_FLAG = 0x8000
	_D3D11_RESOURCE_MISC_TILE_POOL                       _D3D11_RESOURCE_MISC_FLAG = 0x20000
	_D3D11_RESOURCE_MISC_TILED                           _D3D11_RESOURCE_MISC_FLAG = 0x40000
	_D3D11_RESOURCE_MISC_HW_PROTECTED                    _D3D11_RESOURCE_MISC_FLAG = 0x80000
	_D3D11_RESOURCE_MISC_SHARED_DISPLAYABLE              _D3D11_RESOURCE_MISC_FLAG = 0x100000
	_D3D11_RESOURCE_MISC_SHARED_EXCLUSIVE_WRITER         _D3D11_RESOURCE_MISC_FLAG = 0x200000
)

type _D3D11_SRV_DIMENSION int32

const (
	_D3D11_SRV_DIMENSION_UNKNOWN          _D3D11_SRV_DIMENSION = 0
	_D3D11_SRV_DIMENSION_BUFFER           _D3D11_SRV_DIMENSION = 1
	_D3D11_SRV_DIMENSION_TEXTURE1D        _D3D11_SRV_DIMENSION = 2
	_D3D11_SRV_DIMENSION_TEXTURE1DARRAY   _D3D11_SRV_DIMENSION = 3
	_D3D11_SRV_DIMENSION_TEXTURE2D        _D3D11_SRV_DIMENSION = 4
	_D3D11_SRV_DIMENSION_TEXTURE2DARRAY   _D3D11_SRV_DIMENSION = 5
	_D3D11_SRV_DIMENSION_TEXTURE2DMS      _D3D11_SRV_DIMENSION = 6
	_D3D11_SRV_DIMENSION_TEXTURE2DMSARRAY _D3D11_SRV_DIMENSION = 7
	_D3D11_SRV_DIMENSION_TEXTURE3D        _D3D11_SRV_DIMENSION = 8
	_D3D11_SRV_DIMENSION_TEXTURECUBE      _D3D11_SRV_DIMENSION = 9
	_D3D11_SRV_DIMENSION_TEXTURECUBEARRAY _D3D11_SRV_DIMENSION = 10
	_D3D11_SRV_DIMENSION_BUFFEREX         _D3D11_SRV_DIMENSION = 11
)

type _D3D11_STENCIL_OP int32

const (
	_D3D11_STENCIL_OP_KEEP     _D3D11_STENCIL_OP = 1
	_D3D11_STENCIL_OP_ZERO     _D3D11_STENCIL_OP = 2
	_D3D11_STENCIL_OP_REPLACE  _D3D11_STENCIL_OP = 3
	_D3D11_STENCIL_OP_INCR_SAT _D3D11_STENCIL_OP = 4
	_D3D11_STENCIL_OP_DECR_SAT _D3D11_STENCIL_OP = 5
	_D3D11_STENCIL_OP_INVERT   _D3D11_STENCIL_OP = 6
	_D3D11_STENCIL_OP_INCR     _D3D11_STENCIL_OP = 7
	_D3D11_STENCIL_OP_DECR     _D3D11_STENCIL_OP = 8
)

type _D3D11_TEXTURE_ADDRESS_MODE int32

const (
	_D3D11_TEXTURE_ADDRESS_WRAP        _D3D11_TEXTURE_ADDRESS_MODE = 1
	_D3D11_TEXTURE_ADDRESS_MIRROR      _D3D11_TEXTURE_ADDRESS_MODE = 2
	_D3D11_TEXTURE_ADDRESS_CLAMP       _D3D11_TEXTURE_ADDRESS_MODE = 3
	_D3D11_TEXTURE_ADDRESS_BORDER      _D3D11_TEXTURE_ADDRESS_MODE = 4
	_D3D11_TEXTURE_ADDRESS_MIRROR_ONCE _D3D11_TEXTURE_ADDRESS_MODE = 5
)

type _D3D11_USAGE int32

const (
	_D3D11_USAGE_DEFAULT   _D3D11_USAGE = 0
	_D3D11_USAGE_IMMUTABLE _D3D11_USAGE = 1
	_D3D11_USAGE_DYNAMIC   _D3D11_USAGE = 2
	_D3D11_USAGE_STAGING   _D3D11_USAGE = 3
)

var (
	_IID_ID3D11Texture2D = windows.GUID{Data1: 0x6f15aaf2, Data2: 0xd208, Data3: 0x4e89, Data4: [...]byte{0x9a, 0xb4, 0x48, 0x95, 0x35, 0xd3, 0x4f, 0x9c}}
)

type _D3D11_BLEND_DESC struct {
	AlphaToCoverageEnable  _BOOL
	IndependentBlendEnable _BOOL
	RenderTarget           [8]_D3D11_RENDER_TARGET_BLEND_DESC
}

type _D3D11_BOX struct {
	left   uint32
	top    uint32
	front  uint32
	right  uint32
	bottom uint32
	back   uint32
}

type _D3D11_BUFFER_DESC struct {
	ByteWidth           uint32
	Usage               _D3D11_USAGE
	BindFlags           uint32
	CPUAccessFlags      uint32
	MiscFlags           uint32
	StructureByteStride uint32
}

type _D3D11_DEPTH_STENCIL_DESC struct {
	DepthEnable      _BOOL
	DepthWriteMask   _D3D11_DEPTH_WRITE_MASK
	DepthFunc        _D3D11_COMPARISON_FUNC
	StencilEnable    _BOOL
	StencilReadMask  uint8
	StencilWriteMask uint8
	FrontFace        _D3D11_DEPTH_STENCILOP_DESC
	BackFace         _D3D11_DEPTH_STENCILOP_DESC
}

type _D3D11_DEPTH_STENCIL_VIEW_DESC struct {
	Format        _DXGI_FORMAT
	ViewDimension _D3D11_DSV_DIMENSION
	Flags         uint32
	_             [3]uint32
}

type _D3D11_DEPTH_STENCILOP_DESC struct {
	StencilFailOp      _D3D11_STENCIL_OP
	StencilDepthFailOp _D3D11_STENCIL_OP
	StencilPassOp      _D3D11_STENCIL_OP
	StencilFunc        _D3D11_COMPARISON_FUNC
}

type _D3D11_INPUT_ELEMENT_DESC struct {
	SemanticName         *byte
	SemanticIndex        uint32
	Format               _DXGI_FORMAT
	InputSlot            uint32
	AlignedByteOffset    uint32
	InputSlotClass       _D3D11_INPUT_CLASSIFICATION
	InstanceDataStepRate uint32
}

type _D3D11_MAPPED_SUBRESOURCE struct {
	pData      unsafe.Pointer
	RowPitch   uint32
	DepthPitch uint32
}

type _D3D11_RECT struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

type _D3D11_RASTERIZER_DESC struct {
	FillMode              _D3D11_FILL_MODE
	CullMode              _D3D11_CULL_MODE
	FrontCounterClockwise _BOOL
	DepthBias             int32
	DepthBiasClamp        float32
	SlopeScaledDepthBias  float32
	DepthClipEnable       _BOOL
	ScissorEnable         _BOOL
	MultisampleEnable     _BOOL
	AntialiasedLineEnable _BOOL
}

type _D3D11_RENDER_TARGET_BLEND_DESC struct {
	BlendEnable           _BOOL
	SrcBlend              _D3D11_BLEND
	DestBlend             _D3D11_BLEND
	BlendOp               _D3D11_BLEND_OP
	SrcBlendAlpha         _D3D11_BLEND
	DestBlendAlpha        _D3D11_BLEND
	BlendOpAlpha          _D3D11_BLEND_OP
	RenderTargetWriteMask uint8
}

type _D3D11_RENDER_TARGET_VIEW_DESC struct {
	Format        _DXGI_FORMAT
	ViewDimension _D3D11_RTV_DIMENSION
	_             [3]uint32
}

type _D3D11_SAMPLER_DESC struct {
	Filter         _D3D11_FILTER
	AddressU       _D3D11_TEXTURE_ADDRESS_MODE
	AddressV       _D3D11_TEXTURE_ADDRESS_MODE
	AddressW       _D3D11_TEXTURE_ADDRESS_MODE
	MipLODBias     float32
	MaxAnisotropy  uint32
	ComparisonFunc _D3D11_COMPARISON_FUNC
	BorderColor    [4]float32
	MinLOD         float32
	MaxLOD         float32
}

type _D3D11_SHADER_RESOURCE_VIEW_DESC struct {
	Format        _DXGI_FORMAT
	ViewDimension _D3D11_SRV_DIMENSION
	_             [4]uint32
}

type _D3D11_SUBRESOURCE_DATA struct {
	pSysMem          unsafe.Pointer
	SysMemPitch      uint32
	SysMemSlicePitch uint32
}

type _D3D11_TEXTURE2D_DESC struct {
	Width          uint32
	Height         uint32
	MipLevels      uint32
	ArraySize      uint32
	Format         _DXGI_FORMAT
	SampleDesc     _DXGI_SAMPLE_DESC
	Usage          _D3D11_USAGE
	BindFlags      uint32
	CPUAccessFlags uint32
	MiscFlags      uint32
}

type _D3D11_VIEWPORT struct {
	TopLeftX float32
	TopLeftY float32
	Width    float32
	Height   float32
	MinDepth float32
	MaxDepth float32
}

var (
	d3d11 = windows.NewLazySystemDLL("d3d11.dll")

	procD3D11CreateDevice = d3d11.NewProc("D3D11CreateDevice")
)

func _D3D11CreateDevice(pAdapter unsafe.Pointer, driverType _D3D_DRIVER_TYPE, software windows.Handle, flags uint32, featureLevels []_D3D_FEATURE_LEVEL, createDevice bool, createImmediateContext bool) (unsafe.Pointer, _D3D_FEATURE_LEVEL, unsafe.Pointer, error) {
	var pFeatureLevels *_D3D_FEATURE_LEVEL
	if len(featureLevels) > 0 {
		pFeatureLevels = &featureLevels[0]
	}

	var device unsafe.Pointer
	var pDevice *unsafe.Pointer
	if createDevice {
		pDevice = &device
	}

	var featureLevel _D3D_FEATURE_LEVEL

	var immediateContext unsafe.Pointer
	var pImmediateContext *unsafe.Pointer
	if createImmediateContext {
		pImmediateContext = &immediateContext
	}

	r, _, _ := procD3D11CreateDevice.Call(uintptr(pAdapter), uintptr(driverType), uintptr(software), uintptr(flags), uintptr(unsafe.Pointer(pFeatureLevels)), uintptr(len(featureLevels)), _D3D11_SDK_VERSION, uintptr(unsafe.Pointer(pDevice)), uintptr(unsafe.Pointer(&featureLevel)), uintptr(unsafe.Pointer(pImmediateContext)))
	if device == nil && uint32(r) != uint32(windows.S_FALSE) {
		return nil, 0, nil, fmt.Errorf("directx: D3D11CreateDevice failed: %w", handleError(windows.Handle(uint32(r))))
	}
	if device != nil && uint32(r) != uint32(windows.S_OK) {
		return nil, 0, nil, fmt.Errorf("directx: D3D11CreateDevice failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return device, featureLevel, immediateContext, nil
}

type _ID3D11BlendState struct {
	vtbl *_ID3D11BlendState_Vtbl
}

type _ID3D11BlendState_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	GetDesc uintptr
}

func (i *_ID3D11BlendState) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _ID3D11Buffer struct {
	vtbl *_ID3D11Buffer_Vtbl
}

type _ID3D11Buffer_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	// ID3D11Resource
	GetType             uintptr
	SetEvictionPriority uintptr
	GetEvictionPriority uintptr

	GetDesc uintptr
}

func (i *_ID3D11Buffer) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _ID3D11ClassInstance struct {
	vtbl *_ID3D11ClassInstance_Vtbl
}

type _ID3D11ClassInstance_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	GetClassLinkage uintptr
	GetDesc         uintptr
	GetInstanceName uintptr
	GetTypeName     uintptr
}

type _ID3D11ClassLinkage struct {
	vtbl *_ID3D11ClassLinkage
}

type _ID3D11ClassLinkage_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	GetClassInstance    uintptr
	CreateClassInstance uintptr
}

type _ID3D11DepthStencilState struct {
	vtbl *_ID3D11DepthStencilState_Vtbl
}

type _ID3D11DepthStencilState_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	GetDesc uintptr
}

type _ID3D11DepthStencilView struct {
	vtbl *_ID3D11DepthStencilView_Vtbl
}

type _ID3D11DepthStencilView_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	// ID3D11View
	GetResource uintptr

	GetDesc uintptr
}

func (i *_ID3D11DepthStencilView) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _ID3D11Device struct {
	vtbl *_ID3D11Device_Vtbl
}

type _ID3D11Device_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	CreateBuffer                         uintptr
	CreateTexture1D                      uintptr
	CreateTexture2D                      uintptr
	CreateTexture3D                      uintptr
	CreateShaderResourceView             uintptr
	CreateUnorderedAccessView            uintptr
	CreateRenderTargetView               uintptr
	CreateDepthStencilView               uintptr
	CreateInputLayout                    uintptr
	CreateVertexShader                   uintptr
	CreateGeometryShader                 uintptr
	CreateGeometryShaderWithStreamOutput uintptr
	CreatePixelShader                    uintptr
	CreateHullShader                     uintptr
	CreateDomainShader                   uintptr
	CreateComputeShader                  uintptr
	CreateClassLinkage                   uintptr
	CreateBlendState                     uintptr
	CreateDepthStencilState              uintptr
	CreateRasterizerState                uintptr
	CreateSamplerState                   uintptr
	CreateQuery                          uintptr
	CreatePredicate                      uintptr
	CreateCounter                        uintptr
	CreateDeferredContext                uintptr
	OpenSharedResource                   uintptr
	CheckFormatSupport                   uintptr
	CheckMultisampleQualityLevels        uintptr
	CheckCounterInfo                     uintptr
	CheckCounter                         uintptr
	CheckFeatureSupport                  uintptr
	GetPrivateData                       uintptr
	SetPrivateData                       uintptr
	SetPrivateDataInterface              uintptr
	GetFeatureLevel                      uintptr
	GetCreationFlags                     uintptr
	GetDeviceRemovedReason               uintptr
	GetImmediateContext                  uintptr
	SetExceptionMode                     uintptr
	GetExceptionMode                     uintptr
}

func (i *_ID3D11Device) CreateBlendState(pBlendStateDesc *_D3D11_BLEND_DESC) (*_ID3D11BlendState, error) {
	var blendState *_ID3D11BlendState
	r, _, _ := syscall.Syscall(i.vtbl.CreateBlendState, 3, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pBlendStateDesc)), uintptr(unsafe.Pointer(&blendState)))
	runtime.KeepAlive(pBlendStateDesc)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateBlendState failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return blendState, nil
}

func (i *_ID3D11Device) CreateBuffer(pDesc *_D3D11_BUFFER_DESC, pInitialData *_D3D11_SUBRESOURCE_DATA) (*_ID3D11Buffer, error) {
	if pDesc.BindFlags&uint32(_D3D11_BIND_CONSTANT_BUFFER) != 0 && pDesc.ByteWidth%16 != 0 {
		return nil, fmt.Errorf("directx: ByteLength for a constant buffer must be multiples of 16 but was %d at ID3D11Device::CreateBuffer", pDesc.ByteWidth)
	}

	var buffer *_ID3D11Buffer
	r, _, _ := syscall.Syscall6(i.vtbl.CreateBuffer, 4, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pDesc)), uintptr(unsafe.Pointer(pInitialData)), uintptr(unsafe.Pointer(&buffer)),
		0, 0)
	runtime.KeepAlive(pDesc)
	runtime.KeepAlive(pInitialData)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateBuffer failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return buffer, nil
}

func (i *_ID3D11Device) CreateDepthStencilState(pDepthStencilDesc *_D3D11_DEPTH_STENCIL_DESC) (*_ID3D11DepthStencilState, error) {
	var dss *_ID3D11DepthStencilState
	r, _, _ := syscall.Syscall(i.vtbl.CreateDepthStencilState, 3, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pDepthStencilDesc)), uintptr(unsafe.Pointer(&dss)))
	runtime.KeepAlive(pDepthStencilDesc)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateDepthStencilState failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return dss, nil
}

func (i *_ID3D11Device) CreateDepthStencilView(pResource unsafe.Pointer, pDesc *_D3D11_DEPTH_STENCIL_VIEW_DESC) (*_ID3D11DepthStencilView, error) {
	var dsv *_ID3D11DepthStencilView
	r, _, _ := syscall.Syscall6(i.vtbl.CreateDepthStencilView, 4, uintptr(unsafe.Pointer(i)),
		uintptr(pResource), uintptr(unsafe.Pointer(pDesc)), uintptr(unsafe.Pointer(&dsv)),
		0, 0)
	runtime.KeepAlive(pResource)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateDepthStencilView failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return dsv, nil
}

func (i *_ID3D11Device) CreateInputLayout(inputElementDescs []_D3D11_INPUT_ELEMENT_DESC, pShaderBytecodeWithInputSignature unsafe.Pointer, bytecodeLength uintptr) (*_ID3D11InputLayout, error) {
	var inputLayout *_ID3D11InputLayout
	var pInputElementDescs *_D3D11_INPUT_ELEMENT_DESC
	if len(inputElementDescs) > 0 {
		pInputElementDescs = &inputElementDescs[0]
	}

	r, _, _ := syscall.Syscall6(i.vtbl.CreateInputLayout, 6, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pInputElementDescs)), uintptr(len(inputElementDescs)), uintptr(pShaderBytecodeWithInputSignature),
		bytecodeLength, uintptr(unsafe.Pointer(&inputLayout)))
	runtime.KeepAlive(pInputElementDescs)

	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateInputLayout failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return inputLayout, nil
}

func (i *_ID3D11Device) CreatePixelShader(pShaderBytecode unsafe.Pointer, bytecodeLength uintptr, pClassLinkage *_ID3D11ClassLinkage) (*_ID3D11PixelShader, error) {
	var pixelShader *_ID3D11PixelShader
	r, _, _ := syscall.Syscall6(i.vtbl.CreatePixelShader, 5, uintptr(unsafe.Pointer(i)),
		uintptr(pShaderBytecode), bytecodeLength, uintptr(unsafe.Pointer(pClassLinkage)),
		uintptr(unsafe.Pointer(&pixelShader)), 0)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreatePixelShader failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return pixelShader, nil
}

func (i *_ID3D11Device) CreateRasterizerState(pRasterizerDesc *_D3D11_RASTERIZER_DESC) (*_ID3D11RasterizerState, error) {
	var rs *_ID3D11RasterizerState
	r, _, _ := syscall.Syscall(i.vtbl.CreateRasterizerState, 3, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pRasterizerDesc)), uintptr(unsafe.Pointer(&rs)))
	runtime.KeepAlive(pRasterizerDesc)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateRasterizerState failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return rs, nil
}

func (i *_ID3D11Device) CreateRenderTargetView(pResource unsafe.Pointer, pDesc *_D3D11_RENDER_TARGET_VIEW_DESC) (*_ID3D11RenderTargetView, error) {
	var rtView *_ID3D11RenderTargetView
	r, _, _ := syscall.Syscall6(i.vtbl.CreateRenderTargetView, 4, uintptr(unsafe.Pointer(i)),
		uintptr(pResource), uintptr(unsafe.Pointer(pDesc)), uintptr(unsafe.Pointer(&rtView)),
		0, 0)
	runtime.KeepAlive(pResource)
	runtime.KeepAlive(pDesc)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateRenderTargetView failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return rtView, nil
}

func (i *_ID3D11Device) CreateSamplerState(pSamplerDesc *_D3D11_SAMPLER_DESC) (*_ID3D11SamplerState, error) {
	var samplerState *_ID3D11SamplerState
	r, _, _ := syscall.Syscall(i.vtbl.CreateSamplerState, 3, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pSamplerDesc)), uintptr(unsafe.Pointer(&samplerState)))
	runtime.KeepAlive(pSamplerDesc)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateSamplerState failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return samplerState, nil
}

func (i *_ID3D11Device) CreateShaderResourceView(pResource unsafe.Pointer, pDesc *_D3D11_SHADER_RESOURCE_VIEW_DESC) (*_ID3D11ShaderResourceView, error) {
	var srView *_ID3D11ShaderResourceView
	r, _, _ := syscall.Syscall6(i.vtbl.CreateShaderResourceView, 4, uintptr(unsafe.Pointer(i)),
		uintptr(pResource), uintptr(unsafe.Pointer(pDesc)), uintptr(unsafe.Pointer(&srView)),
		0, 0)
	runtime.KeepAlive(pResource)
	runtime.KeepAlive(pDesc)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateShaderResourceView failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return srView, nil
}

func (i *_ID3D11Device) CreateTexture2D(pDesc *_D3D11_TEXTURE2D_DESC, pInitialData *_D3D11_SUBRESOURCE_DATA) (*_ID3D11Texture2D, error) {
	var texture *_ID3D11Texture2D
	r, _, _ := syscall.Syscall6(i.vtbl.CreateTexture2D, 4, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pDesc)), uintptr(unsafe.Pointer(pInitialData)), uintptr(unsafe.Pointer(&texture)),
		0, 0)
	runtime.KeepAlive(pDesc)
	runtime.KeepAlive(pInitialData)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateTexture2D failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return texture, nil
}

func (i *_ID3D11Device) CreateVertexShader(pShaderBytecode unsafe.Pointer, bytecodeLength uintptr, pClassLinkage *_ID3D11ClassLinkage) (*_ID3D11VertexShader, error) {
	var vertexShader *_ID3D11VertexShader
	r, _, _ := syscall.Syscall6(i.vtbl.CreateVertexShader, 5, uintptr(unsafe.Pointer(i)),
		uintptr(pShaderBytecode), bytecodeLength, uintptr(unsafe.Pointer(pClassLinkage)),
		uintptr(unsafe.Pointer(&vertexShader)), 0)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::CreateVertexShader failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return vertexShader, nil
}

func (i *_ID3D11Device) QueryInterface(riid *windows.GUID) (unsafe.Pointer, error) {
	var v unsafe.Pointer
	r, _, _ := syscall.Syscall(i.vtbl.QueryInterface, 3, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(riid)), uintptr(unsafe.Pointer(&v)))
	runtime.KeepAlive(riid)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: ID3D11Device::QueryInterface failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return v, nil
}

type _ID3D11DeviceContext struct {
	vtbl *_ID3D11DeviceContext_Vtbl
}

type _ID3D11DeviceContext_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	VSSetConstantBuffers                      uintptr
	PSSetShaderResources                      uintptr
	PSSetShader                               uintptr
	PSSetSamplers                             uintptr
	VSSetShader                               uintptr
	DrawIndexed                               uintptr
	Draw                                      uintptr
	Map                                       uintptr
	Unmap                                     uintptr
	PSSetConstantBuffers                      uintptr
	IASetInputLayout                          uintptr
	IASetVertexBuffers                        uintptr
	IASetIndexBuffer                          uintptr
	DrawIndexedInstanced                      uintptr
	DrawInstanced                             uintptr
	GSSetConstantBuffers                      uintptr
	GSSetShader                               uintptr
	IASetPrimitiveTopology                    uintptr
	VSSetShaderResources                      uintptr
	VSSetSamplers                             uintptr
	Begin                                     uintptr
	End                                       uintptr
	GetData                                   uintptr
	SetPredication                            uintptr
	GSSetShaderResources                      uintptr
	GSSetSamplers                             uintptr
	OMSetRenderTargets                        uintptr
	OMSetRenderTargetsAndUnorderedAccessViews uintptr
	OMSetBlendState                           uintptr
	OMSetDepthStencilState                    uintptr
	SOSetTargets                              uintptr
	DrawAuto                                  uintptr
	DrawIndexedInstancedIndirect              uintptr
	DrawInstancedIndirect                     uintptr
	Dispatch                                  uintptr
	DispatchIndirect                          uintptr
	RSSetState                                uintptr
	RSSetViewports                            uintptr
	RSSetScissorRects                         uintptr
	CopySubresourceRegion                     uintptr
	CopyResource                              uintptr
	UpdateSubresource                         uintptr
	CopyStructureCount                        uintptr
	ClearRenderTargetView                     uintptr
	ClearUnorderedAccessViewUint              uintptr
	ClearUnorderedAccessViewFloat             uintptr
	ClearDepthStencilView                     uintptr
	GenerateMips                              uintptr
	SetResourceMinLOD                         uintptr
	GetResourceMinLOD                         uintptr
	ResolveSubresource                        uintptr
	ExecuteCommandList                        uintptr
	HSSetShaderResources                      uintptr
	HSSetShader                               uintptr
	HSSetSamplers                             uintptr
	HSSetConstantBuffers                      uintptr
	DSSetShaderResources                      uintptr
	DSSetShader                               uintptr
	DSSetSamplers                             uintptr
	DSSetConstantBuffers                      uintptr
	CSSetShaderResources                      uintptr
	CSSetUnorderedAccessViews                 uintptr
	CSSetShader                               uintptr
	CSSetSamplers                             uintptr
	CSSetConstantBuffers                      uintptr
	VSGetConstantBuffers                      uintptr
	PSGetShaderResources                      uintptr
	PSGetShader                               uintptr
	PSGetSamplers                             uintptr
	VSGetShader                               uintptr
	PSGetConstantBuffers                      uintptr
	IAGetInputLayout                          uintptr
	IAGetVertexBuffers                        uintptr
	IAGetIndexBuffer                          uintptr
	GSGetConstantBuffers                      uintptr
	GSGetShader                               uintptr
	IAGetPrimitiveTopology                    uintptr
	VSGetShaderResources                      uintptr
	VSGetSamplers                             uintptr
	GetPredication                            uintptr
	GSGetShaderResources                      uintptr
	GSGetSamplers                             uintptr
	OMGetRenderTargets                        uintptr
	OMGetRenderTargetsAndUnorderedAccessViews uintptr
	OMGetBlendState                           uintptr
	OMGetDepthStencilState                    uintptr
	SOGetTargets                              uintptr
	RSGetState                                uintptr
	RSGetViewports                            uintptr
	RSGetScissorRects                         uintptr
	HSGetShaderResources                      uintptr
	HSGetShader                               uintptr
	HSGetSamplers                             uintptr
	HSGetConstantBuffers                      uintptr
	DSGetShaderResources                      uintptr
	DSGetShader                               uintptr
	DSGetSamplers                             uintptr
	DSGetConstantBuffers                      uintptr
	CSGetShaderResources                      uintptr
	CSGetUnorderedAccessViews                 uintptr
	CSGetShader                               uintptr
	CSGetSamplers                             uintptr
	CSGetConstantBuffers                      uintptr
	ClearState                                uintptr
	Flush                                     uintptr
	GetType                                   uintptr
	GetContextFlags                           uintptr
	FinishCommandList                         uintptr
}

func (i *_ID3D11DeviceContext) ClearState() {
	_, _, _ = syscall.Syscall(i.vtbl.ClearState, 1, uintptr(unsafe.Pointer(i)),
		0, 0)
}

func (i *_ID3D11DeviceContext) ClearDepthStencilView(pDepthStencilView *_ID3D11DepthStencilView, clearFlags uint8, depth float32, stencil uint8) {
	_, _, _ = syscall.Syscall6(i.vtbl.ClearDepthStencilView, 5, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pDepthStencilView)), uintptr(clearFlags), uintptr(math.Float32bits(depth)),
		uintptr(stencil), 0)
	runtime.KeepAlive(pDepthStencilView)
}

func (i *_ID3D11DeviceContext) CopySubresourceRegion(pDstResource unsafe.Pointer, dstSubresource uint32, dstX uint32, dstY uint32, dstZ uint32, pSrcResource unsafe.Pointer, srcSubresource uint32, pSrcBox *_D3D11_BOX) {
	_, _, _ = syscall.Syscall9(i.vtbl.CopySubresourceRegion, 9, uintptr(unsafe.Pointer(i)),
		uintptr(pDstResource), uintptr(dstSubresource), uintptr(dstX),
		uintptr(dstY), uintptr(dstZ), uintptr(pSrcResource),
		uintptr(srcSubresource), uintptr(unsafe.Pointer(pSrcBox)))
	runtime.KeepAlive(pDstResource)
	runtime.KeepAlive(pSrcResource)
}

func (i *_ID3D11DeviceContext) DrawIndexed(indexCount uint32, startIndexLocation uint32, baseVertexLocation int32) {
	_, _, _ = syscall.Syscall6(i.vtbl.DrawIndexed, 4, uintptr(unsafe.Pointer(i)),
		uintptr(indexCount), uintptr(startIndexLocation), uintptr(baseVertexLocation),
		0, 0)
}

func (i *_ID3D11DeviceContext) IASetIndexBuffer(pIndexBuffer *_ID3D11Buffer, format _DXGI_FORMAT, offset uint32) {
	_, _, _ = syscall.Syscall6(i.vtbl.IASetIndexBuffer, 4, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pIndexBuffer)), uintptr(format), uintptr(offset),
		0, 0)
	runtime.KeepAlive(pIndexBuffer)
}

func (i *_ID3D11DeviceContext) IASetInputLayout(pInputLayout *_ID3D11InputLayout) {
	_, _, _ = syscall.Syscall(i.vtbl.IASetInputLayout, 2, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pInputLayout)), 0)
	runtime.KeepAlive(pInputLayout)
}

func (i *_ID3D11DeviceContext) IASetPrimitiveTopology(topology _D3D11_PRIMITIVE_TOPOLOGY) {
	_, _, _ = syscall.Syscall(i.vtbl.IASetPrimitiveTopology, 2, uintptr(unsafe.Pointer(i)),
		uintptr(topology), 0)
}

func (i *_ID3D11DeviceContext) IASetVertexBuffers(startSlot uint32, vertexBuffers []*_ID3D11Buffer, strides []uint32, offsets []uint32) {
	_, _, _ = syscall.Syscall6(i.vtbl.IASetVertexBuffers, 6, uintptr(unsafe.Pointer(i)),
		uintptr(startSlot), uintptr(len(vertexBuffers)), uintptr(unsafe.Pointer(&vertexBuffers[0])),
		uintptr(unsafe.Pointer(&strides[0])), uintptr(unsafe.Pointer(&offsets[0])))
	runtime.KeepAlive(vertexBuffers)
	runtime.KeepAlive(strides)
	runtime.KeepAlive(offsets)
}

func (i *_ID3D11DeviceContext) Map(pResource unsafe.Pointer, subresource uint32, mapType _D3D11_MAP, mapFlags uint32, pMappedResource *_D3D11_MAPPED_SUBRESOURCE) error {
	r, _, _ := syscall.Syscall6(i.vtbl.Map, 6, uintptr(unsafe.Pointer(i)),
		uintptr(pResource), uintptr(subresource), uintptr(mapType),
		uintptr(mapFlags), uintptr(unsafe.Pointer(pMappedResource)))
	runtime.KeepAlive(pResource)
	runtime.KeepAlive(pMappedResource)
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("directx: ID3D11DeviceContext::Map failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (i *_ID3D11DeviceContext) OMSetBlendState(pBlendState *_ID3D11BlendState, blendFactor *[4]float32, sampleMask uint32) {
	var pBlendFactor *float32
	if blendFactor != nil {
		pBlendFactor = &blendFactor[0]
	}
	_, _, _ = syscall.Syscall6(i.vtbl.OMSetBlendState, 4, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pBlendState)), uintptr(unsafe.Pointer(pBlendFactor)), uintptr(sampleMask),
		0, 0)
	runtime.KeepAlive(pBlendState)
}

func (i *_ID3D11DeviceContext) OMSetDepthStencilState(pDepthStencilState *_ID3D11DepthStencilState, stencilRef uint32) {
	_, _, _ = syscall.Syscall(i.vtbl.OMSetDepthStencilState, 3, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pDepthStencilState)), uintptr(stencilRef))
	runtime.KeepAlive(pDepthStencilState)
}

func (i *_ID3D11DeviceContext) OMSetRenderTargets(renderTargetViews []*_ID3D11RenderTargetView, depthStencilView *_ID3D11DepthStencilView) {
	var ppRenderTargetViews **_ID3D11RenderTargetView
	if len(renderTargetViews) > 0 {
		ppRenderTargetViews = &renderTargetViews[0]
	}
	_, _, _ = syscall.Syscall6(i.vtbl.OMSetRenderTargets, 4, uintptr(unsafe.Pointer(i)),
		uintptr(len(renderTargetViews)), uintptr(unsafe.Pointer(ppRenderTargetViews)), uintptr(unsafe.Pointer(depthStencilView)),
		0, 0)
	runtime.KeepAlive(renderTargetViews)
}

func (i *_ID3D11DeviceContext) PSSetConstantBuffers(startSlot uint32, constantBuffers []*_ID3D11Buffer) {
	var ppConstantBuffers **_ID3D11Buffer
	if len(constantBuffers) > 0 {
		ppConstantBuffers = &constantBuffers[0]
	}
	_, _, _ = syscall.Syscall6(i.vtbl.PSSetConstantBuffers, 4, uintptr(unsafe.Pointer(i)),
		uintptr(startSlot), uintptr(len(constantBuffers)), uintptr(unsafe.Pointer(ppConstantBuffers)),
		0, 0)
	runtime.KeepAlive(constantBuffers)
}

func (i *_ID3D11DeviceContext) PSSetSamplers(startSlot uint32, samplers []*_ID3D11SamplerState) {
	var ppSamplers **_ID3D11SamplerState
	if len(samplers) > 0 {
		ppSamplers = &samplers[0]
	}
	_, _, _ = syscall.Syscall6(i.vtbl.PSSetSamplers, 4, uintptr(unsafe.Pointer(i)),
		uintptr(startSlot), uintptr(len(samplers)), uintptr(unsafe.Pointer(ppSamplers)),
		0, 0)
	runtime.KeepAlive(samplers)
}

func (i *_ID3D11DeviceContext) PSSetShader(pPixelShader *_ID3D11PixelShader, classInstances []*_ID3D11ClassInstance) {
	var ppClassInstances **_ID3D11ClassInstance
	if len(classInstances) > 0 {
		ppClassInstances = &classInstances[0]
	}
	_, _, _ = syscall.Syscall6(i.vtbl.PSSetShader, 4, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pPixelShader)), uintptr(unsafe.Pointer(ppClassInstances)), uintptr(len(classInstances)),
		0, 0)
	runtime.KeepAlive(pPixelShader)
}

func (i *_ID3D11DeviceContext) PSSetShaderResources(startSlot uint32, shaderResourceViews []*_ID3D11ShaderResourceView) {
	var ppShaderResourceViews **_ID3D11ShaderResourceView
	if len(shaderResourceViews) > 0 {
		ppShaderResourceViews = &shaderResourceViews[0]
	}
	_, _, _ = syscall.Syscall6(i.vtbl.PSSetShaderResources, 4, uintptr(unsafe.Pointer(i)),
		uintptr(startSlot), uintptr(len(shaderResourceViews)), uintptr(unsafe.Pointer(ppShaderResourceViews)),
		0, 0)
	runtime.KeepAlive(shaderResourceViews)
}

func (i *_ID3D11DeviceContext) RSSetScissorRects(rects []_D3D11_RECT) {
	var pRects *_D3D11_RECT
	if len(rects) > 0 {
		pRects = &rects[0]
	}
	_, _, _ = syscall.Syscall(i.vtbl.RSSetScissorRects, 3, uintptr(unsafe.Pointer(i)),
		uintptr(len(rects)), uintptr(unsafe.Pointer(pRects)))
	runtime.KeepAlive(rects)
}

func (i *_ID3D11DeviceContext) RSSetState(pRasterizerState *_ID3D11RasterizerState) {
	_, _, _ = syscall.Syscall(i.vtbl.RSSetState, 2, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pRasterizerState)), 0)
	runtime.KeepAlive(pRasterizerState)
}

func (i *_ID3D11DeviceContext) RSSetViewports(viewports []_D3D11_VIEWPORT) {
	var pViewports *_D3D11_VIEWPORT
	if len(viewports) > 0 {
		pViewports = &viewports[0]
	}
	_, _, _ = syscall.Syscall(i.vtbl.RSSetViewports, 3, uintptr(unsafe.Pointer(i)),
		uintptr(len(viewports)), uintptr(unsafe.Pointer(pViewports)))
	runtime.KeepAlive(viewports)
}

func (i *_ID3D11DeviceContext) Unmap(pResource unsafe.Pointer, subresource uint32) {
	_, _, _ = syscall.Syscall(i.vtbl.Unmap, 3, uintptr(unsafe.Pointer(i)),
		uintptr(pResource), uintptr(subresource))
	runtime.KeepAlive(pResource)
}

func (i *_ID3D11DeviceContext) UpdateSubresource(pDstResource unsafe.Pointer, dstSubresource uint32, pDstBox *_D3D11_BOX, pSrcData unsafe.Pointer, srcRowPitch uint32, srcDepthPitch uint32) {
	_, _, _ = syscall.Syscall9(i.vtbl.UpdateSubresource, 7, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pDstResource)), uintptr(dstSubresource), uintptr(unsafe.Pointer(pDstBox)),
		uintptr(pSrcData), uintptr(srcRowPitch), uintptr(srcDepthPitch),
		0, 0)
	runtime.KeepAlive(pDstResource)
	runtime.KeepAlive(pDstBox)
	runtime.KeepAlive(pSrcData)
}

func (i *_ID3D11DeviceContext) VSSetConstantBuffers(startSlot uint32, constantBuffers []*_ID3D11Buffer) {
	var ppConstantBuffers **_ID3D11Buffer
	if len(constantBuffers) > 0 {
		ppConstantBuffers = &constantBuffers[0]
	}
	_, _, _ = syscall.Syscall6(i.vtbl.VSSetConstantBuffers, 4, uintptr(unsafe.Pointer(i)),
		uintptr(startSlot), uintptr(len(constantBuffers)), uintptr(unsafe.Pointer(ppConstantBuffers)),
		0, 0)
	runtime.KeepAlive(constantBuffers)
}

func (i *_ID3D11DeviceContext) VSSetShader(pVertexShader *_ID3D11VertexShader, classInstances []*_ID3D11ClassInstance) {
	var ppClassInstances **_ID3D11ClassInstance
	if len(classInstances) > 0 {
		ppClassInstances = &classInstances[0]
	}
	_, _, _ = syscall.Syscall6(i.vtbl.VSSetShader, 4, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(pVertexShader)), uintptr(unsafe.Pointer(ppClassInstances)), uintptr(len(classInstances)),
		0, 0)
	runtime.KeepAlive(pVertexShader)
}

type _ID3D11InputLayout struct {
	vtbl *_ID3D11InputLayout_Vtbl
}

type _ID3D11InputLayout_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr
}

func (i *_ID3D11InputLayout) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _ID3D11PixelShader struct {
	vtbl *_ID3D11PixelShader_Vtbl
}

type _ID3D11PixelShader_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr
}

func (i *_ID3D11PixelShader) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _ID3D11RasterizerState struct {
	vtbl *_ID3D11RasterizerState_Vtbl
}

type _ID3D11RasterizerState_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	GetDesc uintptr
}

func (i *_ID3D11RasterizerState) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _ID3D11RenderTargetView struct {
	vtbl *_ID3D11RenderTargetView_Vtbl
}

type _ID3D11RenderTargetView_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	// ID3D11View
	GetResource uintptr

	GetDesc uintptr
}

func (i *_ID3D11RenderTargetView) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _ID3D11SamplerState struct {
	vtbl *_ID3D11SamplerState_Vtbl
}

type _ID3D11SamplerState_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	GetDesc uintptr
}

func (i *_ID3D11SamplerState) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _ID3D11ShaderResourceView struct {
	vtbl *_ID3D11ShaderResourceView_Vtbl
}

type _ID3D11ShaderResourceView_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	// ID3D11View
	GetResource uintptr

	GetDesc uintptr
}

func (i *_ID3D11ShaderResourceView) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _ID3D11Texture2D struct {
	vtbl *_ID3D11Texture2D_Vtbl
}

type _ID3D11Texture2D_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr

	// ID3D11Resource
	GetType             uintptr
	SetEvictionPriority uintptr
	GetEvictionPriority uintptr

	GetDesc uintptr
}

func (i *_ID3D11Texture2D) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _ID3D11VertexShader struct {
	vtbl *_ID3D11VertexShader_Vtbl
}

type _ID3D11VertexShader_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	// ID3D11DeviceChild
	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr
}

func (i *_ID3D11VertexShader) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}
