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
	"image/color"

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type Image struct {
	texture     *texture
	framebuffer *framebuffer
}

func NewImage(width, height int, filter opengl.Filter) (*Image, error) {
	i := &Image{}
	c := &newImageCommand{
		result: i,
		width:  width,
		height: height,
		filter: filter,
	}
	theCommandQueue.Enqueue(c)
	return i, nil
}

func NewImageFromImage(img *image.RGBA, filter opengl.Filter) (*Image, error) {
	i := &Image{}
	c := &newImageFromImageCommand{
		result: i,
		img:    img,
		filter: filter,
	}
	theCommandQueue.Enqueue(c)
	return i, nil
}

func NewScreenFramebufferImage(width, height int) (*Image, error) {
	i := &Image{}
	c := &newScreenFramebufferImageCommand{
		result: i,
		width:  width,
		height: height,
	}
	theCommandQueue.Enqueue(c)
	return i, nil
}

func (i *Image) Dispose() error {
	c := &disposeCommand{
		target: i,
	}
	theCommandQueue.Enqueue(c)
	return nil
}

func (i *Image) Fill(clr color.Color) error {
	c := &fillCommand{
		dst:   i,
		color: clr,
	}
	theCommandQueue.Enqueue(c)
	return nil
}

func (i *Image) DrawImage(src *Image, vertices []int16, geo, clr Matrix, mode opengl.CompositeMode) error {
	c := &drawImageCommand{
		dst:      i,
		src:      src,
		vertices: vertices,
		geo:      geo,
		color:    clr,
		mode:     mode,
	}
	theCommandQueue.Enqueue(c)
	return nil
}

func (i *Image) Pixels(context *opengl.Context) ([]uint8, error) {
	// Flush the enqueued commands so that pixels are certainly read.
	if err := theCommandQueue.Flush(context); err != nil {
		return nil, err
	}
	f := i.framebuffer
	return context.FramebufferPixels(f.native, f.width, f.height)
}

func (i *Image) ReplacePixels(p []uint8) error {
	c := &replacePixelsCommand{
		dst:    i,
		pixels: p,
	}
	theCommandQueue.Enqueue(c)
	return nil
}

func (i *Image) IsInvalidated(context *opengl.Context) bool {
	return !context.IsTexture(i.texture.native)
}
