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

// +build darwin linux windows
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

var ZeroFramebuffer Framebuffer

// TODO: Remove this after the GopherJS bug was fixed (#159)
func (p Program) Equals(other Program) bool {
	return p == other
}

type UniformLocation int32
type AttribLocation int32

type programID uint32

func (p Program) id() programID {
	return programID(p)
}

type context struct {
	locationCache *locationCache
	funcs         chan func()
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
		Triangles:          gl.TRIANGLES,
		Lines:              gl.LINES,
	}
	c.locationCache = newLocationCache()
	c.funcs = make(chan func())
	return c
}

func (c *Context) Loop() {
	for {
		select {
		case f := <-c.funcs:
			f()
		}
	}
}

func (c *Context) RunOnContextThread(f func()) {
	ch := make(chan struct{})
	c.funcs <- func() {
		f()
		close(ch)
	}
	<-ch
	return
}

func (c *Context) Init() {
	c.RunOnContextThread(func() {
		// This initialization must be done after Loop is called.
		// This is why Init is separated from NewContext.

		if err := gl.Init(); err != nil {
			panic(fmt.Sprintf("opengl: initializing error %v", err))
		}
		// Textures' pixel formats are alpha premultiplied.
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
	})
}

func (c *Context) Check() {
	c.RunOnContextThread(func() {
		if e := gl.GetError(); e != gl.NO_ERROR {
			panic(fmt.Sprintf("check failed: %d", e))
		}
	})
}

func (c *Context) NewTexture(width, height int, pixels []uint8, filter Filter) (texture Texture, err error) {
	c.RunOnContextThread(func() {
		var t uint32
		gl.GenTextures(1, &t)
		// TOOD: Use gl.IsTexture
		if t <= 0 {
			err = errors.New("opengl: creating texture failed")
			return
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

		texture = Texture(t)
		return
	})
	return
}

func (c *Context) FramebufferPixels(f Framebuffer, width, height int) (pixels []uint8, err error) {
	c.RunOnContextThread(func() {
		gl.Flush()
		gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(f))
		pixels = make([]uint8, 4*width*height)
		gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
		if e := gl.GetError(); e != gl.NO_ERROR {
			pixels = nil
			err = fmt.Errorf("opengl: glReadPixels: %d", e)
			return
		}
		return
	})
	return
}

func (c *Context) BindTexture(t Texture) {
	c.RunOnContextThread(func() {
		gl.BindTexture(gl.TEXTURE_2D, uint32(t))
	})
}

func (c *Context) DeleteTexture(t Texture) {
	c.RunOnContextThread(func() {
		tt := uint32(t)
		gl.DeleteTextures(1, &tt)
	})
}

func (c *Context) TexSubImage2D(p []uint8, width, height int) {
	c.RunOnContextThread(func() {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(p))
	})
}

func (c *Context) BindZeroFramebuffer() {
	c.RunOnContextThread(func() {
		gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(ZeroFramebuffer))
	})
}

func (c *Context) NewFramebuffer(texture Texture) (framebuffer Framebuffer, err error) {
	c.RunOnContextThread(func() {
		var f uint32
		gl.GenFramebuffers(1, &f)
		// TODO: Use gl.IsFramebuffer
		if f <= 0 {
			err = errors.New("opengl: creating framebuffer failed: gl.IsFramebuffer returns false")
			return
		}
		gl.BindFramebuffer(gl.FRAMEBUFFER, f)

		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, uint32(texture), 0)
		s := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
		if s != gl.FRAMEBUFFER_COMPLETE {
			if s != 0 {
				err = fmt.Errorf("opengl: creating framebuffer failed: %v", s)
				return
			}
			if e := gl.GetError(); e != gl.NO_ERROR {
				err = fmt.Errorf("opengl: creating framebuffer failed: (glGetError) %d", e)
				return
			}
			err = fmt.Errorf("opengl: creating framebuffer failed: unknown error")
			return
		}
		framebuffer = Framebuffer(f)
		return
	})
	return
}

func (c *Context) SetViewport(f Framebuffer, width, height int) (err error) {
	c.RunOnContextThread(func() {
		gl.Flush()
		gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(f))
		if st := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); st != gl.FRAMEBUFFER_COMPLETE {
			if e := gl.GetError(); e != 0 {
				err = fmt.Errorf("opengl: glBindFramebuffer failed: %d", e)
				return
			}
			err = errors.New("opengl: glBindFramebuffer failed: the context is different?")
			return
		}
		gl.Viewport(0, 0, int32(width), int32(height))
		return
	})
	return
}

func (c *Context) FillFramebuffer(r, g, b, a float64) error {
	c.RunOnContextThread(func() {
		gl.ClearColor(float32(r), float32(g), float32(b), float32(a))
		gl.Clear(gl.COLOR_BUFFER_BIT)
		return
	})
	return nil
}

func (c *Context) DeleteFramebuffer(f Framebuffer) {
	c.RunOnContextThread(func() {
		ff := uint32(f)
		gl.DeleteFramebuffers(1, &ff)
	})
}

