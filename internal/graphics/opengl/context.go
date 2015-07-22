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

	"github.com/go-gl/gl/v2.1/gl"
)

type Texture uint32
type Framebuffer uint32
type Shader uint32
type Program uint32
type Buffer uint32

var ZeroFramebuffer Framebuffer = 0

// TODO: Remove this after the GopherJS bug was fixed (#159)
func (p Program) Equals(other Program) bool {
	return p == other
}

type UniformLocation int32
type AttribLocation int32

type ProgramID int

func GetProgramID(p Program) ProgramID {
	return ProgramID(p)
}

type context struct{}

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
		Triangles:          gl.TRIANGLES,
		Lines:              gl.LINES,
	}
	c.init()
	return c
}

func (c *Context) init() {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	// Textures' pixel formats are alpha premultiplied.
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
}

func (c *Context) Check() {
	if e := gl.GetError(); e != gl.NO_ERROR {
		panic(fmt.Sprintf("check failed: %d", e))
	}
}

func (c *Context) NewTexture(width, height int, pixels []uint8, filter Filter) (Texture, error) {
	var t uint32
	gl.GenTextures(1, &t)
	if t < 0 {
		return 0, errors.New("glGenTexture failed")
	}
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	gl.BindTexture(gl.TEXTURE_2D, t)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, int32(filter))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, int32(filter))

	var p interface{}
	if pixels != nil {
		p = pixels
	}
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(p))

	return Texture(t), nil
}

func (c *Context) FramebufferPixels(f Framebuffer, width, height int) ([]uint8, error) {
	gl.Flush()

	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(f))

	pixels := make([]uint8, 4*width*height)
	gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
	if e := gl.GetError(); e != gl.NO_ERROR {
		return nil, errors.New(fmt.Sprintf("glReadPixels: %d", e))
	}
	return pixels, nil
}

func (c *Context) BindTexture(t Texture) {
	gl.BindTexture(gl.TEXTURE_2D, uint32(t))
}

func (c *Context) DeleteTexture(t Texture) {
	tt := uint32(t)
	gl.DeleteTextures(1, &tt)
}

func (c *Context) TexSubImage2D(p []uint8, width, height int) {
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(p))
}

func (c *Context) NewFramebuffer(texture Texture) (Framebuffer, error) {
	var f uint32
	gl.GenFramebuffers(1, &f)
	gl.BindFramebuffer(gl.FRAMEBUFFER, f)

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, uint32(texture), 0)
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		return 0, errors.New("creating framebuffer failed")
	}

	return Framebuffer(f), nil
}

func (c *Context) SetViewport(f Framebuffer, width, height int) error {
	gl.Flush()
	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(f))
	if err := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); err != gl.FRAMEBUFFER_COMPLETE {
		if e := gl.GetError(); e != 0 {
			return errors.New(fmt.Sprintf("glBindFramebuffer failed: %d", e))
		}
		return errors.New("glBindFramebuffer failed: the context is different?")
	}
	gl.Viewport(0, 0, int32(width), int32(height))
	return nil
}

func (c *Context) FillFramebuffer(r, g, b, a float64) error {
	gl.ClearColor(float32(r), float32(g), float32(b), float32(a))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	return nil
}

func (c *Context) DeleteFramebuffer(f Framebuffer) {
	ff := uint32(f)
	gl.DeleteFramebuffers(1, &ff)
}

func (c *Context) NewShader(shaderType ShaderType, source string) (Shader, error) {
	s := gl.CreateShader(uint32(shaderType))
	if s == 0 {
		return 0, errors.New("glCreateShader failed")
	}

	glSource := gl.Str(source + "\x00")
	gl.ShaderSource(uint32(s), 1, &glSource, nil)
	gl.CompileShader(s)

	var v int32
	gl.GetShaderiv(s, gl.COMPILE_STATUS, &v)
	if v == gl.FALSE {
		log := []uint8{}
		gl.GetShaderiv(uint32(s), gl.INFO_LOG_LENGTH, &v)
		if v != 0 {
			log = make([]uint8, int(v))
			gl.GetShaderInfoLog(uint32(s), v, nil, (*uint8)(gl.Ptr(log)))
		}
		return 0, errors.New(fmt.Sprintf("shader compile failed: %s", string(log)))
	}
	return Shader(s), nil
}

