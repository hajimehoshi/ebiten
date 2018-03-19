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
	emath "github.com/hajimehoshi/ebiten/internal/math"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/web"
)

// arrayBufferLayoutPart is a part of an array buffer layout.
type arrayBufferLayoutPart struct {
	// TODO: This struct should belong to a program and know it.
	name     string
	dataType opengl.DataType
	num      int
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
		opengl.GetContext().VertexAttribPointer(program, p.name, p.num, p.dataType, total, offset)
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
				name:     "vertex",
				dataType: opengl.Float,
				num:      2,
			},
			{
				name:     "tex_coord",
				dataType: opengl.Float,
				num:      4,
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

	// programNearest is OpenGL's program for rendering a texture with nearest filter.
	programNearest opengl.Program

	// programLinear is OpenGL's program for rendering a texture with linear filter.
	programLinear opengl.Program

	programScreen opengl.Program

	lastProgram                opengl.Program
	lastProjectionMatrix       []float32
	lastColorMatrix            []float32
	lastColorMatrixTranslation []float32
	lastSourceWidth            int
	lastSourceHeight           int

	indices []uint16
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
	s.lastSourceWidth = 0
	s.lastSourceHeight = 0

	// When context lost happens, deleting programs or buffers is not necessary.
	// However, it is not assumed that reset is called only when context lost happens.
	// Let's delete them explicitly.
	if s.programNearest != zeroProgram {
		opengl.GetContext().DeleteProgram(s.programNearest)
	}
	if s.programLinear != zeroProgram {
		opengl.GetContext().DeleteProgram(s.programLinear)
	}
	if s.programScreen != zeroProgram {
		opengl.GetContext().DeleteProgram(s.programScreen)
	}

	// On browsers (at least Chrome), buffers are already detached from the context
	// and must not be deleted by DeleteBuffer.
	if !web.IsBrowser() {
		if s.arrayBuffer != zeroBuffer {
			opengl.GetContext().DeleteBuffer(s.arrayBuffer)
		}
		if s.elementArrayBuffer != zeroBuffer {
			opengl.GetContext().DeleteBuffer(s.elementArrayBuffer)
		}
	}

	shaderVertexModelviewNative, err := opengl.GetContext().NewShader(opengl.VertexShader, shader(shaderVertexModelview))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer opengl.GetContext().DeleteShader(shaderVertexModelviewNative)

	shaderFragmentNearestNative, err := opengl.GetContext().NewShader(opengl.FragmentShader, shader(shaderFragmentNearest))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer opengl.GetContext().DeleteShader(shaderFragmentNearestNative)

	shaderFragmentLinearNative, err := opengl.GetContext().NewShader(opengl.FragmentShader, shader(shaderFragmentLinear))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer opengl.GetContext().DeleteShader(shaderFragmentLinearNative)

	shaderFragmentScreenNative, err := opengl.GetContext().NewShader(opengl.FragmentShader, shader(shaderFragmentScreen))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer opengl.GetContext().DeleteShader(shaderFragmentScreenNative)

	s.programNearest, err = opengl.GetContext().NewProgram([]opengl.Shader{
		shaderVertexModelviewNative,
		shaderFragmentNearestNative,
	})
	if err != nil {
		return err
	}

	s.programLinear, err = opengl.GetContext().NewProgram([]opengl.Shader{
		shaderVertexModelviewNative,
		shaderFragmentLinearNative,
	})
	if err != nil {
		return err
	}

	s.programScreen, err = opengl.GetContext().NewProgram([]opengl.Shader{
		shaderVertexModelviewNative,
		shaderFragmentScreenNative,
	})
	if err != nil {
		return err
	}

	s.arrayBuffer = theArrayBufferLayout.newArrayBuffer()

	s.indices = make([]uint16, 6*maxQuads)
	for i := uint16(0); i < maxQuads; i++ {
		s.indices[6*i+0] = 4*i + 0
		s.indices[6*i+1] = 4*i + 1
		s.indices[6*i+2] = 4*i + 2
		s.indices[6*i+3] = 4*i + 1
		s.indices[6*i+4] = 4*i + 2
		s.indices[6*i+5] = 4*i + 3
	}
	// Note that the indices passed to NewElementArrayBuffer is not under GC management
	// in opengl package due to unsafe-way.
	// See NewElementArrayBuffer in context_mobile.go.
	s.elementArrayBuffer = opengl.GetContext().NewElementArrayBuffer(s.indices)

	return nil
}

// areSameFloat32Array returns a boolean indicating if a and b are deeply equal.
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

// useProgram uses the program (programTexture).
func (s *openGLState) useProgram(proj []float32, texture opengl.Texture, dst, src *Image, colorM *affine.ColorM, filter Filter) {
	c := opengl.GetContext()

	var program opengl.Program
	switch filter {
	case FilterNearest:
		program = s.programNearest
	case FilterLinear:
		program = s.programLinear
	case FilterScreen:
		program = s.programScreen
	default:
		panic("not reached")
	}

	if s.lastProgram != program {
		c.UseProgram(program)
		if s.lastProgram != zeroProgram {
			theArrayBufferLayout.disable(s.lastProgram)
		}
		theArrayBufferLayout.enable(program)

		s.lastProgram = program
		s.lastProjectionMatrix = nil
		s.lastColorMatrix = nil
		s.lastColorMatrixTranslation = nil
		s.lastSourceWidth = 0
		s.lastSourceHeight = 0
		c.BindElementArrayBuffer(s.elementArrayBuffer)
		c.UniformInt(program, "texture", 0)
	}

	if !areSameFloat32Array(s.lastProjectionMatrix, proj) {
		c.UniformFloats(program, "projection_matrix", proj)
		if s.lastProjectionMatrix == nil {
			s.lastProjectionMatrix = make([]float32, 16)
		}
		// (*framebuffer).projectionMatrix is always same for the same framebuffer.
		// It's OK to hold the reference without copying.
		s.lastProjectionMatrix = proj
	}

	esBody, esTranslate := colorM.UnsafeElements()

	if !areSameFloat32Array(s.lastColorMatrix, esBody) {
		c.UniformFloats(program, "color_matrix", esBody)
		if s.lastColorMatrix == nil {
			s.lastColorMatrix = make([]float32, 16)
		}
		// ColorM's elements are immutable. It's OK to hold the reference without copying.
		s.lastColorMatrix = esBody
	}
	if !areSameFloat32Array(s.lastColorMatrixTranslation, esTranslate) {
		c.UniformFloats(program, "color_matrix_translation", esTranslate)
		if s.lastColorMatrixTranslation == nil {
			s.lastColorMatrixTranslation = make([]float32, 4)
		}
		// ColorM's elements are immutable. It's OK to hold the reference without copying.
		s.lastColorMatrixTranslation = esTranslate
	}

	sw, sh := src.Size()
	sw = emath.NextPowerOf2Int(sw)
	sh = emath.NextPowerOf2Int(sh)

	if s.lastSourceWidth != sw || s.lastSourceHeight != sh {
		c.UniformFloats(program, "source_size", []float32{float32(sw), float32(sh)})
		s.lastSourceWidth = sw
		s.lastSourceHeight = sh
	}

	if program == s.programScreen {
		sw, _ := src.Size()
		dw, _ := dst.Size()
		scale := float32(dw) / float32(sw)
		c.UniformFloat(program, "scale", scale)
	}

	// We don't have to call gl.ActiveTexture here: GL_TEXTURE0 is the default active texture
	// See also: https://www.opengl.org/sdk/docs/man2/xhtml/glActiveTexture.xml
	c.BindTexture(texture)
}