func (c *Context) NewShader(shaderType ShaderType, source string) (shader Shader, err error) {
	c.RunOnContextThread(func() {
		s := gl.CreateShader(uint32(shaderType))
		if s == 0 {
			err = errors.New("opengl: glCreateShader failed")
			return
		}
		cSources, free := gl.Strs(source + "\x00")
		gl.ShaderSource(uint32(s), 1, cSources, nil)
		free()
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
			err = fmt.Errorf("opengl: shader compile failed: %s", log)
			return
		}
		shader = Shader(s)
		return
	})
	return
}

func (c *Context) DeleteShader(s Shader) {
	c.RunOnContextThread(func() {
		gl.DeleteShader(uint32(s))
	})
}

func (c *Context) GlslHighpSupported() bool {
	return false
}

func (c *Context) NewProgram(shaders []Shader) (program Program, err error) {
	c.RunOnContextThread(func() {
		p := gl.CreateProgram()
		if p == 0 {
			err = errors.New("opengl: glCreateProgram failed")
			return
		}

		for _, shader := range shaders {
			gl.AttachShader(p, uint32(shader))
		}
		gl.LinkProgram(p)
		var v int32
		gl.GetProgramiv(p, gl.LINK_STATUS, &v)
		if v == gl.FALSE {
			err = errors.New("opengl: program error")
			return
		}
		program = Program(p)
		return
	})
	return
}

func (c *Context) UseProgram(p Program) {
	c.RunOnContextThread(func() {
		gl.UseProgram(uint32(p))
	})
}

func (c *Context) getUniformLocation(p Program, location string) UniformLocation {
	uniform := UniformLocation(gl.GetUniformLocation(uint32(p), gl.Str(location+"\x00")))
	if uniform == -1 {
		panic("opengl: invalid uniform location: " + location)
	}
	return uniform
}

func (c *Context) UniformInt(p Program, location string, v int) {
	c.RunOnContextThread(func() {
		l := int32(c.locationCache.GetUniformLocation(c, p, location))
		gl.Uniform1i(l, int32(v))
	})
}

func (c *Context) UniformFloats(p Program, location string, v []float32) {
	c.RunOnContextThread(func() {
		l := int32(c.locationCache.GetUniformLocation(c, p, location))
		switch len(v) {
		case 4:
			gl.Uniform4fv(l, 1, (*float32)(gl.Ptr(v)))
		case 16:
			gl.UniformMatrix4fv(l, 1, false, (*float32)(gl.Ptr(v)))
		default:
			panic("not reach")
		}
	})
}

func (c *Context) getAttribLocation(p Program, location string) AttribLocation {
	attrib := AttribLocation(gl.GetAttribLocation(uint32(p), gl.Str(location+"\x00")))
	if attrib == -1 {
		panic("invalid attrib location: " + location)
	}
	return attrib
}

func (c *Context) VertexAttribPointer(p Program, location string, normalize bool, stride int, size int, v int) {
	c.RunOnContextThread(func() {
		l := c.locationCache.GetAttribLocation(c, p, location)
		gl.VertexAttribPointer(uint32(l), int32(size), gl.SHORT, normalize, int32(stride), gl.PtrOffset(v))
	})
}

func (c *Context) EnableVertexAttribArray(p Program, location string) {
	c.RunOnContextThread(func() {
		l := c.locationCache.GetAttribLocation(c, p, location)
		gl.EnableVertexAttribArray(uint32(l))
	})
}

func (c *Context) DisableVertexAttribArray(p Program, location string) {
	c.RunOnContextThread(func() {
		l := c.locationCache.GetAttribLocation(c, p, location)
		gl.DisableVertexAttribArray(uint32(l))
	})
}

func (c *Context) NewBuffer(bufferType BufferType, v interface{}, bufferUsage BufferUsage) (buffer Buffer) {
	c.RunOnContextThread(func() {
		var b uint32
		gl.GenBuffers(1, &b)
		gl.BindBuffer(uint32(bufferType), b)
		switch v := v.(type) {
		case int:
			gl.BufferData(uint32(bufferType), v, nil, uint32(bufferUsage))
		case []uint16:
			gl.BufferData(uint32(bufferType), 2*len(v), gl.Ptr(v), uint32(bufferUsage))
		default:
			panic("not reach")
		}
		buffer = Buffer(b)
		return
	})
	return
}

func (c *Context) BindElementArrayBuffer(b Buffer) {
	c.RunOnContextThread(func() {
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, uint32(b))
	})
}

func (c *Context) BufferSubData(bufferType BufferType, data []int16) {
	c.RunOnContextThread(func() {
		gl.BufferSubData(uint32(bufferType), 0, 2*len(data), gl.Ptr(data))
	})
}

func (c *Context) DrawElements(mode Mode, len int) {
	c.RunOnContextThread(func() {
		gl.DrawElements(uint32(mode), int32(len), gl.UNSIGNED_SHORT, gl.PtrOffset(0))
	})
}

func (c *Context) Finish() {
	c.RunOnContextThread(func() {
		gl.Finish()
	})
}
