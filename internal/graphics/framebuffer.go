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

type Framebuffer struct {
	native    opengl.Framebuffer
	width     int
	height    int
	flipY     bool
	proMatrix *[4][4]float64
}

func NewZeroFramebuffer(c *opengl.Context, width, height int) (*Framebuffer, error) {
	f := &Framebuffer{
		width:  width,
		height: height,
		flipY:  true,
	}
	return f, nil
}

func NewFramebufferFromTexture(c *opengl.Context, texture *Texture) (*Framebuffer, error) {
	native, err := c.NewFramebuffer(opengl.Texture(texture.native))
	if err != nil {
		return nil, err
	}
	f := &Framebuffer{
		native: native,
		width:  texture.width,
		height: texture.height,
	}
	return f, nil
}

func (f *Framebuffer) Dispose(c *opengl.Context) error {
	// Don't delete the default framebuffer.
	if f.native == opengl.ZeroFramebuffer {
		return nil
	}
	c.DeleteFramebuffer(f.native)
	return nil
}

func (f *Framebuffer) setAsViewport(c *opengl.Context) error {
	width := int(NextPowerOf2Int32(int32(f.width)))
	height := int(NextPowerOf2Int32(int32(f.height)))
	return c.SetViewport(f.native, width, height)
}

func (f *Framebuffer) projectionMatrix() *[4][4]float64 {
	if f.proMatrix != nil {
		return f.proMatrix
	}
	width := int(NextPowerOf2Int32(int32(f.width)))
	height := int(NextPowerOf2Int32(int32(f.height)))
	m := orthoProjectionMatrix(0, width, 0, height)
	if f.flipY {
		m[1][1] *= -1
		m[1][3] += float64(f.height) / float64(NextPowerOf2Int32(int32(f.height))) * 2
	}
	f.proMatrix = m
	return f.proMatrix
}

func (f *Framebuffer) Fill(context *opengl.Context, clr color.Color) error {
	c := &fillCommand{
		dst:   f,
		color: clr,
	}
	theCommandQueue.Enqueue(c)
	return nil
}

func (f *Framebuffer) DrawTexture(context *opengl.Context, t *Texture, vertices []int16, geo, clr Matrix, mode opengl.CompositeMode) error {
	c := &drawImageCommand{
		dst:      f,
		src:      t,
		vertices: vertices,
		geo:      geo,
		color:    clr,
		mode:     mode,
	}
	theCommandQueue.Enqueue(c)
	// Drawing a texture to the default buffer must be the last command.
	// TODO(hajimehoshi): This seems a little hacky. Refactor.
	if f.native == opengl.ZeroFramebuffer {
		if err := theCommandQueue.Flush(context); err != nil {
			return err
		}
	}
	return nil
}

func (f *Framebuffer) Pixels(context *opengl.Context) ([]uint8, error) {
	// Flush the enqueued commands so that pixels are certainly read.
	if err := theCommandQueue.Flush(context); err != nil {
		return nil, err
	}
	return context.FramebufferPixels(f.native, f.width, f.height)
}

func (f *Framebuffer) ReplacePixels(context *opengl.Context, t *Texture, p []uint8) error {
	c := &replacePixelsCommand{
		dst:     f,
		texture: t,
		pixels:  p,
	}
	theCommandQueue.Enqueue(c)
	return nil
}
