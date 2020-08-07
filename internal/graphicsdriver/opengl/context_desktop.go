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

// +build darwin freebsd linux windows
// +build !js
// +build !android
// +build !ios

package opengl

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/opengl/gl"
	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

type (
	textureNative     uint32
	framebufferNative uint32
	shader            uint32
	program           uint32
	buffer            uint32
)

func (t textureNative) equal(rhs textureNative) bool {
	return t == rhs
}

func (f framebufferNative) equal(rhs framebufferNative) bool {
	return f == rhs
}

func (s shader) equal(rhs shader) bool {
	return s == rhs
}

func (b buffer) equal(rhs buffer) bool {
	return b == rhs
}

func (u uniformLocation) equal(rhs uniformLocation) bool {
	return u == rhs
}

func (p program) equal(rhs program) bool {
	return p == rhs
}

var InvalidTexture textureNative

type (
	uniformLocation int32
	attribLocation  int32
)

type programID uint32

const (
	invalidTexture     = 0
	invalidFramebuffer = (1 << 32) - 1
	invalidUniform     = -1
)

func getProgramID(p program) programID {
	return programID(p)
}

const (
	vertexShader       = shaderType(gl.VERTEX_SHADER)
	fragmentShader     = shaderType(gl.FRAGMENT_SHADER)
	arrayBuffer        = bufferType(gl.ARRAY_BUFFER)
	elementArrayBuffer = bufferType(gl.ELEMENT_ARRAY_BUFFER)
	dynamicDraw        = bufferUsage(gl.DYNAMIC_DRAW)
	short              = dataType(gl.SHORT)
	float              = dataType(gl.FLOAT)

	zero             = operation(gl.ZERO)
	one              = operation(gl.ONE)
	srcAlpha         = operation(gl.SRC_ALPHA)
	dstAlpha         = operation(gl.DST_ALPHA)
	oneMinusSrcAlpha = operation(gl.ONE_MINUS_SRC_ALPHA)
	oneMinusDstAlpha = operation(gl.ONE_MINUS_DST_ALPHA)
	dstColor         = operation(gl.DST_COLOR)
)

type contextImpl struct {
	init bool
}

func (c *context) reset() error {
	if err := c.t.Call(func() error {
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
		return err
	}
	c.locationCache = newLocationCache()
	c.lastTexture = invalidTexture
	c.lastFramebuffer = invalidFramebuffer
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = driver.CompositeModeUnknown
	_ = c.t.Call(func() error {
		gl.Enable(gl.BLEND)
		return nil
	})
	c.blendFunc(driver.CompositeModeSourceOver)
	_ = c.t.Call(func() error {
		f := int32(0)
		gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &f)
		c.screenFramebuffer = framebufferNative(f)
		return nil
	})
	return nil
}

func (c *context) blendFunc(mode driver.CompositeMode) {
	_ = c.t.Call(func() error {
		if c.lastCompositeMode == mode {
			return nil
		}
		c.lastCompositeMode = mode
		s, d := mode.Operations()
		s2, d2 := convertOperation(s), convertOperation(d)
		gl.BlendFunc(uint32(s2), uint32(d2))
		return nil
	})
}

func (c *context) newTexture(width, height int) (textureNative, error) {
	var texture textureNative
	if err := c.t.Call(func() error {
		var t uint32
		gl.GenTextures(1, &t)
		// TODO: Use gl.IsTexture
		if t <= 0 {
			return errors.New("opengl: creating texture failed")
		}
		gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
		texture = textureNative(t)
		return nil
	}); err != nil {
		return 0, err
	}
	c.bindTexture(texture)
	_ = c.t.Call(func() error {
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		// If data is nil, this just allocates memory and the content is undefined.
		// https://www.khronos.org/registry/OpenGL-Refpages/gl4/html/glTexImage2D.xhtml
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
		return nil
	})
	return texture, nil
}

func (c *context) bindFramebufferImpl(f framebufferNative) {
	_ = c.t.Call(func() error {
		gl.BindFramebufferEXT(gl.FRAMEBUFFER, uint32(f))
		return nil
	})
}

func (c *context) framebufferPixels(f *framebuffer, width, height int) ([]byte, error) {
	var pixels []byte
	_ = c.t.Call(func() error {
		gl.Flush()
		return nil
	})
	c.bindFramebuffer(f.native)
	if err := c.t.Call(func() error {
		pixels = make([]byte, 4*width*height)
		gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
		return nil
	}); err != nil {
		return nil, err
	}
	return pixels, nil
}

