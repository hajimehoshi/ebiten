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

type Texture struct {
	*js.Object
}

type Framebuffer struct {
	*js.Object
}

var ZeroFramebuffer Framebuffer

type Shader struct {
	*js.Object
}

type Program struct {
	*js.Object
}

type Buffer struct {
	*js.Object
}

// TODO: Remove this after the GopherJS bug was fixed (#159)
func (p Program) Equals(other Program) bool {
	return p.Object == other.Object
}

type UniformLocation struct {
	*js.Object
}

type AttribLocation int

type ProgramID int

func GetProgramID(p Program) ProgramID {
	return ProgramID(p.Get("__ebiten_programId").Int())
}

type context struct {
	gl              *webgl.Context
	lastFramebuffer Framebuffer
}

func NewContext() *Context {
	var gl *webgl.Context

	if js.Global.Get("require") == js.Undefined {
		// TODO: Define id?
		canvas := js.Global.Get("document").Call("querySelector", "canvas")
		var err error
		gl, err = webgl.NewContext(canvas, &webgl.ContextAttributes{
			Alpha:              true,
			PremultipliedAlpha: true,
		})
		if err != nil {
			panic(err)
		}
	} else {
		// Use headless-gl for testing.
		webglContext := js.Global.Call("require", "gl").Invoke(16, 16)
		gl = &webgl.Context{Object: webglContext}
	}

	c := &Context{
		Nearest:            Filter(gl.NEAREST),
		Linear:             Filter(gl.LINEAR),
		VertexShader:       ShaderType(gl.VERTEX_SHADER),
		FragmentShader:     ShaderType(gl.FRAGMENT_SHADER),
		ArrayBuffer:        BufferType(gl.ARRAY_BUFFER),
		ElementArrayBuffer: BufferType(gl.ELEMENT_ARRAY_BUFFER),
		DynamicDraw:        BufferUsage(gl.DYNAMIC_DRAW),
		StaticDraw:         BufferUsage(gl.STATIC_DRAW),
		Triangles:          Mode(gl.TRIANGLES),
		Lines:              Mode(gl.LINES),
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

func (c *Context) Check() {
	gl := c.gl
	if e := gl.GetError(); e != gl.NO_ERROR {
		panic(fmt.Sprintf("check failed: %d", e))
	}
}

func (c *Context) NewTexture(width, height int, pixels []uint8, filter Filter) (Texture, error) {
	gl := c.gl
	t := gl.CreateTexture()
	if t == nil {
		return Texture{nil}, errors.New("glGenTexture failed")
	}
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	gl.BindTexture(gl.TEXTURE_2D, t)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, int(filter))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, int(filter))

	// TODO: Can we use glTexSubImage2D with linear filtering?

	// void texImage2D(GLenum target, GLint level, GLenum internalformat,
	//     GLsizei width, GLsizei height, GLint border, GLenum format,
	//     GLenum type, ArrayBufferView? pixels);
	var p interface{}
	if pixels != nil {
		p = pixels
	}
	gl.Call("texImage2D", gl.TEXTURE_2D, 0, gl.RGBA, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, p)

	return Texture{t}, nil
}

func (c *Context) bindFramebuffer(f Framebuffer) {
	gl := c.gl
	// TODO: Fix this after the GopherJS bug was fixed (#159)
	if c.lastFramebuffer.Object != f.Object {
		gl.BindFramebuffer(gl.FRAMEBUFFER, f.Object)
		c.lastFramebuffer = f
	}
}

func (c *Context) FramebufferPixels(f Framebuffer, width, height int) ([]uint8, error) {
	gl := c.gl
	gl.Flush()

	c.bindFramebuffer(f)

	pixels := js.Global.Get("Uint8Array").New(4 * width * height)
	gl.ReadPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, pixels)
	if e := gl.GetError(); e != gl.NO_ERROR {
		return nil, errors.New(fmt.Sprintf("gl error: %d", e))
	}
	return pixels.Interface().([]uint8), nil
}

func (c *Context) BindTexture(t Texture) {
	gl := c.gl
	gl.BindTexture(gl.TEXTURE_2D, t.Object)
}

func (c *Context) DeleteTexture(t Texture) {
	gl := c.gl
	gl.DeleteTexture(t.Object)
}

func (c *Context) TexSubImage2D(p []uint8, width, height int) {
	gl := c.gl
	// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
	//                    GLsizei width, GLsizei height,
	//                    GLenum format, GLenum type, ArrayBufferView? pixels);
	gl.Call("texSubImage2D", gl.TEXTURE_2D, 0, 0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, p)
}

func (c *Context) NewFramebuffer(t Texture) (Framebuffer, error) {
	gl := c.gl
	f := gl.CreateFramebuffer()
	c.bindFramebuffer(Framebuffer{f})

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, t.Object, 0)
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		return Framebuffer{nil}, errors.New("creating framebuffer failed")
	}

	return Framebuffer{f}, nil
}

