// Copyright 2022 The Ebiten Authors
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
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/hlsl"
)

const frameCount = 2

// NewGraphics creates an implementation of graphicsdriver.Graphics for DirectX.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics() (graphicsdriver.Graphics, error) {
	g := &Graphics{}
	if err := g.initialize(); err != nil {
		return nil, err
	}
	return g, nil
}

var inputElementDescs []_D3D12_INPUT_ELEMENT_DESC

func init() {
	position := []byte("POSITION\000")
	texcoord := []byte("TEXCOORD\000")
	color := []byte("COLOR\000")
	inputElementDescs = []_D3D12_INPUT_ELEMENT_DESC{
		{
			SemanticName:         &position[0],
			SemanticIndex:        0,
			Format:               _DXGI_FORMAT_R32G32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D12_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D12_INPUT_CLASSIFICATION_PER_VERTEX_DATA,
			InstanceDataStepRate: 0,
		},
		{
			SemanticName:         &texcoord[0],
			SemanticIndex:        0,
			Format:               _DXGI_FORMAT_R32G32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D12_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D12_INPUT_CLASSIFICATION_PER_VERTEX_DATA,
			InstanceDataStepRate: 0,
		},
		{
			SemanticName:         &color[0],
			SemanticIndex:        0,
			Format:               _DXGI_FORMAT_R32G32B32A32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D12_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D12_INPUT_CLASSIFICATION_PER_VERTEX_DATA,
			InstanceDataStepRate: 0,
		},
	}
}

type Graphics struct {
	debug              *_ID3D12Debug
	device             *_ID3D12Device
	commandQueue       *_ID3D12CommandQueue
	rtvDescriptorHeap  *_ID3D12DescriptorHeap
	rtvDescriptorSize  uint32
	renderTargets      [frameCount]*_ID3D12Resource
	framePipelineToken _D3D12XBOX_FRAME_PIPELINE_TOKEN

	fence          *_ID3D12Fence
	fenceValues    [frameCount]uint64
	fenceWaitEvent windows.Handle

	allowTearing bool

	// drawCommandAllocators are command allocators for a 3D engine (DrawIndexedInstanced).
	// For the word 'engine', see https://docs.microsoft.com/en-us/windows/win32/direct3d12/user-mode-heap-synchronization.
	// The term 'draw' is used instead of '3D' in this package.
	drawCommandAllocators [frameCount]*_ID3D12CommandAllocator

	// copyCommandAllocators are command allocators for a copy engine (CopyTextureRegion).
	copyCommandAllocators [frameCount]*_ID3D12CommandAllocator

	// drawCommandList is a command list for a 3D engine (DrawIndexedInstanced).
	drawCommandList *_ID3D12GraphicsCommandList

	needFlushDrawCommandList bool

	// copyCommandList is a command list for a copy engine (CopyTextureRegion).
	copyCommandList *_ID3D12GraphicsCommandList

	needFlushCopyCommandList bool

	// drawCommandList and copyCommandList are exclusive: if one is not empty, the other must be empty.

	vertices [frameCount][]*_ID3D12Resource
	indices  [frameCount][]*_ID3D12Resource

	factory   *_IDXGIFactory4
	swapChain *_IDXGISwapChain4

	window windows.HWND

	frameIndex          int
	prevBeginFrameIndex int

	// frameStarted is true since Begin until End with present
	frameStarted bool

	images         map[graphicsdriver.ImageID]*Image
	screenImage    *Image
	nextImageID    graphicsdriver.ImageID
	disposedImages [frameCount][]*Image

	shaders         map[graphicsdriver.ShaderID]*Shader
	nextShaderID    graphicsdriver.ShaderID
	disposedShaders [frameCount][]*Shader

	vsyncEnabled bool
	transparent  bool

	// occluded reports whether the screen is invisible or not.
	occluded bool

	// lastTime is the last time for rendering.
	lastTime time.Time

	newScreenWidth  int
	newScreenHeight int

	suspendingCh chan struct{}
	suspendedCh  chan struct{}
	resumeCh     chan struct{}

	pipelineStates
}

func (g *Graphics) initialize() (ferr error) {
	var (
		useWARP       bool
		useDebugLayer bool
	)
	env := os.Getenv("EBITENGINE_DIRECTX")
	if env == "" {
		// For backward compatibility, read the EBITEN_ version.
		env = os.Getenv("EBITEN_DIRECTX")
	}
	for _, t := range strings.Split(env, ",") {
		switch strings.TrimSpace(t) {
		case "warp":
			// TODO: Is WARP available on Xbox?
			useWARP = true
		case "debug":
			useDebugLayer = true
		}
	}

	// Specify the level 11 by default.
	// Some old cards don't work well with the default feature level (#2447, #2486).
	var featureLevel _D3D_FEATURE_LEVEL = _D3D_FEATURE_LEVEL_11_0
	if env := os.Getenv("EBITENGINE_DIRECTX_FEATURE_LEVEL"); env != "" {
		switch env {
		case "11_0":
			featureLevel = _D3D_FEATURE_LEVEL_11_0
		case "11_1":
			featureLevel = _D3D_FEATURE_LEVEL_11_1
		case "12_0":
			featureLevel = _D3D_FEATURE_LEVEL_12_0
		case "12_1":
			featureLevel = _D3D_FEATURE_LEVEL_12_1
		case "12_2":
			featureLevel = _D3D_FEATURE_LEVEL_12_2
		}
	}

	// Initialize not only a device but also other members like a fence.
	// Even if initializing a device succeeds, initializing a fence might fail (#2142).

	if microsoftgdk.IsXbox() {
		if err := g.initializeXbox(useWARP, useDebugLayer); err != nil {
			return err
		}
	} else {
		if err := g.initializeDesktop(useWARP, useDebugLayer, featureLevel); err != nil {
			return err
		}
	}

	return nil
}

func (g *Graphics) initializeDesktop(useWARP bool, useDebugLayer bool, featureLevel _D3D_FEATURE_LEVEL) (ferr error) {
	if err := d3d12.Load(); err != nil {
		return err
	}

	// As g's lifetime is the same as the process's lifetime, debug and other objects are never released
	// if this initialization succeeds.

	// The debug interface is optional and might not exist.
	if useDebugLayer {
		d, err := _D3D12GetDebugInterface()
		if err != nil {
			return err
		}
		g.debug = d
		defer func() {
			if ferr != nil {
				g.debug.Release()
				g.debug = nil
			}
		}()
		g.debug.EnableDebugLayer()
	}

	var flag uint32
	if g.debug != nil {
		flag = _DXGI_CREATE_FACTORY_DEBUG
	}
	f, err := _CreateDXGIFactory2(flag)
	if err != nil {
		return err
	}
	g.factory = f
	defer func() {
		if ferr != nil {
			g.factory.Release()
			g.factory = nil
		}
	}()

	var adapter *_IDXGIAdapter1
	if useWARP {
		a, err := g.factory.EnumWarpAdapter()
		if err != nil {
			return err
		}
		defer a.Release()
		adapter = a
	} else {
		for i := 0; ; i++ {
			a, err := g.factory.EnumAdapters1(uint32(i))
			if errors.Is(err, _DXGI_ERROR_NOT_FOUND) {
				break
			}
			if err != nil {
				return err
			}
			defer a.Release()

			desc, err := a.GetDesc1()
			if err != nil {
				return err
			}
			if desc.Flags&_DXGI_ADAPTER_FLAG_SOFTWARE != 0 {
				continue
			}
			// Test D3D12CreateDevice without creating an actual device.
			if _, err := _D3D12CreateDevice(unsafe.Pointer(a), featureLevel, &_IID_ID3D12Device, false); err != nil {
				continue
			}

			adapter = a
			break
		}
	}

	if adapter == nil {
		return errors.New("directx: DirectX 12 is not supported")
	}

	d, err := _D3D12CreateDevice(unsafe.Pointer(adapter), featureLevel, &_IID_ID3D12Device, true)
	if err != nil {
		return err
	}
	g.device = (*_ID3D12Device)(d)

	if f, err := g.factory.QueryInterface(&_IID_IDXGIFactory5); err == nil && f != nil {
		factory := (*_IDXGIFactory5)(f)
		defer factory.Release()
		var allowTearing int32
		if err := factory.CheckFeatureSupport(_DXGI_FEATURE_PRESENT_ALLOW_TEARING, unsafe.Pointer(&allowTearing), uint32(unsafe.Sizeof(allowTearing))); err == nil && allowTearing != 0 {
			g.allowTearing = true
		}
	}

	if err := g.initializeMembers(); err != nil {
		return err
	}

	// GetCopyableFootprints might return an invalid value with Wine (#2114).
	// To check this early, call NewImage here.
	i, err := g.NewImage(1, 1)
	if err != nil {
		return err
	}
	i.Dispose()

	return nil
}

