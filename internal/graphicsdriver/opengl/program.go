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
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
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

// totalBytes returns the size in bytes for one element of the array buffer.
func (a *arrayBufferLayout) totalBytes() int {
	if a.total != 0 {
		return a.total
	}
	t := 0
	for _, p := range a.parts {
		t += floatSizeInBytes * p.num
	}
	a.total = t
	return a.total
}

// newArrayBuffer creates OpenGL's buffer object for the array buffer.
func (a *arrayBufferLayout) newArrayBuffer(context *context) buffer {
	return context.newArrayBuffer(a.totalBytes() * graphics.IndicesCount)
}

// enable starts using the array buffer.
func (a *arrayBufferLayout) enable(context *context) {
	for i := range a.parts {
		context.enableVertexAttribArray(i)
	}
	total := a.totalBytes()
	offset := 0
	for i, p := range a.parts {
		context.vertexAttribPointer(i, p.num, total, offset)
		offset += floatSizeInBytes * p.num
	}
}

// disable stops using the array buffer.
func (a *arrayBufferLayout) disable(context *context) {
	// TODO: Disabling should be done in reversed order?
	for i := range a.parts {
		context.disableVertexAttribArray(i)
	}
}

// theArrayBufferLayout is the array buffer layout for Ebiten.
var theArrayBufferLayout = arrayBufferLayout{
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

func init() {
	vertexFloatCount := theArrayBufferLayout.totalBytes() / floatSizeInBytes
	if graphics.VertexFloatCount != vertexFloatCount {
		panic(fmt.Sprintf("vertex float num must be %d but %d", graphics.VertexFloatCount, vertexFloatCount))
	}
}

type programKey struct {
	useColorM bool
	filter    graphicsdriver.Filter
	address   graphicsdriver.Address
}

// openGLState is a state for
type openGLState struct {
	// arrayBuffer is OpenGL's array buffer (vertices data).
	arrayBuffer buffer

	// elementArrayBuffer is OpenGL's element array buffer (indices data).
	elementArrayBuffer buffer

	// programs is OpenGL's program for rendering a texture.
	programs map[programKey]program

	lastProgram       program
	lastUniforms      map[string][]float32
	lastActiveTexture int
}

var (
	zeroBuffer  buffer
	zeroProgram program
)

// reset resets or initializes the OpenGL state.
func (s *openGLState) reset(context *context) error {
	if err := context.reset(); err != nil {
		return err
	}

	s.lastProgram = zeroProgram
	context.useProgram(zeroProgram)
	for key := range s.lastUniforms {
		delete(s.lastUniforms, key)
	}

	// When context lost happens, deleting programs or buffers is not necessary.
	// However, it is not assumed that reset is called only when context lost happens.
	// Let's delete them explicitly.
	if s.programs == nil {
		s.programs = map[programKey]program{}
	} else {
		for k, p := range s.programs {
			context.deleteProgram(p)
			delete(s.programs, k)
		}
	}

	// On browsers (at least Chrome), buffers are already detached from the context
	// and must not be deleted by DeleteBuffer.
	if runtime.GOOS != "js" {
		if !s.arrayBuffer.equal(zeroBuffer) {
			context.deleteBuffer(s.arrayBuffer)
		}
		if !s.elementArrayBuffer.equal(zeroBuffer) {
			context.deleteBuffer(s.elementArrayBuffer)
		}
	}

	shaderVertexModelviewNative, err := context.newVertexShader(vertexShaderStr())
	if err != nil {
		panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
	}
	defer context.deleteShader(shaderVertexModelviewNative)

	for _, c := range []bool{false, true} {
		for _, a := range []graphicsdriver.Address{
			graphicsdriver.AddressClampToZero,
			graphicsdriver.AddressRepeat,
			graphicsdriver.AddressUnsafe,
		} {
			for _, f := range []graphicsdriver.Filter{
				graphicsdriver.FilterNearest,
				graphicsdriver.FilterLinear,
				graphicsdriver.FilterScreen,
			} {
				shaderFragmentColorMatrixNative, err := context.newFragmentShader(fragmentShaderStr(c, f, a))
				if err != nil {
					panic(fmt.Sprintf("graphics: shader compiling error:\n%s", err))
				}
				defer context.deleteShader(shaderFragmentColorMatrixNative)

				program, err := context.newProgram([]shader{
					shaderVertexModelviewNative,
					shaderFragmentColorMatrixNative,
				}, theArrayBufferLayout.names())

				if err != nil {
					return err
				}

				s.programs[programKey{
					useColorM: c,
					filter:    f,
					address:   a,
				}] = program
			}
		}
	}

	s.arrayBuffer = theArrayBufferLayout.newArrayBuffer(context)

	// Note that the indices passed to NewElementArrayBuffer is not under GC management
	// in opengl package due to unsafe-way.
	// See NewElementArrayBuffer in context_mobile.go.
	s.elementArrayBuffer = context.newElementArrayBuffer(graphics.IndicesCount * 2)

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

type uniformVariable struct {
	name  string
	value []float32
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
func (g *Graphics) useProgram(program program, uniforms []uniformVariable, textures [graphics.ShaderImageCount]textureVariable) error {
	if !g.state.lastProgram.equal(program) {
		g.context.useProgram(program)
		if g.state.lastProgram.equal(zeroProgram) {
			theArrayBufferLayout.enable(&g.context)
			g.context.bindArrayBuffer(g.state.arrayBuffer)
			g.context.bindElementArrayBuffer(g.state.elementArrayBuffer)
		}

		g.state.lastProgram = program
		for k := range g.state.lastUniforms {
			delete(g.state.lastUniforms, k)
		}
		g.state.lastActiveTexture = 0
		g.context.activeTexture(0)
	}

	for _, u := range uniforms {
		if got, expected := len(u.value), u.typ.FloatCount(); got != expected {
			// Copy a shaderir.Type value once. Do not pass u.typ directly to fmt.Errorf arguments, or
			// the value u would be allocated on heap.
			typ := u.typ
			return fmt.Errorf("opengl: length of a uniform variables %s (%s) doesn't match: expected %d but %d", u.name, typ.String(), expected, got)
		}

		cached, ok := g.state.lastUniforms[u.name]
		if ok && areSameFloat32Array(cached, u.value) {
			continue
		}
		g.context.uniformFloats(program, u.name, u.value, u.typ)
		if g.state.lastUniforms == nil {
			g.state.lastUniforms = map[string][]float32{}
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
			if t.native.equal(at.textureNative) {
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
			g.context.activeTexture(idx)
			g.state.lastActiveTexture = idx
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
