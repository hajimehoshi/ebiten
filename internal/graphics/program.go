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
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

// arrayBufferLayoutPart is a part of an array buffer layout.
type arrayBufferLayoutPart struct {
	// TODO: This struct should belong to a program and know it.
	name      string
	dataType  opengl.DataType
	num       int
	normalize bool
}

// arrayBufferLayout is an array buffer layout.
//
// An array buffer in OpenGL is a buffer representing vertices and
// is passed to a vertex shader.
type arrayBufferLayout struct {
	parts []arrayBufferLayoutPart
	total int
}

// totalBytes returns the size in bytes for one element of the array buffer.
func (a *arrayBufferLayout) totalBytes() int {
	if a.total != 0 {
		return a.total
	}
	t := 0
	for _, p := range a.parts {
		t += p.dataType.SizeInBytes() * p.num
	}
	a.total = t
	return a.total
}

// newArrayBuffer creates OpenGL's buffer object for the array buffer.
func (a *arrayBufferLayout) newArrayBuffer() opengl.Buffer {
	return opengl.GetContext().NewArrayBuffer(a.totalBytes() * 4 * maxQuads)
}

// enable binds the array buffer the given program to use the array buffer.
func (a *arrayBufferLayout) enable(program opengl.Program) {
	for _, p := range a.parts {
		opengl.GetContext().EnableVertexAttribArray(program, p.name)
	}
	total := a.totalBytes()
	offset := 0
	for _, p := range a.parts {
		opengl.GetContext().VertexAttribPointer(program, p.name, p.num, p.dataType, p.normalize, total, offset)
		offset += p.dataType.SizeInBytes() * p.num
	}
}

// disable stops using the array buffer.
func (a *arrayBufferLayout) disable(program opengl.Program) {
	// TODO: Disabling should be done in reversed order?
	for _, p := range a.parts {
		opengl.GetContext().DisableVertexAttribArray(program, p.name)
	}
}

var (
	// theArrayBufferLayout is the array buffer layout for Ebiten.
	theArrayBufferLayout = arrayBufferLayout{
		// Note that GL_MAX_VERTEX_ATTRIBS is at least 16.
		parts: []arrayBufferLayoutPart{
			{
				name:      "vertex",
				dataType:  opengl.Float,
				num:       2,
				normalize: false,
			},
			{
				name:      "tex_coord",
				dataType:  opengl.Float,
				num:       2,
				normalize: false,
			},
			{
				name:      "geo_matrix_body",
				dataType:  opengl.Float,
				num:       4,
				normalize: false,
			},
			{
				name:      "geo_matrix_translation",
				dataType:  opengl.Float,
				num:       2,
				normalize: false,
			},
		},
	}
)

// openGLState is a state for OpenGL.
type openGLState struct {
	// arrayBuffer is OpenGL's array buffer (vertices data).
	arrayBuffer opengl.Buffer

	// elementArrayBuffer is OpenGL's element array buffer (indices data).
	elementArrayBuffer opengl.Buffer

	// programTexture is OpenGL's program for rendering a texture.
	programTexture opengl.Program

	lastProgram                opengl.Program
	lastProjectionMatrix       []float32
	lastColorMatrix            []float32
	lastColorMatrixTranslation []float32
}

var (
	// theOpenGLState is the OpenGL state in the current process.
	theOpenGLState openGLState

	zeroBuffer  opengl.Buffer
	zeroProgram opengl.Program
)

const (
	indicesNum = 1 << 16
	maxQuads   = indicesNum / 6
)

// ResetGLState resets or initializes the current OpenGL state.
func ResetGLState() error {
	return theOpenGLState.reset()
}

