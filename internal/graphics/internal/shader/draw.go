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
	"github.com/hajimehoshi/ebiten/internal/graphics/internal/opengl"
	"image/color"
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
	Vertex(i int) (x0, y0, x1, y1 int)
	Texture(i int) (u0, v0, u1, v1 int)
}

var vertices = make([]int16, 0, 4*8*quadsMaxNum)

var initialized = false

func DrawTexture(c *opengl.Context, texture opengl.Texture, projectionMatrix *[4][4]float64, quads TextureQuads, geo Matrix, color Matrix) error {
	// TODO: WebGL doesn't seem to have Check gl.MAX_ELEMENTS_VERTICES or gl.MAX_ELEMENTS_INDICES so far.
	// Let's use them to compare to len(quads) in the future.

	if !initialized {
		if err := initialize(c); err != nil {
			return err
		}
		initialized = true
	}

	if quads.Len() == 0 {
		return nil
	}
	if quadsMaxNum < quads.Len() {
		return errors.New(fmt.Sprintf("len(quads) must be equal to or less than %d", quadsMaxNum))
	}

	f := useProgramForTexture(c, glMatrix(projectionMatrix), texture, geo, color)
	defer f.FinishProgram()

	vertices := vertices[0:0]
	num := 0
	for i := 0; i < quads.Len(); i++ {
		x0, y0, x1, y1 := quads.Vertex(i)
		u0, v0, u1, v1 := quads.Texture(i)
		if x0 == x1 || y0 == y1 || u0 == u1 || v0 == v1 {
			continue
		}
		vertices = append(vertices,
			int16(x0), int16(y0), int16(u0), int16(v0),
			int16(x1), int16(y0), int16(u1), int16(v0),
			int16(x0), int16(y1), int16(u0), int16(v1),
			int16(x1), int16(y1), int16(u1), int16(v1),
		)
		num++
	}
	if len(vertices) == 0 {
		return nil
	}
	c.BufferSubData(c.ArrayBuffer, vertices)
	c.DrawElements(c.Triangles, 6*num)
	return nil
}

type Lines interface {
	Len() int
	Points(i int) (x0, y0, x1, y1 int)
	Color(i int) color.Color
}

func DrawLines(c *opengl.Context, projectionMatrix *[4][4]float64, lines Lines) error {
	if !initialized {
		if err := initialize(c); err != nil {
			return err
		}
		initialized = true
	}

	if lines.Len() == 0 {
		return nil
	}

	f := useProgramForLines(c, glMatrix(projectionMatrix))
	defer f.FinishProgram()

	vertices := vertices[0:0]
	num := 0
	for i := 0; i < lines.Len(); i++ {
		x0, y0, x1, y1 := lines.Points(i)
		if x0 == x1 && y0 == y1 {
			continue
		}
		r, g, b, a := lines.Color(i).RGBA()
		vertices = append(vertices,
			int16(x0), int16(y0), int16(r), int16(g), int16(b), int16(a),
			int16(x1), int16(y1), int16(r), int16(g), int16(b), int16(a),
		)
		num++
	}
	if len(vertices) == 0 {
		return nil
	}
	c.BufferSubData(c.ArrayBuffer, vertices)
	c.DrawElements(c.Lines, 2*num)

	return nil
}

type Rects interface {
	Len() int
	Rect(i int) (x, y, width, height int)
	Color(i int) color.Color
}

func DrawFilledRects(c *opengl.Context, projectionMatrix *[4][4]float64, rects Rects) error {
	if !initialized {
		if err := initialize(c); err != nil {
			return err
		}
		initialized = true
	}

	if rects.Len() == 0 {
		return nil
	}

	f := useProgramForRects(c, glMatrix(projectionMatrix))
	defer f.FinishProgram()

	vertices := vertices[0:0]
	num := 0
	for i := 0; i < rects.Len(); i++ {
		x, y, w, h := rects.Rect(i)
		if w == 0 || h == 0 {
			continue
		}
		x0, y0, x1, y1 := x, y, x+w, y+h
		r, g, b, a := rects.Color(i).RGBA()
		vertices = append(vertices,
			int16(x0), int16(y0), int16(r), int16(g), int16(b), int16(a),
			int16(x1), int16(y0), int16(r), int16(g), int16(b), int16(a),
			int16(x0), int16(y1), int16(r), int16(g), int16(b), int16(a),
			int16(x1), int16(y1), int16(r), int16(g), int16(b), int16(a),
		)
		num++
	}
	if len(vertices) == 0 {
		return nil
	}
	c.BufferSubData(c.ArrayBuffer, vertices)
	c.DrawElements(c.Triangles, 6*num)

	return nil
}
