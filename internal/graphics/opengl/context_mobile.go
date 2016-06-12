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

// +build android

package opengl

import (
	"errors"
	"fmt"

	mgl "golang.org/x/mobile/gl"
)

type Texture mgl.Texture
type Framebuffer mgl.Framebuffer
type Shader mgl.Shader
type Program mgl.Program
type Buffer mgl.Buffer

type uniformLocation mgl.Uniform
type attribLocation mgl.Attrib

type programID uint32

func (p Program) id() programID {
	return programID(p.Value)
}

type context struct {
	gl          mgl.Context
	worker      mgl.Worker
	initialized chan struct{}
}

func NewContext() (*Context, error) {
	c := &Context{
		Nearest:            mgl.NEAREST,
		Linear:             mgl.LINEAR,
		VertexShader:       mgl.VERTEX_SHADER,
		FragmentShader:     mgl.FRAGMENT_SHADER,
		ArrayBuffer:        mgl.ARRAY_BUFFER,
		ElementArrayBuffer: mgl.ELEMENT_ARRAY_BUFFER,
		DynamicDraw:        mgl.DYNAMIC_DRAW,
		StaticDraw:         mgl.STATIC_DRAW,
		Triangles:          mgl.TRIANGLES,
		Lines:              mgl.LINES,
		zero:               mgl.ZERO,
		one:                mgl.ONE,
		srcAlpha:           mgl.SRC_ALPHA,
		dstAlpha:           mgl.DST_ALPHA,
		oneMinusSrcAlpha:   mgl.ONE_MINUS_SRC_ALPHA,
		oneMinusDstAlpha:   mgl.ONE_MINUS_DST_ALPHA,
		locationCache:      newLocationCache(),
		lastCompositeMode:  CompositeModeUnknown,
	}
	c.gl, c.worker = mgl.NewContext()
	c.initialized = make(chan struct{})
	go func() {
		// GL calls will just enqueue an task to the worker.
		// Since the worker is not avaialbe, this enqueuing should be done
		// in a goroutine.

		// Textures' pixel formats are alpha premultiplied.
		c.gl.Enable(mgl.BLEND)
		c.BlendFunc(CompositeModeSourceOver)
		close(c.initialized)
	}()
	return c, nil
}

func (c *Context) Resume() {
	c.locationCache = newLocationCache()
	c.lastFramebuffer = ZeroFramebuffer
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = CompositeModeUnknown
	c.gl.Enable(mgl.BLEND)
	c.BlendFunc(CompositeModeSourceOver)
}

func (c *Context) WaitUntilInitializingDone() {
	// TODO: Call this function at an approriate place
	<-c.initialized
}

func (c *Context) Worker() mgl.Worker {
	return c.worker
}

func (c *Context) BlendFunc(mode CompositeMode) {
	gl := c.gl
	if c.lastCompositeMode == mode {
		return
	}
	c.lastCompositeMode = mode
	s, d := c.operations(mode)
	gl.BlendFunc(mgl.Enum(s), mgl.Enum(d))
}

func (c *Context) NewTexture(width, height int, pixels []uint8, filter Filter) (Texture, error) {
	gl := c.gl
	t := gl.CreateTexture()
	if t.Value <= 0 {
		return Texture{}, errors.New("opengl: creating texture failed")
	}
	gl.PixelStorei(mgl.UNPACK_ALIGNMENT, 4)
	gl.BindTexture(mgl.TEXTURE_2D, t)

	gl.TexParameteri(mgl.TEXTURE_2D, mgl.TEXTURE_MAG_FILTER, int(filter))
	gl.TexParameteri(mgl.TEXTURE_2D, mgl.TEXTURE_MIN_FILTER, int(filter))

	var p []uint8
	if pixels != nil {
		p = pixels
	}
	gl.TexImage2D(mgl.TEXTURE_2D, 0, width, height, mgl.RGBA, mgl.UNSIGNED_BYTE, p)

	return Texture(t), nil
}

func (c *Context) bindFramebufferImpl(f Framebuffer) {
	gl := c.gl
	gl.BindFramebuffer(mgl.FRAMEBUFFER, mgl.Framebuffer(f))
}

func (c *Context) FramebufferPixels(f Framebuffer, width, height int) ([]uint8, error) {
	gl := c.gl
	gl.Flush()

	c.bindFramebuffer(f)

	pixels := make([]uint8, 4*width*height)
	gl.ReadPixels(pixels, 0, 0, width, height, mgl.RGBA, mgl.UNSIGNED_BYTE)
	if e := gl.GetError(); e != mgl.NO_ERROR {
		return nil, fmt.Errorf("opengl: glReadPixels: %d", e)
	}
	return pixels, nil
}