func (g *Graphics) initializeXbox(useWARP bool, useDebugLayer bool) (ferr error) {
	if err := d3d12x.Load(); err != nil {
		return err
	}

	params := &_D3D12XBOX_CREATE_DEVICE_PARAMETERS{
		Version:                           microsoftgdk.D3D12SDKVersion(),
		GraphicsCommandQueueRingSizeBytes: _D3D12XBOX_DEFAULT_SIZE_BYTES,
		GraphicsScratchMemorySizeBytes:    _D3D12XBOX_DEFAULT_SIZE_BYTES,
		ComputeScratchMemorySizeBytes:     _D3D12XBOX_DEFAULT_SIZE_BYTES,
	}
	if useDebugLayer {
		params.ProcessDebugFlags = _D3D12_PROCESS_DEBUG_FLAG_DEBUG_LAYER_ENABLED
	}
	d, err := _D3D12XboxCreateDevice(nil, params, &_IID_ID3D12Device)
	if err != nil {
		return err
	}
	g.device = (*_ID3D12Device)(d)

	if err := g.initializeMembers(); err != nil {
		return err
	}

	if err := g.registerFrameEventForXbox(); err != nil {
		return err
	}

	g.suspendingCh = make(chan struct{})
	g.suspendedCh = make(chan struct{})
	g.resumeCh = make(chan struct{})
	if _, err := _RegisterAppStateChangeNotification(func(quiesced bool, context unsafe.Pointer) uintptr {
		if quiesced {
			g.suspendingCh <- struct{}{}
			// Confirm the suspension completed before the callback ends.
			<-g.suspendedCh
		} else {
			g.resumeCh <- struct{}{}
		}
		return 0
	}, nil); err != nil {
		return err
	}

	return nil
}

func (g *Graphics) registerFrameEventForXbox() error {
	d, err := g.device.QueryInterface(&_IID_IDXGIDevice)
	if err != nil {
		return err
	}
	dxgiDevice := (*_IDXGIDevice)(d)
	defer dxgiDevice.Release()

	dxgiAdapter, err := dxgiDevice.GetAdapter()
	if err != nil {
		return err
	}
	defer dxgiAdapter.Release()

	dxgiOutput, err := dxgiAdapter.EnumOutputs(0)
	if err != nil {
		return err
	}
	defer dxgiOutput.Release()

	if err := g.device.SetFrameIntervalX(dxgiOutput, _D3D12XBOX_FRAME_INTERVAL_60_HZ, frameCount-1, _D3D12XBOX_FRAME_INTERVAL_FLAG_NONE); err != nil {
		return err
	}
	if err := g.device.ScheduleFrameEventX(_D3D12XBOX_FRAME_EVENT_ORIGIN, 0, nil, _D3D12XBOX_SCHEDULE_FRAME_EVENT_FLAG_NONE); err != nil {
		return err
	}

	return nil
}

func (g *Graphics) initializeMembers() (ferr error) {
	// Create an event for a fence.
	e, err := windows.CreateEventEx(nil, nil, 0, windows.EVENT_MODIFY_STATE|windows.SYNCHRONIZE)
	if err != nil {
		return fmt.Errorf("directx: CreateEvent failed: %w", err)
	}
	g.fenceWaitEvent = e

	// Create a command queue.
	desc := _D3D12_COMMAND_QUEUE_DESC{
		Type:  _D3D12_COMMAND_LIST_TYPE_DIRECT,
		Flags: _D3D12_COMMAND_QUEUE_FLAG_NONE,
	}
	c, err := g.device.CreateCommandQueue(&desc)
	if err != nil {
		return err
	}
	g.commandQueue = c
	defer func() {
		if ferr != nil {
			g.commandQueue.Release()
			g.commandQueue = nil
		}
	}()

	// Create command allocators.
	for i := 0; i < frameCount; i++ {
		dca, err := g.device.CreateCommandAllocator(_D3D12_COMMAND_LIST_TYPE_DIRECT)
		if err != nil {
			return err
		}
		g.drawCommandAllocators[i] = dca
		defer func(i int) {
			if ferr != nil {
				g.drawCommandAllocators[i].Release()
				g.drawCommandAllocators[i] = nil
			}
		}(i)

		cca, err := g.device.CreateCommandAllocator(_D3D12_COMMAND_LIST_TYPE_DIRECT)
		if err != nil {
			return err
		}
		g.copyCommandAllocators[i] = cca
		defer func(i int) {
			if ferr != nil {
				g.copyCommandAllocators[i].Release()
				g.copyCommandAllocators[i] = nil
			}
		}(i)
	}

	// Create a frame fence.
	f, err := g.device.CreateFence(0, _D3D12_FENCE_FLAG_NONE)
	if err != nil {
		return err
	}
	g.fence = f
	defer func() {
		if ferr != nil {
			g.fence.Release()
			g.fence = nil
		}
	}()
	g.fenceValues[g.frameIndex]++

	// Create command lists.
	dcl, err := g.device.CreateCommandList(0, _D3D12_COMMAND_LIST_TYPE_DIRECT, g.drawCommandAllocators[0], nil)
	if err != nil {
		return err
	}
	g.drawCommandList = dcl
	defer func() {
		if ferr != nil {
			g.drawCommandList.Release()
			g.drawCommandList = nil
		}
	}()

	ccl, err := g.device.CreateCommandList(0, _D3D12_COMMAND_LIST_TYPE_DIRECT, g.copyCommandAllocators[0], nil)
	if err != nil {
		return err
	}
	g.copyCommandList = ccl
	defer func() {
		if ferr != nil {
			g.copyCommandList.Release()
			g.copyCommandList = nil
		}
	}()

	// Close the command list once as this is immediately Reset at Begin.
	if err := g.drawCommandList.Close(); err != nil {
		return err
	}
	if err := g.copyCommandList.Close(); err != nil {
		return err
	}

	// Create a descriptor heap for RTV.
	h, err := g.device.CreateDescriptorHeap(&_D3D12_DESCRIPTOR_HEAP_DESC{
		Type:           _D3D12_DESCRIPTOR_HEAP_TYPE_RTV,
		NumDescriptors: frameCount,
		Flags:          _D3D12_DESCRIPTOR_HEAP_FLAG_NONE,
		NodeMask:       0,
	})
	if err != nil {
		return err
	}
	g.rtvDescriptorHeap = h
	defer func() {
		if ferr != nil {
			g.rtvDescriptorHeap.Release()
			g.rtvDescriptorHeap = nil
		}
	}()
	g.rtvDescriptorSize = g.device.GetDescriptorHandleIncrementSize(_D3D12_DESCRIPTOR_HEAP_TYPE_RTV)

	if err := g.pipelineStates.initialize(g.device); err != nil {
		return err
	}

	return nil
}

func (g *Graphics) Initialize() (err error) {
	// Initialization should already be done.
	return nil
}

