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
	"errors"
	"fmt"
	"github.com/go-gl/gl"
)

type FilterType int
type ShaderType int
type BufferType int
type BufferUsageType int

type Texture gl.Texture

func (t Texture) Pixels(width, height int) ([]uint8, error) {
	// TODO: Use glGetTexLevelParameteri and GL_TEXTURE_WIDTH?
	pixels := make([]uint8, 4*width*height)
	gl.Texture(t).Bind(gl.TEXTURE_2D)
	gl.GetTexImage(gl.TEXTURE_2D, 0, gl.RGBA, gl.UNSIGNED_BYTE, pixels)
	if e := gl.GetError(); e != gl.NO_ERROR {
		// TODO: Use glu.ErrorString
		return nil, errors.New(fmt.Sprintf("gl error: %d", e))
	}
	return pixels, nil
}

func (t Texture) Delete() {
	gl.Texture(t).Delete()
}

type Framebuffer gl.Framebuffer

func (f Framebuffer) SetAsViewport(width, height int) error {
	gl.Flush()
	gl.Framebuffer(f).Bind()
	err := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	if err != gl.FRAMEBUFFER_COMPLETE {
		if gl.GetError() != 0 {
			return errors.New(fmt.Sprintf("glBindFramebuffer failed: %d", gl.GetError()))
		}
		return errors.New("glBindFramebuffer failed: the context is different?")
	}
	gl.Viewport(0, 0, width, height)
	return nil
}

func (f Framebuffer) Fill(r, g, b, a float64) error {
	gl.ClearColor(gl.GLclampf(r), gl.GLclampf(g), gl.GLclampf(b), gl.GLclampf(a))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	return nil
}

func (f Framebuffer) Delete() {
	gl.Framebuffer(f).Delete()
}

type Shader gl.Shader

func (s Shader) Delete() {
	gl.Shader(s).Delete()
}

type Program gl.Program

func (p Program) Use() {
	gl.Program(p).Use()
}

func (p Program) GetAttributeLocation(name string) AttribLocation {
	return AttribLocation(gl.Program(p).GetAttribLocation(name))
}

func (p Program) GetUniformLocation(name string) UniformLocation {
	return UniformLocation(gl.Program(p).GetUniformLocation(name))
}

type AttribLocation int

func (a AttribLocation) EnableArray() {
	gl.AttribLocation(a).EnableArray()
}

func (a AttribLocation) DisableArray() {
	gl.AttribLocation(a).DisableArray()
}

func (a AttribLocation) AttribPointer(stride int, x uintptr) {
	gl.AttribLocation(a).AttribPointer(2, gl.FLOAT, false, stride, x)
}

type UniformLocation int

func (u UniformLocation) UniformMatrix4fv(matrix [16]float32) {
	gl.UniformLocation(u).UniformMatrix4fv(false, matrix)
}

func (u UniformLocation) Uniform4fv(count int, v []float32) {
	gl.UniformLocation(u).Uniform4fv(count, v)
}

func (u UniformLocation) Uniform1i(v int) {
	gl.UniformLocation(u).Uniform1i(v)
}

type Context struct {
	Nearest            FilterType
	Linear             FilterType
	VertexShader       ShaderType
	FragmentShader     ShaderType
	ArrayBuffer        BufferType
	ElementArrayBuffer BufferType
	DynamicDraw        BufferUsageType
	StaticDraw         BufferUsageType
}

func NewContext() *Context {
	c := &Context{
		Nearest:            gl.NEAREST,
		Linear:             gl.LINEAR,
		VertexShader:       gl.VERTEX_SHADER,
		FragmentShader:     gl.FRAGMENT_SHADER,
		ArrayBuffer:        gl.ARRAY_BUFFER,
		ElementArrayBuffer: gl.ELEMENT_ARRAY_BUFFER,
		DynamicDraw:        gl.DYNAMIC_DRAW,
		StaticDraw:         gl.STATIC_DRAW,
	}
	c.init()
	return c
}

func (c *Context) init() {
	gl.Init()
	gl.Enable(gl.TEXTURE_2D)
	// Textures' pixel formats are alpha premultiplied.
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
}

func (c *Context) NewTexture(width, height int, pixels []uint8, filter FilterType) (Texture, error) {
	t := gl.GenTexture()
	if t < 0 {
		return 0, errors.New("glGenTexture failed")
	}
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	t.Bind(gl.TEXTURE_2D)
	defer gl.Texture(0).Bind(gl.TEXTURE_2D)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, int(filter))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, int(filter))

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, pixels)

	return Texture(t), nil
}

func (c *Context) NewFramebuffer(texture Texture) (Framebuffer, error) {
	f := gl.GenFramebuffer()
	f.Bind()

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, gl.Texture(texture), 0)
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		return 0, errors.New("creating framebuffer failed")
	}

	return Framebuffer(f), nil
}

func (c *Context) NewShader(shaderType ShaderType, source string) (Shader, error) {
	s := gl.CreateShader(gl.GLenum(shaderType))
	if s == 0 {
		println(gl.GetError())
		return 0, errors.New("glCreateShader failed")
	}

	s.Source(source)
	s.Compile()

	if s.Get(gl.COMPILE_STATUS) == gl.FALSE {
		log := ""
		if s.Get(gl.INFO_LOG_LENGTH) != 0 {
			log = s.GetInfoLog()
		}
		return 0, errors.New(fmt.Sprintf("shader compile failed: %s", log))
	}
	return Shader(s), nil
}

func (c *Context) NewProgram(shaders []Shader) (Program, error) {
	p := gl.CreateProgram()
	if p == 0 {
		return 0, errors.New("glCreateProgram failed")
	}

	for _, shader := range shaders {
		p.AttachShader(gl.Shader(shader))
	}
	p.Link()
	if p.Get(gl.LINK_STATUS) == gl.FALSE {
		return 0, errors.New("program error")
	}
	return Program(p), nil
}

func (c *Context) NewBuffer(bufferType BufferType, size int, ptr interface{}, bufferUsageType BufferUsageType) {
	gl.GenBuffer().Bind(gl.GLenum(bufferType))
	gl.BufferData(gl.GLenum(bufferType), size, ptr, gl.GLenum(bufferUsageType))
}
