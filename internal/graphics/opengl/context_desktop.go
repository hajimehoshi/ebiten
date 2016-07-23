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

// +build darwin,!arm,!arm64 linux windows
// +build !js
// +build !android
// +build !ios

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

type uniformLocation int32
type attribLocation int32

type programID uint32

const (
	invalidTexture     = 0
	invalidFramebuffer = (1 << 32) - 1
)

func (p Program) id() programID {
	return programID(p)
}

func init() {
	Nearest = gl.NEAREST
	Linear = gl.LINEAR
	VertexShader = gl.VERTEX_SHADER
	FragmentShader = gl.FRAGMENT_SHADER
	ArrayBuffer = gl.ARRAY_BUFFER
	ElementArrayBuffer = gl.ELEMENT_ARRAY_BUFFER
	DynamicDraw = gl.DYNAMIC_DRAW
	StaticDraw = gl.STATIC_DRAW
	Triangles = gl.TRIANGLES
	Lines = gl.LINES

	zero = gl.ZERO
	one = gl.ONE
	srcAlpha = gl.SRC_ALPHA
	dstAlpha = gl.DST_ALPHA
	oneMinusSrcAlpha = gl.ONE_MINUS_SRC_ALPHA
	oneMinusDstAlpha = gl.ONE_MINUS_DST_ALPHA
}

type context struct {
	init            bool
	runOnMainThread func(func() error) error
}

func NewContext(runOnMainThread func(func() error) error) (*Context, error) {
	c := &Context{}
	c.runOnMainThread = runOnMainThread
	return c, nil
}

func (c *Context) RunOnContextThread(f func() error) error {
	return c.runOnMainThread(f)
}

func (c *Context) Reset() error {
	if err := c.RunOnContextThread(func() error {
		if c.init {
			return nil
		}
		// Note that this initialization must be done after Loop is called.
		if err := gl.Init(); err != nil {
			return fmt.Errorf("opengl: initializing error %v", err)
		}
		c.init = true
		return nil
	}); err != nil {
		return nil
	}
	c.locationCache = newLocationCache()
	c.lastTexture = invalidTexture
	c.lastFramebuffer = invalidFramebuffer
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = CompositeModeUnknown
	if err := c.RunOnContextThread(func() error {
		gl.Enable(gl.BLEND)
		return nil
	}); err != nil {
		return err
	}
	c.BlendFunc(CompositeModeSourceOver)
	if err := c.RunOnContextThread(func() error {
		f := int32(0)
		gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &f)
		c.screenFramebuffer = Framebuffer(f)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (c *Context) BlendFunc(mode CompositeMode) {
	c.RunOnContextThread(func() error {
		if c.lastCompositeMode == mode {
			return nil
		}
		c.lastCompositeMode = mode
		s, d := operations(mode)
		gl.BlendFunc(uint32(s), uint32(d))
		return nil
	})
}

func (c *Context) NewTexture(width, height int, pixels []uint8, filter Filter) (Texture, error) {
	var texture Texture
	if err := c.RunOnContextThread(func() error {
		var t uint32
		gl.GenTextures(1, &t)
		// TODO: Use gl.IsTexture
		if t <= 0 {
			return errors.New("opengl: creating texture failed")
		}
		gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
		texture = Texture(t)
		return nil
	}); err != nil {
		return 0, err
	}
	if err := c.BindTexture(texture); err != nil {
		return 0, err
	}
	if err := c.RunOnContextThread(func() error {
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, int32(filter))
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, int32(filter))

		var p interface{}
		if pixels != nil {
			p = pixels
		}
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(p))
		return nil
	}); err != nil {
		return 0, err
	}
	return texture, nil
}

