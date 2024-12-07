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

//go:build !playstation5

package opengl

import (
	"fmt"
	"math"
	"runtime"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

const floatSizeInBytes = 4

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

func (a *arrayBufferLayout) names() []string {
	ns := make([]string, len(a.parts))
	for i, p := range a.parts {
		ns[i] = p.name
	}
	return ns
}

// float32Count returns the total float32 count for one element of the array buffer.
func (a *arrayBufferLayout) float32Count() int {
	if a.total != 0 {
		return a.total
	}
	t := 0
	for _, p := range a.parts {
		t += p.num
	}
	a.total = t
	return a.total
}

func (a *arrayBufferLayout) addPart(part arrayBufferLayoutPart) {
	a.parts = append(a.parts, part)
	a.total = 0
}

// enable starts using the array buffer.
func (a *arrayBufferLayout) enable(context *context) {
	for i := range a.parts {
		context.ctx.EnableVertexAttribArray(uint32(i))
	}
	total := a.float32Count()
	offset := 0
	for i, p := range a.parts {
		context.ctx.VertexAttribPointer(uint32(i), int32(p.num), gl.FLOAT, false, int32(floatSizeInBytes*total), offset)
		offset += floatSizeInBytes * p.num
	}
}

// disable stops using the array buffer.
func (a *arrayBufferLayout) disable(context *context) {
	// TODO: Disabling should be done in reversed order?
	for i := range a.parts {
		context.ctx.DisableVertexAttribArray(uint32(i))
	}
}

// theArrayBufferLayout is the array buffer layout for Ebitengine.
var theArrayBufferLayout arrayBufferLayout

func init() {
	theArrayBufferLayout = arrayBufferLayout{
		// Note that GL_MAX_VERTEX_ATTRIBS is at least 16.
		parts: []arrayBufferLayoutPart{
			{
				name: "A0",
				num:  2,
			},
			{
				name: "A1",
				num:  2,
			},
			{
				name: "A2",
				num:  4,
			},
		},
	}
	n := theArrayBufferLayout.float32Count()
	diff := graphics.VertexFloatCount - n
	if diff == 0 {
		return
	}
	if diff%4 != 0 {
		panic("opengl: unexpected attribute layout")
	}
	for i := 0; i < diff/4; i++ {
		theArrayBufferLayout.addPart(arrayBufferLayoutPart{
			name: fmt.Sprintf("A%d", i+3),
			num:  4,
		})
	}
}

type openGLState struct {
	vertexArray uint32

	// arrayBuffer is OpenGL's array buffer (vertices data).
	arrayBuffer buffer

	arrayBufferSizeInBytes int

	// elementArrayBuffer is OpenGL's element array buffer (indices data).
	elementArrayBuffer buffer

	elementArrayBufferSizeInBytes int

	lastProgram       program
	lastUniforms      map[string][]uint32
	lastActiveTexture int
}

// reset resets or initializes the OpenGL state.
func (s *openGLState) reset(context *context) error {
	if err := context.reset(); err != nil {
		return err
	}

	s.lastProgram = 0
	context.ctx.UseProgram(0)
	for key := range s.lastUniforms {
		delete(s.lastUniforms, key)
	}

	// On browsers (at least Chrome), buffers are already detached from the context
	// and must not be deleted by DeleteBuffer.
	if runtime.GOOS != "js" {
		if s.arrayBuffer != 0 {
			context.ctx.DeleteBuffer(uint32(s.arrayBuffer))
		}
		if s.elementArrayBuffer != 0 {
			context.ctx.DeleteBuffer(uint32(s.elementArrayBuffer))
		}
		if s.vertexArray != 0 {
			context.ctx.DeleteVertexArray(s.vertexArray)
		}
	}

	s.arrayBuffer = 0
	s.arrayBufferSizeInBytes = 0
	s.elementArrayBuffer = 0
	s.elementArrayBufferSizeInBytes = 0
	s.vertexArray = 0

	return nil
}

func pow2(x int) int {
	if x > (math.MaxInt+1)/2 {
		return math.MaxInt
	}

	p2 := 1
	for p2 < x {
		p2 *= 2
	}
	return p2
}

func (s *openGLState) setVertices(context *context, vertices []float32, indices []uint32) {
	if s.vertexArray == 0 {
		s.vertexArray = context.ctx.CreateVertexArray()
	}
	context.ctx.BindVertexArray(s.vertexArray)

	if size := len(vertices) * int(unsafe.Sizeof(vertices[0])); s.arrayBufferSizeInBytes < size {
		if s.arrayBuffer != 0 {
			context.ctx.DeleteBuffer(uint32(s.arrayBuffer))
		}

		newSize := pow2(size)
		// newArrayBuffer calls BindBuffer.
		s.arrayBuffer = context.newArrayBuffer(newSize)
		s.arrayBufferSizeInBytes = newSize

		// Reenable the array buffer layout explicitly after resetting the array buffer.
		theArrayBufferLayout.enable(context)
	}

	if size := len(indices) * int(unsafe.Sizeof(indices[0])); s.elementArrayBufferSizeInBytes < size {
		if s.elementArrayBuffer != 0 {
			context.ctx.DeleteBuffer(uint32(s.elementArrayBuffer))
		}

		newSize := pow2(size)
		// newElementArrayBuffer calls BindBuffer.
		s.elementArrayBuffer = context.newElementArrayBuffer(newSize)
		s.elementArrayBufferSizeInBytes = newSize
	}

	// Note that the vertices and the indices passed to BufferSubData is not under GC management in the gl package.
	vs := unsafe.Slice((*byte)(unsafe.Pointer(&vertices[0])), len(vertices)*int(unsafe.Sizeof(vertices[0])))
	context.ctx.BufferSubData(gl.ARRAY_BUFFER, 0, vs)
	is := unsafe.Slice((*byte)(unsafe.Pointer(&indices[0])), len(indices)*int(unsafe.Sizeof(indices[0])))
	context.ctx.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, 0, is)
}

