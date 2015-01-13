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

// +build js

package opengl

import (
	"errors"
	"fmt"
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/webgl"
)

type Texture js.Object
type Framebuffer js.Object
type Shader js.Object
type Program js.Object
type UniformLocation js.Object
type AttribLocation int

type ProgramID int

func GetProgramID(p Program) ProgramID {
	return ProgramID(p.Get("__ebiten_programId").Int())
}

type context struct {
	gl *webgl.Context
}

func NewContext(gl *webgl.Context) *Context {
	c := &Context{
		Nearest:            FilterType(gl.NEAREST),
		Linear:             FilterType(gl.LINEAR),
		VertexShader:       ShaderType(gl.VERTEX_SHADER),
		FragmentShader:     ShaderType(gl.FRAGMENT_SHADER),
		ArrayBuffer:        BufferType(gl.ARRAY_BUFFER),
		ElementArrayBuffer: BufferType(gl.ELEMENT_ARRAY_BUFFER),
		DynamicDraw:        BufferUsageType(gl.DYNAMIC_DRAW),
		StaticDraw:         BufferUsageType(gl.STATIC_DRAW),
	}
	c.gl = gl
	c.init()
	return c
}

func (c *Context) init() {
	gl := c.gl
	// Textures' pixel formats are alpha premultiplied.
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
}

func (c *Context) NewTexture(width, height int, pixels []uint8, filter FilterType) (Texture, error) {
	gl := c.gl
	t := gl.CreateTexture()
	if t == nil {
		return nil, errors.New("glGenTexture failed")
	}
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	gl.BindTexture(gl.TEXTURE_2D, t)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, int(filter))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, int(filter))

	// void texImage2D(GLenum target, GLint level, GLenum internalformat,
	//     GLsizei width, GLsizei height, GLint border, GLenum format,
	//     GLenum type, ArrayBufferView? pixels);
	var p interface{}
	if pixels != nil {
		p = pixels
	}
	gl.Call("texImage2D", gl.TEXTURE_2D, 0, gl.RGBA, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, p)

	return Texture(t), nil
}

func (c *Context) TexturePixels(t Texture, width, height int) ([]uint8, error) {
	gl := c.gl
	gl.Flush()
	// TODO: Use glGetTexLevelParameteri and GL_TEXTURE_WIDTH?
	pixels := js.Global.Get("Uint8Array").New(4 * width * height)
	gl.BindTexture(gl.TEXTURE_2D, t)
	gl.ReadPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, pixels)
	if e := gl.GetError(); e != gl.NO_ERROR {
		return nil, errors.New(fmt.Sprintf("gl error: %d", e))
	}
	return pixels.Interface().([]uint8), nil
}

func (c *Context) BindTexture(t Texture) {
	gl := c.gl
	gl.BindTexture(gl.TEXTURE_2D, t)
}

func (c *Context) DeleteTexture(t Texture) {
	gl := c.gl
	gl.DeleteTexture(t)
}

func (c *Context) GlslHighpSupported() bool {
	gl := c.gl
	// node-gl library may not define getShaderPrecisionFormat.
	if gl.Get("getShaderPrecisionFormat") == js.Undefined {
		return false
	}
	return gl.Call("getShaderPrecisionFormat", gl.FRAGMENT_SHADER, gl.HIGH_FLOAT).Get("precision").Int() != 0
}

func (c *Context) NewFramebuffer(texture Texture) (Framebuffer, error) {
	gl := c.gl
	f := gl.CreateFramebuffer()
	gl.BindFramebuffer(gl.FRAMEBUFFER, f)

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, texture, 0)
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		return nil, errors.New("creating framebuffer failed")
	}

	return Framebuffer(f), nil
}

var lastFramebuffer Framebuffer

func (c *Context) SetViewport(f Framebuffer, width, height int) error {
	gl := c.gl
	if lastFramebuffer != f {
		gl.Flush()
		lastFramebuffer = f
	}
	if f != nil {
		gl.BindFramebuffer(gl.FRAMEBUFFER, f)
	} else {
		gl.BindFramebuffer(gl.FRAMEBUFFER, nil)
	}
	// gl.CheckFramebufferStatus might cause a performance problem. Don't call this here.
	gl.Viewport(0, 0, width, height)
	return nil
}

