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

package opengl

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	emath "github.com/hajimehoshi/ebiten/internal/math"
	"github.com/hajimehoshi/ebiten/internal/web"
)

// arrayBufferLayoutPart is a part of an array buffer layout.
type arrayBufferLayoutPart struct {
	// TODO: This struct should belong to a program and know it.
	name     string
	dataType DataType
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
func (a *arrayBufferLayout) newArrayBuffer() buffer {
	return GetContext().newArrayBuffer(a.totalBytes() * IndicesNum)
}

// enable binds the array buffer the given program to use the array buffer.
func (a *arrayBufferLayout) enable(program program) {
	for _, p := range a.parts {
		GetContext().enableVertexAttribArray(program, p.name)
	}
	total := a.totalBytes()
	offset := 0
	for _, p := range a.parts {
		GetContext().vertexAttribPointer(program, p.name, p.num, p.dataType, total, offset)
		offset += p.dataType.SizeInBytes() * p.num
	}
}

// disable stops using the array buffer.
func (a *arrayBufferLayout) disable(program program) {
	// TODO: Disabling should be done in reversed order?
	for _, p := range a.parts {
		GetContext().disableVertexAttribArray(program, p.name)
	}
}

// theArrayBufferLayout is the array buffer layout for Ebiten.
var theArrayBufferLayout arrayBufferLayout

func initializeArrayBuferLayout() {
	theArrayBufferLayout = arrayBufferLayout{
		// Note that GL_MAX_VERTEX_ATTRIBS is at least 16.
		parts: []arrayBufferLayoutPart{
			{
				name:     "vertex",
				dataType: Float,
				num:      2,
			},
			{
				name:     "tex_coord",
				dataType: Float,
				num:      4,
			},
			{
				name:     "color_scale",
				dataType: Float,
				num:      4,
			},
		},
	}
}

func ArrayBufferLayoutTotalBytes() int {
	return theArrayBufferLayout.totalBytes()
}

// openGLState is a state for
type openGLState struct {
	// arrayBuffer is OpenGL's array buffer (vertices data).
	arrayBuffer buffer

	// elementArrayBuffer is OpenGL's element array buffer (indices data).
	elementArrayBuffer buffer

	// programNearest is OpenGL's program for rendering a texture with nearest filter.
	programNearest program

	// programLinear is OpenGL's program for rendering a texture with linear filter.
	programLinear program

	programScreen program

	lastProgram                program
	lastProjectionMatrix       []float32
	lastColorMatrix            []float32
	lastColorMatrixTranslation []float32
	lastSourceWidth            int
	lastSourceHeight           int
}

var (
	// theOpenGLState is the OpenGL state in the current process.
	theOpenGLState openGLState

	zeroBuffer  buffer
	zeroProgram program
)

const (
	IndicesNum   = (1 << 16) / 3 * 3 // Adjust num for triangles.
	maxTriangles = IndicesNum / 3
	maxQuads     = maxTriangles / 2
)

// ResetGLState resets or initializes the current OpenGL state.
func ResetGLState() error {
	return theOpenGLState.reset()
}

// reset resets or initializes the OpenGL state.
func (s *openGLState) reset() error {
	if err := GetContext().reset(); err != nil {
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
		GetContext().deleteProgram(s.programNearest)
	}
	if s.programLinear != zeroProgram {
		GetContext().deleteProgram(s.programLinear)
	}
	if s.programScreen != zeroProgram {
		GetContext().deleteProgram(s.programScreen)
	}

	// On browsers (at least Chrome), buffers are already detached from the context
	// and must not be deleted by DeleteBuffer.
	if !web.IsBrowser() {
		if s.arrayBuffer != zeroBuffer {
			GetContext().deleteBuffer(s.arrayBuffer)
		}
		if s.elementArrayBuffer != zeroBuffer {
			GetContext().deleteBuffer(s.elementArrayBuffer)
		}
	}

	shaderVertexModelviewNative, err := GetContext().newShader(VertexShader, shaderStr(shaderVertexModelview))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer GetContext().deleteShader(shaderVertexModelviewNative)

	shaderFragmentNearestNative, err := GetContext().newShader(FragmentShader, shaderStr(shaderFragmentNearest))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer GetContext().deleteShader(shaderFragmentNearestNative)

	shaderFragmentLinearNative, err := GetContext().newShader(FragmentShader, shaderStr(shaderFragmentLinear))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer GetContext().deleteShader(shaderFragmentLinearNative)

	shaderFragmentScreenNative, err := GetContext().newShader(FragmentShader, shaderStr(shaderFragmentScreen))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer GetContext().deleteShader(shaderFragmentScreenNative)

	s.programNearest, err = GetContext().newProgram([]shader{
		shaderVertexModelviewNative,
		shaderFragmentNearestNative,
	})
	if err != nil {
		return err
	}

	s.programLinear, err = GetContext().newProgram([]shader{
		shaderVertexModelviewNative,
		shaderFragmentLinearNative,
	})
	if err != nil {
		return err
	}

	s.programScreen, err = GetContext().newProgram([]shader{
		shaderVertexModelviewNative,
		shaderFragmentScreenNative,
	})
	if err != nil {
		return err
	}

	s.arrayBuffer = theArrayBufferLayout.newArrayBuffer()

	// Note that the indices passed to NewElementArrayBuffer is not under GC management
	// in opengl package due to unsafe-way.
	// See NewElementArrayBuffer in context_mobile.go.
	s.elementArrayBuffer = GetContext().newElementArrayBuffer(IndicesNum * 2)

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

func UseProgram(proj []float32, texture Texture, dstW, dstH, srcW, srcH int, colorM *affine.ColorM, filter graphics.Filter) {
	theOpenGLState.useProgram(proj, texture, dstW, dstH, srcW, srcH, colorM, filter)
}

// useProgram uses the program (programTexture).
func (s *openGLState) useProgram(proj []float32, texture Texture, dstW, dstH, srcW, srcH int, colorM *affine.ColorM, filter graphics.Filter) {
	c := GetContext()

	var program program
	switch filter {
	case graphics.FilterNearest:
		program = s.programNearest
	case graphics.FilterLinear:
		program = s.programLinear
	case graphics.FilterScreen:
		program = s.programScreen
	default:
		panic("not reached")
	}

	if s.lastProgram != program {
		c.useProgram(program)
		if s.lastProgram != zeroProgram {
			theArrayBufferLayout.disable(s.lastProgram)
		}
		theArrayBufferLayout.enable(program)

		if s.lastProgram == zeroProgram {
			c.BindBuffer(ArrayBuffer, s.arrayBuffer)
			c.BindBuffer(ElementArrayBuffer, s.elementArrayBuffer)
			c.uniformInt(program, "texture", 0)
		}

		s.lastProgram = program
		s.lastProjectionMatrix = nil
		s.lastColorMatrix = nil
		s.lastColorMatrixTranslation = nil
		s.lastSourceWidth = 0
		s.lastSourceHeight = 0
	}

	if !areSameFloat32Array(s.lastProjectionMatrix, proj) {
		c.uniformFloats(program, "projection_matrix", proj)
		if s.lastProjectionMatrix == nil {
			s.lastProjectionMatrix = make([]float32, 16)
		}
		// (*framebuffer).projectionMatrix is always same for the same framebuffer.
		// It's OK to hold the reference without copying.
		s.lastProjectionMatrix = proj
	}

	esBody, esTranslate := colorM.UnsafeElements()

	if !areSameFloat32Array(s.lastColorMatrix, esBody) {
		c.uniformFloats(program, "color_matrix_body", esBody)
		if s.lastColorMatrix == nil {
			s.lastColorMatrix = make([]float32, 16)
		}
		// ColorM's elements are immutable. It's OK to hold the reference without copying.
		s.lastColorMatrix = esBody
	}
	if !areSameFloat32Array(s.lastColorMatrixTranslation, esTranslate) {
		c.uniformFloats(program, "color_matrix_translation", esTranslate)
		if s.lastColorMatrixTranslation == nil {
			s.lastColorMatrixTranslation = make([]float32, 4)
		}
		// ColorM's elements are immutable. It's OK to hold the reference without copying.
		s.lastColorMatrixTranslation = esTranslate
	}

	sw := emath.NextPowerOf2Int(srcW)
	sh := emath.NextPowerOf2Int(srcH)

	if s.lastSourceWidth != sw || s.lastSourceHeight != sh {
		c.uniformFloats(program, "source_size", []float32{float32(sw), float32(sh)})
		s.lastSourceWidth = sw
		s.lastSourceHeight = sh
	}

	if program == s.programScreen {
		scale := float32(dstW) / float32(srcW)
		c.uniformFloat(program, "scale", scale)
	}

	// We don't have to call gl.ActiveTexture here: GL_TEXTURE0 is the default active texture
	// See also: https://www.opengl.org/sdk/docs/man2/xhtml/glActiveTexture.xml
	c.BindTexture(texture)
}
