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
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/hlsl"
)

type resourceWithSize struct {
	value       *_ID3D12Resource
	sizeInBytes uint32
}

func (r *resourceWithSize) release() {
	r.value.Release()
	r.value = nil
	r.sizeInBytes = 0
}

type graphics12 struct {
	debug              *_ID3D12Debug
	device             *_ID3D12Device
	commandQueue       *_ID3D12CommandQueue
	rtvDescriptorHeap  *_ID3D12DescriptorHeap
	rtvDescriptorSize  uint32
	renderTargets      [frameCount]*_ID3D12Resource
	framePipelineToken _D3D12XBOX_FRAME_PIPELINE_TOKEN

	fences         [frameCount]*_ID3D12Fence
	fenceValues    [frameCount]uint64
	fenceWaitEvent windows.Handle

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

	vertices [frameCount][]*resourceWithSize
	indices  [frameCount][]*resourceWithSize

	graphicsInfra *graphicsInfra

	window windows.HWND

	frameIndex          int
	prevBeginFrameIndex int

	// frameStarted is true since Begin until End with present
	frameStarted bool

	images         map[graphicsdriver.ImageID]*image12
	screenImage    *image12
	nextImageID    graphicsdriver.ImageID
	disposedImages [frameCount][]*image12

	shaders         map[graphicsdriver.ShaderID]*shader12
	nextShaderID    graphicsdriver.ShaderID
	disposedShaders [frameCount][]*shader12
	tmpUniforms     []uint32

	vsyncEnabled bool

	newScreenWidth  int
	newScreenHeight int

	suspendingCh chan struct{}
	suspendedCh  chan struct{}
	resumeCh     chan struct{}

	pipelineStates
}

func newGraphics12(useWARP bool, useDebugLayer bool, featureLevel _D3D_FEATURE_LEVEL) (*graphics12, error) {
	g := &graphics12{}

	// Initialize not only a device but also other members like a fence.
	// Even if initializing a device succeeds, initializing a fence might fail (#2142).
	if microsoftgdk.IsXbox() {
		if err := g.initializeXbox(useWARP, useDebugLayer); err != nil {
			return nil, err
		}
	} else {
		if err := g.initializeDesktop(useWARP, useDebugLayer, featureLevel); err != nil {
			return nil, err
		}
	}

	return g, nil
}

