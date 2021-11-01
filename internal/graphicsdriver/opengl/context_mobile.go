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

//go:build android || ios
// +build android ios

package opengl

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gles"
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

func (p program) equal(rhs program) bool {
	return p == rhs
}

var InvalidTexture textureNative

type (
	uniformLocation int32
	attribLocation  int32
)

func (u uniformLocation) equal(rhs uniformLocation) bool {
	return u == rhs
}

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
	zero             = operation(gles.ZERO)
	one              = operation(gles.ONE)
	srcAlpha         = operation(gles.SRC_ALPHA)
	dstAlpha         = operation(gles.DST_ALPHA)
	oneMinusSrcAlpha = operation(gles.ONE_MINUS_SRC_ALPHA)
	oneMinusDstAlpha = operation(gles.ONE_MINUS_DST_ALPHA)
	dstColor         = operation(gles.DST_COLOR)
)

type contextImpl struct {
	ctx gles.Context
}

func (c *context) reset() error {
	c.locationCache = newLocationCache()
	c.lastTexture = invalidTexture
	c.lastFramebuffer = invalidFramebuffer
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = driver.CompositeModeUnknown
	c.ctx.Enable(gles.BLEND)
	c.ctx.Enable(gles.SCISSOR_TEST)
	c.blendFunc(driver.CompositeModeSourceOver)
	f := make([]int32, 1)
	c.ctx.GetIntegerv(f, gles.FRAMEBUFFER_BINDING)
	c.screenFramebuffer = framebufferNative(f[0])
	// TODO: Need to update screenFramebufferWidth/Height?
	return nil
}

func (c *context) blendFunc(mode driver.CompositeMode) {
	if c.lastCompositeMode == mode {
		return
	}
	c.lastCompositeMode = mode
	s, d := mode.Operations()
	s2, d2 := convertOperation(s), convertOperation(d)
	c.ctx.BlendFunc(uint32(s2), uint32(d2))
}

func (c *context) scissor(x, y, width, height int) {
	c.ctx.Scissor(int32(x), int32(y), int32(width), int32(height))
}

func (c *context) newTexture(width, height int) (textureNative, error) {
	t := c.ctx.GenTextures(1)[0]
	if t <= 0 {
		return 0, errors.New("opengl: creating texture failed")
	}
	c.bindTexture(textureNative(t))

	c.ctx.TexParameteri(gles.TEXTURE_2D, gles.TEXTURE_MAG_FILTER, gles.NEAREST)
	c.ctx.TexParameteri(gles.TEXTURE_2D, gles.TEXTURE_MIN_FILTER, gles.NEAREST)
	c.ctx.TexParameteri(gles.TEXTURE_2D, gles.TEXTURE_WRAP_S, gles.CLAMP_TO_EDGE)
	c.ctx.TexParameteri(gles.TEXTURE_2D, gles.TEXTURE_WRAP_T, gles.CLAMP_TO_EDGE)
	c.ctx.PixelStorei(gles.UNPACK_ALIGNMENT, 4)
	c.ctx.TexImage2D(gles.TEXTURE_2D, 0, gles.RGBA, int32(width), int32(height), gles.RGBA, gles.UNSIGNED_BYTE, nil)

	return textureNative(t), nil
}

func (c *context) bindFramebufferImpl(f framebufferNative) {
	c.ctx.BindFramebuffer(gles.FRAMEBUFFER, uint32(f))
}

func (c *context) framebufferPixels(f *framebuffer, width, height int) []byte {
	c.ctx.Flush()

	c.bindFramebuffer(f.native)

	pixels := make([]byte, 4*width*height)
	c.ctx.ReadPixels(pixels, 0, 0, int32(width), int32(height), gles.RGBA, gles.UNSIGNED_BYTE)
	return pixels
}

func (c *context) framebufferPixelsToBuffer(f *framebuffer, buffer buffer, width, height int) {
	c.ctx.Flush()

	c.bindFramebuffer(f.native)

	c.ctx.BindBuffer(gles.PIXEL_PACK_BUFFER, uint32(buffer))
	c.ctx.ReadPixels(nil, 0, 0, int32(width), int32(height), gles.RGBA, gles.UNSIGNED_BYTE)
	c.ctx.BindBuffer(gles.PIXEL_PACK_BUFFER, 0)
}