func (c *Context) FillFramebuffer(f Framebuffer, r, g, b, a float64) error {
	gl := c.gl
	gl.ClearColor(float32(r), float32(g), float32(b), float32(a))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	return nil
}

func (c *Context) DeleteFramebuffer(f Framebuffer) {
	gl := c.gl
	gl.DeleteFramebuffer(f)
}

func (c *Context) NewShader(shaderType ShaderType, source string) (Shader, error) {
	gl := c.gl
	s := gl.CreateShader(int(shaderType))
	if s == nil {
		println(gl.GetError())
		return nil, errors.New("glCreateShader failed")
	}

	gl.ShaderSource(s, source)
	gl.CompileShader(s)

	if !gl.GetShaderParameterb(s, gl.COMPILE_STATUS) {
		log := gl.GetShaderInfoLog(s)
		return nil, errors.New(fmt.Sprintf("shader compile failed: %s", log))
	}
	return Shader(s), nil
}

func (c *Context) DeleteShader(s Shader) {
	gl := c.gl
	gl.DeleteShader(s)
}

var lastProgramID ProgramID = 0

func (c *Context) NewProgram(shaders []Shader) (Program, error) {
	gl := c.gl
	p := gl.CreateProgram()
	if p == nil {
		return nil, errors.New("glCreateProgram failed")
	}
	p.Set("__ebiten_programId", lastProgramID)
	lastProgramID++

	for _, shader := range shaders {
		gl.AttachShader(p, shader)
	}
	gl.LinkProgram(p)
	if !gl.GetProgramParameterb(p, gl.LINK_STATUS) {
		return nil, errors.New("program error")
	}
	return Program(p), nil
}

func (c *Context) UseProgram(p Program) {
	gl := c.gl
	gl.UseProgram(p)
}

func (c *Context) GetUniformLocation(p Program, location string) UniformLocation {
	gl := c.gl
	return gl.GetUniformLocation(p, location)
}

func (c *Context) UniformInt(p Program, location string, v int) {
	gl := c.gl
	l := GetUniformLocation(c, p, location)
	gl.Uniform1i(l, v)
}

func (c *Context) UniformFloats(p Program, location string, v []float32) {
	gl := c.gl
	l := GetUniformLocation(c, p, location)
	switch len(v) {
	case 4:
		gl.Call("uniform4fv", l, v)
	case 16:
		gl.UniformMatrix4fv(l, false, v)
	default:
		panic("not reach")
	}
}

func (c *Context) GetAttribLocation(p Program, location string) AttribLocation {
	gl := c.gl
	return AttribLocation(gl.GetAttribLocation(p, location))
}

func (c *Context) VertexAttribPointer(p Program, location string, stride int, v uintptr) {
	gl := c.gl
	l := GetAttribLocation(c, p, location)
	gl.VertexAttribPointer(int(l), 2, gl.FLOAT, false, stride, int(v))
}

func (c *Context) EnableVertexAttribArray(p Program, location string) {
	gl := c.gl
	l := GetAttribLocation(c, p, location)
	gl.EnableVertexAttribArray(int(l))
}

func (c *Context) DisableVertexAttribArray(p Program, location string) {
	gl := c.gl
	l := GetAttribLocation(c, p, location)
	gl.DisableVertexAttribArray(int(l))
}

func (c *Context) NewBuffer(bufferType BufferType, v interface{}, bufferUsageType BufferUsageType) {
	gl := c.gl
	b := gl.CreateBuffer()
	gl.BindBuffer(int(bufferType), b)
	gl.BufferData(int(bufferType), v, int(bufferUsageType))
}

func (c *Context) BufferSubData(bufferType BufferType, data []float32) {
	gl := c.gl
	const float32Size = 4
	gl.BufferSubData(int(bufferType), 0, data)
}

func (c *Context) DrawElements(len int) {
	gl := c.gl
	gl.DrawElements(gl.TRIANGLES, len, gl.UNSIGNED_SHORT, 0)
}

func (c *Context) Flush() {
	gl := c.gl
	gl.Flush()
}
