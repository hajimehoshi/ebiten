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

//go:build (darwin || freebsd || linux || windows) && !android && !ios
// +build darwin freebsd linux windows
// +build !android
// +build !ios

package opengl

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type (
	textureNative      uint32
	renderbufferNative uint32
	framebufferNative  uint32
	shader             uint32
	program            uint32
	buffer             uint32
)

func (t textureNative) equal(rhs textureNative) bool {
	return t == rhs
}

func (r renderbufferNative) equal(rhs renderbufferNative) bool {
	return r == rhs
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
	if !c.init {
		// Note that this initialization must be done after Loop is called.
		if err := gl.Init(); err != nil {
			return fmt.Errorf("opengl: initializing error %v", err)
		}
		c.init = true
	}

	c.locationCache = newLocationCache()
	c.lastTexture = invalidTexture
	c.lastFramebuffer = invalidFramebuffer
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = driver.CompositeModeUnknown
	gl.Enable(gl.BLEND)
	gl.Enable(gl.SCISSOR_TEST)

	c.blendFunc(driver.CompositeModeSourceOver)

	f := int32(0)
	gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &f)
	c.screenFramebuffer = framebufferNative(f)
	return nil
}

func (c *context) blendFunc(mode driver.CompositeMode) {
	if c.lastCompositeMode == mode {
		return
	}
	c.lastCompositeMode = mode
	s, d := mode.Operations()
	s2, d2 := convertOperation(s), convertOperation(d)
	gl.BlendFunc(uint32(s2), uint32(d2))
}

func (c *context) scissor(x, y, width, height int) {
	gl.Scissor(int32(x), int32(y), int32(width), int32(height))
}

func (c *context) newTexture(width, height int) (textureNative, error) {
	var t uint32
	gl.GenTextures(1, &t)
	// TODO: Use gl.IsTexture
	if t <= 0 {
		return 0, errors.New("opengl: creating texture failed")
	}
	texture := textureNative(t)
	c.bindTexture(texture)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	// If data is nil, this just allocates memory and the content is undefined.
	// https://www.khronos.org/registry/OpenGL-Refpages/gl4/html/glTexImage2D.xhtml
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	return texture, nil
}

func (c *context) bindFramebufferImpl(f framebufferNative) {
	gl.BindFramebufferEXT(gl.FRAMEBUFFER, uint32(f))
}

func (c *context) framebufferPixels(f *framebuffer, width, height int) []byte {
	gl.Flush()
	c.bindFramebuffer(f.native)
	pixels := make([]byte, 4*width*height)
	gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
	return pixels
}

func (c *context) framebufferPixelsToBuffer(f *framebuffer, buffer buffer, width, height int) {
	gl.Flush()
	c.bindFramebuffer(f.native)
	gl.BindBuffer(gl.PIXEL_PACK_BUFFER, uint32(buffer))
	gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.BindBuffer(gl.PIXEL_PACK_BUFFER, 0)
}

func (c *context) activeTexture(idx int) {
	gl.ActiveTexture(gl.TEXTURE0 + uint32(idx))
}

func (c *context) bindTextureImpl(t textureNative) {
	gl.BindTexture(gl.TEXTURE_2D, uint32(t))
}

func (c *context) deleteTexture(t textureNative) {
	tt := uint32(t)
	if !gl.IsTexture(tt) {
		return
	}
	if c.lastTexture == t {
		c.lastTexture = invalidTexture
	}
	gl.DeleteTextures(1, &tt)
}

func (c *context) isTexture(t textureNative) bool {
	panic("opengl: isTexture is not implemented")
}

func (c *context) newRenderbuffer(width, height int) (renderbufferNative, error) {
	var r uint32
	gl.GenRenderbuffersEXT(1, &r)
	if r <= 0 {
		return 0, errors.New("opengl: creating renderbuffer failed")
	}

	renderbuffer := renderbufferNative(r)
	c.bindRenderbuffer(renderbuffer)

	// GL_STENCIL_INDEX8 might not be available with OpenGL 2.1.
	// https://www.khronos.org/opengl/wiki/Image_Format
	gl.RenderbufferStorageEXT(gl.RENDERBUFFER, gl.DEPTH24_STENCIL8, int32(width), int32(height))

	return renderbuffer, nil
}