func (c *context) activeTexture(idx int) {
	c.ctx.ActiveTexture(uint32(gles.TEXTURE0 + idx))
}

func (c *context) bindTextureImpl(t textureNative) {
	c.ctx.BindTexture(gles.TEXTURE_2D, uint32(t))
}

func (c *context) deleteTexture(t textureNative) {
	if !c.ctx.IsTexture(uint32(t)) {
		return
	}
	if c.lastTexture == t {
		c.lastTexture = invalidTexture
	}
	c.ctx.DeleteTextures([]uint32{uint32(t)})
}

func (c *context) isTexture(t textureNative) bool {
	return c.ctx.IsTexture(uint32(t))
}

func (c *context) newRenderbuffer(width, height int) (renderbufferNative, error) {
	r := c.ctx.GenRenderbuffers(1)[0]
	if r <= 0 {
		return 0, errors.New("opengl: creating renderbuffer failed")
	}

	renderbuffer := renderbufferNative(r)
	c.bindRenderbuffer(renderbuffer)

	c.ctx.RenderbufferStorage(gles.RENDERBUFFER, gles.STENCIL_INDEX8, int32(width), int32(height))

	return renderbuffer, nil
}

func (c *context) bindRenderbufferImpl(r renderbufferNative) {
	c.ctx.BindRenderbuffer(gles.RENDERBUFFER, uint32(r))
}

func (c *context) deleteRenderbuffer(r renderbufferNative) {
	if !c.ctx.IsRenderbuffer(uint32(r)) {
		return
	}
	if c.lastRenderbuffer.equal(r) {
		c.lastRenderbuffer = 0
	}
	c.ctx.DeleteRenderbuffers([]uint32{uint32(r)})
}

func (c *context) newFramebuffer(texture textureNative) (framebufferNative, error) {
	f := c.ctx.GenFramebuffers(1)[0]
	if f <= 0 {
		return 0, fmt.Errorf("opengl: creating framebuffer failed: the returned value is not positive but %d", f)
	}
	c.bindFramebuffer(framebufferNative(f))

	c.ctx.FramebufferTexture2D(gles.FRAMEBUFFER, gles.COLOR_ATTACHMENT0, gles.TEXTURE_2D, uint32(texture), 0)
	s := c.ctx.CheckFramebufferStatus(gles.FRAMEBUFFER)
	if s != gles.FRAMEBUFFER_COMPLETE {
		if s != 0 {
			return 0, fmt.Errorf("opengl: creating framebuffer failed: %v", s)
		}
		if e := c.ctx.GetError(); e != gles.NO_ERROR {
			return 0, fmt.Errorf("opengl: creating framebuffer failed: (glGetError) %d", e)
		}
		return 0, fmt.Errorf("opengl: creating framebuffer failed: unknown error")
	}
	return framebufferNative(f), nil
}

func (c *context) bindStencilBuffer(f framebufferNative, r renderbufferNative) error {
	c.bindFramebuffer(f)

	c.ctx.FramebufferRenderbuffer(gles.FRAMEBUFFER, gles.STENCIL_ATTACHMENT, gles.RENDERBUFFER, uint32(r))
	if s := c.ctx.CheckFramebufferStatus(gles.FRAMEBUFFER); s != gles.FRAMEBUFFER_COMPLETE {
		return errors.New(fmt.Sprintf("opengl: glFramebufferRenderbuffer failed: %d", s))
	}
	return nil
}

func (c *context) setViewportImpl(width, height int) {
	c.ctx.Viewport(0, 0, int32(width), int32(height))
}

