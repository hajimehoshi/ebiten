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
	renderTargetView   *_ID3D11RenderTargetView
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
	if i.renderTargetView != nil {
		i.renderTargetView.Release()
		i.renderTargetView = nil
	}
	if i.shaderResourceView != nil {
		i.shaderResourceView.Release()
		i.shaderResourceView = nil
	}
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

func (i *image11) setAsRenderTarget() error {
	if i.renderTargetView == nil {
		rtv, err := i.graphics.device.CreateRenderTargetView(unsafe.Pointer(i.texture), nil)
		if err != nil {
			return err
		}
		i.renderTargetView = rtv
	}

	i.graphics.deviceContext.OMSetRenderTargets([]*_ID3D11RenderTargetView{i.renderTargetView})
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