func (c *context) bindRenderbufferImpl(r renderbufferNative) {
	gl.BindRenderbufferEXT(gl.RENDERBUFFER, uint32(r))
}

func (c *context) deleteRenderbuffer(r renderbufferNative) {
	rr := uint32(r)
	if !gl.IsRenderbufferEXT(rr) {
		return
	}
	if c.lastRenderbuffer.equal(r) {
		c.lastRenderbuffer = 0
	}
	gl.DeleteRenderbuffersEXT(1, &rr)
}

func (c *context) newFramebuffer(texture textureNative) (framebufferNative, error) {
	var f uint32
	gl.GenFramebuffersEXT(1, &f)
	// TODO: Use gl.IsFramebuffer
	if f <= 0 {
		return 0, errors.New("opengl: creating framebuffer failed: gl.IsFramebuffer returns false")
	}
	c.bindFramebuffer(framebufferNative(f))
	gl.FramebufferTexture2DEXT(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, uint32(texture), 0)
	s := gl.CheckFramebufferStatusEXT(gl.FRAMEBUFFER)
	if s != gl.FRAMEBUFFER_COMPLETE {
		if s != 0 {
			return 0, fmt.Errorf("opengl: creating framebuffer failed: %v", s)
		}
		if e := gl.GetError(); e != gl.NO_ERROR {
			return 0, fmt.Errorf("opengl: creating framebuffer failed: (glGetError) %d", e)
		}
		return 0, fmt.Errorf("opengl: creating framebuffer failed: unknown error")
	}
	return framebufferNative(f), nil
}

func (c *context) bindStencilBuffer(f framebufferNative, r renderbufferNative) error {
	c.bindFramebuffer(f)

	gl.FramebufferRenderbufferEXT(gl.FRAMEBUFFER, gl.STENCIL_ATTACHMENT, gl.RENDERBUFFER, uint32(r))
	if s := gl.CheckFramebufferStatusEXT(gl.FRAMEBUFFER); s != gl.FRAMEBUFFER_COMPLETE {
		return errors.New(fmt.Sprintf("opengl: glFramebufferRenderbuffer failed: %d", s))
	}
	return nil
}

func (c *context) setViewportImpl(width, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
}

func (c *context) deleteFramebuffer(f framebufferNative) {
	ff := uint32(f)
	if !gl.IsFramebufferEXT(ff) {
		return
	}
	if c.lastFramebuffer == f {
		c.lastFramebuffer = invalidFramebuffer
		c.lastViewportWidth = 0
		c.lastViewportHeight = 0
	}
	gl.DeleteFramebuffersEXT(1, &ff)
}

func (c *context) newVertexShader(source string) (shader, error) {
	return c.newShader(gl.VERTEX_SHADER, source)
}

func (c *context) newFragmentShader(source string) (shader, error) {
	return c.newShader(gl.FRAGMENT_SHADER, source)
}

func (c *context) newShader(shaderType uint32, source string) (shader, error) {
	s := gl.CreateShader(shaderType)
	if s == 0 {
		return 0, fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
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
		return 0, fmt.Errorf("opengl: shader compile failed: %s", log)
	}
	return shader(s), nil
}

func (c *context) deleteShader(s shader) {
	gl.DeleteShader(uint32(s))
}

func (c *context) newProgram(shaders []shader, attributes []string) (program, error) {
	p := gl.CreateProgram()
	if p == 0 {
		return 0, errors.New("opengl: glCreateProgram failed")
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
		return 0, fmt.Errorf("opengl: program error: %s", log)
	}
	return program(p), nil
}

func (c *context) useProgram(p program) {
	gl.UseProgram(uint32(p))
}

func (c *context) deleteProgram(p program) {
	c.locationCache.deleteProgram(p)

	if !gl.IsProgram(uint32(p)) {
		return
	}
	gl.DeleteProgram(uint32(p))
}

func (c *context) getUniformLocationImpl(p program, location string) uniformLocation {
	l, free := gl.Strs(location + "\x00")
	uniform := uniformLocation(gl.GetUniformLocation(uint32(p), *l))
	free()
	return uniform
}