func (g *graphics12) initializeDesktop(useWARP bool, useDebugLayer bool, featureLevel _D3D_FEATURE_LEVEL) (ferr error) {
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

	f, err := _CreateDXGIFactory()
	if err != nil {
		return err
	}
	gi, err := newGraphicsInfra(f)
	if err != nil {
		return err
	}
	g.graphicsInfra = gi
	defer func() {
		if ferr != nil {
			g.graphicsInfra.release()
			g.graphicsInfra = nil
		}
	}()

	adapters, err := g.graphicsInfra.appendAdapters(nil, useWARP)
	if err != nil {
		return err
	}
	defer func() {
		for _, a := range adapters {
			a.Release()
		}
	}()

	var adapter *_IDXGIAdapter1
	if useWARP {
		if len(adapters) > 0 {
			adapter = adapters[0]
		}
	} else {
		for _, a := range adapters {
			desc, err := a.GetDesc1()
			if err != nil {
				continue
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

func (g *graphics12) initializeXbox(useWARP bool, useDebugLayer bool) (ferr error) {
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

func (g *graphics12) registerFrameEventForXbox() error {
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

func (g *graphics12) initializeMembers() (ferr error) {
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
	for i := range frameCount {
		f, err := g.device.CreateFence(0, _D3D12_FENCE_FLAG_NONE)
		if err != nil {
			return err
		}
		g.fences[i] = f
		defer func() {
			if ferr != nil {
				g.fences[i].Release()
				g.fences[i] = nil
			}
		}()
	}

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

func (g *graphics12) Initialize() (err error) {
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

func (g *graphics12) updateSwapChain(width, height int) error {
	if g.window == 0 {
		return errors.New("directx: the window handle is not initialized yet")
	}

	if microsoftgdk.IsXbox() {
		if err := g.initSwapChainXbox(width, height); err != nil {
			return err
		}
		return nil
	}

	if !g.graphicsInfra.isSwapChainInited() {
		if err := g.initSwapChainDesktop(width, height); err != nil {
			return err
		}
		return nil
	}

	g.newScreenWidth = width
	g.newScreenHeight = height

	return nil
}

func (g *graphics12) initSwapChainDesktop(width, height int) error {
	if err := g.graphicsInfra.initSwapChain(width, height, unsafe.Pointer(g.commandQueue), g.window); err != nil {
		return err
	}

	// TODO: Get the current buffer index?

	if err := g.createRenderTargetViewsDesktop(); err != nil {
		return err
	}

	idx, err := g.graphicsInfra.currentBackBufferIndex()
	if err != nil {
		return err
	}
	g.frameIndex = idx

	return nil
}

func (g *graphics12) initSwapChainXbox(width, height int) (ferr error) {
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
		}, _D3D12_RESOURCE_STATE_PRESENT(), &_D3D12_CLEAR_VALUE{
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

func (g *graphics12) resizeSwapChainDesktop(width, height int) error {
	// All resources must be released before ResizeBuffers.
	if err := g.waitForCommandQueue(); err != nil {
		return err
	}
	g.releaseResources(g.frameIndex)

	for _, r := range g.renderTargets {
		r.Release()
	}

	if err := g.graphicsInfra.resizeSwapChain(width, height); err != nil {
		return err
	}

	if err := g.createRenderTargetViewsDesktop(); err != nil {
		return err
	}

	return nil
}

func (g *graphics12) createRenderTargetViewsDesktop() (ferr error) {
	// Create frame resources.
	h, err := g.rtvDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}
	for i := 0; i < frameCount; i++ {
		r, err := g.graphicsInfra.getBuffer(uint32(i), &_IID_ID3D12Resource)
		if err != nil {
			return err
		}
		g.renderTargets[i] = (*_ID3D12Resource)(r)
		defer func(i int) {
			if ferr != nil {
				g.renderTargets[i].Release()
				g.renderTargets[i] = nil
			}
		}(i)

		g.device.CreateRenderTargetView((*_ID3D12Resource)(r), nil, h)
		h.Offset(1, g.rtvDescriptorSize)
	}

	return nil
}

func (g *graphics12) SetWindow(window uintptr) {
	g.window = windows.HWND(window)
	// TODO: need to update the swap chain?
}

func (g *graphics12) Begin() error {
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

func (g *graphics12) End(present bool) error {
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
		if rb, ok := g.screenImage.transiteState(_D3D12_RESOURCE_STATE_PRESENT()); ok {
			g.drawCommandList.ResourceBarrier([]_D3D12_RESOURCE_BARRIER_Transition{rb})
		}
	}

	if err := g.drawCommandList.Close(); err != nil {
		return err
	}
	g.commandQueue.ExecuteCommandLists([]*_ID3D12GraphicsCommandList{g.drawCommandList})

	// Release vertices and indices buffers when too many ones were created.
	// The threshold is an arbitrary number.
	// This is needed especially for testings, where present is always false.
	if len(g.vertices[g.frameIndex]) >= 16 {
		if err := g.waitForCommandQueue(); err != nil {
			return err
		}
		g.releaseResources(g.frameIndex)
		g.resetVerticesAndIndices(g.frameIndex, true)
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
			g.screenImage.width = g.newScreenWidth
			g.screenImage.height = g.newScreenHeight
			g.newScreenWidth = 0
			g.newScreenHeight = 0
		}

		if err := g.moveToNextFrame(); err != nil {
			return err
		}

		g.releaseResources(g.frameIndex)
		g.resetVerticesAndIndices(g.frameIndex, false)

		g.frameStarted = false
	}

	return nil
}

func (g *graphics12) presentDesktop() error {
	return g.graphicsInfra.present(g.vsyncEnabled)
}

func (g *graphics12) presentXbox() error {
	var pinner runtime.Pinner
	pinner.Pin(&g.renderTargets[g.frameIndex])
	defer pinner.Unpin()
	return g.commandQueue.PresentX(1, &_D3D12XBOX_PRESENT_PLANE_PARAMETERS{
		Token:         g.framePipelineToken,
		ResourceCount: 1,
		ppResources:   &g.renderTargets[g.frameIndex],
	}, nil)
}

func (g *graphics12) moveToNextFrame() error {
	g.fenceValues[g.frameIndex]++
	fv := g.fenceValues[g.frameIndex]
	if err := g.commandQueue.Signal(g.fences[g.frameIndex], fv); err != nil {
		return err
	}

	// Update the frame index.
	if microsoftgdk.IsXbox() {
		g.frameIndex = (g.frameIndex + 1) % frameCount
	} else {
		idx, err := g.graphicsInfra.currentBackBufferIndex()
		if err != nil {
			return err
		}
		g.frameIndex = idx
	}

	if g.fences[g.frameIndex].GetCompletedValue() < g.fenceValues[g.frameIndex] {
		if err := g.fences[g.frameIndex].SetEventOnCompletion(g.fenceValues[g.frameIndex], g.fenceWaitEvent); err != nil {
			return err
		}
		if _, err := windows.WaitForSingleObject(g.fenceWaitEvent, windows.INFINITE); err != nil {
			return err
		}
	}
	return nil
}

func (g *graphics12) releaseResources(frameIndex int) {
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

func (g *graphics12) resetVerticesAndIndices(frameIndex int, release bool) {
	if release {
		for i := range g.vertices[frameIndex] {
			g.vertices[frameIndex][i].release()
			g.vertices[frameIndex][i] = nil
		}
	}
	g.vertices[frameIndex] = g.vertices[frameIndex][:0]

	if release {
		for i := range g.indices[frameIndex] {
			g.indices[frameIndex][i].release()
			g.indices[frameIndex][i] = nil
		}
	}
	g.indices[frameIndex] = g.indices[frameIndex][:0]
}

// flushCommandList executes commands in the command list and waits for its completion.
//
// TODO: This is not efficient. Is it possible to make two command lists work in parallel?
func (g *graphics12) flushCommandList(commandList *_ID3D12GraphicsCommandList) error {
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

func (g *graphics12) waitForCommandQueue() error {
	g.fenceValues[g.frameIndex]++
	fv := g.fenceValues[g.frameIndex]
	if err := g.commandQueue.Signal(g.fences[g.frameIndex], fv); err != nil {
		return err
	}
	if err := g.fences[g.frameIndex].SetEventOnCompletion(fv, g.fenceWaitEvent); err != nil {
		return err
	}
	if _, err := windows.WaitForSingleObject(g.fenceWaitEvent, windows.INFINITE); err != nil {
		return err
	}
	return nil
}

func (g *graphics12) SetTransparent(transparent bool) {
	// TODO: Implement this?
}

func (g *graphics12) SetVertices(vertices []float32, indices []uint32) (ferr error) {
	// Create buffers if necessary.
	vidx := len(g.vertices[g.frameIndex])
	if cap(g.vertices[g.frameIndex]) > vidx {
		g.vertices[g.frameIndex] = g.vertices[g.frameIndex][:vidx+1]
	} else {
		g.vertices[g.frameIndex] = append(g.vertices[g.frameIndex], nil)
	}
	vsize := pow2(uint32(len(vertices)) * uint32(unsafe.Sizeof(vertices[0])))
	if g.vertices[g.frameIndex][vidx] != nil && g.vertices[g.frameIndex][vidx].sizeInBytes < vsize {
		g.vertices[g.frameIndex][vidx].release()
		g.vertices[g.frameIndex][vidx] = nil
	}
	if g.vertices[g.frameIndex][vidx] == nil {
		// TODO: Use the default heap for efficiently. See the official example HelloTriangle.
		vs, err := createBuffer(g.device, uint64(vsize), _D3D12_HEAP_TYPE_UPLOAD)
		if err != nil {
			return err
		}
		g.vertices[g.frameIndex][vidx] = &resourceWithSize{
			value:       vs,
			sizeInBytes: vsize,
		}
		defer func() {
			if ferr != nil {
				g.vertices[g.frameIndex][vidx].release()
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
	isize := pow2(uint32(len(indices)) * uint32(unsafe.Sizeof(indices[0])))
	if g.indices[g.frameIndex][iidx] != nil && g.indices[g.frameIndex][iidx].sizeInBytes < isize {
		g.indices[g.frameIndex][iidx].release()
		g.indices[g.frameIndex][iidx] = nil
	}
	if g.indices[g.frameIndex][iidx] == nil {
		is, err := createBuffer(g.device, uint64(isize), _D3D12_HEAP_TYPE_UPLOAD)
		if err != nil {
			return err
		}
		g.indices[g.frameIndex][iidx] = &resourceWithSize{
			value:       is,
			sizeInBytes: isize,
		}
		defer func() {
			if ferr != nil {
				g.indices[g.frameIndex][iidx].release()
				g.indices[g.frameIndex][iidx] = nil
			}
		}()
	}

	m, err := g.vertices[g.frameIndex][vidx].value.Map(0, &_D3D12_RANGE{0, 0})
	if err != nil {
		return err
	}
	copy(unsafe.Slice((*float32)(unsafe.Pointer(m)), len(vertices)), vertices)
	g.vertices[g.frameIndex][vidx].value.Unmap(0, nil)

	m, err = g.indices[g.frameIndex][iidx].value.Map(0, &_D3D12_RANGE{0, 0})
	if err != nil {
		return err
	}
	copy(unsafe.Slice((*uint32)(unsafe.Pointer(m)), len(indices)), indices)
	g.indices[g.frameIndex][iidx].value.Unmap(0, nil)

	return nil
}

func (g *graphics12) NewImage(width, height int) (graphicsdriver.Image, error) {
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

	i := &image12{
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

func (g *graphics12) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
	imageWidth := width
	imageHeight := height
	if g.screenImage != nil {
		imageWidth = g.screenImage.width
		imageHeight = g.screenImage.height
		g.screenImage.Dispose()
		g.screenImage = nil
	}

	if err := g.updateSwapChain(width, height); err != nil {
		return nil, err
	}

	i := &image12{
		graphics: g,
		id:       g.genNextImageID(),
		width:    imageWidth,
		height:   imageHeight,
		screen:   true,
		states:   [frameCount]_D3D12_RESOURCE_STATES{0, 0},
	}
	g.addImage(i)
	g.screenImage = i
	return i, nil
}

func (g *graphics12) addImage(img *image12) {
	if g.images == nil {
		g.images = map[graphicsdriver.ImageID]*image12{}
	}
	if _, ok := g.images[img.id]; ok {
		panic(fmt.Sprintf("directx: image ID %d was already registered", img.id))
	}
	g.images[img.id] = img
}

func (g *graphics12) removeImage(img *image12) {
	delete(g.images, img.id)
	g.disposedImages[g.frameIndex] = append(g.disposedImages[g.frameIndex], img)
}

func (g *graphics12) addShader(s *shader12) {
	if g.shaders == nil {
		g.shaders = map[graphicsdriver.ShaderID]*shader12{}
	}
	if _, ok := g.shaders[s.id]; ok {
		panic(fmt.Sprintf("directx: shader ID %d was already registered", s.id))
	}
	g.shaders[s.id] = s
}

func (g *graphics12) removeShader(s *shader12) {
	delete(g.shaders, s.id)
	g.disposedShaders[g.frameIndex] = append(g.disposedShaders[g.frameIndex], s)
}

func (g *graphics12) SetVsyncEnabled(enabled bool) {
	g.vsyncEnabled = enabled
}

func (g *graphics12) NeedsClearingScreen() bool {
	// TODO: Confirm this is really true.
	return true
}

func (g *graphics12) MaxImageSize() int {
	return _D3D12_REQ_TEXTURE2D_U_OR_V_DIMENSION
}

func (g *graphics12) NewShader(program *shaderir.Program) (graphicsdriver.Shader, error) {
	vsh, psh, err := compileShader(program)
	if err != nil {
		return nil, err
	}

	s := &shader12{
		graphics:       g,
		id:             g.genNextShaderID(),
		uniformTypes:   program.Uniforms,
		uniformOffsets: hlsl.UniformVariableOffsetsInDwords(program),
		vertexShader:   vsh,
		pixelShader:    psh,
	}
	g.addShader(s)
	return s, nil
}

func (g *graphics12) DrawTriangles(dstID graphicsdriver.ImageID, srcs [graphics.ShaderSrcImageCount]graphicsdriver.ImageID, shaderID graphicsdriver.ShaderID, dstRegions []graphicsdriver.DstRegion, indexOffset int, blend graphicsdriver.Blend, uniforms []uint32, fillRule graphicsdriver.FillRule) error {
	if shaderID == graphicsdriver.InvalidShaderID {
		return fmt.Errorf("directx: shader ID is invalid")
	}

	if err := g.flushCommandList(g.copyCommandList); err != nil {
		return err
	}

	// Release constant buffers when too many ones will be created.
	numPipelines := 1
	if fillRule != graphicsdriver.FillRuleFillAll {
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

	var srcImages [graphics.ShaderSrcImageCount]*image12
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

	if err := dst.setAsRenderTarget(g.drawCommandList, g.device, fillRule != graphicsdriver.FillRuleFillAll); err != nil {
		return err
	}

	shader := g.shaders[shaderID]
	g.tmpUniforms = appendAdjustedUniforms(g.tmpUniforms[:0], shader.uniformTypes, shader.uniformOffsets, uniforms)

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
			BufferLocation: g.vertices[g.frameIndex][len(g.vertices[g.frameIndex])-1].value.GetGPUVirtualAddress(),
			SizeInBytes:    g.vertices[g.frameIndex][len(g.vertices[g.frameIndex])-1].sizeInBytes,
			StrideInBytes:  graphics.VertexFloatCount * uint32(unsafe.Sizeof(float32(0))),
		},
	})
	g.drawCommandList.IASetIndexBuffer(&_D3D12_INDEX_BUFFER_VIEW{
		BufferLocation: g.indices[g.frameIndex][len(g.indices[g.frameIndex])-1].value.GetGPUVirtualAddress(),
		SizeInBytes:    g.indices[g.frameIndex][len(g.indices[g.frameIndex])-1].sizeInBytes,
		Format:         _DXGI_FORMAT_R32_UINT,
	})

	if err := g.pipelineStates.drawTriangles(g.device, g.drawCommandList, g.frameIndex, dst.screen, srcImages, shader, dstRegions, g.tmpUniforms, blend, indexOffset, fillRule); err != nil {
		return err
	}

	return nil
}

func (g *graphics12) genNextImageID() graphicsdriver.ImageID {
	g.nextImageID++
	return g.nextImageID
}

func (g *graphics12) genNextShaderID() graphicsdriver.ShaderID {
	g.nextShaderID++
	return g.nextShaderID
}