func (c *context) deleteFramebuffer(f framebufferNative) {
	if !c.ctx.IsFramebuffer(uint32(f)) {
		return
	}
	// If a framebuffer to be deleted is bound, a newly bound framebuffer
	// will be a default framebuffer.
	// https://www.khronos.org/opengles/sdk/docs/man/xhtml/glDeleteFramebuffers.xml
	if c.lastFramebuffer == f {
		c.lastFramebuffer = invalidFramebuffer
		c.lastViewportWidth = 0
		c.lastViewportHeight = 0
	}
	c.ctx.DeleteFramebuffers([]uint32{uint32(f)})
}

func (c *context) newVertexShader(source string) (shader, error) {
	return c.newShader(gles.VERTEX_SHADER, source)
}

func (c *context) newFragmentShader(source string) (shader, error) {
	return c.newShader(gles.FRAGMENT_SHADER, source)
}

func (c *context) newShader(shaderType uint32, source string) (shader, error) {
	s := c.ctx.CreateShader(shaderType)
	if s == 0 {
		return 0, fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
	}
	c.ctx.ShaderSource(s, source)
	c.ctx.CompileShader(s)

	v := make([]int32, 1)
	c.ctx.GetShaderiv(v, s, gles.COMPILE_STATUS)
	if v[0] == gles.FALSE {
		log := c.ctx.GetShaderInfoLog(s)
		return 0, fmt.Errorf("opengl: shader compile failed: %s", log)
	}
	return shader(s), nil
}

func (c *context) deleteShader(s shader) {
	c.ctx.DeleteShader(uint32(s))
}

func (c *context) newProgram(shaders []shader, attributes []string) (program, error) {
	p := c.ctx.CreateProgram()
	if p == 0 {
		return 0, errors.New("opengl: glCreateProgram failed")
	}

	for _, shader := range shaders {
		c.ctx.AttachShader(p, uint32(shader))
	}

	for i, name := range attributes {
		c.ctx.BindAttribLocation(p, uint32(i), name)
	}

	c.ctx.LinkProgram(p)
	v := make([]int32, 1)
	c.ctx.GetProgramiv(v, p, gles.LINK_STATUS)
	if v[0] == gles.FALSE {
		info := c.ctx.GetProgramInfoLog(p)
		return 0, fmt.Errorf("opengl: program error: %s", info)
	}
	return program(p), nil
}

func (c *context) useProgram(p program) {
	c.ctx.UseProgram(uint32(p))
}

func (c *context) deleteProgram(p program) {
	c.locationCache.deleteProgram(p)

	if !c.ctx.IsProgram(uint32(p)) {
		return
	}
	c.ctx.DeleteProgram(uint32(p))
}

func (c *context) getUniformLocationImpl(p program, location string) uniformLocation {
	u := uniformLocation(c.ctx.GetUniformLocation(uint32(p), location))
	return u
}

func (c *context) uniformInt(p program, location string, v int) bool {
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l == invalidUniform {
		return false
	}
	c.ctx.Uniform1i(int32(l), int32(v))
	return true
}

func (c *context) uniformFloat(p program, location string, v float32) bool {
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l == invalidUniform {
		return false
	}
	c.ctx.Uniform1f(int32(l), v)
	return true
}

func (c *context) uniformFloats(p program, location string, v []float32, typ shaderir.Type) bool {
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l == invalidUniform {
		return false
	}

	base := typ.Main
	if base == shaderir.Array {
		base = typ.Sub[0].Main
	}

	switch base {
	case shaderir.Float:
		c.ctx.Uniform1fv(int32(l), v)
	case shaderir.Vec2:
		c.ctx.Uniform2fv(int32(l), v)
	case shaderir.Vec3:
		c.ctx.Uniform3fv(int32(l), v)
	case shaderir.Vec4:
		c.ctx.Uniform4fv(int32(l), v)
	case shaderir.Mat2:
		c.ctx.UniformMatrix2fv(int32(l), false, v)
	case shaderir.Mat3:
		c.ctx.UniformMatrix3fv(int32(l), false, v)
	case shaderir.Mat4:
		c.ctx.UniformMatrix4fv(int32(l), false, v)
	default:
		panic(fmt.Sprintf("opengl: unexpected type: %s", typ.String()))
	}
	return true
}