func (s *openGLState) resetLastUniforms() {
	for k := range s.lastUniforms {
		delete(s.lastUniforms, k)
	}
}

// areSameUint32Array returns a boolean indicating if a and b are deeply equal.
func areSameUint32Array(a, b []uint32) bool {
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

type uniformVariable struct {
	name  string
	value []uint32
	typ   shaderir.Type
}

type textureVariable struct {
	valid  bool
	native textureNative
}

func (g *Graphics) textureVariableName(idx int) string {
	if v, ok := g.textureVariableNameCache[idx]; ok {
		return v
	}
	if g.textureVariableNameCache == nil {
		g.textureVariableNameCache = map[int]string{}
	}
	name := fmt.Sprintf("T%d", idx)
	g.textureVariableNameCache[idx] = name
	return name
}

// useProgram uses the program (programTexture).
func (g *Graphics) useProgram(program program, uniforms []uniformVariable, textures [graphics.ShaderSrcImageCount]textureVariable) error {
	if g.state.lastProgram != program {
		g.context.ctx.UseProgram(uint32(program))

		g.state.lastProgram = program
		for k := range g.state.lastUniforms {
			delete(g.state.lastUniforms, k)
		}
		g.state.lastActiveTexture = 0
		g.context.ctx.ActiveTexture(gl.TEXTURE0)
		g.context.lastTexture = 0 // Make sure next bindTexture call actually does something.
	}

	for _, u := range uniforms {
		if u.value == nil {
			continue
		}
		if got, expected := len(u.value), u.typ.DwordCount(); got != expected {
			// Copy a shaderir.Type value once. Do not pass u.typ directly to fmt.Errorf arguments, or
			// the value u would be allocated on heap.
			typ := u.typ
			return fmt.Errorf("opengl: length of a uniform variables %s (%s) doesn't match: expected %d but %d", u.name, typ.String(), expected, got)
		}

		cached, ok := g.state.lastUniforms[u.name]
		if ok && areSameUint32Array(cached, u.value) {
			continue
		}
		g.context.uniforms(program, u.name, u.value, u.typ)
		if g.state.lastUniforms == nil {
			g.state.lastUniforms = map[string][]uint32{}
		}
		g.state.lastUniforms[u.name] = u.value
	}

	var idx int
loop:
	for i, t := range textures {
		if !t.valid {
			continue
		}

		// If the texture is already bound, set the texture variable to point to the texture.
		// Rebinding the same texture seems problematic (#1193).
		for _, at := range g.activatedTextures {
			if t.native == at.textureNative {
				g.context.uniformInt(program, g.textureVariableName(i), at.index)
				continue loop
			}
		}

		g.activatedTextures = append(g.activatedTextures, activatedTexture{
			textureNative: t.native,
			index:         idx,
		})
		g.context.uniformInt(program, g.textureVariableName(i), idx)
		if g.state.lastActiveTexture != idx {
			g.context.ctx.ActiveTexture(uint32(gl.TEXTURE0 + idx))
			g.state.lastActiveTexture = idx
			g.context.lastTexture = 0 // Make sure next bindTexture call actually does something.
		}

		// Apparently, a texture must be bound every time. The cache is not used here.
		g.context.bindTexture(t.native)

		idx++
	}

	for i := range g.activatedTextures {
		g.activatedTextures[i] = activatedTexture{}
	}
	g.activatedTextures = g.activatedTextures[:0]

	return nil
}

func uint32sToFloat32s(s []uint32) []float32 {
	return unsafe.Slice((*float32)(unsafe.Pointer(&s[0])), len(s))
}

func uint32sToInt32s(s []uint32) []int32 {
	return unsafe.Slice((*int32)(unsafe.Pointer(&s[0])), len(s))
}
