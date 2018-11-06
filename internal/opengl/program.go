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
	name string
	num  int
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
		t += float.SizeInBytes() * p.num
	}
	a.total = t
	return a.total
}

// newArrayBuffer creates OpenGL's buffer object for the array buffer.
func (a *arrayBufferLayout) newArrayBuffer() buffer {
	return theContext.newArrayBuffer(a.totalBytes() * graphics.IndicesNum)
}

// enable binds the array buffer the given program to use the array buffer.
func (a *arrayBufferLayout) enable(program program) {
	for _, p := range a.parts {
		theContext.enableVertexAttribArray(program, p.name)
	}
	total := a.totalBytes()
	offset := 0
	for _, p := range a.parts {
		theContext.vertexAttribPointer(program, p.name, p.num, float, total, offset)
		offset += float.SizeInBytes() * p.num
	}
}

// disable stops using the array buffer.
func (a *arrayBufferLayout) disable(program program) {
	// TODO: Disabling should be done in reversed order?
	for _, p := range a.parts {
		theContext.disableVertexAttribArray(program, p.name)
	}
}

// theArrayBufferLayout is the array buffer layout for Ebiten.
var theArrayBufferLayout arrayBufferLayout

func initializeArrayBuferLayout() {
	theArrayBufferLayout = arrayBufferLayout{
		// Note that GL_MAX_VERTEX_ATTRIBS is at least 16.
		parts: []arrayBufferLayoutPart{
			{
				name: "vertex",
				num:  2,
			},
			{
				name: "tex_coord",
				num:  4,
			},
			{
				name: "color_scale",
				num:  4,
			},
		},
	}
}

func init() {
	vertexFloatNum := theArrayBufferLayout.totalBytes() / float.SizeInBytes()
	if graphics.VertexFloatNum != vertexFloatNum {
		panic(fmt.Sprintf("vertex float num must be %d but %d", graphics.VertexFloatNum, vertexFloatNum))
	}
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

	source      *Image
	destination *Image
}

var (
	zeroBuffer  buffer
	zeroProgram program
)

const (
	maxTriangles = graphics.IndicesNum / 3
	maxQuads     = maxTriangles / 2
)

