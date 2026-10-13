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
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type readback11 struct {
	graphics *graphics11
	staging  *_ID3D11Texture2D
	region   image.Rectangle
	args     []graphicsdriver.PixelsArgs

	// reusable is set only after the map and the copy have succeeded.
	reusable bool
}

func (i *image11) ReadPixelsAsync(args []graphicsdriver.PixelsArgs) (graphicsdriver.PixelReadback, error) {
	if i.screen {
		return nil, errors.New("directx: ReadPixelsAsync cannot be called on the screen")
	}
	region, err := readbackRegion(args)
	if err != nil {
		return nil, err
	}
	r := &readback11{
		graphics: i.graphics,
		region:   region,
		args:     args,
	}
	if region.Empty() {
		return r, nil
	}
	staging, err := i.graphics.acquireStagingTexture(region.Size())
	if err != nil {
		return nil, err
	}
	r.staging = staging
	i.graphics.deviceContext.CopySubresourceRegion(unsafe.Pointer(staging), 0, 0, 0, 0, unsafe.Pointer(i.texture), 0, &_D3D11_BOX{
		left:   uint32(region.Min.X),
		top:    uint32(region.Min.Y),
		right:  uint32(region.Max.X),
		bottom: uint32(region.Max.Y),
		back:   1,
	})
	// Submit the copy even if no later draw or presentation flushes the immediate context.
	i.graphics.deviceContext.Flush()
	return r, nil
}

func (r *readback11) Wait() error {
	_, err := r.readPixels(0)
	return err
}

func (r *readback11) Poll() (bool, error) {
	return r.readPixels(uint32(_D3D11_MAP_FLAG_DO_NOT_WAIT))
}

func (r *readback11) readPixels(flags uint32) (bool, error) {
	if r.staging == nil {
		return true, nil
	}
	var mapped _D3D11_MAPPED_SUBRESOURCE
	if err := r.graphics.deviceContext.Map(unsafe.Pointer(r.staging), 0, _D3D11_MAP_READ, flags, &mapped); err != nil {
		if flags&uint32(_D3D11_MAP_FLAG_DO_NOT_WAIT) != 0 && errors.Is(err, _DXGI_ERROR_WAS_STILL_DRAWING) {
			return false, nil
		}
		return true, err
	}
	defer r.graphics.deviceContext.Unmap(unsafe.Pointer(r.staging), 0)
	stride := int(mapped.RowPitch)
	copyReadbackPixels(r.args, r.region, unsafe.Slice((*byte)(mapped.pData), stride*r.region.Dy()), stride)
	r.reusable = true
	return true, nil
}

func (r *readback11) Dispose() {
	if r.staging != nil {
		if r.reusable {
			r.graphics.releaseStagingTexture(r.staging, r.region.Size())
		} else {
			// The immediate context retains resources referenced by its outstanding commands.
			r.staging.Release()
		}
		r.staging = nil
	}
	r.args = nil
}

// acquireStagingTexture returns a staging texture of size pixels that the CPU can read.
func (g *graphics11) acquireStagingTexture(size image.Point) (*_ID3D11Texture2D, error) {
	if t, ok := g.stagingTextures.Take(size); ok {
		return t, nil
	}
	return g.device.CreateTexture2D(&_D3D11_TEXTURE2D_DESC{
		Width:          uint32(size.X),
		Height:         uint32(size.Y),
		MipLevels:      1,
		ArraySize:      1,
		Format:         _DXGI_FORMAT_R8G8B8A8_UNORM,
		SampleDesc:     _DXGI_SAMPLE_DESC{Count: 1},
		Usage:          _D3D11_USAGE_STAGING,
		CPUAccessFlags: uint32(_D3D11_CPU_ACCESS_READ),
	}, nil)
}

// releaseStagingTexture caches t, an unmapped texture of size pixels whose read completed
// successfully, for reuse, or releases it.
func (g *graphics11) releaseStagingTexture(t *_ID3D11Texture2D, size image.Point) {
	for _, dropped := range g.stagingTextures.Put(size, t, 4*size.X*size.Y) {
		dropped.Release()
	}
}