func createBuffer(device *_ID3D12Device, bufferSize uint64, heapType _D3D12_HEAP_TYPE) (*_ID3D12Resource, error) {
	state := _D3D12_RESOURCE_STATE_GENERIC_READ()
	if heapType == _D3D12_HEAP_TYPE_READBACK {
		state = _D3D12_RESOURCE_STATE_COPY_DEST
	}

	r, err := device.CreateCommittedResource(&_D3D12_HEAP_PROPERTIES{
		Type:                 heapType,
		CPUPageProperty:      _D3D12_CPU_PAGE_PROPERTY_UNKNOWN,
		MemoryPoolPreference: _D3D12_MEMORY_POOL_UNKNOWN,
		CreationNodeMask:     1,
		VisibleNodeMask:      1,
	}, _D3D12_HEAP_FLAG_NONE, &_D3D12_RESOURCE_DESC{
		Dimension:        _D3D12_RESOURCE_DIMENSION_BUFFER,
		Alignment:        0,
		Width:            bufferSize,
		Height:           1,
		DepthOrArraySize: 1,
		MipLevels:        1,
		Format:           _DXGI_FORMAT_UNKNOWN,
		SampleDesc: _DXGI_SAMPLE_DESC{
			Count:   1,
			Quality: 0,
		},
		Layout: _D3D12_TEXTURE_LAYOUT_ROW_MAJOR,
		Flags:  _D3D12_RESOURCE_FLAG_NONE,
	}, state, nil)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (g *Graphics) updateSwapChain(width, height int) error {
	if g.window == 0 {
		return errors.New("directx: the window handle is not initialized yet")
	}

	if g.swapChain == nil {
		if microsoftgdk.IsXbox() {
			if err := g.initSwapChainXbox(width, height); err != nil {
				return err
			}
		} else {
			if err := g.initSwapChainDesktop(width, height); err != nil {
				return err
			}
		}
		return nil
	}

	if microsoftgdk.IsXbox() {
		return errors.New("directx: resizing should never happen on Xbox")
	}

	g.newScreenWidth = width
	g.newScreenHeight = height

	return nil
}

func (g *Graphics) initSwapChainDesktop(width, height int) (ferr error) {
	// Create a swap chain.
	//
	// DXGI_ALPHA_MODE_PREMULTIPLIED doesn't work with a HWND well.
	//
	//     IDXGIFactory::CreateSwapChain: Alpha blended swapchains must be created with CreateSwapChainForComposition,
	//     or CreateSwapChainForCoreWindow with the DXGI_SWAP_CHAIN_FLAG_FOREGROUND_LAYER flag
	desc := &_DXGI_SWAP_CHAIN_DESC1{
		Width:       uint32(width),
		Height:      uint32(height),
		Format:      _DXGI_FORMAT_B8G8R8A8_UNORM,
		BufferUsage: _DXGI_USAGE_RENDER_TARGET_OUTPUT,
		BufferCount: frameCount,
		SwapEffect:  _DXGI_SWAP_EFFECT_FLIP_DISCARD,
		SampleDesc: _DXGI_SAMPLE_DESC{
			Count:   1,
			Quality: 0,
		},
	}
	if g.allowTearing {
		desc.Flags |= uint32(_DXGI_SWAP_CHAIN_FLAG_ALLOW_TEARING)
	}
	s, err := g.factory.CreateSwapChainForHwnd(unsafe.Pointer(g.commandQueue), g.window, desc, nil, nil)
	if err != nil {
		return err
	}
	s.As(&g.swapChain)
	defer func() {
		if ferr != nil {
			g.swapChain.Release()
			g.swapChain = nil
		}
	}()

	// MakeWindowAssociation should be called after swap chain creation.
	// https://docs.microsoft.com/en-us/windows/win32/api/dxgi/nf-dxgi-idxgifactory-makewindowassociation
	if err := g.factory.MakeWindowAssociation(g.window, _DXGI_MWA_NO_WINDOW_CHANGES|_DXGI_MWA_NO_ALT_ENTER); err != nil {
		return err
	}

	// TODO: Get the current buffer index?

	if err := g.createRenderTargetViewsDesktop(); err != nil {
		return err
	}

	g.frameIndex = int(g.swapChain.GetCurrentBackBufferIndex())

	return nil
}

func (g *Graphics) initSwapChainXbox(width, height int) (ferr error) {
	h, err := g.rtvDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}

	for i := 0; i < frameCount; i++ {
		r, err := g.device.CreateCommittedResource(&_D3D12_HEAP_PROPERTIES{
			Type:                 _D3D12_HEAP_TYPE_DEFAULT,
			CPUPageProperty:      _D3D12_CPU_PAGE_PROPERTY_UNKNOWN,
			MemoryPoolPreference: _D3D12_MEMORY_POOL_UNKNOWN,
			CreationNodeMask:     1,
			VisibleNodeMask:      1,
		}, _D3D12_HEAP_FLAG_ALLOW_DISPLAY, &_D3D12_RESOURCE_DESC{
			Dimension:        _D3D12_RESOURCE_DIMENSION_TEXTURE2D,
			Alignment:        0,
			Width:            uint64(width),
			Height:           uint32(height),
			DepthOrArraySize: 1,
			MipLevels:        1, // Use a single mipmap level
			Format:           _DXGI_FORMAT_B8G8R8A8_UNORM,
			SampleDesc: _DXGI_SAMPLE_DESC{
				Count:   1,
				Quality: 0,
			},
			Layout: _D3D12_TEXTURE_LAYOUT_UNKNOWN,
			Flags:  _D3D12_RESOURCE_FLAG_ALLOW_RENDER_TARGET,
		}, _D3D12_RESOURCE_STATE_PRESENT, &_D3D12_CLEAR_VALUE{
			Format: _DXGI_FORMAT_B8G8R8A8_UNORM,
		})
		if err != nil {
			return err
		}

		g.renderTargets[i] = r
		defer func(i int) {
			if ferr != nil {
				g.renderTargets[i].Release()
				g.renderTargets[i] = nil
			}
		}(i)

		g.device.CreateRenderTargetView(r, &_D3D12_RENDER_TARGET_VIEW_DESC{
			Format:        _DXGI_FORMAT_B8G8R8A8_UNORM,
			ViewDimension: _D3D12_RTV_DIMENSION_TEXTURE2D,
		}, h)
		h.Offset(1, g.rtvDescriptorSize)
	}

	return nil
}

func (g *Graphics) resizeSwapChainDesktop(width, height int) error {
	// All resources must be released before ResizeBuffers.
	if err := g.waitForCommandQueue(); err != nil {
		return err
	}
	g.releaseResources(g.frameIndex)

	for i := 0; i < frameCount; i++ {
		g.fenceValues[i] = g.fenceValues[g.frameIndex]
	}

	for _, r := range g.renderTargets {
		r.Release()
	}

	var flag uint32
	if g.allowTearing {
		flag |= uint32(_DXGI_SWAP_CHAIN_FLAG_ALLOW_TEARING)
	}
	if err := g.swapChain.ResizeBuffers(frameCount, uint32(width), uint32(height), _DXGI_FORMAT_B8G8R8A8_UNORM, flag); err != nil {
		return err
	}

	if err := g.createRenderTargetViewsDesktop(); err != nil {
		return err
	}

	return nil
}

func (g *Graphics) createRenderTargetViewsDesktop() (ferr error) {
	// Create frame resources.
	h, err := g.rtvDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}
	for i := 0; i < frameCount; i++ {
		r, err := g.swapChain.GetBuffer(uint32(i))
		if err != nil {
			return err
		}
		g.renderTargets[i] = r
		defer func(i int) {
			if ferr != nil {
				g.renderTargets[i].Release()
				g.renderTargets[i] = nil
			}
		}(i)

		g.device.CreateRenderTargetView(r, nil, h)
		h.Offset(1, g.rtvDescriptorSize)
	}

	return nil
}

func (g *Graphics) SetWindow(window uintptr) {
	g.window = windows.HWND(window)
	// TODO: need to update the swap chain?
}

