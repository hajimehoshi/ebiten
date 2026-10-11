// Copyright 2026 The Ebitengine Authors
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
	"image"
	"math"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type readback12 struct {
	graphics   *graphics12
	staging    *_ID3D12Resource
	fence      *_ID3D12Fence
	region     image.Rectangle
	args       []graphicsdriver.PixelsArgs
	stride     int
	totalBytes uint64
	recorded   bool
}

var _ graphicsdriver.AsyncPixelsReader = (*image12)(nil)

func (i *image12) ReadPixelsAsync(args []graphicsdriver.PixelsArgs) (_ graphicsdriver.PixelsReadback, err error) {
	if i.screen {
		return nil, errors.New("directx: ReadPixelsAsync cannot be called on the screen")
	}
	region, err := readbackRegion(args)
	if err != nil {
		return nil, err
	}
	r := &readback12{graphics: i.graphics, region: region, args: append([]graphicsdriver.PixelsArgs(nil), args...)}
	if region.Empty() {
		return r, nil
	}
	defer func() {
		if err != nil {
			r.Dispose()
		}
	}()
	desc := _D3D12_RESOURCE_DESC{
		Dimension:        _D3D12_RESOURCE_DIMENSION_TEXTURE2D,
		Width:            uint64(region.Dx()),
		Height:           uint32(region.Dy()),
		DepthOrArraySize: 1,
		MipLevels:        1,
		Format:           _DXGI_FORMAT_R8G8B8A8_UNORM,
		SampleDesc:       _DXGI_SAMPLE_DESC{Count: 1},
		Layout:           _D3D12_TEXTURE_LAYOUT_UNKNOWN,
	}
	layout, _, _, totalBytes := i.graphics.device.GetCopyableFootprints(&desc, 0, 1, 0)
	r.stride = int(layout.Footprint.RowPitch)
	r.totalBytes = totalBytes
	r.staging, err = createBuffer(i.graphics.device, totalBytes, _D3D12_HEAP_TYPE_READBACK)
	if err != nil {
		return nil, err
	}
	r.fence, err = i.graphics.device.CreateFence(0, _D3D12_FENCE_FLAG_NONE)
	if err != nil {
		return nil, err
	}
	// Submit earlier draws without resetting their allocator or waiting for the GPU.
	if err = i.graphics.submitCommandList(i.graphics.drawCommandList); err != nil {
		return nil, err
	}
	if rb, ok := i.transiteState(_D3D12_RESOURCE_STATE_COPY_SOURCE); ok {
		i.graphics.copyCommandList.ResourceBarrier([]_D3D12_RESOURCE_BARRIER_Transition{rb})
	}
	dst := _D3D12_TEXTURE_COPY_LOCATION_PlacedFootprint{
		pResource:       r.staging,
		Type:            _D3D12_TEXTURE_COPY_TYPE_PLACED_FOOTPRINT,
		PlacedFootprint: layout,
	}
	src := _D3D12_TEXTURE_COPY_LOCATION_SubresourceIndex{
		pResource: i.texture,
		Type:      _D3D12_TEXTURE_COPY_TYPE_SUBRESOURCE_INDEX,
	}
	r.recorded = true
	i.graphics.needFlushCopyCommandList = true
	i.graphics.copyCommandList.CopyTextureRegion_PlacedFootprint_SubresourceIndex(&dst, 0, 0, 0, &src, &_D3D12_BOX{
		left:   uint32(region.Min.X),
		top:    uint32(region.Min.Y),
		right:  uint32(region.Max.X),
		bottom: uint32(region.Max.Y),
		back:   1,
	})
	if err = i.graphics.submitCommandList(i.graphics.copyCommandList); err != nil {
		return nil, err
	}
	// A private fence keeps completion independent of frame-fence reuse and presentation.
	if err = i.graphics.commandQueue.Signal(r.fence, 1); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *readback12) Poll() (bool, error) {
	if r.staging == nil {
		return true, nil
	}
	switch r.fence.GetCompletedValue() {
	case 0:
		return false, nil
	case math.MaxUint64:
		if err := r.graphics.device.GetDeviceRemovedReason(); err != nil {
			return true, err
		}
		return true, errors.New("directx: the device was removed during a pixel read-back")
	}
	m, err := r.staging.mapPointer(0, &_D3D12_RANGE{Begin: 0, End: uintptr(r.totalBytes)})
	if err != nil {
		return true, err
	}
	defer r.staging.Unmap(0, &_D3D12_RANGE{})
	copyReadbackPixels(r.args, r.region, unsafe.Slice((*byte)(m), r.totalBytes), r.stride)
	return true, nil
}

func (r *readback12) Dispose() {
	r.args = nil
	if r.recorded && r.fence.GetCompletedValue() == 0 {
		// D3D12 does not retain resources for submitted commands. An aborted read-back must
		// keep its buffer and fence until the frame's existing queue-completion wait.
		g := r.graphics
		g.disposedReadbacks[g.frameIndex] = append(g.disposedReadbacks[g.frameIndex], r)
		return
	}
	r.disposeImpl()
}

func (r *readback12) disposeImpl() {
	if r.staging != nil {
		r.staging.Release()
		r.staging = nil
	}
	if r.fence != nil {
		r.fence.Release()
		r.fence = nil
	}
}
