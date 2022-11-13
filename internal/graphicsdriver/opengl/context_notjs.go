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

//go:build !js

package opengl

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
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

type contextImpl struct {
	ctx  gl.Context
	init bool
}

func (c *context) reset() error {
	if !c.init {
		// Initialize OpenGL after WGL is initialized especially for Windows (#2452).
		if err := c.ctx.Init(); err != nil {
			return err
		}
		c.init = true
	}

	c.locationCache = newLocationCache()
	c.lastTexture = invalidTexture
	c.lastFramebuffer = invalidFramebuffer
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastBlend = graphicsdriver.Blend{}
	c.ctx.Enable(gl.BLEND)
	c.ctx.Enable(gl.SCISSOR_TEST)
	c.blend(graphicsdriver.BlendSourceOver)
	f := make([]int32, 1)
	c.ctx.GetIntegerv(f, gl.FRAMEBUFFER_BINDING)
	c.screenFramebuffer = framebufferNative(f[0])
	// TODO: Need to update screenFramebufferWidth/Height?
	return nil
}

func (c *context) blend(blend graphicsdriver.Blend) {
	if c.lastBlend == blend {
		return
	}
	c.lastBlend = blend
	c.ctx.BlendFuncSeparate(
		uint32(convertBlendFactor(blend.BlendFactorSourceRGB)),
		uint32(convertBlendFactor(blend.BlendFactorDestinationRGB)),
		uint32(convertBlendFactor(blend.BlendFactorSourceAlpha)),
		uint32(convertBlendFactor(blend.BlendFactorDestinationAlpha)),
	)
	c.ctx.BlendEquationSeparate(
		uint32(convertBlendOperation(blend.BlendOperationRGB)),
		uint32(convertBlendOperation(blend.BlendOperationAlpha)),
	)
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

	c.ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	c.ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	c.ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	c.ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	c.ctx.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	c.ctx.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, nil)

	return textureNative(t), nil
}

func (c *context) bindFramebufferImpl(f framebufferNative) {
	c.ctx.BindFramebuffer(gl.FRAMEBUFFER, uint32(f))
}

func (c *context) framebufferPixels(buf []byte, f *framebuffer, x, y, width, height int) {
	c.ctx.Flush()

	c.bindFramebuffer(f.native)

	c.ctx.ReadPixels(buf, int32(x), int32(y), int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE)
}

func (c *context) framebufferPixelsToBuffer(f *framebuffer, buffer buffer, width, height int) {
	c.ctx.Flush()

	c.bindFramebuffer(f.native)

	c.ctx.BindBuffer(gl.PIXEL_PACK_BUFFER, uint32(buffer))
	c.ctx.ReadPixels(nil, 0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE)
	c.ctx.BindBuffer(gl.PIXEL_PACK_BUFFER, 0)
}

func (c *context) activeTexture(idx int) {
	c.ctx.ActiveTexture(uint32(gl.TEXTURE0 + idx))
}

func (c *context) bindTextureImpl(t textureNative) {
	c.ctx.BindTexture(gl.TEXTURE_2D, uint32(t))
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

	// GL_STENCIL_INDEX8 might not be available with OpenGL 2.1.
	// https://www.khronos.org/opengl/wiki/Image_Format
	c.ctx.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH24_STENCIL8, int32(width), int32(height))

	return renderbuffer, nil
}

func (c *context) bindRenderbufferImpl(r renderbufferNative) {
	c.ctx.BindRenderbuffer(gl.RENDERBUFFER, uint32(r))
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

	c.ctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, uint32(texture), 0)
	s := c.ctx.CheckFramebufferStatus(gl.FRAMEBUFFER)
	if s != gl.FRAMEBUFFER_COMPLETE {
		if s != 0 {
			return 0, fmt.Errorf("opengl: creating framebuffer failed: %v", s)
		}
		if e := c.ctx.GetError(); e != gl.NO_ERROR {
			return 0, fmt.Errorf("opengl: creating framebuffer failed: (glGetError) %d", e)
		}
		return 0, fmt.Errorf("opengl: creating framebuffer failed: unknown error")
	}
	return framebufferNative(f), nil
}

func (c *context) bindStencilBuffer(f framebufferNative, r renderbufferNative) error {
	c.bindFramebuffer(f)

	c.ctx.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.STENCIL_ATTACHMENT, gl.RENDERBUFFER, uint32(r))
	if s := c.ctx.CheckFramebufferStatus(gl.FRAMEBUFFER); s != gl.FRAMEBUFFER_COMPLETE {
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
	return c.newShader(gl.VERTEX_SHADER, source)
}

func (c *context) newFragmentShader(source string) (shader, error) {
	return c.newShader(gl.FRAGMENT_SHADER, source)
}