func (c *Context) bindFramebufferImpl(f Framebuffer) error {
	if err := c.RunOnContextThread(func() error {
		gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(f))
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (c *Context) FramebufferPixels(f Framebuffer, width, height int) ([]uint8, error) {
	var pixels []uint8
	if err := c.RunOnContextThread(func() error {
		gl.Flush()
		return nil
	}); err != nil {
		return nil, err
	}
	c.bindFramebuffer(f)
	if err := c.RunOnContextThread(func() error {
		pixels = make([]uint8, 4*width*height)
		gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
		if e := gl.GetError(); e != gl.NO_ERROR {
			pixels = nil
			return fmt.Errorf("opengl: glReadPixels: %d", e)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return pixels, nil
}

func (c *Context) bindTextureImpl(t Texture) error {
	c.RunOnContextThread(func() error {
		gl.BindTexture(gl.TEXTURE_2D, uint32(t))
		return nil
	})
	return nil
}

func (c *Context) DeleteTexture(t Texture) {
	c.RunOnContextThread(func() error {
		tt := uint32(t)
		if !gl.IsTexture(tt) {
			return nil
		}
		if c.lastTexture == t {
			c.lastTexture = invalidTexture
		}
		gl.DeleteTextures(1, &tt)
		return nil
	})
}

func (c *Context) IsTexture(t Texture) bool {
	r := false
	c.RunOnContextThread(func() error {
		r = gl.IsTexture(uint32(t))
		return nil
	})
	return r
}

func (c *Context) TexSubImage2D(p []uint8, width, height int) {
	c.RunOnContextThread(func() error {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(p))
		return nil
	})
}

func (c *Context) BindScreenFramebuffer() error {
	return c.bindFramebuffer(c.screenFramebuffer)
}

func (c *Context) NewFramebuffer(texture Texture) (Framebuffer, error) {
	var framebuffer Framebuffer
	var f uint32
	if err := c.RunOnContextThread(func() error {
		gl.GenFramebuffers(1, &f)
		// TODO: Use gl.IsFramebuffer
		if f <= 0 {
			return errors.New("opengl: creating framebuffer failed: gl.IsFramebuffer returns false")
		}
		return nil
	}); err != nil {
		return 0, err
	}
	c.bindFramebuffer(Framebuffer(f))
	if err := c.RunOnContextThread(func() error {
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, uint32(texture), 0)
		s := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
		if s != gl.FRAMEBUFFER_COMPLETE {
			if s != 0 {
				return fmt.Errorf("opengl: creating framebuffer failed: %v", s)
			}
			if e := gl.GetError(); e != gl.NO_ERROR {
				return fmt.Errorf("opengl: creating framebuffer failed: (glGetError) %d", e)
			}
			return fmt.Errorf("opengl: creating framebuffer failed: unknown error")
		}
		framebuffer = Framebuffer(f)
		return nil
	}); err != nil {
		return 0, err
	}
	return framebuffer, nil
}

func (c *Context) setViewportImpl(width, height int) error {
	return c.RunOnContextThread(func() error {
		gl.Viewport(0, 0, int32(width), int32(height))
		return nil
	})
}

func (c *Context) FillFramebuffer(r, g, b, a float64) error {
	return c.RunOnContextThread(func() error {
		gl.ClearColor(float32(r), float32(g), float32(b), float32(a))
		gl.Clear(gl.COLOR_BUFFER_BIT)
		return nil
	})
}

func (c *Context) DeleteFramebuffer(f Framebuffer) {
	c.RunOnContextThread(func() error {
		ff := uint32(f)
		if !gl.IsFramebuffer(ff) {
			return nil
		}
		if c.lastFramebuffer == f {
			c.lastFramebuffer = invalidFramebuffer
			c.lastViewportWidth = 0
			c.lastViewportHeight = 0
		}
		gl.DeleteFramebuffers(1, &ff)
		return nil
	})
}

func (c *Context) NewShader(shaderType ShaderType, source string) (Shader, error) {
	var shader Shader
	if err := c.RunOnContextThread(func() error {
		s := gl.CreateShader(uint32(shaderType))
		if s == 0 {
			return errors.New("opengl: glCreateShader failed")
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
			return fmt.Errorf("opengl: shader compile failed: %s", log)
		}
		shader = Shader(s)
		return nil
	}); err != nil {
		return 0, err
	}
	return shader, nil
}

func (c *Context) DeleteShader(s Shader) {
	c.RunOnContextThread(func() error {
		gl.DeleteShader(uint32(s))
		return nil
	})
}

func (c *Context) GlslHighpSupported() bool {
	return false
}

func (c *Context) NewProgram(shaders []Shader) (Program, error) {
	var program Program
	if err := c.RunOnContextThread(func() error {
		p := gl.CreateProgram()
		if p == 0 {
			return errors.New("opengl: glCreateProgram failed")
		}

		for _, shader := range shaders {
			gl.AttachShader(p, uint32(shader))
		}
		gl.LinkProgram(p)
		var v int32
		gl.GetProgramiv(p, gl.LINK_STATUS, &v)
		if v == gl.FALSE {
			return errors.New("opengl: program error")
		}
		program = Program(p)
		return nil
	}); err != nil {
		return 0, err
	}
	return program, nil
}

func (c *Context) UseProgram(p Program) {
	c.RunOnContextThread(func() error {
		gl.UseProgram(uint32(p))
		return nil
	})
}

func (c *Context) DeleteProgram(p Program) {
	c.RunOnContextThread(func() error {
		if !gl.IsProgram(uint32(p)) {
			return nil
		}
		gl.DeleteProgram(uint32(p))
		return nil
	})
}

func (c *Context) getUniformLocationImpl(p Program, location string) uniformLocation {
	uniform := uniformLocation(gl.GetUniformLocation(uint32(p), gl.Str(location+"\x00")))
	if uniform == -1 {
		panic("opengl: invalid uniform location: " + location)
	}
	return uniform
}

func (c *Context) UniformInt(p Program, location string, v int) {
	c.RunOnContextThread(func() error {
		l := int32(c.locationCache.GetUniformLocation(c, p, location))
		gl.Uniform1i(l, int32(v))
		return nil
	})
}

func (c *Context) UniformFloats(p Program, location string, v []float32) {
	c.RunOnContextThread(func() error {
		l := int32(c.locationCache.GetUniformLocation(c, p, location))
		switch len(v) {
		case 4:
			gl.Uniform4fv(l, 1, (*float32)(gl.Ptr(v)))
		case 16:
			gl.UniformMatrix4fv(l, 1, false, (*float32)(gl.Ptr(v)))
		default:
			panic("not reach")
		}
		return nil
	})
}

func (c *Context) getAttribLocationImpl(p Program, location string) attribLocation {
	attrib := attribLocation(gl.GetAttribLocation(uint32(p), gl.Str(location+"\x00")))
	if attrib == -1 {
		panic("opengl: invalid attrib location: " + location)
	}
	return attrib
}

func (c *Context) VertexAttribPointer(p Program, location string, normalize bool, stride int, size int, v int) {
	c.RunOnContextThread(func() error {
		l := c.locationCache.GetAttribLocation(c, p, location)
		gl.VertexAttribPointer(uint32(l), int32(size), gl.SHORT, normalize, int32(stride), gl.PtrOffset(v))
		return nil
	})
}

func (c *Context) EnableVertexAttribArray(p Program, location string) {
	c.RunOnContextThread(func() error {
		l := c.locationCache.GetAttribLocation(c, p, location)
		gl.EnableVertexAttribArray(uint32(l))
		return nil
	})
}

func (c *Context) DisableVertexAttribArray(p Program, location string) {
	c.RunOnContextThread(func() error {
		l := c.locationCache.GetAttribLocation(c, p, location)
		gl.DisableVertexAttribArray(uint32(l))
		return nil
	})
}

func (c *Context) NewBuffer(bufferType BufferType, v interface{}, bufferUsage BufferUsage) Buffer {
	var buffer Buffer
	c.RunOnContextThread(func() error {
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
		return nil
	})
	return buffer
}

func (c *Context) BindElementArrayBuffer(b Buffer) {
	c.RunOnContextThread(func() error {
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, uint32(b))
		return nil
	})
}

func (c *Context) BufferSubData(bufferType BufferType, data []int16) {
	c.RunOnContextThread(func() error {
		gl.BufferSubData(uint32(bufferType), 0, 2*len(data), gl.Ptr(data))
		return nil
	})
}

func (c *Context) DeleteBuffer(b Buffer) {
	c.RunOnContextThread(func() error {
		bb := uint32(b)
		gl.DeleteBuffers(1, &bb)
		return nil
	})
}

func (c *Context) DrawElements(mode Mode, len int, offsetInBytes int) {
	c.RunOnContextThread(func() error {
		gl.DrawElements(uint32(mode), int32(len), gl.UNSIGNED_SHORT, gl.PtrOffset(offsetInBytes))
		return nil
	})
}

func (c *Context) Flush() {
	c.RunOnContextThread(func() error {
		gl.Flush()
		return nil
	})
}
