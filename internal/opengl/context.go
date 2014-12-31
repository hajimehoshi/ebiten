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

// +build !js

package opengl

import (
	"errors"
	"fmt"
	"github.com/go-gl/gl"
)

type Texture gl.Texture
type Framebuffer gl.Framebuffer
type Shader gl.Shader
type Program gl.Program

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

func (c *Context) TexturePixels(t Texture, width, height int) ([]uint8, error) {
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

func (c *Context) BindTexture(t Texture) {
	gl.Texture(t).Bind(gl.TEXTURE_2D)
}

func (c *Context) DeleteTexture(t Texture) {
	gl.Texture(t).Delete()
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

func (c *Context) SetViewport(f Framebuffer, width, height int) error {
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

func (c *Context) FillFramebuffer(f Framebuffer, r, g, b, a float64) error {
	gl.ClearColor(gl.GLclampf(r), gl.GLclampf(g), gl.GLclampf(b), gl.GLclampf(a))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	return nil
}

func (c *Context) DeleteFramebuffer(f Framebuffer) {
	gl.Framebuffer(f).Delete()
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

func (c *Context) DeleteShader(s Shader) {
	gl.Shader(s).Delete()
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

func (c *Context) UseProgram(p Program) {
	gl.Program(p).Use()
}

func (c *Context) Uniform1i(p Program, location string, v int) {
	// TODO: Cache the location names.
	gl.Program(p).GetUniformLocation(location).Uniform1i(v)
}

func (c *Context) Uniform4fv(p Program, location string, v [4]float32) {
	gl.Program(p).GetUniformLocation(location).Uniform4fv(1, v[:])
}

func (c *Context) UniformMatrix4fv(p Program, location string, v [16]float32) {
	gl.Program(p).GetUniformLocation(location).UniformMatrix4fv(false, v)
}

func (c *Context) VertexAttribPointer(p Program, location string, stride int, v uintptr) {
	gl.Program(p).GetAttribLocation(location).AttribPointer(2, gl.FLOAT, false, stride, v)
}

func (c *Context) EnableVertexAttribArray(p Program, location string) {
	gl.Program(p).GetAttribLocation(location).EnableArray()
}

func (c *Context) DisableVertexAttribArray(p Program, location string) {
	gl.Program(p).GetAttribLocation(location).DisableArray()
}

func (c *Context) NewBuffer(bufferType BufferType, size int, ptr interface{}, bufferUsageType BufferUsageType) {
	gl.GenBuffer().Bind(gl.GLenum(bufferType))
	gl.BufferData(gl.GLenum(bufferType), size, ptr, gl.GLenum(bufferUsageType))
}

func (c *Context) BufferSubData(bufferType BufferType, data []float32) {
	const float32Size = 4
	gl.BufferSubData(gl.GLenum(bufferType), 0, float32Size*len(data), data)
}

func (c *Context) DrawElements(len int) {
	gl.DrawElements(gl.TRIANGLES, len, gl.UNSIGNED_SHORT, uintptr(0))
}

func (c *Context) Flush() {
	gl.Flush()
}
