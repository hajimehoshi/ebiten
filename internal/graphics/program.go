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
	"math"
)

var (
	indexBufferLines opengl.Buffer
	indexBufferQuads opengl.Buffer
)

var (
	programTexture   opengl.Program
	programSolidRect opengl.Program
	programSolidLine opengl.Program
)

const indicesNum = math.MaxUint16 + 1
const quadsMaxNum = indicesNum / 6

// unsafe.SizeOf can't be used because unsafe doesn't work with GopherJS.
const int16Size = 2
const float32Size = 4

func initialize(c *opengl.Context) error {
	shaderVertexModelviewNative, err := c.NewShader(c.VertexShader, shader(c, shaderVertexModelview))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderVertexModelviewNative)

	shaderVertexColorNative, err := c.NewShader(c.VertexShader, shader(c, shaderVertexColor))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderVertexColorNative)

	shaderVertexColorLineNative, err := c.NewShader(c.VertexShader, shader(c, shaderVertexColorLine))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderVertexColorLineNative)

	shaderFragmentTextureNative, err := c.NewShader(c.FragmentShader, shader(c, shaderFragmentTexture))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderFragmentTextureNative)

	shaderFragmentSolidNative, err := c.NewShader(c.FragmentShader, shader(c, shaderFragmentSolid))
	if err != nil {
		return err
	}
	defer c.DeleteShader(shaderFragmentSolidNative)

	programTexture, err = c.NewProgram([]opengl.Shader{
		shaderVertexModelviewNative,
		shaderFragmentTextureNative,
	})
	if err != nil {
		return err
	}

	programSolidRect, err = c.NewProgram([]opengl.Shader{
		shaderVertexColorNative,
		shaderFragmentSolidNative,
	})
	if err != nil {
		return err
	}

	programSolidLine, err = c.NewProgram([]opengl.Shader{
		shaderVertexColorLineNative,
		shaderFragmentSolidNative,
	})
	if err != nil {
		return err
	}

	// 16 [bytse] is an arbitrary number which seems enough to draw anything. Fix this if necessary.
	const stride = 16
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
	indexBufferQuads = c.NewBuffer(c.ElementArrayBuffer, indices, c.StaticDraw)

	indices = make([]uint16, indicesNum)
	for i := 0; i < len(indices); i++ {
		indices[i] = uint16(i)
	}
	indexBufferLines = c.NewBuffer(c.ElementArrayBuffer, indices, c.StaticDraw)

	return nil
}

