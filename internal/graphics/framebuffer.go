// Copyright 2014 Hajime Hoshi
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
	"image/color"

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

func orthoProjectionMatrix(left, right, bottom, top int) *[4][4]float64 {
	e11 := float64(2) / float64(right-left)
	e22 := float64(2) / float64(top-bottom)
	e14 := -1 * float64(right+left) / float64(right-left)
	e24 := -1 * float64(top+bottom) / float64(top-bottom)

	return &[4][4]float64{
		{e11, 0, 0, e14},
		{0, e22, 0, e24},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}

type framebuffer struct {
	native    opengl.Framebuffer
	width     int
	height    int
	flipY     bool
	proMatrix *[4][4]float64
}

func NewZeroFramebufferImage(width, height int) (*Image, error) {
	f := &framebuffer{
		width:  width,
		height: height,
		flipY:  true,
	}
	return &Image{
		framebuffer: f,
	}, nil
}

func (f *framebuffer) initFromTexture(context *opengl.Context, texture *texture) error {
	native, err := context.NewFramebuffer(opengl.Texture(texture.native))
	if err != nil {
		return err
	}
	f.native = native
	f.width = texture.width
	f.height = texture.height
	return nil
}

func (i *Image) Dispose() error {
	c := &disposeCommand{
		framebuffer: i.framebuffer,
		texture:     i.texture,
	}
	theCommandQueue.Enqueue(c)
	return nil
}

const viewportSize = 4096

func (f *framebuffer) setAsViewport(c *opengl.Context) error {
	width := viewportSize
	height := viewportSize
	return c.SetViewport(f.native, width, height)
}

func (f *framebuffer) projectionMatrix() *[4][4]float64 {
	if f.proMatrix != nil {
		return f.proMatrix
	}
	width := viewportSize
	height := viewportSize
	m := orthoProjectionMatrix(0, width, 0, height)
	if f.flipY {
		m[1][1] *= -1
		m[1][3] += float64(f.height) / float64(height) * 2
	}
	f.proMatrix = m
	return f.proMatrix
}

func (i *Image) Fill(clr color.Color) error {
	c := &fillCommand{
		dst:   i.framebuffer,
		color: clr,
	}
	theCommandQueue.Enqueue(c)
	return nil
}

func (i *Image) DrawImage(src *Image, vertices []int16, geo, clr Matrix, mode opengl.CompositeMode) error {
	c := &drawImageCommand{
		dst:      i.framebuffer,
		src:      src.texture,
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
		dst:     i.framebuffer,
		texture: i.texture,
		pixels:  p,
	}
	theCommandQueue.Enqueue(c)
	return nil
}