func (c *Context) DeleteShader(s Shader) {
	gl.DeleteShader(uint32(s))
}

func (c *Context) GlslHighpSupported() bool {
	return false
}

func (c *Context) NewProgram(shaders []Shader) (Program, error) {
	p := gl.CreateProgram()
	if p == 0 {
		return 0, errors.New("glCreateProgram failed")
	}

	for _, shader := range shaders {
		gl.AttachShader(p, uint32(shader))
	}
	gl.LinkProgram(p)
	var v int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &v)
	if v == gl.FALSE {
		return 0, errors.New("program error")
	}
	return Program(p), nil
}

func (c *Context) UseProgram(p Program) {
	gl.UseProgram(uint32(p))
}

func (c *Context) GetUniformLocation(p Program, location string) UniformLocation {
	u := UniformLocation(gl.GetUniformLocation(uint32(p), gl.Str(location+"\x00")))
	if u == -1 {
		panic("invalid uniform location: " + location)
	}
	return u
}

func (c *Context) UniformInt(p Program, location string, v int) {
	l := int32(GetUniformLocation(c, p, location))
	gl.Uniform1i(l, int32(v))
}

func (c *Context) UniformFloats(p Program, location string, v []float32) {
	l := int32(GetUniformLocation(c, p, location))
	switch len(v) {
	case 4:
		gl.Uniform4fv(l, 1, (*float32)(gl.Ptr(v)))
	case 16:
		gl.UniformMatrix4fv(l, 1, false, (*float32)(gl.Ptr(v)))
	default:
		panic("not reach")
	}
}

func (c *Context) GetAttribLocation(p Program, location string) AttribLocation {
	a := AttribLocation(gl.GetAttribLocation(uint32(p), gl.Str(location+"\x00")))
	if a == -1 {
		panic("invalid attrib location: " + location)
	}
	return a
}

func (c *Context) VertexAttribPointer(p Program, location string, signed bool, normalize bool, stride int, size int, v int) {
	l := GetAttribLocation(c, p, location)
	t := gl.SHORT
	if !signed {
		t = gl.UNSIGNED_SHORT
	}
	gl.VertexAttribPointer(uint32(l), int32(size), uint32(t), normalize, int32(stride), gl.PtrOffset(v))
}

func (c *Context) EnableVertexAttribArray(p Program, location string) {
	l := GetAttribLocation(c, p, location)
	gl.EnableVertexAttribArray(uint32(l))
}

func (c *Context) DisableVertexAttribArray(p Program, location string) {
	l := GetAttribLocation(c, p, location)
	gl.DisableVertexAttribArray(uint32(l))
}

func (c *Context) NewBuffer(bufferType BufferType, v interface{}, bufferUsage BufferUsage) Buffer {
	var b uint32
	gl.GenBuffers(1, &b)
	gl.BindBuffer(uint32(bufferType), b)
	size := 0
	ptr := v
	switch v := v.(type) {
	case int:
		size = v
		ptr = nil
	case []uint16:
		size = 2 * len(v)
	case []float32:
		size = 4 * len(v)
	default:
		panic("not reach")
	}
	gl.BufferData(uint32(bufferType), size, gl.Ptr(ptr), uint32(bufferUsage))
	return Buffer(b)
}

func (c *Context) BindElementArrayBuffer(b Buffer) {
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, uint32(b))
}

func (c *Context) BufferSubData(bufferType BufferType, data []int16) {
	const int16Size = 2
	gl.BufferSubData(uint32(bufferType), 0, int16Size*len(data), gl.Ptr(data))
}

func (c *Context) DrawElements(mode Mode, len int) {
	gl.DrawElements(uint32(mode), int32(len), gl.UNSIGNED_SHORT, gl.PtrOffset(0))
}
