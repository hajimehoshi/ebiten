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
	var t int
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
	var offset int
	for i, p := range a.parts {
		context.ctx.VertexAttribPointer(uint32(i), int32(p.num), gl.FLOAT, false, int32(floatSizeInBytes*total), offset)
		offset += floatSizeInBytes * p.num
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
	for i := range diff / 4 {
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
	lastUniforms      uniformCache
	lastActiveTexture int
}

// uniformCache keeps the last values of the uniform variables of each program, the sampler variables
// included, so that a draw does not send a value that the program already has.
// OpenGL keeps uniform values for each program object, so a switch of programs does not invalidate them.
// A slot names a variable of a program: samplerSlot for a sampler variable and uniformSlot for a uniform variable.
type uniformCache struct {
	programs       map[program]*programUniforms
	current        *programUniforms
	currentProgram program

	// generation invalidates every cached value at once when it changes.
	generation uint64
}

// programUniforms holds the cached values of one program, indexed by slot.
type programUniforms struct {
	values []cachedUniform
}

type cachedUniform struct {
	// value is an owned copy, as the caller's slice is reused for the next frame.
	value      []uint32
	generation uint64
}

// samplerSlot returns the slot of the sampler variable of the source image i.
func samplerSlot(i int) int {
	return i
}

// uniformSlot returns the slot of the uniform variable i.
func uniformSlot(i int) int {
	return graphics.ShaderSrcImageCount + i
}

// use makes p the program of the next calls of isSame and set.
func (c *uniformCache) use(p program) {
	if c.programs == nil {
		c.programs = map[program]*programUniforms{}
	}
	m, ok := c.programs[p]
	if !ok {
		m = &programUniforms{}
		c.programs[p] = m
	}
	c.current = m
	c.currentProgram = p
}

// isSame reports whether the current program has the value for the variable of the slot.
func (c *uniformCache) isSame(slot int, value []uint32) bool {
	if c.current == nil || slot >= len(c.current.values) {
		return false
	}
	cached := &c.current.values[slot]
	return cached.generation == c.generation && areSameUint32Array(cached.value, value)
}

// set records that the current program has the value for the variable of the slot.
func (c *uniformCache) set(slot int, value []uint32) {
	if slot >= len(c.current.values) {
		c.current.values = append(c.current.values, make([]cachedUniform, slot+1-len(c.current.values))...)
	}
	cached := &c.current.values[slot]
	cached.value = append(cached.value[:0], value...)
	cached.generation = c.generation
}

// invalidate forgets the values of every program, and keeps the storage for the next values.
func (c *uniformCache) invalidate() {
	c.generation++
}

// deleteProgram forgets the values of p, as a new program can reuse its name.
func (c *uniformCache) deleteProgram(p program) {
	delete(c.programs, p)
	if c.currentProgram == p {
		c.current = nil
		c.currentProgram = 0
	}
}

// clear forgets every program, for a new context.
func (c *uniformCache) clear() {
	clear(c.programs)
	c.current = nil
	c.currentProgram = 0
}

// reset resets or initializes the OpenGL state.
func (s *openGLState) reset(context *context) error {
	if err := context.reset(); err != nil {
		return err
	}

	s.lastProgram = 0
	context.ctx.UseProgram(0)
	s.lastUniforms.clear()

	if s.arrayBuffer != 0 {
		context.ctx.DeleteBuffer(uint32(s.arrayBuffer))
	}
	if s.elementArrayBuffer != 0 {
		context.ctx.DeleteBuffer(uint32(s.elementArrayBuffer))
	}
	if s.vertexArray != 0 {
		context.ctx.DeleteVertexArray(s.vertexArray)
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

	// Note that the vertices and the indices passed to BufferSubData are not under GC management in the gl package.
	vs := unsafe.Slice((*byte)(unsafe.Pointer(&vertices[0])), len(vertices)*int(unsafe.Sizeof(vertices[0])))
	context.ctx.BufferSubData(gl.ARRAY_BUFFER, 0, vs)
	is := unsafe.Slice((*byte)(unsafe.Pointer(&indices[0])), len(indices)*int(unsafe.Sizeof(indices[0])))
	context.ctx.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, 0, is)
}

func (s *openGLState) resetLastUniforms() {
	s.lastUniforms.invalidate()
}

// areSameUint32Array returns a boolean indicating if a and b are deeply equal.
func areSameUint32Array(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
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

// textureVariableNames holds the names of the sampler variables of the source images.
var textureVariableNames = func() [graphics.ShaderSrcImageCount]string {
	var names [graphics.ShaderSrcImageCount]string
	for i := range names {
		names[i] = fmt.Sprintf("T%d", i)
	}
	return names
}()

func (g *Graphics) deleteProgram(p program) {
	// A name of a deleted program can be reused for a new program.
	// Reset lastProgram so that useProgram doesn't skip the state updates for such a new program.
	if g.state.lastProgram == p {
		g.state.lastProgram = 0
	}
	g.state.lastUniforms.deleteProgram(p)
	g.context.deleteProgram(p)
}

// useProgram uses the program with the given uniforms and textures.
func (g *Graphics) useProgram(program program, uniforms []uniformVariable, textures [graphics.ShaderSrcImageCount]textureVariable) error {
	if g.state.lastProgram != program {
		g.context.ctx.UseProgram(uint32(program))

		g.state.lastProgram = program
		g.state.lastUniforms.use(program)
		g.state.lastActiveTexture = 0
		g.context.ctx.ActiveTexture(gl.TEXTURE0)
		g.context.lastTexture = 0 // Make sure next bindTexture call actually does something.
	}

	for i, u := range uniforms {
		if u.value == nil {
			continue
		}
		if got, expected := len(u.value), u.typ.DwordCount(); got != expected {
			// Copy a shaderir.Type value once. Do not pass u.typ directly to fmt.Errorf arguments, or
			// the value u would be allocated on heap.
			typ := u.typ
			return fmt.Errorf("opengl: length of a uniform variables %s (%s) doesn't match: expected %d but %d", u.name, typ.String(), expected, got)
		}

		if g.state.lastUniforms.isSame(uniformSlot(i), u.value) {
			continue
		}
		g.context.uniforms(program, u.name, u.value, u.typ)
		g.state.lastUniforms.set(uniformSlot(i), u.value)
	}

	var idx int
loop:
	for i, t := range textures {
		native := t.native
		// A sampler for a slot without an image must not sample a texture left by an earlier draw,
		// as the texture might be the destination (#3923). Let such samplers sample an empty texture.
		if !t.valid {
			if !g.context.hasUniform(program, textureVariableNames[i]) {
				continue
			}
			if g.context.emptyTexture == 0 {
				// Creating a texture binds it to the active texture unit. Activate the unit idx first:
				// the new texture takes it, and no texture of this draw is bound to it yet.
				g.activateTextureUnit(idx)
				if err := g.context.ensureEmptyTexture(); err != nil {
					return err
				}
			}
			native = g.context.emptyTexture
		}

		// If the texture is already bound, set the texture variable to point to the texture.
		// Rebinding the same texture seems problematic (#1193).
		for _, at := range g.activatedTextures {
			if native == at.textureNative {
				g.setSampler(program, i, at.index)
				continue loop
			}
		}

		g.activatedTextures = append(g.activatedTextures, activatedTexture{
			textureNative: native,
			index:         idx,
		})
		g.setSampler(program, i, idx)
		g.activateTextureUnit(idx)

		// Apparently, a texture must be bound every time. The cache is not used here.
		g.context.bindTexture(native)

		idx++
	}

	for i := range g.activatedTextures {
		g.activatedTextures[i] = activatedTexture{}
	}
	g.activatedTextures = g.activatedTextures[:0]

	return nil
}

// activateTextureUnit makes the texture unit at index active.
func (g *Graphics) activateTextureUnit(index int) {
	if g.state.lastActiveTexture == index {
		return
	}
	g.context.ctx.ActiveTexture(uint32(gl.TEXTURE0 + index))
	g.state.lastActiveTexture = index
	g.context.lastTexture = 0 // Make sure next bindTexture call actually does something.
}

// setSampler sets the sampler variable of the texture i to the texture unit, unless the program has it already.
func (g *Graphics) setSampler(program program, i int, unit int) {
	v := [...]uint32{uint32(unit)}
	if g.state.lastUniforms.isSame(samplerSlot(i), v[:]) {
		return
	}
	g.context.uniformInt(program, textureVariableNames[i], unit)
	g.state.lastUniforms.set(samplerSlot(i), v[:])
}

func uint32sToFloat32s(s []uint32) []float32 {
	return unsafe.Slice((*float32)(unsafe.Pointer(&s[0])), len(s))
}

func uint32sToInt32s(s []uint32) []int32 {
	return unsafe.Slice((*int32)(unsafe.Pointer(&s[0])), len(s))
}