func areSameFloat32Array(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var (
	lastProgram          opengl.Program
	lastProjectionMatrix []float32
	lastModelviewMatrix  []float32
	lastColorMatrix      []float32
)

type programFinisher func()

func (p programFinisher) FinishProgram() {
	p()
}

func useProgramForTexture(c *opengl.Context, projectionMatrix []float32, texture opengl.Texture, geo Matrix, color Matrix) programFinisher {
	if !lastProgram.Equals(programTexture) {
		c.UseProgram(programTexture)
		lastProgram = programTexture
		lastProjectionMatrix = nil
		lastModelviewMatrix = nil
		lastColorMatrix = nil
	}
	program := programTexture

	c.BindElementArrayBuffer(indexBufferQuads)

	if !areSameFloat32Array(lastProjectionMatrix, projectionMatrix) {
		c.UniformFloats(program, "projection_matrix", projectionMatrix)
		if lastProjectionMatrix == nil {
			lastProjectionMatrix = make([]float32, 16)
		}
		copy(lastProjectionMatrix, projectionMatrix)
	}

	ma := float32(geo.Element(0, 0))
	mb := float32(geo.Element(0, 1))
	mc := float32(geo.Element(1, 0))
	md := float32(geo.Element(1, 1))
	tx := float32(geo.Element(0, 2))
	ty := float32(geo.Element(1, 2))
	modelviewMatrix := []float32{
		ma, mc, 0, 0,
		mb, md, 0, 0,
		0, 0, 1, 0,
		tx, ty, 0, 1,
	}
	if !areSameFloat32Array(lastModelviewMatrix, modelviewMatrix) {
		c.UniformFloats(program, "modelview_matrix", modelviewMatrix)
		if lastModelviewMatrix == nil {
			lastModelviewMatrix = make([]float32, 16)
		}
		copy(lastModelviewMatrix, modelviewMatrix)
	}

	c.UniformInt(program, "texture", 0)

	e := [4][5]float32{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			e[i][j] = float32(color.Element(i, j))
		}
	}

	colorMatrix := []float32{
		e[0][0], e[1][0], e[2][0], e[3][0],
		e[0][1], e[1][1], e[2][1], e[3][1],
		e[0][2], e[1][2], e[2][2], e[3][2],
		e[0][3], e[1][3], e[2][3], e[3][3],
	}
	if !areSameFloat32Array(lastColorMatrix, colorMatrix) {
		c.UniformFloats(program, "color_matrix", colorMatrix)
		if lastColorMatrix == nil {
			lastColorMatrix = make([]float32, 16)
		}
		copy(lastColorMatrix, colorMatrix)
	}
	colorMatrixTranslation := []float32{
		e[0][4], e[1][4], e[2][4], e[3][4],
	}
	c.UniformFloats(program, "color_matrix_translation", colorMatrixTranslation)

	// We don't have to call gl.ActiveTexture here: GL_TEXTURE0 is the default active texture
	// See also: https://www.opengl.org/sdk/docs/man2/xhtml/glActiveTexture.xml
	c.BindTexture(texture)

	c.EnableVertexAttribArray(program, "vertex")
	c.EnableVertexAttribArray(program, "tex_coord")

	c.VertexAttribPointer(program, "vertex", true, false, int16Size*4, 2, int16Size*0)
	c.VertexAttribPointer(program, "tex_coord", true, true, int16Size*4, 2, int16Size*2)

	return func() {
		c.DisableVertexAttribArray(program, "tex_coord")
		c.DisableVertexAttribArray(program, "vertex")
	}
}

func useProgramForLines(c *opengl.Context, projectionMatrix []float32) programFinisher {
	if !lastProgram.Equals(programSolidLine) {
		c.UseProgram(programSolidLine)
		lastProgram = programSolidLine
	}
	program := programSolidLine

	c.BindElementArrayBuffer(indexBufferLines)

	c.UniformFloats(program, "projection_matrix", projectionMatrix)

	c.EnableVertexAttribArray(program, "vertex")
	c.EnableVertexAttribArray(program, "color")

	// TODO: Change to floats?
	c.VertexAttribPointer(program, "vertex", true, false, int16Size*6, 2, int16Size*0)
	c.VertexAttribPointer(program, "color", false, true, int16Size*6, 4, int16Size*2)

	return func() {
		c.DisableVertexAttribArray(program, "color")
		c.DisableVertexAttribArray(program, "vertex")
	}
}

func useProgramForRects(c *opengl.Context, projectionMatrix []float32) programFinisher {
	if !lastProgram.Equals(programSolidRect) {
		c.UseProgram(programSolidRect)
		lastProgram = programSolidRect
	}
	program := programSolidRect

	c.BindElementArrayBuffer(indexBufferQuads)

	c.UniformFloats(program, "projection_matrix", projectionMatrix)

	c.EnableVertexAttribArray(program, "vertex")
	c.EnableVertexAttribArray(program, "color")

	c.VertexAttribPointer(program, "vertex", true, false, int16Size*6, 2, int16Size*0)
	c.VertexAttribPointer(program, "color", false, true, int16Size*6, 4, int16Size*2)

	return func() {
		c.DisableVertexAttribArray(program, "color")
		c.DisableVertexAttribArray(program, "vertex")
	}
}
