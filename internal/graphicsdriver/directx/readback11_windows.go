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
}

var _ graphicsdriver.AsyncPixelsReader = (*image11)(nil)

func (i *image11) ReadPixelsAsync(args []graphicsdriver.PixelsArgs) (graphicsdriver.PixelsReadback, error) {
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
	staging, err := i.graphics.device.CreateTexture2D(&_D3D11_TEXTURE2D_DESC{
		Width:          uint32(region.Dx()),
		Height:         uint32(region.Dy()),
		MipLevels:      1,
		ArraySize:      1,
		Format:         _DXGI_FORMAT_R8G8B8A8_UNORM,
		SampleDesc:     _DXGI_SAMPLE_DESC{Count: 1},
		Usage:          _D3D11_USAGE_STAGING,
		CPUAccessFlags: uint32(_D3D11_CPU_ACCESS_READ),
	}, nil)
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

func (r *readback11) Poll() (bool, error) {
	if r.staging == nil {
		return true, nil
	}
	var mapped _D3D11_MAPPED_SUBRESOURCE
	if err := r.graphics.deviceContext.Map(unsafe.Pointer(r.staging), 0, _D3D11_MAP_READ, uint32(_D3D11_MAP_FLAG_DO_NOT_WAIT), &mapped); err != nil {
		if errors.Is(err, _DXGI_ERROR_WAS_STILL_DRAWING) {
			return false, nil
		}
		return true, err
	}
	defer r.graphics.deviceContext.Unmap(unsafe.Pointer(r.staging), 0)
	stride := int(mapped.RowPitch)
	copyReadbackPixels(r.args, r.region, unsafe.Slice((*byte)(mapped.pData), stride*r.region.Dy()), stride)
	return true, nil
}

func (r *readback11) Dispose() {
	// The immediate context retains resources referenced by its outstanding commands.
	if r.staging != nil {
		r.staging.Release()
		r.staging = nil
	}
	r.args = nil
}