func (g *Graphics) Begin() error {
	if microsoftgdk.IsXbox() && !g.frameStarted {
		select {
		case <-g.suspendingCh:
			if err := g.commandQueue.SuspendX(0); err != nil {
				return err
			}
			g.suspendedCh <- struct{}{}
			<-g.resumeCh
			if err := g.commandQueue.ResumeX(); err != nil {
				return err
			}
			if err := g.registerFrameEventForXbox(); err != nil {
				return err
			}
		default:
		}

		g.framePipelineToken = _D3D12XBOX_FRAME_PIPELINE_TOKEN_NULL
		if err := g.device.WaitFrameEventX(_D3D12XBOX_FRAME_EVENT_ORIGIN, windows.INFINITE, nil, _D3D12XBOX_WAIT_FRAME_EVENT_FLAG_NONE, &g.framePipelineToken); err != nil {
			return err
		}
	}
	g.frameStarted = true

	if g.prevBeginFrameIndex != g.frameIndex {
		if err := g.drawCommandAllocators[g.frameIndex].Reset(); err != nil {
			return err
		}
		if err := g.copyCommandAllocators[g.frameIndex].Reset(); err != nil {
			return err
		}
	}
	g.prevBeginFrameIndex = g.frameIndex

	if err := g.drawCommandList.Reset(g.drawCommandAllocators[g.frameIndex], nil); err != nil {
		return err
	}
	if err := g.copyCommandList.Reset(g.copyCommandAllocators[g.frameIndex], nil); err != nil {
		return err
	}
	return nil
}

func (g *Graphics) End(present bool) error {
	// The swap chain might still be nil when Begin-End is invoked not by a frame (e.g., Image.At).

	// As copyCommandList and drawCommandList are exclusive, the order should not matter here.
	if err := g.flushCommandList(g.copyCommandList); err != nil {
		return err
	}
	if err := g.copyCommandList.Close(); err != nil {
		return err
	}

	// screenImage can be nil in tests.
	if present && g.screenImage != nil {
		if rb, ok := g.screenImage.transiteState(_D3D12_RESOURCE_STATE_PRESENT); ok {
			g.drawCommandList.ResourceBarrier([]_D3D12_RESOURCE_BARRIER_Transition{rb})
		}
	}

	if err := g.drawCommandList.Close(); err != nil {
		return err
	}
	g.commandQueue.ExecuteCommandLists([]*_ID3D12GraphicsCommandList{g.drawCommandList})

	// Release vertices and indices buffers when too many ones were created.
	// The threshold is an arbitrary number.
	// This is needed espciallly for testings, where present is always false.
	if len(g.vertices[g.frameIndex]) >= 16 {
		if err := g.waitForCommandQueue(); err != nil {
			return err
		}
		g.releaseResources(g.frameIndex)
		g.releaseVerticesAndIndices(g.frameIndex)
	}

	g.pipelineStates.resetConstantBuffers(g.frameIndex)

	if present {
		if microsoftgdk.IsXbox() {
			if err := g.presentXbox(); err != nil {
				return err
			}
		} else {
			if err := g.presentDesktop(); err != nil {
				return err
			}
		}

		if g.newScreenWidth != 0 && g.newScreenHeight != 0 {
			if err := g.resizeSwapChainDesktop(g.newScreenWidth, g.newScreenHeight); err != nil {
				return err
			}
			g.newScreenWidth = 0
			g.newScreenHeight = 0
		}

		if err := g.moveToNextFrame(); err != nil {
			return err
		}

		g.releaseResources(g.frameIndex)
		g.releaseVerticesAndIndices(g.frameIndex)

		g.frameStarted = false
	}

	return nil
}

func (g *Graphics) presentDesktop() error {
	if g.swapChain == nil {
		return fmt.Errorf("directx: the swap chain is not initialized yet at End")
	}

	var syncInterval uint32
	var flags _DXGI_PRESENT
	if g.occluded {
		// The screen is not visible. Test whether we can resume.
		flags |= _DXGI_PRESENT_TEST
	} else {
		// Do actual rendering only when the screen is visible.
		if g.vsyncEnabled {
			syncInterval = 1
		} else if g.allowTearing {
			flags |= _DXGI_PRESENT_ALLOW_TEARING
		}
	}

	occluded, err := g.swapChain.Present(syncInterval, uint32(flags))
	if err != nil {
		return err
	}
	g.occluded = occluded

	// Reduce FPS when the screen is invisible.
	now := time.Now()
	if g.occluded {
		if delta := 100*time.Millisecond - now.Sub(g.lastTime); delta > 0 {
			time.Sleep(delta)
		}
	}
	g.lastTime = now

	return nil
}

func (g *Graphics) presentXbox() error {
	return g.commandQueue.PresentX(1, &_D3D12XBOX_PRESENT_PLANE_PARAMETERS{
		Token:         g.framePipelineToken,
		ResourceCount: 1,
		ppResources:   &g.renderTargets[g.frameIndex],
	}, nil)
}

func (g *Graphics) moveToNextFrame() error {
	fv := g.fenceValues[g.frameIndex]
	if err := g.commandQueue.Signal(g.fence, fv); err != nil {
		return err
	}

	// Update the frame index.
	if microsoftgdk.IsXbox() {
		g.frameIndex = (g.frameIndex + 1) % frameCount
	} else {
		g.frameIndex = int(g.swapChain.GetCurrentBackBufferIndex())
	}

	if g.fence.GetCompletedValue() < g.fenceValues[g.frameIndex] {
		if err := g.fence.SetEventOnCompletion(g.fenceValues[g.frameIndex], g.fenceWaitEvent); err != nil {
			return err
		}
		if _, err := windows.WaitForSingleObject(g.fenceWaitEvent, windows.INFINITE); err != nil {
			return err
		}
	}
	g.fenceValues[g.frameIndex] = fv + 1
	return nil
}

func (g *Graphics) releaseResources(frameIndex int) {
	for i, img := range g.disposedImages[frameIndex] {
		img.disposeImpl()
		g.disposedImages[frameIndex][i] = nil
	}
	g.disposedImages[frameIndex] = g.disposedImages[frameIndex][:0]

	for i, s := range g.disposedShaders[frameIndex] {
		s.disposeImpl()
		g.disposedShaders[frameIndex][i] = nil
	}
	g.disposedShaders[frameIndex] = g.disposedShaders[frameIndex][:0]
}

func (g *Graphics) releaseVerticesAndIndices(frameIndex int) {
	for i := range g.vertices[frameIndex] {
		g.vertices[frameIndex][i].Release()
		g.vertices[frameIndex][i] = nil
	}
	g.vertices[frameIndex] = g.vertices[frameIndex][:0]

	for i := range g.indices[frameIndex] {
		g.indices[frameIndex][i].Release()
		g.indices[frameIndex][i] = nil
	}
	g.indices[frameIndex] = g.indices[frameIndex][:0]
}

// flushCommandList executes commands in the command list and waits for its completion.
//
// TODO: This is not efficient. Is it possible to make two command lists work in parallel?
func (g *Graphics) flushCommandList(commandList *_ID3D12GraphicsCommandList) error {
	switch commandList {
	case g.drawCommandList:
		if !g.needFlushDrawCommandList {
			return nil
		}
		g.needFlushDrawCommandList = false
	case g.copyCommandList:
		if !g.needFlushCopyCommandList {
			return nil
		}
		g.needFlushCopyCommandList = false
	}

	if err := commandList.Close(); err != nil {
		return err
	}

	g.commandQueue.ExecuteCommandLists([]*_ID3D12GraphicsCommandList{commandList})

	if err := g.waitForCommandQueue(); err != nil {
		return err
	}

	switch commandList {
	case g.drawCommandList:
		if err := g.drawCommandAllocators[g.frameIndex].Reset(); err != nil {
			return err
		}
		if err := commandList.Reset(g.drawCommandAllocators[g.frameIndex], nil); err != nil {
			return err
		}
	case g.copyCommandList:
		if err := g.copyCommandAllocators[g.frameIndex].Reset(); err != nil {
			return err
		}
		if err := commandList.Reset(g.copyCommandAllocators[g.frameIndex], nil); err != nil {
			return err
		}

		for _, img := range g.images {
			img.releaseUploadingStagingBuffers()
		}
	}

	return nil
}

func (g *Graphics) waitForCommandQueue() error {
	fv := g.fenceValues[g.frameIndex]
	if err := g.commandQueue.Signal(g.fence, fv); err != nil {
		return err
	}
	if err := g.fence.SetEventOnCompletion(fv, g.fenceWaitEvent); err != nil {
		return err
	}
	if _, err := windows.WaitForSingleObject(g.fenceWaitEvent, windows.INFINITE); err != nil {
		return err
	}
	g.fenceValues[g.frameIndex]++
	return nil
}

