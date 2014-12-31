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
	"github.com/hajimehoshi/ebiten/internal"
	"github.com/hajimehoshi/ebiten/internal/graphics/internal/shader"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

func orthoProjectionMatrix(left, right, bottom, top int) [4][4]float64 {
	e11 := float64(2) / float64(right-left)
	e22 := float64(2) / float64(top-bottom)
	e14 := -1 * float64(right+left) / float64(right-left)
	e24 := -1 * float64(top+bottom) / float64(top-bottom)

	return [4][4]float64{
		{e11, 0, 0, e14},
		{0, e22, 0, e24},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}

type Framebuffer struct {
	native opengl.Framebuffer
	width  int
	height int
	flipY  bool
}

func NewZeroFramebuffer(c *opengl.Context, width, height int) (*Framebuffer, error) {
	r := &Framebuffer{
		width:  width,
		height: height,
		flipY:  true,
	}
	return r, nil
}

func NewFramebufferFromTexture(c *opengl.Context, texture *Texture) (*Framebuffer, error) {
	f, err := c.NewFramebuffer(opengl.Texture(texture.native))
	if err != nil {
		return nil, err
	}
	w, h := texture.Size()
	return &Framebuffer{
		native: f,
		width:  w,
		height: h,
	}, nil
}

func (f *Framebuffer) Size() (width, height int) {
	return f.width, f.height
}

func (f *Framebuffer) Dispose() {
	f.native.Delete()
}

func (f *Framebuffer) setAsViewport() error {
	width := internal.NextPowerOf2Int(f.width)
	height := internal.NextPowerOf2Int(f.height)
	return f.native.SetAsViewport(width, height)
}

func (f *Framebuffer) projectionMatrix() [4][4]float64 {
	width := internal.NextPowerOf2Int(f.width)
	height := internal.NextPowerOf2Int(f.height)
	m := orthoProjectionMatrix(0, width, 0, height)
	if f.flipY {
		m[1][1] *= -1
		m[1][3] += float64(f.height) / float64(internal.NextPowerOf2Int(f.height)) * 2
	}
	return m
}

type Matrix interface {
	Element(i, j int) float64
}

type TextureQuads interface {
	Len() int
	Vertex(i int) (x0, y0, x1, y1 float32)
	Texture(i int) (u0, v0, u1, v1 float32)
}

func (f *Framebuffer) Fill(r, g, b, a float64) error {
	if err := f.setAsViewport(); err != nil {
		return err
	}
	return f.native.Fill(r, g, b, a)
}

func (f *Framebuffer) DrawTexture(c *opengl.Context, t *Texture, quads TextureQuads, geo, clr Matrix) error {
	if err := f.setAsViewport(); err != nil {
		return err
	}
	projectionMatrix := f.projectionMatrix()
	// TODO: Define texture.Draw()
	return shader.DrawTexture(c, t.native, projectionMatrix, quads, geo, clr)
}
