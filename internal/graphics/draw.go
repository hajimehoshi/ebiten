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
	"errors"
	"fmt"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
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

func drawTexture(c *opengl.Context, texture opengl.Texture, projectionMatrix *[4][4]float64, vertices []int16, geo Matrix, color Matrix, mode opengl.CompositeMode) error {
	c.BlendFunc(mode)

	// NOTE: WebGL doesn't seem to have Check gl.MAX_ELEMENTS_VERTICES or gl.MAX_ELEMENTS_INDICES so far.
	// Let's use them to compare to len(quads) in the future.
	n := len(vertices) / 16
	if n == 0 {
		return nil
	}
	if MaxQuads < n/16 {
		return errors.New(fmt.Sprintf("len(quads) must be equal to or less than %d", MaxQuads))
	}

	p := programContext{
		state:            &theOpenGLState,
		program:          theOpenGLState.programTexture,
		context:          c,
		projectionMatrix: glMatrix(projectionMatrix),
		texture:          texture,
		geoM:             geo,
		colorM:           color,
	}
	p.begin()
	defer p.end()
	c.BufferSubData(c.ArrayBuffer, vertices)
	c.DrawElements(c.Triangles, 6*n)
	return nil
}
