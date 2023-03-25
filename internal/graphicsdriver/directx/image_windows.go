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
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type Image struct {
	graphics *Graphics
	id       graphicsdriver.ImageID
	width    int
	height   int
	screen   bool

	*image12
}

type image12 struct {
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