func (g *Graphics) SetTransparent(transparent bool) {
	g.transparent = transparent
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint16) (ferr error) {
	// Create buffers if necessary.
	vidx := len(g.vertices[g.frameIndex])
	if cap(g.vertices[g.frameIndex]) > vidx {
		g.vertices[g.frameIndex] = g.vertices[g.frameIndex][:vidx+1]
	} else {
		g.vertices[g.frameIndex] = append(g.vertices[g.frameIndex], nil)
	}
	if g.vertices[g.frameIndex][vidx] == nil {
		// TODO: Use the default heap for efficiently. See the official example HelloTriangle.
		vs, err := createBuffer(g.device, graphics.IndicesCount*graphics.VertexFloatCount*uint64(unsafe.Sizeof(float32(0))), _D3D12_HEAP_TYPE_UPLOAD)
		if err != nil {
			return err
		}
		g.vertices[g.frameIndex][vidx] = vs
		defer func() {
			if ferr != nil {
				g.vertices[g.frameIndex][vidx].Release()
				g.vertices[g.frameIndex][vidx] = nil
			}
		}()
	}

	iidx := len(g.indices[g.frameIndex])
	if cap(g.indices[g.frameIndex]) > iidx {
		g.indices[g.frameIndex] = g.indices[g.frameIndex][:iidx+1]
	} else {
		g.indices[g.frameIndex] = append(g.indices[g.frameIndex], nil)
	}
	if g.indices[g.frameIndex][iidx] == nil {
		is, err := createBuffer(g.device, graphics.IndicesCount*uint64(unsafe.Sizeof(uint16(0))), _D3D12_HEAP_TYPE_UPLOAD)
		if err != nil {
			return err
		}
		g.indices[g.frameIndex][iidx] = is
		defer func() {
			if ferr != nil {
				g.indices[g.frameIndex][iidx].Release()
				g.indices[g.frameIndex][iidx] = nil
			}
		}()
	}

	m, err := g.vertices[g.frameIndex][vidx].Map(0, &_D3D12_RANGE{0, 0})
	if err != nil {
		return err
	}
	copy(unsafe.Slice((*float32)(unsafe.Pointer(m)), len(vertices)), vertices)
	g.vertices[g.frameIndex][vidx].Unmap(0, nil)

	m, err = g.indices[g.frameIndex][iidx].Map(0, &_D3D12_RANGE{0, 0})
	if err != nil {
		return err
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(m)), len(indices)), indices)
	g.indices[g.frameIndex][iidx].Unmap(0, nil)

	return nil
}

func (g *Graphics) NewImage(width, height int) (graphicsdriver.Image, error) {
	desc := _D3D12_RESOURCE_DESC{
		Dimension:        _D3D12_RESOURCE_DIMENSION_TEXTURE2D,
		Alignment:        0,
		Width:            uint64(graphics.InternalImageSize(width)),
		Height:           uint32(graphics.InternalImageSize(height)),
		DepthOrArraySize: 1,
		MipLevels:        0,
		Format:           _DXGI_FORMAT_R8G8B8A8_UNORM,
		SampleDesc: _DXGI_SAMPLE_DESC{
			Count:   1,
			Quality: 0,
		},
		Layout: _D3D12_TEXTURE_LAYOUT_UNKNOWN,
		Flags:  _D3D12_RESOURCE_FLAG_ALLOW_RENDER_TARGET,
	}

	state := _D3D12_RESOURCE_STATE_PIXEL_SHADER_RESOURCE
	t, err := g.device.CreateCommittedResource(&_D3D12_HEAP_PROPERTIES{
		Type:                 _D3D12_HEAP_TYPE_DEFAULT, // Upload?
		CPUPageProperty:      _D3D12_CPU_PAGE_PROPERTY_UNKNOWN,
		MemoryPoolPreference: _D3D12_MEMORY_POOL_UNKNOWN,
		CreationNodeMask:     1,
		VisibleNodeMask:      1,
	}, _D3D12_HEAP_FLAG_NONE, &desc, state, nil)
	if err != nil {
		return nil, err
	}

	i := &Image{
		graphics: g,
		id:       g.genNextImageID(),
		width:    width,
		height:   height,
		texture:  t,
		states:   [frameCount]_D3D12_RESOURCE_STATES{state},
	}
	g.addImage(i)
	return i, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
	if err := g.updateSwapChain(width, height); err != nil {
		return nil, err
	}

	i := &Image{
		graphics: g,
		id:       g.genNextImageID(),
		width:    width,
		height:   height,
		screen:   true,
		states:   [frameCount]_D3D12_RESOURCE_STATES{0, 0},
	}
	g.addImage(i)
	g.screenImage = i
	return i, nil
}

func (g *Graphics) addImage(img *Image) {
	if g.images == nil {
		g.images = map[graphicsdriver.ImageID]*Image{}
	}
	if _, ok := g.images[img.id]; ok {
		panic(fmt.Sprintf("directx: image ID %d was already registered", img.id))
	}
	g.images[img.id] = img
}

func (g *Graphics) removeImage(img *Image) {
	delete(g.images, img.id)
	g.disposedImages[g.frameIndex] = append(g.disposedImages[g.frameIndex], img)
}

func (g *Graphics) addShader(s *Shader) {
	if g.shaders == nil {
		g.shaders = map[graphicsdriver.ShaderID]*Shader{}
	}
	if _, ok := g.shaders[s.id]; ok {
		panic(fmt.Sprintf("directx: shader ID %d was already registered", s.id))
	}
	g.shaders[s.id] = s
}

func (g *Graphics) removeShader(s *Shader) {
	delete(g.shaders, s.id)
	g.disposedShaders[g.frameIndex] = append(g.disposedShaders[g.frameIndex], s)
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	g.vsyncEnabled = enabled
}

func (g *Graphics) NeedsRestoring() bool {
	return false
}

func (g *Graphics) NeedsClearingScreen() bool {
	// TODO: Confirm this is really true.
	return true
}

func (g *Graphics) IsGL() bool {
	return false
}

func (g *Graphics) IsDirectX() bool {
	return true
}

func (g *Graphics) MaxImageSize() int {
	return _D3D12_REQ_TEXTURE2D_U_OR_V_DIMENSION
}

func (g *Graphics) NewShader(program *shaderir.Program) (graphicsdriver.Shader, error) {
	vs, ps, offsets := hlsl.Compile(program)
	vsh, psh, err := newShader(vs, ps)
	if err != nil {
		return nil, err
	}

	s := &Shader{
		graphics:       g,
		id:             g.genNextShaderID(),
		uniformTypes:   program.Uniforms,
		uniformOffsets: offsets,
		vertexShader:   vsh,
		pixelShader:    psh,
	}
	g.addShader(s)
	return s, nil
}