func (c *context) uniformInt(p program, location string, v int) bool {
	l := int32(c.locationCache.GetUniformLocation(c, p, location))
	if l == invalidUniform {
		return false
	}
	gl.Uniform1i(l, int32(v))
	return true
}

func (c *context) uniformFloat(p program, location string, v float32) bool {
	l := int32(c.locationCache.GetUniformLocation(c, p, location))
	if l == invalidUniform {
		return false
	}
	gl.Uniform1f(l, v)
	return true
}

func (c *context) uniformFloats(p program, location string, v []float32, typ shaderir.Type) bool {
	l := int32(c.locationCache.GetUniformLocation(c, p, location))
	if l == invalidUniform {
		return false
	}

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
	return true
}

func (c *context) vertexAttribPointer(index int, size int, stride int, offset int) {
	gl.VertexAttribPointer(uint32(index), int32(size), gl.FLOAT, false, int32(stride), uintptr(offset))
}

func (c *context) enableVertexAttribArray(index int) {
	gl.EnableVertexAttribArray(uint32(index))
}

func (c *context) disableVertexAttribArray(index int) {
	gl.DisableVertexAttribArray(uint32(index))
}

func (c *context) newArrayBuffer(size int) buffer {
	var b uint32
	gl.GenBuffers(1, &b)
	gl.BindBuffer(gl.ARRAY_BUFFER, b)
	gl.BufferData(gl.ARRAY_BUFFER, size, nil, gl.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) newElementArrayBuffer(size int) buffer {
	var b uint32
	gl.GenBuffers(1, &b)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, b)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, size, nil, gl.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) bindArrayBuffer(b buffer) {
	gl.BindBuffer(gl.ARRAY_BUFFER, uint32(b))
}

func (c *context) bindElementArrayBuffer(b buffer) {
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, uint32(b))
}

func (c *context) arrayBufferSubData(data []float32) {
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(data)*4, gl.Ptr(data))
}

func (c *context) elementArrayBufferSubData(data []uint16) {
	gl.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, 0, len(data)*2, gl.Ptr(data))
}

func (c *context) deleteBuffer(b buffer) {
	bb := uint32(b)
	gl.DeleteBuffers(1, &bb)
}

func (c *context) drawElements(len int, offsetInBytes int) {
	gl.DrawElements(gl.TRIANGLES, int32(len), gl.UNSIGNED_SHORT, uintptr(offsetInBytes))
}

func (c *context) maxTextureSizeImpl() int {
	s := int32(0)
	gl.GetIntegerv(gl.MAX_TEXTURE_SIZE, &s)
	return int(s)
}

func (c *context) getShaderPrecisionFormatPrecision() int {
	// glGetShaderPrecisionFormat is not defined at OpenGL 2.0. Assume that desktop environments always have
	// enough highp precision.
	return highpPrecision
}

func (c *context) flush() {
	gl.Flush()
}

func (c *context) needsRestoring() bool {
	return false
}

func (c *context) texSubImage2D(t textureNative, args []*driver.ReplacePixelsArgs) {
	c.bindTexture(t)
	for _, a := range args {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(a.X), int32(a.Y), int32(a.Width), int32(a.Height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(a.Pixels))
	}
}

func (c *context) enableStencilTest() {
	gl.Enable(gl.STENCIL_TEST)
}

func (c *context) disableStencilTest() {
	gl.Disable(gl.STENCIL_TEST)
}

func (c *context) beginStencilWithEvenOddRule() {
	gl.Clear(gl.STENCIL_BUFFER_BIT)
	gl.StencilFunc(gl.ALWAYS, 0x00, 0xff)
	gl.StencilOp(gl.KEEP, gl.KEEP, gl.INVERT)
	gl.ColorMask(false, false, false, false)
}

func (c *context) endStencilWithEvenOddRule() {
	gl.StencilFunc(gl.NOTEQUAL, 0x00, 0xff)
	gl.StencilOp(gl.KEEP, gl.KEEP, gl.KEEP)
	gl.ColorMask(true, true, true, true)
}
