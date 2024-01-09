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
	"image"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type image11 struct {
	graphics *graphics11
	id       graphicsdriver.ImageID
	width    int
	height   int
	screen   bool

	texture            *_ID3D11Texture2D
	stencil            *_ID3D11Texture2D
	renderTargetView   *_ID3D11RenderTargetView
	stencilView        *_ID3D11DepthStencilView
	shaderResourceView *_ID3D11ShaderResourceView
}

func (i *image11) internalSize() (int, int) {
	if i.screen {
		return i.width, i.height
	}
	return graphics.InternalImageSize(i.width), graphics.InternalImageSize(i.height)
}

func (i *image11) ID() graphicsdriver.ImageID {
	return i.id
}

func (i *image11) Dispose() {
	i.disposeBuffers()
	i.graphics.removeImage(i)
}

func (i *image11) disposeBuffers() {
	if i.texture != nil {
		i.texture.Release()
		i.texture = nil
	}
	if i.stencil != nil {
		i.stencil.Release()
		i.stencil = nil
	}
	if i.renderTargetView != nil {
		i.renderTargetView.Release()
		i.renderTargetView = nil
	}
	if i.stencilView != nil {
		i.stencilView.Release()
		i.stencilView = nil
	}
	if i.shaderResourceView != nil {
		i.shaderResourceView.Release()
		i.shaderResourceView = nil
	}
}

func (i *image11) ReadPixels(args []graphicsdriver.PixelsArgs) error {
	var unionRegion image.Rectangle
	for _, a := range args {
		unionRegion = unionRegion.Union(a.Region)
	}

	staging, err := i.graphics.device.CreateTexture2D(&_D3D11_TEXTURE2D_DESC{
		Width:     uint32(unionRegion.Dx()),
		Height:    uint32(unionRegion.Dy()),
		MipLevels: 0,
		ArraySize: 1,
		Format:    _DXGI_FORMAT_R8G8B8A8_UNORM,
		SampleDesc: _DXGI_SAMPLE_DESC{
			Count:   1,
			Quality: 0,
		},
		Usage:          _D3D11_USAGE_STAGING,
		BindFlags:      0,
		CPUAccessFlags: uint32(_D3D11_CPU_ACCESS_READ),
		MiscFlags:      0,
	}, nil)
	if err != nil {
		return err
	}
	defer staging.Release()

	i.graphics.deviceContext.CopySubresourceRegion(unsafe.Pointer(staging), 0, 0, 0, 0, unsafe.Pointer(i.texture), 0, &_D3D11_BOX{
		left:   uint32(unionRegion.Min.X),
		top:    uint32(unionRegion.Min.Y),
		front:  0,
		right:  uint32(unionRegion.Max.X),
		bottom: uint32(unionRegion.Max.Y),
		back:   1,
	})

	var mapped _D3D11_MAPPED_SUBRESOURCE
	if err := i.graphics.deviceContext.Map(unsafe.Pointer(staging), 0, _D3D11_MAP_READ, 0, &mapped); err != nil {
		return err
	}

	stride := int(mapped.RowPitch)
	srcPix := unsafe.Slice((*byte)(mapped.pData), stride*unionRegion.Dy())
	for _, a := range args {
		w := a.Region.Dx()
		if unionRegion == a.Region && stride == 4*w {
			copy(a.Pixels, srcPix)
			continue
		}
		offset := 4*(a.Region.Min.X-unionRegion.Min.X) + stride*(a.Region.Min.Y-unionRegion.Min.Y)
		for j := 0; j < a.Region.Dy(); j++ {
			copy(a.Pixels[j*4*w:(j+1)*4*w], srcPix[offset+j*stride:])
		}
	}

	i.graphics.deviceContext.Unmap(unsafe.Pointer(staging), 0)

	return nil
}

func (i *image11) WritePixels(args []graphicsdriver.PixelsArgs) error {
	for _, a := range args {
		i.graphics.deviceContext.UpdateSubresource(unsafe.Pointer(i.texture), 0, &_D3D11_BOX{
			left:   uint32(a.Region.Min.X),
			top:    uint32(a.Region.Min.Y),
			front:  0,
			right:  uint32(a.Region.Max.X),
			bottom: uint32(a.Region.Max.Y),
			back:   1,
		}, unsafe.Pointer(&a.Pixels[0]), uint32(4*a.Region.Dx()), 0)
	}
	return nil
}

func (i *image11) setAsRenderTarget(useStencil bool) error {
	if i.renderTargetView == nil {
		rtv, err := i.graphics.device.CreateRenderTargetView(unsafe.Pointer(i.texture), nil)
		if err != nil {
			return err
		}
		i.renderTargetView = rtv
	}

	if !useStencil {
		i.graphics.deviceContext.OMSetRenderTargets([]*_ID3D11RenderTargetView{i.renderTargetView}, nil)
		return nil
	}

	if i.screen {
		return fmt.Errorf("directx: a stencil buffer is not available for a screen image")
	}

	if i.stencil == nil {
		w, h := i.internalSize()
		s, err := i.graphics.device.CreateTexture2D(&_D3D11_TEXTURE2D_DESC{
			Width:     uint32(w),
			Height:    uint32(h),
			MipLevels: 0,
			ArraySize: 1,
			Format:    _DXGI_FORMAT_D24_UNORM_S8_UINT,
			SampleDesc: _DXGI_SAMPLE_DESC{
				Count:   1,
				Quality: 0,
			},
			Usage:          _D3D11_USAGE_DEFAULT,
			BindFlags:      uint32(_D3D11_BIND_DEPTH_STENCIL),
			CPUAccessFlags: 0,
			MiscFlags:      0,
		}, nil)
		if err != nil {
			return err
		}
		i.stencil = s
	}

	if i.stencilView == nil {
		sv, err := i.graphics.device.CreateDepthStencilView(unsafe.Pointer(i.stencil), nil)
		if err != nil {
			return err
		}
		i.stencilView = sv
	}

	i.graphics.deviceContext.OMSetRenderTargets([]*_ID3D11RenderTargetView{i.renderTargetView}, i.stencilView)
	i.graphics.deviceContext.ClearDepthStencilView(i.stencilView, uint8(_D3D11_CLEAR_STENCIL), 0, 0)

	return nil
}

func (i *image11) getShaderResourceView() (*_ID3D11ShaderResourceView, error) {
	if i.shaderResourceView == nil {
		srv, err := i.graphics.device.CreateShaderResourceView(unsafe.Pointer(i.texture), nil)
		if err != nil {
			return nil, err
		}
		i.shaderResourceView = srv
	}
	return i.shaderResourceView, nil
}
