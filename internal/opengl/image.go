// Copyright 2018 The Ebiten Authors
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

package opengl

import (
	"github.com/hajimehoshi/ebiten/internal/math"
)

type Image struct {
	Texture     Texture
	Framebuffer *Framebuffer
	width       int
	height      int
}

func NewImage(width, height int) *Image {
	return &Image{
		width:  width,
		height: height,
	}
}

func (i *Image) Size() (int, int) {
	return i.width, i.height
}

func (i *Image) IsInvalidated() bool {
	return !theContext.isTexture(i.Texture)
}

func (i *Image) Delete() {
	if i.Framebuffer != nil {
		i.Framebuffer.delete()
	}
	if i.Texture != *new(Texture) {
		theContext.deleteTexture(i.Texture)
	}
}

func (i *Image) SetViewport() error {
	if err := i.ensureFramebuffer(); err != nil {
		return err
	}
	theContext.setViewport(i.Framebuffer)
	return nil
}

func (i *Image) Pixels() ([]byte, error) {
	if err := i.ensureFramebuffer(); err != nil {
		return nil, err
	}
	w, h := i.Size()
	p, err := theContext.framebufferPixels(i.Framebuffer, w, h)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (i *Image) ProjectionMatrix() []float32 {
	if i.Framebuffer == nil {
		panic("not reached")
	}
	return i.Framebuffer.projectionMatrix()
}

func (i *Image) ensureFramebuffer() error {
	if i.Framebuffer != nil {
		return nil
	}
	w, h := i.Size()
	f, err := newFramebufferFromTexture(i.Texture, math.NextPowerOf2Int(w), math.NextPowerOf2Int(h))
	if err != nil {
		return err
	}
	i.Framebuffer = f
	return nil
}