// reset resets or initializes the OpenGL state.
func (s *openGLState) reset() error {
	if err := opengl.GetContext().Reset(); err != nil {
		return err
	}
	s.lastProgram = zeroProgram
	s.lastProjectionMatrix = nil
	s.lastColorMatrix = nil
	s.lastColorMatrixTranslation = nil

	shaderVertexModelviewNative, err := opengl.GetContext().NewShader(opengl.VertexShader, shader(shaderVertexModelview))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer opengl.GetContext().DeleteShader(shaderVertexModelviewNative)

	shaderFragmentTextureNative, err := opengl.GetContext().NewShader(opengl.FragmentShader, shader(shaderFragmentTexture))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer opengl.GetContext().DeleteShader(shaderFragmentTextureNative)

	s.programTexture, err = opengl.GetContext().NewProgram([]opengl.Shader{
		shaderVertexModelviewNative,
		shaderFragmentTextureNative,
	})
	if err != nil {
		return err
	}

	if s.arrayBuffer != zeroBuffer {
		opengl.GetContext().DeleteBuffer(s.arrayBuffer)
	}
	s.arrayBuffer = theArrayBufferLayout.newArrayBuffer()

	if s.elementArrayBuffer != zeroBuffer {
		opengl.GetContext().DeleteBuffer(s.elementArrayBuffer)
	}
	indices := make([]uint16, 6*maxQuads)
	for i := uint16(0); i < maxQuads; i++ {
		indices[6*i+0] = 4*i + 0
		indices[6*i+1] = 4*i + 1
		indices[6*i+2] = 4*i + 2
		indices[6*i+3] = 4*i + 1
		indices[6*i+4] = 4*i + 2
		indices[6*i+5] = 4*i + 3
	}
	s.elementArrayBuffer = opengl.GetContext().NewElementArrayBuffer(indices)

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

type programContext struct {
	state            *openGLState
	program          opengl.Program
	projectionMatrix []float32
	texture          opengl.Texture
	colorM           affine.ColorM
}

func (p *programContext) begin() error {
	c := opengl.GetContext()
	if p.state.lastProgram != p.program {
		c.UseProgram(p.program)
		if p.state.lastProgram != zeroProgram {
			theArrayBufferLayout.disable(p.state.lastProgram)
		}
		theArrayBufferLayout.enable(p.program)

		p.state.lastProgram = p.state.programTexture
		p.state.lastProjectionMatrix = nil
		p.state.lastColorMatrix = nil
		p.state.lastColorMatrixTranslation = nil
		c.BindElementArrayBuffer(p.state.elementArrayBuffer)
		c.UniformInt(p.program, "texture", 0)
	}

	if !areSameFloat32Array(p.state.lastProjectionMatrix, p.projectionMatrix) {
		c.UniformFloats(p.program, "projection_matrix", p.projectionMatrix)
		if p.state.lastProjectionMatrix == nil {
			p.state.lastProjectionMatrix = make([]float32, 16)
		}
		copy(p.state.lastProjectionMatrix, p.projectionMatrix)
	}

	e := [4][5]float32{}
	es := p.colorM.UnsafeElements()
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			e[i][j] = float32(es[i*affine.ColorMDim+j])
		}
	}

	colorMatrix := []float32{
		e[0][0], e[1][0], e[2][0], e[3][0],
		e[0][1], e[1][1], e[2][1], e[3][1],
		e[0][2], e[1][2], e[2][2], e[3][2],
		e[0][3], e[1][3], e[2][3], e[3][3],
	}
	if !areSameFloat32Array(p.state.lastColorMatrix, colorMatrix) {
		c.UniformFloats(p.program, "color_matrix", colorMatrix)
		if p.state.lastColorMatrix == nil {
			p.state.lastColorMatrix = make([]float32, 16)
		}
		copy(p.state.lastColorMatrix, colorMatrix)
	}
	colorMatrixTranslation := []float32{
		e[0][4], e[1][4], e[2][4], e[3][4],
	}
	if !areSameFloat32Array(p.state.lastColorMatrixTranslation, colorMatrixTranslation) {
		c.UniformFloats(p.program, "color_matrix_translation", colorMatrixTranslation)
		if p.state.lastColorMatrixTranslation == nil {
			p.state.lastColorMatrixTranslation = make([]float32, 4)
		}
		copy(p.state.lastColorMatrixTranslation, colorMatrixTranslation)
	}

	// We don't have to call gl.ActiveTexture here: GL_TEXTURE0 is the default active texture
	// See also: https://www.opengl.org/sdk/docs/man2/xhtml/glActiveTexture.xml
	if err := c.BindTexture(p.texture); err != nil {
		return err
	}
	return nil
}