func (c *context) activeTexture(idx int) {
	_ = c.t.Call(func() error {
		gl.ActiveTexture(gl.TEXTURE0 + uint32(idx))
		return nil
	})
}

func (c *context) bindTextureImpl(t textureNative) {
	_ = c.t.Call(func() error {
		gl.BindTexture(gl.TEXTURE_2D, uint32(t))
		return nil
	})
}

func (c *context) deleteTexture(t textureNative) {
	_ = c.t.Call(func() error {
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

func (c *context) isTexture(t textureNative) bool {
	panic("opengl: isTexture is not implemented")
}

func (c *context) newFramebuffer(texture textureNative) (framebufferNative, error) {
	var framebuffer framebufferNative
	var f uint32
	if err := c.t.Call(func() error {
		gl.GenFramebuffersEXT(1, &f)
		// TODO: Use gl.IsFramebuffer
		if f <= 0 {
			return errors.New("opengl: creating framebuffer failed: gl.IsFramebuffer returns false")
		}
		return nil
	}); err != nil {
		return 0, err
	}
	c.bindFramebuffer(framebufferNative(f))
	if err := c.t.Call(func() error {
		gl.FramebufferTexture2DEXT(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, uint32(texture), 0)
		s := gl.CheckFramebufferStatusEXT(gl.FRAMEBUFFER)
		if s != gl.FRAMEBUFFER_COMPLETE {
			if s != 0 {
				return fmt.Errorf("opengl: creating framebuffer failed: %v", s)
			}
			if e := gl.GetError(); e != gl.NO_ERROR {
				return fmt.Errorf("opengl: creating framebuffer failed: (glGetError) %d", e)
			}
			return fmt.Errorf("opengl: creating framebuffer failed: unknown error")
		}
		framebuffer = framebufferNative(f)
		return nil
	}); err != nil {
		return 0, err
	}
	return framebuffer, nil
}

func (c *context) setViewportImpl(width, height int) {
	_ = c.t.Call(func() error {
		gl.Viewport(0, 0, int32(width), int32(height))
		return nil
	})
}

func (c *context) deleteFramebuffer(f framebufferNative) {
	_ = c.t.Call(func() error {
		ff := uint32(f)
		if !gl.IsFramebufferEXT(ff) {
			return nil
		}
		if c.lastFramebuffer == f {
			c.lastFramebuffer = invalidFramebuffer
			c.lastViewportWidth = 0
			c.lastViewportHeight = 0
		}
		gl.DeleteFramebuffersEXT(1, &ff)
		return nil
	})
}

func (c *context) newShader(shaderType shaderType, source string) (shader, error) {
	var sh shader
	if err := c.t.Call(func() error {
		s := gl.CreateShader(uint32(shaderType))
		if s == 0 {
			return fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
		}
		cSources, free := gl.Strs(source + "\x00")
		gl.ShaderSource(uint32(s), 1, cSources, nil)
		free()
		gl.CompileShader(s)

		var v int32
		gl.GetShaderiv(s, gl.COMPILE_STATUS, &v)
		if v == gl.FALSE {
			var l int32
			var log []byte
			gl.GetShaderiv(uint32(s), gl.INFO_LOG_LENGTH, &l)
			if l != 0 {
				log = make([]byte, l)
				gl.GetShaderInfoLog(s, l, nil, (*uint8)(gl.Ptr(log)))
			}
			return fmt.Errorf("opengl: shader compile failed: %s", log)
		}
		sh = shader(s)
		return nil
	}); err != nil {
		return 0, err
	}
	return sh, nil
}

func (c *context) deleteShader(s shader) {
	_ = c.t.Call(func() error {
		gl.DeleteShader(uint32(s))
		return nil
	})
}

func (c *context) newProgram(shaders []shader, attributes []string) (program, error) {
	var pr program
	if err := c.t.Call(func() error {
		p := gl.CreateProgram()
		if p == 0 {
			return errors.New("opengl: glCreateProgram failed")
		}

		for _, shader := range shaders {
			gl.AttachShader(p, uint32(shader))
		}

		for i, name := range attributes {
			l, free := gl.Strs(name + "\x00")
			gl.BindAttribLocation(p, uint32(i), *l)
			free()
		}

		gl.LinkProgram(p)
		var v int32
		gl.GetProgramiv(p, gl.LINK_STATUS, &v)
		if v == gl.FALSE {
			var l int32
			var log []byte
			gl.GetProgramiv(p, gl.INFO_LOG_LENGTH, &l)
			if l != 0 {
				log = make([]byte, l)
				gl.GetProgramInfoLog(p, l, nil, (*uint8)(gl.Ptr(log)))
			}
			return fmt.Errorf("opengl: program error: %s", log)
		}
		pr = program(p)
		return nil
	}); err != nil {
		return 0, err
	}
	return pr, nil
}

func (c *context) useProgram(p program) {
	_ = c.t.Call(func() error {
		gl.UseProgram(uint32(p))
		return nil
	})
}

func (c *context) deleteProgram(p program) {
	_ = c.t.Call(func() error {
		if !gl.IsProgram(uint32(p)) {
			return nil
		}
		gl.DeleteProgram(uint32(p))
		return nil
	})
}

func (c *context) getUniformLocationImpl(p program, location string) uniformLocation {
	l, free := gl.Strs(location + "\x00")
	uniform := uniformLocation(gl.GetUniformLocation(uint32(p), *l))
	free()
	return uniform
}

func (c *context) uniformInt(p program, location string, v int) bool {
	var r bool
	_ = c.t.Call(func() error {
		l := int32(c.locationCache.GetUniformLocation(c, p, location))
		if l == invalidUniform {
			return nil
		}
		r = true
		gl.Uniform1i(l, int32(v))
		return nil
	})
	return r
}

func (c *context) uniformFloat(p program, location string, v float32) bool {
	var r bool
	_ = c.t.Call(func() error {
		l := int32(c.locationCache.GetUniformLocation(c, p, location))
		if l == invalidUniform {
			return nil
		}
		r = true
		gl.Uniform1f(l, v)
		return nil
	})
	return r
}

func (c *context) uniformFloats(p program, location string, v []float32, typ shaderir.Type) bool {
	var r bool
	_ = c.t.Call(func() error {
		l := int32(c.locationCache.GetUniformLocation(c, p, location))
		if l == invalidUniform {
			return nil
		}
		r = true

		base := typ.Main
		len := int32(1)
		if base == shaderir.Array {
			base = typ.Sub[0].Main
			len = int32(typ.Length)
		}

		switch base {
		case shaderir.Float:
			gl.Uniform1fv(l, len, (*float32)(gl.Ptr(v)))
		case shaderir.Vec2:
			gl.Uniform2fv(l, len, (*float32)(gl.Ptr(v)))
		case shaderir.Vec3:
			gl.Uniform3fv(l, len, (*float32)(gl.Ptr(v)))
		case shaderir.Vec4:
			gl.Uniform4fv(l, len, (*float32)(gl.Ptr(v)))
		case shaderir.Mat2:
			gl.UniformMatrix2fv(l, len, false, (*float32)(gl.Ptr(v)))
		case shaderir.Mat3:
			gl.UniformMatrix3fv(l, len, false, (*float32)(gl.Ptr(v)))
		case shaderir.Mat4:
			gl.UniformMatrix4fv(l, len, false, (*float32)(gl.Ptr(v)))
		default:
			panic(fmt.Sprintf("opengl: unexpected type: %s", typ.String()))
		}
		return nil
	})
	return r
}

func (c *context) vertexAttribPointer(p program, index int, size int, dataType dataType, stride int, offset int) {
	_ = c.t.Call(func() error {
		gl.VertexAttribPointer(uint32(index), int32(size), uint32(dataType), false, int32(stride), uintptr(offset))
		return nil
	})
}

func (c *context) enableVertexAttribArray(p program, index int) {
	_ = c.t.Call(func() error {
		gl.EnableVertexAttribArray(uint32(index))
		return nil
	})
}

func (c *context) disableVertexAttribArray(p program, index int) {
	_ = c.t.Call(func() error {
		gl.DisableVertexAttribArray(uint32(index))
		return nil
	})
}

func (c *context) newArrayBuffer(size int) buffer {
	var bf buffer
	_ = c.t.Call(func() error {
		var b uint32
		gl.GenBuffers(1, &b)
		gl.BindBuffer(uint32(arrayBuffer), b)
		gl.BufferData(uint32(arrayBuffer), size, nil, uint32(dynamicDraw))
		bf = buffer(b)
		return nil
	})
	return bf
}

func (c *context) newElementArrayBuffer(size int) buffer {
	var bf buffer
	_ = c.t.Call(func() error {
		var b uint32
		gl.GenBuffers(1, &b)
		gl.BindBuffer(uint32(elementArrayBuffer), b)
		gl.BufferData(uint32(elementArrayBuffer), size, nil, uint32(dynamicDraw))
		bf = buffer(b)
		return nil
	})
	return bf
}

func (c *context) bindBuffer(bufferType bufferType, b buffer) {
	_ = c.t.Call(func() error {
		gl.BindBuffer(uint32(bufferType), uint32(b))
		return nil
	})
}

func (c *context) arrayBufferSubData(data []float32) {
	_ = c.t.Call(func() error {
		gl.BufferSubData(uint32(arrayBuffer), 0, len(data)*4, gl.Ptr(data))
		return nil
	})
}

func (c *context) elementArrayBufferSubData(data []uint16) {
	_ = c.t.Call(func() error {
		gl.BufferSubData(uint32(elementArrayBuffer), 0, len(data)*2, gl.Ptr(data))
		return nil
	})
}

func (c *context) deleteBuffer(b buffer) {
	_ = c.t.Call(func() error {
		bb := uint32(b)
		gl.DeleteBuffers(1, &bb)
		return nil
	})
}

func (c *context) drawElements(len int, offsetInBytes int) {
	_ = c.t.Call(func() error {
		gl.DrawElements(gl.TRIANGLES, int32(len), gl.UNSIGNED_SHORT, uintptr(offsetInBytes))
		return nil
	})
}

func (c *context) maxTextureSizeImpl() int {
	size := 0
	_ = c.t.Call(func() error {
		s := int32(0)
		gl.GetIntegerv(gl.MAX_TEXTURE_SIZE, &s)
		size = int(s)
		return nil
	})
	return size
}

func (c *context) getShaderPrecisionFormatPrecision() int {
	// glGetShaderPrecisionFormat is not defined at OpenGL 2.0. Assume that desktop environments always have
	// enough highp precision.
	return highpPrecision
}

func (c *context) flush() {
	_ = c.t.Call(func() error {
		gl.Flush()
		return nil
	})
}

func (c *context) needsRestoring() bool {
	return false
}

func (c *context) canUsePBO() bool {
	var available bool
	_ = c.t.Call(func() error {
		available = isPBOAvailable()
		return nil
	})

	return available
}

func (c *context) texSubImage2D(t textureNative, width, height int, args []*driver.ReplacePixelsArgs) {
	c.bindTexture(t)
	_ = c.t.Call(func() error {
		for _, a := range args {
			gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(a.X), int32(a.Y), int32(a.Width), int32(a.Height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(a.Pixels))
		}
		return nil
	})
}

func (c *context) newPixelBufferObject(width, height int) buffer {
	var bf buffer
	_ = c.t.Call(func() error {
		var b uint32
		gl.GenBuffers(1, &b)
		gl.BindBuffer(gl.PIXEL_UNPACK_BUFFER, b)
		gl.BufferData(gl.PIXEL_UNPACK_BUFFER, 4*width*height, nil, gl.STREAM_DRAW)
		gl.BindBuffer(gl.PIXEL_UNPACK_BUFFER, 0)
		bf = buffer(b)
		return nil
	})
	return bf
}

func (c *context) replacePixelsWithPBO(buffer buffer, t textureNative, width, height int, args []*driver.ReplacePixelsArgs) {
	c.bindTexture(t)
	_ = c.t.Call(func() error {
		gl.BindBuffer(gl.PIXEL_UNPACK_BUFFER, uint32(buffer))

		stride := 4 * width
		for _, a := range args {
			offset := 4 * (a.Y*width + a.X)
			for j := 0; j < a.Height; j++ {
				gl.BufferSubData(gl.PIXEL_UNPACK_BUFFER, offset+stride*j, 4*a.Width, gl.Ptr(a.Pixels[4*a.Width*j:4*a.Width*(j+1)]))
			}
		}

		gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, nil)
		gl.BindBuffer(gl.PIXEL_UNPACK_BUFFER, 0)
		return nil
	})
}