func (g *Graphics) DrawTriangles(dstID graphicsdriver.ImageID, srcs [graphics.ShaderImageCount]graphicsdriver.ImageID, shaderID graphicsdriver.ShaderID, dstRegions []graphicsdriver.DstRegion, indexOffset int, blend graphicsdriver.Blend, uniforms []uint32, evenOdd bool) error {
	if shaderID == graphicsdriver.InvalidShaderID {
		return fmt.Errorf("directx: shader ID is invalid")
	}

	if err := g.flushCommandList(g.copyCommandList); err != nil {
		return err
	}

	// Release constant buffers when too many ones will be created.
	numPipelines := 1
	if evenOdd {
		numPipelines = 2
	}
	if len(g.pipelineStates.constantBuffers[g.frameIndex])+numPipelines > numDescriptorsPerFrame {
		if err := g.flushCommandList(g.drawCommandList); err != nil {
			return err
		}
		g.pipelineStates.releaseConstantBuffers(g.frameIndex)
	}

	dst := g.images[dstID]
	var resourceBarriers []_D3D12_RESOURCE_BARRIER_Transition
	if rb, ok := dst.transiteState(_D3D12_RESOURCE_STATE_RENDER_TARGET); ok {
		resourceBarriers = append(resourceBarriers, rb)
	}

	var srcImages [graphics.ShaderImageCount]*Image
	for i, srcID := range srcs {
		src := g.images[srcID]
		if src == nil {
			continue
		}
		srcImages[i] = src
		if rb, ok := src.transiteState(_D3D12_RESOURCE_STATE_PIXEL_SHADER_RESOURCE); ok {
			resourceBarriers = append(resourceBarriers, rb)
		}
	}

	if len(resourceBarriers) > 0 {
		g.drawCommandList.ResourceBarrier(resourceBarriers)
	}

	if err := dst.setAsRenderTarget(g.drawCommandList, g.device, evenOdd); err != nil {
		return err
	}

	shader := g.shaders[shaderID]
	adjustedUniforms := shader.adjustUniforms(uniforms, shader)

	w, h := dst.internalSize()
	g.needFlushDrawCommandList = true
	g.drawCommandList.RSSetViewports([]_D3D12_VIEWPORT{
		{
			TopLeftX: 0,
			TopLeftY: 0,
			Width:    float32(w),
			Height:   float32(h),
			MinDepth: _D3D12_MIN_DEPTH,
			MaxDepth: _D3D12_MAX_DEPTH,
		},
	})
	g.drawCommandList.IASetPrimitiveTopology(_D3D_PRIMITIVE_TOPOLOGY_TRIANGLELIST)
	g.drawCommandList.IASetVertexBuffers(0, []_D3D12_VERTEX_BUFFER_VIEW{
		{
			BufferLocation: g.vertices[g.frameIndex][len(g.vertices[g.frameIndex])-1].GetGPUVirtualAddress(),
			SizeInBytes:    graphics.IndicesCount * graphics.VertexFloatCount * uint32(unsafe.Sizeof(float32(0))),
			StrideInBytes:  graphics.VertexFloatCount * uint32(unsafe.Sizeof(float32(0))),
		},
	})
	g.drawCommandList.IASetIndexBuffer(&_D3D12_INDEX_BUFFER_VIEW{
		BufferLocation: g.indices[g.frameIndex][len(g.indices[g.frameIndex])-1].GetGPUVirtualAddress(),
		SizeInBytes:    graphics.IndicesCount * uint32(unsafe.Sizeof(uint16(0))),
		Format:         _DXGI_FORMAT_R16_UINT,
	})

	if err := g.pipelineStates.drawTriangles(g.device, g.drawCommandList, g.frameIndex, dst.screen, srcImages, shader, dstRegions, adjustedUniforms, blend, indexOffset, evenOdd); err != nil {
		return err
	}

	return nil
}

func (g *Graphics) genNextImageID() graphicsdriver.ImageID {
	g.nextImageID++
	return g.nextImageID
}

func (g *Graphics) genNextShaderID() graphicsdriver.ShaderID {
	g.nextShaderID++
	return g.nextShaderID
}

type Image struct {
	graphics *Graphics
	id       graphicsdriver.ImageID
	width    int
	height   int
	screen   bool

	states            [frameCount]_D3D12_RESOURCE_STATES
	texture           *_ID3D12Resource
	stencil           *_ID3D12Resource
	rtvDescriptorHeap *_ID3D12DescriptorHeap
	dsvDescriptorHeap *_ID3D12DescriptorHeap

	uploadingStagingBuffers []*_ID3D12Resource
}

func (i *Image) ID() graphicsdriver.ImageID {
	return i.id
}

func (i *Image) Dispose() {
	// Dipose the images later as this image might still be used.
	i.graphics.removeImage(i)
}

func (i *Image) disposeImpl() {
	if i.dsvDescriptorHeap != nil {
		i.dsvDescriptorHeap.Release()
		i.dsvDescriptorHeap = nil
	}
	if i.rtvDescriptorHeap != nil {
		i.rtvDescriptorHeap.Release()
		i.rtvDescriptorHeap = nil
	}
	if i.stencil != nil {
		i.stencil.Release()
		i.stencil = nil
	}
	if i.texture != nil {
		i.texture.Release()
		i.texture = nil
	}
}

func (*Image) IsInvalidated() bool {
	return false
}

func (i *Image) ReadPixels(buf []byte, x, y, width, height int) error {
	if i.screen {
		return errors.New("directx: Pixels cannot be called on the screen")
	}

	if err := i.graphics.flushCommandList(i.graphics.drawCommandList); err != nil {
		return err
	}

	desc := _D3D12_RESOURCE_DESC{
		Dimension:        _D3D12_RESOURCE_DIMENSION_TEXTURE2D,
		Alignment:        0,
		Width:            uint64(width),
		Height:           uint32(height),
		DepthOrArraySize: 1,
		MipLevels:        0,
		Format:           _DXGI_FORMAT_R8G8B8A8_UNORM,
		SampleDesc: _DXGI_SAMPLE_DESC{
			Count:   1,
			Quality: 0,
		},
		Layout: _D3D12_TEXTURE_LAYOUT_UNKNOWN,
		Flags:  _D3D12_RESOURCE_FLAG_ALLOW_RENDER_TARGET,
	}
	layouts, _, _, totalBytes := i.graphics.device.GetCopyableFootprints(&desc, 0, 1, 0)
	readingStagingBuffer, err := createBuffer(i.graphics.device, totalBytes, _D3D12_HEAP_TYPE_READBACK)
	if err != nil {
		return err
	}
	defer func() {
		readingStagingBuffer.Release()
	}()

	if rb, ok := i.transiteState(_D3D12_RESOURCE_STATE_COPY_SOURCE); ok {
		i.graphics.copyCommandList.ResourceBarrier([]_D3D12_RESOURCE_BARRIER_Transition{rb})
	}

	m, err := readingStagingBuffer.Map(0, &_D3D12_RANGE{0, 0})
	if err != nil {
		return err
	}

	dst := _D3D12_TEXTURE_COPY_LOCATION_PlacedFootPrint{
		pResource:       readingStagingBuffer,
		Type:            _D3D12_TEXTURE_COPY_TYPE_PLACED_FOOTPRINT,
		PlacedFootprint: layouts,
	}
	src := _D3D12_TEXTURE_COPY_LOCATION_SubresourceIndex{
		pResource:        i.texture,
		Type:             _D3D12_TEXTURE_COPY_TYPE_SUBRESOURCE_INDEX,
		SubresourceIndex: 0,
	}
	i.graphics.needFlushCopyCommandList = true
	i.graphics.copyCommandList.CopyTextureRegion_PlacedFootPrint_SubresourceIndex(
		&dst, 0, 0, 0, &src, &_D3D12_BOX{
			left:   uint32(x),
			top:    uint32(y),
			front:  0,
			right:  uint32(x + width),
			bottom: uint32(y + height),
			back:   1,
		})

	if err := i.graphics.flushCommandList(i.graphics.copyCommandList); err != nil {
		return err
	}

	dstBytes := unsafe.Slice((*byte)(unsafe.Pointer(m)), totalBytes)
	for j := 0; j < height; j++ {
		copy(buf[j*width*4:(j+1)*width*4], dstBytes[j*int(layouts.Footprint.RowPitch):])
	}

	readingStagingBuffer.Unmap(0, nil)

	return nil
}