func (c *context) newShader(shaderType uint32, source string) (shader, error) {
	s := c.ctx.CreateShader(shaderType)
	if s == 0 {
		return 0, fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
	}
	c.ctx.ShaderSource(s, source)
	c.ctx.CompileShader(s)

	v := make([]int32, 1)
	c.ctx.GetShaderiv(v, s, gl.COMPILE_STATUS)
	if v[0] == gl.FALSE {
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
	c.ctx.GetProgramiv(v, p, gl.LINK_STATUS)
	if v[0] == gl.FALSE {
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

func (c *context) uniforms(p program, location string, v []uint32, typ shaderir.Type) bool {
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
		c.ctx.Uniform1fv(int32(l), uint32sToFloat32s(v))
	case shaderir.Int:
		c.ctx.Uniform1iv(int32(l), uint32sToInt32s(v))
	case shaderir.Vec2:
		c.ctx.Uniform2fv(int32(l), uint32sToFloat32s(v))
	case shaderir.Vec3:
		c.ctx.Uniform3fv(int32(l), uint32sToFloat32s(v))
	case shaderir.Vec4:
		c.ctx.Uniform4fv(int32(l), uint32sToFloat32s(v))
	case shaderir.Mat2:
		c.ctx.UniformMatrix2fv(int32(l), false, uint32sToFloat32s(v))
	case shaderir.Mat3:
		c.ctx.UniformMatrix3fv(int32(l), false, uint32sToFloat32s(v))
	case shaderir.Mat4:
		c.ctx.UniformMatrix4fv(int32(l), false, uint32sToFloat32s(v))
	default:
		panic(fmt.Sprintf("opengl: unexpected type: %s", typ.String()))
	}
	return true
}

func (c *context) vertexAttribPointer(index int, size int, stride int, offset int) {
	c.ctx.VertexAttribPointer(uint32(index), int32(size), gl.FLOAT, false, int32(stride), offset)
}

func (c *context) enableVertexAttribArray(index int) {
	c.ctx.EnableVertexAttribArray(uint32(index))
}

func (c *context) disableVertexAttribArray(index int) {
	c.ctx.DisableVertexAttribArray(uint32(index))
}

func (c *context) newArrayBuffer(size int) buffer {
	b := c.ctx.GenBuffers(1)[0]
	c.ctx.BindBuffer(gl.ARRAY_BUFFER, b)
	c.ctx.BufferData(gl.ARRAY_BUFFER, size, nil, gl.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) newElementArrayBuffer(size int) buffer {
	b := c.ctx.GenBuffers(1)[0]
	c.ctx.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, b)
	c.ctx.BufferData(gl.ELEMENT_ARRAY_BUFFER, size, nil, gl.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) bindArrayBuffer(b buffer) {
	c.ctx.BindBuffer(gl.ARRAY_BUFFER, uint32(b))
}

func (c *context) bindElementArrayBuffer(b buffer) {
	c.ctx.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, uint32(b))
}

func (c *context) arrayBufferSubData(data []float32) {
	s := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*4)
	c.ctx.BufferSubData(gl.ARRAY_BUFFER, 0, s)
}

func (c *context) elementArrayBufferSubData(data []uint16) {
	s := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*2)
	c.ctx.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, 0, s)
}

func (c *context) deleteBuffer(b buffer) {
	c.ctx.DeleteBuffers([]uint32{uint32(b)})
}

func (c *context) drawElements(len int, offsetInBytes int) {
	c.ctx.DrawElements(gl.TRIANGLES, int32(len), gl.UNSIGNED_SHORT, offsetInBytes)
}

func (c *context) maxTextureSizeImpl() int {
	v := make([]int32, 1)
	c.ctx.GetIntegerv(v, gl.MAX_TEXTURE_SIZE)
	return int(v[0])
}

func (c *context) flush() {
	c.ctx.Flush()
}

func (c *context) texSubImage2D(t textureNative, args []*graphicsdriver.WritePixelsArgs) {
	c.bindTexture(t)
	for _, a := range args {
		c.ctx.TexSubImage2D(gl.TEXTURE_2D, 0, int32(a.X), int32(a.Y), int32(a.Width), int32(a.Height), gl.RGBA, gl.UNSIGNED_BYTE, a.Pixels)
	}
}

func (c *context) enableStencilTest() {
	c.ctx.Enable(gl.STENCIL_TEST)
}

func (c *context) disableStencilTest() {
	c.ctx.Disable(gl.STENCIL_TEST)
}

func (c *context) beginStencilWithEvenOddRule() {
	c.ctx.Clear(gl.STENCIL_BUFFER_BIT)
	c.ctx.StencilFunc(gl.ALWAYS, 0x00, 0xff)
	c.ctx.StencilOp(gl.KEEP, gl.KEEP, gl.INVERT)
	c.ctx.ColorMask(false, false, false, false)
}

func (c *context) endStencilWithEvenOddRule() {
	c.ctx.StencilFunc(gl.NOTEQUAL, 0x00, 0xff)
	c.ctx.StencilOp(gl.KEEP, gl.KEEP, gl.KEEP)
	c.ctx.ColorMask(true, true, true, true)
}

func (c *context) isES() bool {
	return c.ctx.IsES()
}
