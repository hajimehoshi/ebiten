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