func (c *Context) BindTexture(t Texture) {
	gl := c.gl
	gl.BindTexture(mgl.TEXTURE_2D, mgl.Texture(t))
}

func (c *Context) DeleteTexture(t Texture) {
	gl := c.gl
	if !gl.IsTexture(mgl.Texture(t)) {
		return
	}
	gl.DeleteTexture(mgl.Texture(t))
}

func (c *Context) IsTexture(t Texture) bool {
	gl := c.gl
	return gl.IsTexture(mgl.Texture(t))
}

func (c *Context) TexSubImage2D(p []uint8, width, height int) {
	gl := c.gl
	gl.TexSubImage2D(mgl.TEXTURE_2D, 0, 0, 0, width, height, mgl.RGBA, mgl.UNSIGNED_BYTE, p)
}

func (c *Context) BindZeroFramebuffer() {
	c.bindFramebuffer(ZeroFramebuffer)
}

func (c *Context) NewFramebuffer(texture Texture) (Framebuffer, error) {
	gl := c.gl
	f := gl.CreateFramebuffer()
	if f.Value <= 0 {
		return Framebuffer{}, errors.New("opengl: creating framebuffer failed: gl.IsFramebuffer returns false")
	}
	c.bindFramebuffer(Framebuffer(f))

	gl.FramebufferTexture2D(mgl.FRAMEBUFFER, mgl.COLOR_ATTACHMENT0, mgl.TEXTURE_2D, mgl.Texture(texture), 0)
	s := gl.CheckFramebufferStatus(mgl.FRAMEBUFFER)
	if s != mgl.FRAMEBUFFER_COMPLETE {
		if s != 0 {
			return Framebuffer{}, fmt.Errorf("opengl: creating framebuffer failed: %v", s)
		}
		if e := gl.GetError(); e != mgl.NO_ERROR {
			return Framebuffer{}, fmt.Errorf("opengl: creating framebuffer failed: (glGetError) %d", e)
		}
		return Framebuffer{}, fmt.Errorf("opengl: creating framebuffer failed: unknown error")
	}

	return Framebuffer(f), nil
}

func (c *Context) SetViewport(f Framebuffer, width, height int) error {
	c.bindFramebuffer(f)
	if c.lastViewportWidth != width || c.lastViewportHeight != height {
		gl := c.gl
		gl.Viewport(0, 0, width, height)
		c.lastViewportWidth = width
		c.lastViewportHeight = height
	}
	return nil
}

func (c *Context) FillFramebuffer(r, g, b, a float64) error {
	gl := c.gl
	gl.ClearColor(float32(r), float32(g), float32(b), float32(a))
	gl.Clear(mgl.COLOR_BUFFER_BIT)
	return nil
}

func (c *Context) DeleteFramebuffer(f Framebuffer) {
	gl := c.gl
	if !gl.IsFramebuffer(mgl.Framebuffer(f)) {
		return
	}
	// If a framebuffer to be delted is bound, a newly bound framebuffer
	// will be a default framebuffer.
	// https://www.khronos.org/opengles/sdk/docs/man/xhtml/glDeleteFramebuffers.xml
	if c.lastFramebuffer == f {
		c.lastFramebuffer = ZeroFramebuffer
		c.lastViewportWidth = 0
		c.lastViewportHeight = 0
	}
	gl.DeleteFramebuffer(mgl.Framebuffer(f))
}

func (c *Context) NewShader(shaderType ShaderType, source string) (Shader, error) {
	gl := c.gl
	s := gl.CreateShader(mgl.Enum(shaderType))
	if s.Value == 0 {
		return Shader{}, errors.New("opengl: glCreateShader failed")
	}
	gl.ShaderSource(s, source)
	gl.CompileShader(s)

	v := gl.GetShaderi(s, mgl.COMPILE_STATUS)
	if v == mgl.FALSE {
		log := gl.GetShaderInfoLog(s)
		return Shader{}, fmt.Errorf("opengl: shader compile failed: %s", log)
	}
	return Shader(s), nil
}

func (c *Context) DeleteShader(s Shader) {
	gl := c.gl
	gl.DeleteShader(mgl.Shader(s))
}

func (c *Context) GlslHighpSupported() bool {
	// TODO: Fix this
	return false
}