func (i *Image) WritePixels(args []*graphicsdriver.WritePixelsArgs) error {
	if i.screen {
		return errors.New("directx: WritePixels cannot be called on the screen")
	}

	if err := i.graphics.flushCommandList(i.graphics.drawCommandList); err != nil {
		return err
	}

	minX := i.width
	minY := i.height
	maxX := 0
	maxY := 0
	for _, a := range args {
		if minX > a.X {
			minX = a.X
		}
		if minY > a.Y {
			minY = a.Y
		}
		if maxX < a.X+a.Width {
			maxX = a.X + a.Width
		}
		if maxY < a.Y+a.Height {
			maxY = a.Y + a.Height
		}
	}

	desc := _D3D12_RESOURCE_DESC{
		Dimension:        _D3D12_RESOURCE_DIMENSION_TEXTURE2D,
		Alignment:        0,
		Width:            uint64(maxX - minX),
		Height:           uint32(maxY - minY),
		DepthOrArraySize: 1,
		MipLevels:        0,
		Format:           _DXGI_FORMAT_R8G8B8A8_UNORM,
		SampleDesc: _DXGI_SAMPLE_DESC{
			Count:   1,
			Quality: 0,
		},
		Layout: _D3D12_TEXTURE_LAYOUT_UNKNOWN,
		Flags:  _D3D12_RESOURCE_FLAG_ALLOW_RENDER_TARGET,
	}
	layouts, _, _, totalBytes := i.graphics.device.GetCopyableFootprints(&desc, 0, 1, 0)
	uploadingStagingBuffer, err := createBuffer(i.graphics.device, totalBytes, _D3D12_HEAP_TYPE_UPLOAD)
	if err != nil {
		return err
	}
	i.uploadingStagingBuffers = append(i.uploadingStagingBuffers, uploadingStagingBuffer)

	if rb, ok := i.transiteState(_D3D12_RESOURCE_STATE_COPY_DEST); ok {
		i.graphics.copyCommandList.ResourceBarrier([]_D3D12_RESOURCE_BARRIER_Transition{rb})
	}

	m, err := uploadingStagingBuffer.Map(0, &_D3D12_RANGE{0, 0})
	if err != nil {
		return err
	}

	i.graphics.needFlushCopyCommandList = true

	srcBytes := unsafe.Slice((*byte)(unsafe.Pointer(m)), totalBytes)
	for _, a := range args {
		for j := 0; j < a.Height; j++ {
			copy(srcBytes[((a.Y-minY)+j)*int(layouts.Footprint.RowPitch)+(a.X-minX)*4:], a.Pixels[j*a.Width*4:(j+1)*a.Width*4])
		}
	}

	for _, a := range args {
		dst := _D3D12_TEXTURE_COPY_LOCATION_SubresourceIndex{
			pResource:        i.texture,
			Type:             _D3D12_TEXTURE_COPY_TYPE_SUBRESOURCE_INDEX,
			SubresourceIndex: 0,
		}
		src := _D3D12_TEXTURE_COPY_LOCATION_PlacedFootPrint{
			pResource:       uploadingStagingBuffer,
			Type:            _D3D12_TEXTURE_COPY_TYPE_PLACED_FOOTPRINT,
			PlacedFootprint: layouts,
		}
		i.graphics.copyCommandList.CopyTextureRegion_SubresourceIndex_PlacedFootPrint(
			&dst, uint32(a.X), uint32(a.Y), 0, &src, &_D3D12_BOX{
				left:   uint32(a.X - minX),
				top:    uint32(a.Y - minY),
				front:  0,
				right:  uint32(a.X - minX + a.Width),
				bottom: uint32(a.Y - minY + a.Height),
				back:   1,
			})
	}

	uploadingStagingBuffer.Unmap(0, nil)

	return nil
}

func (i *Image) resource() *_ID3D12Resource {
	if i.screen {
		return i.graphics.renderTargets[i.graphics.frameIndex]
	}
	return i.texture
}

func (i *Image) state() _D3D12_RESOURCE_STATES {
	if i.screen {
		return i.states[i.graphics.frameIndex]
	}
	return i.states[0]
}

func (i *Image) setState(newState _D3D12_RESOURCE_STATES) {
	if i.screen {
		i.states[i.graphics.frameIndex] = newState
		return
	}
	i.states[0] = newState
}

func (i *Image) transiteState(newState _D3D12_RESOURCE_STATES) (_D3D12_RESOURCE_BARRIER_Transition, bool) {
	if i.state() == newState {
		return _D3D12_RESOURCE_BARRIER_Transition{}, false
	}
	oldState := i.state()
	i.setState(newState)

	return _D3D12_RESOURCE_BARRIER_Transition{
		Type:  _D3D12_RESOURCE_BARRIER_TYPE_TRANSITION,
		Flags: _D3D12_RESOURCE_BARRIER_FLAG_NONE,
		Transition: _D3D12_RESOURCE_TRANSITION_BARRIER{
			pResource:   i.resource(),
			Subresource: _D3D12_RESOURCE_BARRIER_ALL_SUBRESOURCES,
			StateBefore: oldState,
			StateAfter:  newState,
		},
	}, true
}

func (i *Image) internalSize() (int, int) {
	if i.screen {
		return i.width, i.height
	}
	return graphics.InternalImageSize(i.width), graphics.InternalImageSize(i.height)
}

func (i *Image) setAsRenderTarget(drawCommandList *_ID3D12GraphicsCommandList, device *_ID3D12Device, useStencil bool) error {
	if err := i.ensureRenderTargetView(device); err != nil {
		return err
	}

	if i.screen {
		if useStencil {
			return fmt.Errorf("directx: stencils are not available on the screen framebuffer")
		}
		rtv, err := i.graphics.rtvDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
		if err != nil {
			return err
		}
		rtv.Offset(int32(i.graphics.frameIndex), i.graphics.rtvDescriptorSize)
		drawCommandList.OMSetRenderTargets([]_D3D12_CPU_DESCRIPTOR_HANDLE{rtv}, false, nil)
		return nil
	}

	rtv, err := i.rtvDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}

	if !useStencil {
		drawCommandList.OMSetRenderTargets([]_D3D12_CPU_DESCRIPTOR_HANDLE{rtv}, false, nil)
		return nil
	}

	if err := i.ensureDepthStencilView(device); err != nil {
		return err
	}
	dsv, err := i.dsvDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}
	drawCommandList.OMSetStencilRef(0)
	drawCommandList.OMSetRenderTargets([]_D3D12_CPU_DESCRIPTOR_HANDLE{rtv}, false, &dsv)
	drawCommandList.ClearDepthStencilView(dsv, _D3D12_CLEAR_FLAG_STENCIL, 0, 0, nil)

	return nil
}

func (i *Image) ensureRenderTargetView(device *_ID3D12Device) error {
	if i.screen {
		return nil
	}

	if i.rtvDescriptorHeap != nil {
		return nil
	}

	h, err := device.CreateDescriptorHeap(&_D3D12_DESCRIPTOR_HEAP_DESC{
		Type:           _D3D12_DESCRIPTOR_HEAP_TYPE_RTV,
		NumDescriptors: 1,
		Flags:          _D3D12_DESCRIPTOR_HEAP_FLAG_NONE,
		NodeMask:       0,
	})
	if err != nil {
		return err
	}
	i.rtvDescriptorHeap = h

	rtv, err := i.rtvDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}
	device.CreateRenderTargetView(i.texture, nil, rtv)

	return nil
}

func (i *Image) ensureDepthStencilView(device *_ID3D12Device) error {
	if i.screen {
		return fmt.Errorf("directx: stencils are not available on the screen framebuffer")
	}

	if i.dsvDescriptorHeap != nil {
		return nil
	}

	h, err := device.CreateDescriptorHeap(&_D3D12_DESCRIPTOR_HEAP_DESC{
		Type:           _D3D12_DESCRIPTOR_HEAP_TYPE_DSV,
		NumDescriptors: 1,
		Flags:          _D3D12_DESCRIPTOR_HEAP_FLAG_NONE,
		NodeMask:       0,
	})
	if err != nil {
		return err
	}
	i.dsvDescriptorHeap = h

	dsv, err := i.dsvDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}
	if i.stencil == nil {
		s, err := device.CreateCommittedResource(&_D3D12_HEAP_PROPERTIES{
			Type:                 _D3D12_HEAP_TYPE_DEFAULT,
			CPUPageProperty:      _D3D12_CPU_PAGE_PROPERTY_UNKNOWN,
			MemoryPoolPreference: _D3D12_MEMORY_POOL_UNKNOWN,
			CreationNodeMask:     1,
			VisibleNodeMask:      1,
		}, _D3D12_HEAP_FLAG_NONE, &_D3D12_RESOURCE_DESC{
			Dimension:        _D3D12_RESOURCE_DIMENSION_TEXTURE2D,
			Alignment:        0,
			Width:            uint64(graphics.InternalImageSize(i.width)),
			Height:           uint32(graphics.InternalImageSize(i.height)),
			DepthOrArraySize: 1,
			MipLevels:        0,
			Format:           _DXGI_FORMAT_D24_UNORM_S8_UINT,
			SampleDesc: _DXGI_SAMPLE_DESC{
				Count:   1,
				Quality: 0,
			},
			Layout: _D3D12_TEXTURE_LAYOUT_UNKNOWN,
			Flags:  _D3D12_RESOURCE_FLAG_ALLOW_DEPTH_STENCIL,
		}, _D3D12_RESOURCE_STATE_DEPTH_WRITE, &_D3D12_CLEAR_VALUE{
			Format: _DXGI_FORMAT_D24_UNORM_S8_UINT,
		})
		if err != nil {
			return err
		}
		i.stencil = s
	}
	device.CreateDepthStencilView(i.stencil, nil, dsv)

	return nil
}

