// Copyright 2016 Hajime Hoshi
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

package graphics

import (
	"image"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/math"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

type Image struct {
	texture     *texture
	framebuffer *framebuffer
	width       int
	height      int
}

const MaxImageSize = viewportSize

func NewImage(width, height int, filter opengl.Filter) *Image {
	i := &Image{
		width:  width,
		height: height,
	}
	c := &newImageCommand{
		result: i,
		width:  width,
		height: height,
		filter: filter,
	}
	theCommandQueue.Enqueue(c)
	return i
}

func NewImageFromImage(img *image.RGBA, width, height int, filter opengl.Filter) *Image {
	i := &Image{
		width:  width,
		height: height,
	}
	c := &newImageFromImageCommand{
		result: i,
		img:    img,
		filter: filter,
	}
	theCommandQueue.Enqueue(c)
	return i
}

func NewScreenFramebufferImage(width, height int, offsetX, offsetY float64) *Image {
	i := &Image{
		width:  width,
		height: height,
	}
	c := &newScreenFramebufferImageCommand{
		result:  i,
		width:   width,
		height:  height,
		offsetX: offsetX,
		offsetY: offsetY,
	}
	theCommandQueue.Enqueue(c)
	return i
}

func (i *Image) Dispose() {
	c := &disposeCommand{
		target: i,
	}
	theCommandQueue.Enqueue(c)
}

func (i *Image) Size() (int, int) {
	return i.width, i.height
}

func (i *Image) Fill(r, g, b, a uint8) {
	c := &fillCommand{
		dst: i,
	}
	c.color.R = r
	c.color.G = g
	c.color.B = b
	c.color.A = a
	theCommandQueue.Enqueue(c)
}

func (i *Image) DrawImage(src *Image, vertices []float32, clr *affine.ColorM, mode opengl.CompositeMode) {
	theCommandQueue.EnqueueDrawImageCommand(i, src, vertices, clr, mode)
}

func (i *Image) Pixels() ([]uint8, error) {
	// Flush the enqueued commands so that pixels are certainly read.
	if err := theCommandQueue.Flush(); err != nil {
		return nil, err
	}
	f, err := i.createFramebufferIfNeeded()
	if err != nil {
		return nil, err
	}
	return opengl.GetContext().FramebufferPixels(f.native, math.NextPowerOf2Int(i.width), math.NextPowerOf2Int(i.height))
}

func (i *Image) ReplacePixels(p []uint8) {
	pixels := make([]uint8, len(p))
	copy(pixels, p)
	c := &replacePixelsCommand{
		dst:    i,
		pixels: pixels,
	}
	theCommandQueue.Enqueue(c)
}

func (i *Image) IsInvalidated() bool {
	return !opengl.GetContext().IsTexture(i.texture.native)
}

func (i *Image) createFramebufferIfNeeded() (*framebuffer, error) {
	if i.framebuffer != nil {
		return i.framebuffer, nil
	}
	f, err := newFramebufferFromTexture(i.texture)
	if err != nil {
		return nil, err
	}
	i.framebuffer = f
	return i.framebuffer, nil
}
