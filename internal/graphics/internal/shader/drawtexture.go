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
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

func glMatrix(m *[4][4]float64) []float32 {
	result := make([]float32, 16)
	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			result[i+j*4] = float32(m[i][j])
		}
	}
	return result
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

// TODO: Use unsafe.SizeOf?
const float32Size = 4

func DrawTexture(c *opengl.Context, texture opengl.Texture, projectionMatrix *[4][4]float64, quads TextureQuads, geo Matrix, color Matrix) error {
	// TODO: Check len(quads) and gl.MAX_ELEMENTS_INDICES?
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

	// TODO: Check performance
	program := useProgramColorMatrix(c, glMatrix(projectionMatrix), geo, color)

	// TODO: Do we have to call gl.ActiveTexture(gl.TEXTURE0)?
	c.BindTexture(texture)

	c.EnableVertexAttribArray(program, "vertex")
	c.EnableVertexAttribArray(program, "tex_coord")
	defer func() {
		c.DisableVertexAttribArray(program, "tex_coord")
		c.DisableVertexAttribArray(program, "vertex")
	}()

	c.VertexAttribPointer(program, "vertex", stride, uintptr(float32Size*0))
	c.VertexAttribPointer(program, "tex_coord", stride, uintptr(float32Size*2))

	vertices := make([]float32, 0, quads.Len())
	for i := 0; i < quads.Len(); i++ {
		x0, y0, x1, y1 := quads.Vertex(i)
		u0, v0, u1, v1 := quads.Texture(i)
		vertices = append(vertices,
			x0, y0, u0, v0,
			x1, y0, u1, v0,
			x0, y1, u0, v1,
			x1, y1, u1, v1,
		)
	}
	c.BufferSubData(c.ArrayBuffer, vertices)
	c.DrawElements(6 * quads.Len())
	return nil
}
