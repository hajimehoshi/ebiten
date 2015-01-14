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

package shader

import (
	"errors"
	"fmt"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

func glMatrix(m *[4][4]float64) []float32 {
	return []float32{
		float32(m[0][0]), float32(m[1][0]), float32(m[2][0]), float32(m[3][0]),
		float32(m[0][1]), float32(m[1][1]), float32(m[2][1]), float32(m[3][1]),
		float32(m[0][2]), float32(m[1][2]), float32(m[2][2]), float32(m[3][2]),
		float32(m[0][3]), float32(m[1][3]), float32(m[2][3]), float32(m[3][3]),
	}
}

type Matrix interface {
	Element(i, j int) float64
}

type TextureQuads interface {
	Len() int
	Vertex(i int) (x0, y0, x1, y1 float32)
	Texture(i int) (u0, v0, u1, v1 float32)
}

var initialized = false

func DrawTexture(c *opengl.Context, texture opengl.Texture, projectionMatrix *[4][4]float64, quads TextureQuads, geo Matrix, color Matrix) error {
	// TODO: WebGL doesn't seem to have Check gl.MAX_ELEMENTS_VERTICES or gl.MAX_ELEMENTS_INDICES so far.
	// Let's use them to compare to len(quads) in the future.

	// TODO: Unify stride or other consts
	const stride = 4 * 4

	if !initialized {
		if err := initialize(c); err != nil {
			return err
		}
		initialized = true
	}

	if quads.Len() == 0 {
		return nil
	}
	// TODO: Change this panic if image.DrawImage allows more than quadsMaxNum parts.
	// TODO: Kinder message
	if quadsMaxNum < 4*quads.Len() {
		return errors.New(fmt.Sprintf("len(quads) must be equal to or less than %d", quadsMaxNum))
	}

	f := useProgramTexture(c, glMatrix(projectionMatrix), texture, geo, color)
	defer f.FinishProgram()

	vertices := make([]float32, 0, stride*quads.Len())
	for i := 0; i < quads.Len(); i++ {
		x0, y0, x1, y1 := quads.Vertex(i)
		u0, v0, u1, v1 := quads.Texture(i)
		if x0 == x1 || y0 == y1 || u0 == u1 || v0 == v1 {
			continue
		}
		vertices = append(vertices,
			x0, y0, u0, v0,
			x1, y0, u1, v0,
			x0, y1, u0, v1,
			x1, y1, u1, v1,
		)
	}
	if len(vertices) == 0 {
		return nil
	}
	c.BufferSubData(c.ArrayBuffer, vertices)
	c.DrawElements(6 * len(vertices) / 16)
	return nil
}

type VertexQuads interface {
	Len() int
	Vertex(i int) (x0, y0, x1, y1 float32)
}

func max(a, b float32) float32 {
	if a < b {
		return b
	}
	return a
}

func DrawRects(c *opengl.Context, projectionMatrix *[4][4]float64, r, g, b, a float64, quads VertexQuads) error {
	const stride = 4 * 4

	if !initialized {
		if err := initialize(c); err != nil {
			return err
		}
		initialized = true
	}

	if quads.Len() == 0 {
		return nil
	}

	f := useProgramRect(c, glMatrix(projectionMatrix), r, g, b, a)
	defer f.FinishProgram()

	vertices := make([]float32, 0, stride*quads.Len())
	for i := 0; i < quads.Len(); i++ {
		x0, y0, x1, y1 := quads.Vertex(i)
		if x0 == x1 || y0 == y1 {
			continue
		}
		vertices = append(vertices,
			x0, y0, 0, 0,
			x1, y0, 1, 0,
			x0, y1, 0, 1,
			x1, y1, 1, 1,
		)
	}
	if len(vertices) == 0 {
		return nil
	}
	c.BufferSubData(c.ArrayBuffer, vertices)
	c.DrawElements(6 * len(vertices) / 16)

	return nil
}