var lastBindedFramebuffer *js.Object

func (c *Context) SetViewport(f Framebuffer, width, height int) error {
	// TODO: Not sure if Flush is needed here.
	if f.Object == nil {
		c.bindFramebuffer(f)
		gl := c.gl
		gl.Viewport(0, 0, width, height)
		return nil
	}
	c.bindFramebuffer(f)
	gl := c.gl
	gl.Viewport(0, 0, width, height)
	return nil
}

func (c *Context) FillFramebuffer(r, g, b, a float64) error {
	// TODO: Use f?
	gl := c.gl
	gl.ClearColor(float32(r), float32(g), float32(b), float32(a))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	return nil
}

func (c *Context) DeleteFramebuffer(f Framebuffer) {
	gl := c.gl
	gl.DeleteFramebuffer(f.Object)
}

func (c *Context) NewShader(shaderType ShaderType, source string) (Shader, error) {
	gl := c.gl
	s := gl.CreateShader(int(shaderType))
	if s == nil {
		return Shader{nil}, errors.New("glCreateShader failed")
	}

	gl.ShaderSource(s, source)
	gl.CompileShader(s)

	if !gl.GetShaderParameterb(s, gl.COMPILE_STATUS) {
		log := gl.GetShaderInfoLog(s)
		return Shader{nil}, errors.New(fmt.Sprintf("shader compile failed: %s", log))
	}
	return Shader{s}, nil
}

func (c *Context) DeleteShader(s Shader) {
	gl := c.gl
	gl.DeleteShader(s.Object)
}

func (c *Context) GlslHighpSupported() bool {
	gl := c.gl
	// headless-gl library may not define getShaderPrecisionFormat.
	if gl.Get("getShaderPrecisionFormat") == js.Undefined {
		return false
	}
	return gl.Call("getShaderPrecisionFormat", gl.FRAGMENT_SHADER, gl.HIGH_FLOAT).Get("precision").Int() != 0
}

var lastProgramID ProgramID = 0

func (c *Context) NewProgram(shaders []Shader) (Program, error) {
	gl := c.gl
	p := gl.CreateProgram()
	if p == nil {
		return Program{nil}, errors.New("glCreateProgram failed")
	}
	p.Set("__ebiten_programId", lastProgramID)
	lastProgramID++

	for _, shader := range shaders {
		gl.AttachShader(p, shader.Object)
	}
	gl.LinkProgram(p)
	if !gl.GetProgramParameterb(p, gl.LINK_STATUS) {
		return Program{nil}, errors.New("program error")
	}
	return Program{p}, nil
}

func (c *Context) UseProgram(p Program) {
	gl := c.gl
	gl.UseProgram(p.Object)
}

func (c *Context) GetUniformLocation(p Program, location string) UniformLocation {
	gl := c.gl
	return UniformLocation{gl.GetUniformLocation(p.Object, location)}
}

func (c *Context) UniformInt(p Program, location string, v int) {
	gl := c.gl
	l := GetUniformLocation(c, p, location)
	gl.Uniform1i(l.Object, v)
}

func (c *Context) UniformFloats(p Program, location string, v []float32) {
	gl := c.gl
	l := GetUniformLocation(c, p, location)
	switch len(v) {
	case 4:
		gl.Call("uniform4fv", l.Object, v)
	case 16:
		gl.UniformMatrix4fv(l.Object, false, v)
	default:
		panic("not reach")
	}
}

func (c *Context) GetAttribLocation(p Program, location string) AttribLocation {
	gl := c.gl
	return AttribLocation(gl.GetAttribLocation(p.Object, location))
}

func (c *Context) VertexAttribPointer(p Program, location string, signed bool, normalize bool, stride int, size int, v int) {
	gl := c.gl
	l := GetAttribLocation(c, p, location)
	t := gl.SHORT
	if !signed {
		t = gl.UNSIGNED_SHORT
	}
	gl.VertexAttribPointer(int(l), size, t, normalize, stride, v)
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

func (c *Context) NewBuffer(bufferType BufferType, v interface{}, bufferUsage BufferUsage) Buffer {
	gl := c.gl
	b := gl.CreateBuffer()
	gl.BindBuffer(int(bufferType), b)
	gl.BufferData(int(bufferType), v, int(bufferUsage))
	return Buffer{b}
}

func (c *Context) BindElementArrayBuffer(b Buffer) {
	gl := c.gl
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, b.Object)
}

func (c *Context) BufferSubData(bufferType BufferType, data []int16) {
	gl := c.gl
	gl.BufferSubData(int(bufferType), 0, data)
}

func (c *Context) DrawElements(mode Mode, len int) {
	gl := c.gl
	gl.DrawElements(int(mode), len, gl.UNSIGNED_SHORT, 0)
}
