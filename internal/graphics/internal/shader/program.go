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

var (
	programTexture opengl.Program
	programRect    opengl.Program
)

const quadsMaxNum = 65536 / 6

// unsafe.SizeOf can't be used because unsafe doesn't work with GopherJS.
const int16Size = 2

func initialize(c *opengl.Context) error {
	shaderVertexNative, err := c.NewShader(c.VertexShader, shader(c, shaderVertex))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderVertexNative)

	shaderVertexColorNative, err := c.NewShader(c.VertexShader, shader(c, shaderVertexColor))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderVertexColorNative)

	shaderFragmentTextureNative, err := c.NewShader(c.FragmentShader, shader(c, shaderFragmentTexture))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderFragmentTextureNative)

	shaderFragmentRectNative, err := c.NewShader(c.FragmentShader, shader(c, shaderFragmentRect))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderFragmentRectNative)

	programTexture, err = c.NewProgram([]opengl.Shader{
		shaderVertexNative,
		shaderFragmentTextureNative,
	})
	if err != nil {
		return err
	}

	programRect, err = c.NewProgram([]opengl.Shader{
		shaderVertexColorNative,
		shaderFragmentRectNative,
	})
	if err != nil {
		return err
	}

	const stride = int16Size * 8 // 8 = (2 for vertex) + (2 for texture) + (4 for color)
	c.NewBuffer(c.ArrayBuffer, 4*stride*quadsMaxNum, c.DynamicDraw)

	indices := make([]uint16, 6*quadsMaxNum)
	for i := uint16(0); i < quadsMaxNum; i++ {
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

type programFinisher func()

func (p programFinisher) FinishProgram() {
	p()
}

func useProgramTexture(c *opengl.Context, projectionMatrix []float32, texture opengl.Texture, geo Matrix, color Matrix) programFinisher {
	if !lastProgram.Equals(programTexture) {
		c.UseProgram(programTexture)
		lastProgram = programTexture
	}
	program := programTexture

	c.UniformFloats(program, "projection_matrix", projectionMatrix)

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
	c.UniformFloats(program, "modelview_matrix", glModelviewMatrix)
	c.UniformInt(program, "texture", 0)

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
	c.UniformFloats(program, "color_matrix", glColorMatrix)
	glColorMatrixTranslation := []float32{
		e[0][4], e[1][4], e[2][4], e[3][4],
	}
	c.UniformFloats(program, "color_matrix_translation", glColorMatrixTranslation)

	// We don't have to call gl.ActiveTexture here: GL_TEXTURE0 is the default active texture
	// See also: https://www.opengl.org/sdk/docs/man2/xhtml/glActiveTexture.xml
	c.BindTexture(texture)

	c.EnableVertexAttribArray(program, "vertex")
	c.EnableVertexAttribArray(program, "tex_coord")

	c.VertexAttribPointer(program, "vertex", true, false, int16Size*4, 2, uintptr(int16Size*0))
	c.VertexAttribPointer(program, "tex_coord", true, true, int16Size*4, 2, uintptr(int16Size*2))

	return func() {
		c.DisableVertexAttribArray(program, "tex_coord")
		c.DisableVertexAttribArray(program, "vertex")
	}
}

func useProgramRect(c *opengl.Context, projectionMatrix []float32) programFinisher {
	if lastProgram != programRect {
		c.UseProgram(programRect)
		lastProgram = programRect
	}
	program := programRect

	c.UniformFloats(program, "projection_matrix", projectionMatrix)

	c.EnableVertexAttribArray(program, "vertex")
	c.EnableVertexAttribArray(program, "color")

	c.VertexAttribPointer(program, "vertex", true, false, int16Size*6, 2, uintptr(int16Size*0))
	c.VertexAttribPointer(program, "color", false, true, int16Size*6, 4, uintptr(int16Size*2))

	return func() {
		c.DisableVertexAttribArray(program, "color")
		c.DisableVertexAttribArray(program, "vertex")
	}
}