func (c *Context) NewProgram(shaders []Shader) (Program, error) {
	gl := c.gl
	p := gl.CreateProgram()
	if p.Value == 0 {
		return Program{}, errors.New("opengl: glCreateProgram failed")
	}

	for _, shader := range shaders {
		gl.AttachShader(p, mgl.Shader(shader))
	}
	gl.LinkProgram(p)
	v := gl.GetProgrami(p, mgl.LINK_STATUS)
	if v == mgl.FALSE {
		return Program{}, errors.New("opengl: program error")
	}
	return Program(p), nil
}

func (c *Context) UseProgram(p Program) {
	gl := c.gl
	gl.UseProgram(mgl.Program(p))
}

func (c *Context) getUniformLocationImpl(p Program, location string) uniformLocation {
	gl := c.gl
	u := uniformLocation(gl.GetUniformLocation(mgl.Program(p), location))
	if u.Value == -1 {
		panic("invalid uniform location: " + location)
	}
	return u
}

func (c *Context) UniformInt(p Program, location string, v int) {
	gl := c.gl
	gl.Uniform1i(mgl.Uniform(c.locationCache.GetUniformLocation(c, p, location)), v)
}

func (c *Context) UniformFloats(p Program, location string, v []float32) {
	gl := c.gl
	l := mgl.Uniform(c.locationCache.GetUniformLocation(c, p, location))
	switch len(v) {
	case 4:
		gl.Uniform4fv(l, v)
	case 16:
		gl.UniformMatrix4fv(l, v)
	default:
		panic("not reach")
	}
}

func (c *Context) getAttribLocationImpl(p Program, location string) attribLocation {
	gl := c.gl
	a := attribLocation(gl.GetAttribLocation(mgl.Program(p), location))
	if a.Value == ^uint(0) {
		panic("invalid attrib location: " + location)
	}
	return a
}

func (c *Context) VertexAttribPointer(p Program, location string, normalize bool, stride int, size int, v int) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.VertexAttribPointer(mgl.Attrib(l), size, mgl.SHORT, normalize, stride, v)
}

func (c *Context) EnableVertexAttribArray(p Program, location string) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.EnableVertexAttribArray(mgl.Attrib(l))
}

func (c *Context) DisableVertexAttribArray(p Program, location string) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.DisableVertexAttribArray(mgl.Attrib(l))
}

func (c *Context) DeleteProgram(p Program) {
	gl := c.gl
	gl.DeleteProgram(mgl.Program(p))
}

func uint16ToBytes(v []uint16) []byte {
	b := make([]byte, len(v)*2)
	for i, x := range v {
		b[2*i] = byte(x)
		b[2*i+1] = byte(x >> 8)
	}
	return b
}

func int16ToBytes(v []int16) []byte {
	b := make([]byte, len(v)*2)
	for i, x := range v {
		b[2*i] = byte(uint16(x))
		b[2*i+1] = byte(uint16(x) >> 8)
	}
	return b
}

func (c *Context) NewBuffer(bufferType BufferType, v interface{}, bufferUsage BufferUsage) Buffer {
	gl := c.gl
	b := gl.CreateBuffer()
	gl.BindBuffer(mgl.Enum(bufferType), b)
	switch v := v.(type) {
	case int:
		gl.BufferInit(mgl.Enum(bufferType), v, mgl.Enum(bufferUsage))
	case []uint16:
		gl.BufferData(mgl.Enum(bufferType), uint16ToBytes(v), mgl.Enum(bufferUsage))
	default:
		panic("not reach")
	}
	return Buffer(b)
}

func (c *Context) BindElementArrayBuffer(b Buffer) {
	gl := c.gl
	gl.BindBuffer(mgl.ELEMENT_ARRAY_BUFFER, mgl.Buffer(b))
}

func (c *Context) BufferSubData(bufferType BufferType, data []int16) {
	gl := c.gl
	gl.BufferSubData(mgl.Enum(bufferType), 0, int16ToBytes(data))
}

func (c *Context) DeleteBuffer(b Buffer) {
	gl := c.gl
	gl.DeleteBuffer(mgl.Buffer(b))
}

func (c *Context) DrawElements(mode Mode, len int, offsetInBytes int) {
	gl := c.gl
	gl.DrawElements(mgl.Enum(mode), len, mgl.UNSIGNED_SHORT, offsetInBytes)
}

func (c *Context) Flush() {
	gl := c.gl
	gl.Flush()
}