func (c *context) vertexAttribPointer(index int, size int, stride int, offset int) {
	c.ctx.VertexAttribPointer(uint32(index), int32(size), gles.FLOAT, false, int32(stride), offset)
}

func (c *context) enableVertexAttribArray(index int) {
	c.ctx.EnableVertexAttribArray(uint32(index))
}

func (c *context) disableVertexAttribArray(index int) {
	c.ctx.DisableVertexAttribArray(uint32(index))
}

func (c *context) newArrayBuffer(size int) buffer {
	b := c.ctx.GenBuffers(1)[0]
	c.ctx.BindBuffer(gles.ARRAY_BUFFER, b)
	c.ctx.BufferData(gles.ARRAY_BUFFER, size, nil, gles.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) newElementArrayBuffer(size int) buffer {
	b := c.ctx.GenBuffers(1)[0]
	c.ctx.BindBuffer(gles.ELEMENT_ARRAY_BUFFER, b)
	c.ctx.BufferData(gles.ELEMENT_ARRAY_BUFFER, size, nil, gles.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) bindArrayBuffer(b buffer) {
	c.ctx.BindBuffer(gles.ARRAY_BUFFER, uint32(b))
}

func (c *context) bindElementArrayBuffer(b buffer) {
	c.ctx.BindBuffer(gles.ELEMENT_ARRAY_BUFFER, uint32(b))
}

func (c *context) arrayBufferSubData(data []float32) {
	c.ctx.BufferSubData(gles.ARRAY_BUFFER, 0, float32sToBytes(data))
}

func (c *context) elementArrayBufferSubData(data []uint16) {
	c.ctx.BufferSubData(gles.ELEMENT_ARRAY_BUFFER, 0, uint16sToBytes(data))
}

func (c *context) deleteBuffer(b buffer) {
	c.ctx.DeleteBuffers([]uint32{uint32(b)})
}

func (c *context) drawElements(len int, offsetInBytes int) {
	c.ctx.DrawElements(gles.TRIANGLES, int32(len), gles.UNSIGNED_SHORT, offsetInBytes)
}

func (c *context) maxTextureSizeImpl() int {
	v := make([]int32, 1)
	c.ctx.GetIntegerv(v, gles.MAX_TEXTURE_SIZE)
	return int(v[0])
}

func (c *context) getShaderPrecisionFormatPrecision() int {
	_, _, p := c.ctx.GetShaderPrecisionFormat(gles.FRAGMENT_SHADER, gles.HIGH_FLOAT)
	return p
}

func (c *context) flush() {
	c.ctx.Flush()
}

func (c *context) needsRestoring() bool {
	return true
}

func (c *context) canUsePBO() bool {
	// On Android, using PBO might slow the applications, especially when coming back from the context lost.
	// Let's not use PBO until we find a good solution.
	return false
}

func (c *context) texSubImage2D(t textureNative, args []*driver.ReplacePixelsArgs) {
	c.bindTexture(t)
	for _, a := range args {
		c.ctx.TexSubImage2D(gles.TEXTURE_2D, 0, int32(a.X), int32(a.Y), int32(a.Width), int32(a.Height), gles.RGBA, gles.UNSIGNED_BYTE, a.Pixels)
	}
}

func (c *context) enableStencilTest() {
	c.ctx.Enable(gles.STENCIL_TEST)
}

func (c *context) disableStencilTest() {
	c.ctx.Disable(gles.STENCIL_TEST)
}

func (c *context) beginStencilWithEvenOddRule() {
	c.ctx.Clear(gles.STENCIL_BUFFER_BIT)
	c.ctx.StencilFunc(gles.ALWAYS, 0x00, 0xff)
	c.ctx.StencilOp(gles.KEEP, gles.KEEP, gles.INVERT)
	c.ctx.ColorMask(false, false, false, false)
}

func (c *context) endStencilWithEvenOddRule() {
	c.ctx.StencilFunc(gles.NOTEQUAL, 0x00, 0xff)
	c.ctx.StencilOp(gles.KEEP, gles.KEEP, gles.KEEP)
	c.ctx.ColorMask(true, true, true, true)
}
