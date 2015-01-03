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

var programColorMatrix opengl.Program

func initialize(c *opengl.Context) error {
	const size = 10000
	const uint16Size = 2

	shaderVertexNative, err := c.NewShader(c.VertexShader, shader(c, shaderVertex))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderVertexNative)

	shaderColorMatrixNative, err := c.NewShader(c.FragmentShader, shader(c, shaderColorMatrix))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderColorMatrixNative)

	shaders := []opengl.Shader{
		shaderVertexNative,
		shaderColorMatrixNative,
	}
	programColorMatrix, err = c.NewProgram(shaders)
	if err != nil {
		return err
	}

	const stride = 4 * 4
	v := make([]float32, stride*size)
	c.NewBuffer(c.ArrayBuffer, v, c.DynamicDraw)

	indices := make([]uint16, 6*size)
	for i := uint16(0); i < size; i++ {
		indices[6*i+0] = 4*i + 0
		indices[6*i+1] = 4*i + 1
		indices[6*i+2] = 4*i + 2
		indices[6*i+3] = 4*i + 1
		indices[6*i+4] = 4*i + 2
		indices[6*i+5] = 4*i + 3
	}
	c.NewBuffer(c.ElementArrayBuffer, indices, c.StaticDraw)

	return nil
}

var lastProgram opengl.Program

func useProgramColorMatrix(c *opengl.Context, projectionMatrix []float32, geo Matrix, color Matrix) opengl.Program {
	if lastProgram != programColorMatrix {
		c.UseProgram(programColorMatrix)
		lastProgram = programColorMatrix
	}
	// TODO: Check the performance.
	program := programColorMatrix

	c.Uniform(program, "projection_matrix", projectionMatrix)

	ma := float32(geo.Element(0, 0))
	mb := float32(geo.Element(0, 1))
	mc := float32(geo.Element(1, 0))
	md := float32(geo.Element(1, 1))
	tx := float32(geo.Element(0, 2))
	ty := float32(geo.Element(1, 2))
	glModelviewMatrix := []float32{
		ma, mc, 0, 0,
		mb, md, 0, 0,
		0, 0, 1, 0,
		tx, ty, 0, 1,
	}
	c.Uniform(program, "modelview_matrix", glModelviewMatrix)
	c.Uniform(program, "texture", 0)

	e := [4][5]float32{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			e[i][j] = float32(color.Element(i, j))
		}
	}

	glColorMatrix := []float32{
		e[0][0], e[1][0], e[2][0], e[3][0],
		e[0][1], e[1][1], e[2][1], e[3][1],
		e[0][2], e[1][2], e[2][2], e[3][2],
		e[0][3], e[1][3], e[2][3], e[3][3],
	}
	c.Uniform(program, "color_matrix", glColorMatrix)
	glColorMatrixTranslation := []float32{
		e[0][4], e[1][4], e[2][4], e[3][4],
	}
	c.Uniform(program, "color_matrix_translation", glColorMatrixTranslation)

	return program
}
