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
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

func orthoProjectionMatrix(left, right, bottom, top int) []float32 {
	e11 := 2 / float32(right-left)
	e22 := 2 / float32(top-bottom)
	e14 := -1 * float32(right+left) / float32(right-left)
	e24 := -1 * float32(top+bottom) / float32(top-bottom)

	return []float32{
		e11, 0, 0, 0,
		0, e22, 0, 0,
		0, 0, 1, 0,
		e14, e24, 0, 1,
	}
}

type framebuffer struct {
	native    opengl.Framebuffer
	flipY     bool
	proMatrix []float32
	offsetX   float64
	offsetY   float64
}

func newFramebufferFromTexture(texture *texture) (*framebuffer, error) {
	native, err := opengl.GetContext().NewFramebuffer(opengl.Texture(texture.native))
	if err != nil {
		return nil, err
	}
	return &framebuffer{
		native: native,
	}, nil
}

const viewportSize = 4096

func (f *framebuffer) setAsViewport() error {
	width := viewportSize
	height := viewportSize
	return opengl.GetContext().SetViewport(f.native, width, height)
}

func (f *framebuffer) projectionMatrix(height int) []float32 {
	if f.proMatrix != nil {
		return f.proMatrix
	}
	m := orthoProjectionMatrix(0, viewportSize, 0, viewportSize)
	if f.flipY {
		m[4*1+1] *= -1
		m[4*3+1] += float32(height) / float32(viewportSize) * 2
	}
	m[4*3+0] += float32(f.offsetX) / float32(viewportSize) * 2
	m[4*3+1] += float32(f.offsetY) / float32(viewportSize) * 2
	f.proMatrix = m
	return f.proMatrix
}
