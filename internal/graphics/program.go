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

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type arrayBufferLayoutPart struct {
	// TODO: This struct should belong to a program and know it.
	name      string
	dataType  opengl.DataType
	num       int
	normalize bool
}

type arrayBufferLayout struct {
	parts []arrayBufferLayoutPart
}

func (a *arrayBufferLayout) newArrayBuffer(c *opengl.Context) opengl.Buffer {
	total := 0
	for _, p := range a.parts {
		total += p.dataType.SizeInBytes() * p.num
	}
	return c.NewBuffer(opengl.ArrayBuffer, total*4*maxQuads, opengl.DynamicDraw)
}

func (a *arrayBufferLayout) enable(c *opengl.Context, program opengl.Program) {
	for _, p := range a.parts {
		c.EnableVertexAttribArray(program, p.name)
	}
	total := 0
	for _, p := range a.parts {
		total += p.dataType.SizeInBytes() * p.num
	}
	offset := 0
	for _, p := range a.parts {
		c.VertexAttribPointer(program, p.name, p.num, p.dataType, p.normalize, total, offset)
		offset += p.dataType.SizeInBytes() * p.num
	}
}

func (a *arrayBufferLayout) disable(c *opengl.Context, program opengl.Program) {
	// TODO: Disabling should be done in reversed order?
	for _, p := range a.parts {
		c.DisableVertexAttribArray(program, p.name)
	}
}

var (
	theArrayBufferLayout = arrayBufferLayout{
		parts: []arrayBufferLayoutPart{
			{
				name:      "vertex",
				dataType:  opengl.Short,
				num:       2,
				normalize: false,
			},
			{
				name:      "tex_coord",
				dataType:  opengl.Short,
				num:       2,
				normalize: true,
			},
		},
	}
)

type openGLState struct {
	arrayBuffer      opengl.Buffer
	indexBufferQuads opengl.Buffer
	programTexture   opengl.Program

	lastProgram                opengl.Program
	lastProjectionMatrix       []float32
	lastModelviewMatrix        []float32
	lastColorMatrix            []float32
	lastColorMatrixTranslation []float32
}

var (
	theOpenGLState openGLState

	zeroBuffer  opengl.Buffer
	zeroProgram opengl.Program
	zeroTexture opengl.Texture
)

const (
	indicesNum = 1 << 16
	maxQuads   = indicesNum / 6
)

func Reset(context *opengl.Context) error {
	return theOpenGLState.reset(context)
}

func (s *openGLState) reset(context *opengl.Context) error {
	if err := context.Reset(); err != nil {
		return err
	}
	s.lastProgram = zeroProgram
	s.lastProjectionMatrix = nil
	s.lastModelviewMatrix = nil
	s.lastColorMatrix = nil
	s.lastColorMatrixTranslation = nil

	if s.arrayBuffer != zeroBuffer {
		context.DeleteBuffer(s.arrayBuffer)
	}
	if s.indexBufferQuads != zeroBuffer {
		context.DeleteBuffer(s.indexBufferQuads)
	}
	if s.programTexture != zeroProgram {
		context.DeleteProgram(s.programTexture)
	}

	shaderVertexModelviewNative, err := context.NewShader(opengl.VertexShader, shader(context, shaderVertexModelview))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer context.DeleteShader(shaderVertexModelviewNative)

	shaderFragmentTextureNative, err := context.NewShader(opengl.FragmentShader, shader(context, shaderFragmentTexture))
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer context.DeleteShader(shaderFragmentTextureNative)

	s.programTexture, err = context.NewProgram([]opengl.Shader{
		shaderVertexModelviewNative,
		shaderFragmentTextureNative,
	})
	if err != nil {
		return err
	}

	s.arrayBuffer = theArrayBufferLayout.newArrayBuffer(context)

	indices := make([]uint16, 6*maxQuads)
	for i := uint16(0); i < maxQuads; i++ {
		indices[6*i+0] = 4*i + 0
		indices[6*i+1] = 4*i + 1
		indices[6*i+2] = 4*i + 2
		indices[6*i+3] = 4*i + 1
		indices[6*i+4] = 4*i + 2
		indices[6*i+5] = 4*i + 3
	}
	s.indexBufferQuads = context.NewBuffer(opengl.ElementArrayBuffer, indices, opengl.StaticDraw)

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
	context          *opengl.Context
	projectionMatrix []float32
	texture          opengl.Texture
	geoM             Matrix
	colorM           Matrix
}

func (p *programContext) begin() error {
	c := p.context
	if p.state.lastProgram != p.program {
		c.UseProgram(p.program)
		if p.state.lastProgram != zeroProgram {
			theArrayBufferLayout.disable(c, p.state.lastProgram)
		}
		theArrayBufferLayout.enable(c, p.program)

		p.state.lastProgram = p.state.programTexture
		p.state.lastProjectionMatrix = nil
		p.state.lastModelviewMatrix = nil
		p.state.lastColorMatrix = nil
		p.state.lastColorMatrixTranslation = nil
		c.BindElementArrayBuffer(p.state.indexBufferQuads)
		c.UniformInt(p.program, "texture", 0)
	}

	if !areSameFloat32Array(p.state.lastProjectionMatrix, p.projectionMatrix) {
		c.UniformFloats(p.program, "projection_matrix", p.projectionMatrix)
		if p.state.lastProjectionMatrix == nil {
			p.state.lastProjectionMatrix = make([]float32, 16)
		}
		copy(p.state.lastProjectionMatrix, p.projectionMatrix)
	}

	ma := float32(p.geoM.Element(0, 0))
	mb := float32(p.geoM.Element(0, 1))
	mc := float32(p.geoM.Element(1, 0))
	md := float32(p.geoM.Element(1, 1))
	tx := float32(p.geoM.Element(0, 2))
	ty := float32(p.geoM.Element(1, 2))
	modelviewMatrix := []float32{
		ma, mc, 0, 0,
		mb, md, 0, 0,
		0, 0, 1, 0,
		tx, ty, 0, 1,
	}
	if !areSameFloat32Array(p.state.lastModelviewMatrix, modelviewMatrix) {
		c.UniformFloats(p.program, "modelview_matrix", modelviewMatrix)
		if p.state.lastModelviewMatrix == nil {
			p.state.lastModelviewMatrix = make([]float32, 16)
		}
		copy(p.state.lastModelviewMatrix, modelviewMatrix)
	}

	e := [4][5]float32{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			e[i][j] = float32(p.colorM.Element(i, j))
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

func (p *programContext) end() {
}