// reset resets or initializes the OpenGL state.
func (s *openGLState) reset() error {
	if err := theContext.reset(); err != nil {
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
		theContext.deleteProgram(s.programNearest)
	}
	if s.programLinear != zeroProgram {
		theContext.deleteProgram(s.programLinear)
	}
	if s.programScreen != zeroProgram {
		theContext.deleteProgram(s.programScreen)
	}

	// On browsers (at least Chrome), buffers are already detached from the context
	// and must not be deleted by DeleteBuffer.
	if !web.IsBrowser() {
		if s.arrayBuffer != zeroBuffer {
			theContext.deleteBuffer(s.arrayBuffer)
		}
		if s.elementArrayBuffer != zeroBuffer {
			theContext.deleteBuffer(s.elementArrayBuffer)
		}
	}

	shaderVertexModelviewNative, err := theContext.newShader(vertexShader, shaderStr(shaderVertexModelview))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer theContext.deleteShader(shaderVertexModelviewNative)

	shaderFragmentNearestNative, err := theContext.newShader(fragmentShader, shaderStr(shaderFragmentNearest))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer theContext.deleteShader(shaderFragmentNearestNative)

	shaderFragmentLinearNative, err := theContext.newShader(fragmentShader, shaderStr(shaderFragmentLinear))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer theContext.deleteShader(shaderFragmentLinearNative)

	shaderFragmentScreenNative, err := theContext.newShader(fragmentShader, shaderStr(shaderFragmentScreen))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer theContext.deleteShader(shaderFragmentScreenNative)

	s.programNearest, err = theContext.newProgram([]shader{
		shaderVertexModelviewNative,
		shaderFragmentNearestNative,
	})
	if err != nil {
		return err
	}

	s.programLinear, err = theContext.newProgram([]shader{
		shaderVertexModelviewNative,
		shaderFragmentLinearNative,
	})
	if err != nil {
		return err
	}

	s.programScreen, err = theContext.newProgram([]shader{
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
	s.elementArrayBuffer = theContext.newElementArrayBuffer(graphics.IndicesNum * 2)

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

func bufferSubData(vertices []float32, indices []uint16) {
	theContext.arrayBufferSubData(vertices)
	theContext.elementArrayBufferSubData(indices)
}

// useProgram uses the program (programTexture).
func (s *openGLState) useProgram(mode graphics.CompositeMode, colorM *affine.ColorM, filter graphics.Filter) error {
	destination := s.destination
	if destination == nil {
		panic("destination image is not set")
	}
	source := s.source
	if source == nil {
		panic("source image is not set")
	}

	// On some environments, viewport size must be within the framebuffer size.
	// e.g. Edge (#71), Chrome on GPD Pocket (#420), macOS Mojave (#691).
	// Use the same size of the framebuffer here.
	if err := destination.setViewport(); err != nil {
		return err
	}
	proj := destination.projectionMatrix()
	dstW := destination.width
	srcW, srcH := source.width, source.height

	theContext.blendFunc(mode)

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
		theContext.useProgram(program)
		if s.lastProgram != zeroProgram {
			theArrayBufferLayout.disable(s.lastProgram)
		}
		theArrayBufferLayout.enable(program)

		if s.lastProgram == zeroProgram {
			theContext.bindBuffer(arrayBuffer, s.arrayBuffer)
			theContext.bindBuffer(elementArrayBuffer, s.elementArrayBuffer)
			theContext.uniformInt(program, "texture", 0)
		}

		s.lastProgram = program
		s.lastProjectionMatrix = nil
		s.lastColorMatrix = nil
		s.lastColorMatrixTranslation = nil
		s.lastSourceWidth = 0
		s.lastSourceHeight = 0
	}

	if !areSameFloat32Array(s.lastProjectionMatrix, proj) {
		theContext.uniformFloats(program, "projection_matrix", proj)
		if s.lastProjectionMatrix == nil {
			s.lastProjectionMatrix = make([]float32, 16)
		}
		// (*framebuffer).projectionMatrix is always same for the same framebuffer.
		// It's OK to hold the reference without copying.
		s.lastProjectionMatrix = proj
	}

	esBody, esTranslate := colorM.UnsafeElements()

	if !areSameFloat32Array(s.lastColorMatrix, esBody) {
		theContext.uniformFloats(program, "color_matrix_body", esBody)
		if s.lastColorMatrix == nil {
			s.lastColorMatrix = make([]float32, 16)
		}
		// ColorM's elements are immutable. It's OK to hold the reference without copying.
		s.lastColorMatrix = esBody
	}
	if !areSameFloat32Array(s.lastColorMatrixTranslation, esTranslate) {
		theContext.uniformFloats(program, "color_matrix_translation", esTranslate)
		if s.lastColorMatrixTranslation == nil {
			s.lastColorMatrixTranslation = make([]float32, 4)
		}
		// ColorM's elements are immutable. It's OK to hold the reference without copying.
		s.lastColorMatrixTranslation = esTranslate
	}

	sw := emath.NextPowerOf2Int(srcW)
	sh := emath.NextPowerOf2Int(srcH)

	if s.lastSourceWidth != sw || s.lastSourceHeight != sh {
		theContext.uniformFloats(program, "source_size", []float32{float32(sw), float32(sh)})
		s.lastSourceWidth = sw
		s.lastSourceHeight = sh
	}

	if program == s.programScreen {
		scale := float32(dstW) / float32(srcW)
		theContext.uniformFloat(program, "scale", scale)
	}

	// We don't have to call gl.ActiveTexture here: GL_TEXTURE0 is the default active texture
	// See also: https://www.opengl.org/sdk/docs/man2/xhtml/glActiveTexture.xml
	theContext.bindTexture(source.textureNative)

	s.source = nil
	s.destination = nil
	return nil
}