func (i *Image) releaseUploadingStagingBuffers() {
	for idx, buf := range i.uploadingStagingBuffers {
		buf.Release()
		i.uploadingStagingBuffers[idx] = nil
	}
	i.uploadingStagingBuffers = i.uploadingStagingBuffers[:0]
}

type stencilMode int

const (
	prepareStencil stencilMode = iota
	drawWithStencil
	noStencil
)

type pipelineStateKey struct {
	blend       graphicsdriver.Blend
	stencilMode stencilMode
	screen      bool
}

type Shader struct {
	graphics       *Graphics
	id             graphicsdriver.ShaderID
	uniformTypes   []shaderir.Type
	uniformOffsets []int
	vertexShader   *_ID3DBlob
	pixelShader    *_ID3DBlob
	pipelineStates map[pipelineStateKey]*_ID3D12PipelineState
}

func (s *Shader) ID() graphicsdriver.ShaderID {
	return s.id
}

func (s *Shader) Dispose() {
	s.graphics.removeShader(s)
}

func (s *Shader) disposeImpl() {
	for c, p := range s.pipelineStates {
		p.Release()
		delete(s.pipelineStates, c)
	}

	if s.pixelShader != nil {
		s.pixelShader.Release()
		s.pixelShader = nil
	}
	if s.vertexShader != nil {
		count := s.vertexShader.Release()
		if count == 0 {
			for k, v := range vertexShaderCache {
				if v == s.vertexShader {
					delete(vertexShaderCache, k)
				}
			}
		}
		s.vertexShader = nil
	}
}

func (s *Shader) pipelineState(blend graphicsdriver.Blend, stencilMode stencilMode, screen bool) (*_ID3D12PipelineState, error) {
	key := pipelineStateKey{
		blend:       blend,
		stencilMode: stencilMode,
		screen:      screen,
	}
	if state, ok := s.pipelineStates[key]; ok {
		return state, nil
	}

	state, err := s.graphics.pipelineStates.newPipelineState(s.graphics.device, s.vertexShader, s.pixelShader, blend, stencilMode, screen)
	if err != nil {
		return nil, err
	}
	if s.pipelineStates == nil {
		s.pipelineStates = map[pipelineStateKey]*_ID3D12PipelineState{}
	}
	s.pipelineStates[key] = state
	return state, nil
}

func (s *Shader) adjustUniforms(uniforms []uint32, shader *Shader) []uint32 {
	var fs []uint32
	var idx int
	for i, typ := range shader.uniformTypes {
		if len(fs) < s.uniformOffsets[i]/4 {
			fs = append(fs, make([]uint32, s.uniformOffsets[i]/4-len(fs))...)
		}

		n := typ.Uint32Count()
		switch typ.Main {
		case shaderir.Float:
			fs = append(fs, uniforms[idx:idx+1]...)
		case shaderir.Int:
			fs = append(fs, uniforms[idx:idx+1]...)
		case shaderir.Vec2, shaderir.IVec2:
			fs = append(fs, uniforms[idx:idx+2]...)
		case shaderir.Vec3, shaderir.IVec3:
			fs = append(fs, uniforms[idx:idx+3]...)
		case shaderir.Vec4, shaderir.IVec4:
			fs = append(fs, uniforms[idx:idx+4]...)
		case shaderir.Mat2:
			fs = append(fs,
				uniforms[idx+0], uniforms[idx+2], 0, 0,
				uniforms[idx+1], uniforms[idx+3],
			)
		case shaderir.Mat3:
			fs = append(fs,
				uniforms[idx+0], uniforms[idx+3], uniforms[idx+6], 0,
				uniforms[idx+1], uniforms[idx+4], uniforms[idx+7], 0,
				uniforms[idx+2], uniforms[idx+5], uniforms[idx+8],
			)
		case shaderir.Mat4:
			if i == graphics.ProjectionMatrixUniformVariableIndex {
				// In DirectX, the NDC's Y direction (upward) and the framebuffer's Y direction (downward) don't
				// match. Then, the Y direction must be inverted.
				// Invert the sign bits as float32 values.
				fs = append(fs,
					uniforms[idx+0], uniforms[idx+4], uniforms[idx+8], uniforms[idx+12],
					uniforms[idx+1]^(1<<31), uniforms[idx+5]^(1<<31), uniforms[idx+9]^(1<<31), uniforms[idx+13]^(1<<31),
					uniforms[idx+2], uniforms[idx+6], uniforms[idx+10], uniforms[idx+14],
					uniforms[idx+3], uniforms[idx+7], uniforms[idx+11], uniforms[idx+15],
				)
			} else {
				fs = append(fs,
					uniforms[idx+0], uniforms[idx+4], uniforms[idx+8], uniforms[idx+12],
					uniforms[idx+1], uniforms[idx+5], uniforms[idx+9], uniforms[idx+13],
					uniforms[idx+2], uniforms[idx+6], uniforms[idx+10], uniforms[idx+14],
					uniforms[idx+3], uniforms[idx+7], uniforms[idx+11], uniforms[idx+15],
				)
			}
		case shaderir.Array:
			// Each element is aligned to the boundary.
			switch typ.Sub[0].Main {
			case shaderir.Float:
				for j := 0; j < typ.Length; j++ {
					fs = append(fs, uniforms[idx+j])
					if j < typ.Length-1 {
						fs = append(fs, 0, 0, 0)
					}
				}
			case shaderir.Int:
				for j := 0; j < typ.Length; j++ {
					fs = append(fs, uniforms[idx+j])
					if j < typ.Length-1 {
						fs = append(fs, 0, 0, 0)
					}
				}
			case shaderir.Vec2, shaderir.IVec2:
				for j := 0; j < typ.Length; j++ {
					fs = append(fs, uniforms[idx+2*j:idx+2*(j+1)]...)
					if j < typ.Length-1 {
						fs = append(fs, 0, 0)
					}
				}
			case shaderir.Vec3, shaderir.IVec3:
				for j := 0; j < typ.Length; j++ {
					fs = append(fs, uniforms[idx+3*j:idx+3*(j+1)]...)
					if j < typ.Length-1 {
						fs = append(fs, 0)
					}
				}
			case shaderir.Vec4, shaderir.IVec4:
				fs = append(fs, uniforms[idx:idx+4*typ.Length]...)
			case shaderir.Mat2:
				for j := 0; j < typ.Length; j++ {
					u := uniforms[idx+4*j : idx+4*(j+1)]
					fs = append(fs,
						u[0], u[2], 0, 0,
						u[1], u[3], 0, 0,
					)
				}
				if typ.Length > 0 {
					fs = fs[:len(fs)-2]
				}
			case shaderir.Mat3:
				for j := 0; j < typ.Length; j++ {
					u := uniforms[idx+9*j : idx+9*(j+1)]
					fs = append(fs,
						u[0], u[3], u[6], 0,
						u[1], u[4], u[7], 0,
						u[2], u[5], u[8], 0,
					)
				}
				if typ.Length > 0 {
					fs = fs[:len(fs)-1]
				}
			case shaderir.Mat4:
				for j := 0; j < typ.Length; j++ {
					u := uniforms[idx+16*j : idx+16*(j+1)]
					fs = append(fs,
						u[0], u[4], u[8], u[12],
						u[1], u[5], u[9], u[13],
						u[2], u[6], u[10], u[14],
						u[3], u[7], u[11], u[15],
					)
				}
			default:
				panic(fmt.Sprintf("directx: not implemented type for uniform variables: %s", typ.String()))
			}
		default:
			panic(fmt.Sprintf("directx: not implemented type for uniform variables: %s", typ.String()))
		}

		idx += n
	}
	return fs
}
