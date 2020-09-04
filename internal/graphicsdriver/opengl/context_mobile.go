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

// +build android ios

package opengl

import (
	"errors"
	"fmt"

	mgl "golang.org/x/mobile/gl"

	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

type (
	textureNative     mgl.Texture
	framebufferNative mgl.Framebuffer
	shader            mgl.Shader
	program           mgl.Program
	buffer            mgl.Buffer
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

func (p program) equal(rhs program) bool {
	return p == rhs
}

var InvalidTexture textureNative

type (
	uniformLocation mgl.Uniform
	attribLocation  mgl.Attrib
)

func (u uniformLocation) equal(rhs uniformLocation) bool {
	return u == rhs
}

type programID uint32

var (
	invalidTexture     = textureNative(mgl.Texture{})
	invalidFramebuffer = framebufferNative(mgl.Framebuffer{Value: (1 << 32) - 1})
	invalidUniform     = uniformLocation(mgl.Uniform{Value: -1})
)

func getProgramID(p program) programID {
	return programID(p.Value)
}

const (
	vertexShader       = shaderType(mgl.VERTEX_SHADER)
	fragmentShader     = shaderType(mgl.FRAGMENT_SHADER)
	arrayBuffer        = bufferType(mgl.ARRAY_BUFFER)
	elementArrayBuffer = bufferType(mgl.ELEMENT_ARRAY_BUFFER)
	dynamicDraw        = bufferUsage(mgl.DYNAMIC_DRAW)
	short              = dataType(mgl.SHORT)
	float              = dataType(mgl.FLOAT)

	zero             = operation(mgl.ZERO)
	one              = operation(mgl.ONE)
	srcAlpha         = operation(mgl.SRC_ALPHA)
	dstAlpha         = operation(mgl.DST_ALPHA)
	oneMinusSrcAlpha = operation(mgl.ONE_MINUS_SRC_ALPHA)
	oneMinusDstAlpha = operation(mgl.ONE_MINUS_DST_ALPHA)
	dstColor         = operation(mgl.DST_COLOR)
)

type contextImpl struct {
	gl mgl.Context
}

func (c *context) reset() error {
	c.locationCache = newLocationCache()
	c.lastTexture = invalidTexture
	c.lastFramebuffer = invalidFramebuffer
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = driver.CompositeModeUnknown
	c.gl.Enable(mgl.BLEND)
	c.blendFunc(driver.CompositeModeSourceOver)
	f := c.gl.GetInteger(mgl.FRAMEBUFFER_BINDING)
	c.screenFramebuffer = framebufferNative(mgl.Framebuffer{uint32(f)})
	// TODO: Need to update screenFramebufferWidth/Height?
	return nil
}

func (c *context) blendFunc(mode driver.CompositeMode) {
	gl := c.gl
	if c.lastCompositeMode == mode {
		return
	}
	c.lastCompositeMode = mode
	s, d := mode.Operations()
	s2, d2 := convertOperation(s), convertOperation(d)
	gl.BlendFunc(mgl.Enum(s2), mgl.Enum(d2))
}

func (c *context) newTexture(width, height int) (textureNative, error) {
	gl := c.gl
	t := gl.CreateTexture()
	if t.Value <= 0 {
		return textureNative{}, errors.New("opengl: creating texture failed")
	}
	gl.PixelStorei(mgl.UNPACK_ALIGNMENT, 4)
	c.bindTexture(textureNative(t))

	gl.TexParameteri(mgl.TEXTURE_2D, mgl.TEXTURE_MAG_FILTER, mgl.NEAREST)
	gl.TexParameteri(mgl.TEXTURE_2D, mgl.TEXTURE_MIN_FILTER, mgl.NEAREST)
	gl.TexParameteri(mgl.TEXTURE_2D, mgl.TEXTURE_WRAP_S, mgl.CLAMP_TO_EDGE)
	gl.TexParameteri(mgl.TEXTURE_2D, mgl.TEXTURE_WRAP_T, mgl.CLAMP_TO_EDGE)
	gl.TexImage2D(mgl.TEXTURE_2D, 0, mgl.RGBA, width, height, mgl.RGBA, mgl.UNSIGNED_BYTE, nil)

	return textureNative(t), nil
}

func (c *context) bindFramebufferImpl(f framebufferNative) {
	gl := c.gl
	gl.BindFramebuffer(mgl.FRAMEBUFFER, mgl.Framebuffer(f))
}

func (c *context) framebufferPixels(f *framebuffer, width, height int) ([]byte, error) {
	gl := c.gl
	gl.Flush()

	c.bindFramebuffer(f.native)

	pixels := make([]byte, 4*width*height)
	gl.ReadPixels(pixels, 0, 0, width, height, mgl.RGBA, mgl.UNSIGNED_BYTE)
	return pixels, nil
}

func (c *context) activeTexture(idx int) {
	gl := c.gl
	gl.ActiveTexture(mgl.Enum(mgl.TEXTURE0 + idx))
}

func (c *context) bindTextureImpl(t textureNative) {
	gl := c.gl
	gl.BindTexture(mgl.TEXTURE_2D, mgl.Texture(t))
}

func (c *context) deleteTexture(t textureNative) {
	gl := c.gl
	if !gl.IsTexture(mgl.Texture(t)) {
		return
	}
	if c.lastTexture == t {
		c.lastTexture = invalidTexture
	}
	gl.DeleteTexture(mgl.Texture(t))
}

func (c *context) isTexture(t textureNative) bool {
	gl := c.gl
	return gl.IsTexture(mgl.Texture(t))
}

func (c *context) newFramebuffer(texture textureNative) (framebufferNative, error) {
	gl := c.gl
	f := gl.CreateFramebuffer()
	if f.Value <= 0 {
		return framebufferNative{}, errors.New("opengl: creating framebuffer failed: gl.IsFramebuffer returns false")
	}
	c.bindFramebuffer(framebufferNative(f))

	gl.FramebufferTexture2D(mgl.FRAMEBUFFER, mgl.COLOR_ATTACHMENT0, mgl.TEXTURE_2D, mgl.Texture(texture), 0)
	s := gl.CheckFramebufferStatus(mgl.FRAMEBUFFER)
	if s != mgl.FRAMEBUFFER_COMPLETE {
		if s != 0 {
			return framebufferNative{}, fmt.Errorf("opengl: creating framebuffer failed: %v", s)
		}
		if e := gl.GetError(); e != mgl.NO_ERROR {
			return framebufferNative{}, fmt.Errorf("opengl: creating framebuffer failed: (glGetError) %d", e)
		}
		return framebufferNative{}, fmt.Errorf("opengl: creating framebuffer failed: unknown error")
	}
	return framebufferNative(f), nil
}

func (c *context) setViewportImpl(width, height int) {
	gl := c.gl
	gl.Viewport(0, 0, width, height)
}

func (c *context) deleteFramebuffer(f framebufferNative) {
	gl := c.gl
	if !gl.IsFramebuffer(mgl.Framebuffer(f)) {
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
	gl.DeleteFramebuffer(mgl.Framebuffer(f))
}

func (c *context) newShader(shaderType shaderType, source string) (shader, error) {
	gl := c.gl
	s := gl.CreateShader(mgl.Enum(shaderType))
	if s.Value == 0 {
		return shader{}, fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
	}
	gl.ShaderSource(s, source)
	gl.CompileShader(s)

	v := gl.GetShaderi(s, mgl.COMPILE_STATUS)
	if v == mgl.FALSE {
		log := gl.GetShaderInfoLog(s)
		return shader{}, fmt.Errorf("opengl: shader compile failed: %s", log)
	}
	return shader(s), nil
}

func (c *context) deleteShader(s shader) {
	gl := c.gl
	gl.DeleteShader(mgl.Shader(s))
}

func (c *context) newProgram(shaders []shader, attributes []string) (program, error) {
	gl := c.gl
	p := gl.CreateProgram()
	if p.Value == 0 {
		return program{}, errors.New("opengl: glCreateProgram failed")
	}

	for _, shader := range shaders {
		gl.AttachShader(p, mgl.Shader(shader))
	}

	for i, name := range attributes {
		gl.BindAttribLocation(p, mgl.Attrib{Value: uint(i)}, name)
	}

	gl.LinkProgram(p)
	v := gl.GetProgrami(p, mgl.LINK_STATUS)
	if v == mgl.FALSE {
		info := gl.GetProgramInfoLog(p)
		return program{}, fmt.Errorf("opengl: program error: %s", info)
	}
	return program(p), nil
}

func (c *context) useProgram(p program) {
	gl := c.gl
	gl.UseProgram(mgl.Program(p))
}

func (c *context) deleteProgram(p program) {
	gl := c.gl
	if !gl.IsProgram(mgl.Program(p)) {
		return
	}
	gl.DeleteProgram(mgl.Program(p))
}

func (c *context) getUniformLocationImpl(p program, location string) uniformLocation {
	gl := c.gl
	u := uniformLocation(gl.GetUniformLocation(mgl.Program(p), location))
	return u
}

func (c *context) uniformInt(p program, location string, v int) bool {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l == invalidUniform {
		return false
	}
	gl.Uniform1i(mgl.Uniform(l), v)
	return true
}

func (c *context) uniformFloat(p program, location string, v float32) bool {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l == invalidUniform {
		return false
	}
	gl.Uniform1f(mgl.Uniform(l), v)
	return true
}

func (c *context) uniformFloats(p program, location string, v []float32, typ shaderir.Type) bool {
	gl := c.gl
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
		gl.Uniform1fv(mgl.Uniform(l), v)
	case shaderir.Vec2:
		gl.Uniform2fv(mgl.Uniform(l), v)
	case shaderir.Vec3:
		gl.Uniform3fv(mgl.Uniform(l), v)
	case shaderir.Vec4:
		gl.Uniform4fv(mgl.Uniform(l), v)
	case shaderir.Mat2:
		gl.UniformMatrix2fv(mgl.Uniform(l), v)
	case shaderir.Mat3:
		gl.UniformMatrix3fv(mgl.Uniform(l), v)
	case shaderir.Mat4:
		gl.UniformMatrix4fv(mgl.Uniform(l), v)
	default:
		panic(fmt.Sprintf("opengl: unexpected type: %s", typ.String()))
	}
	return true
}

func (c *context) vertexAttribPointer(p program, index int, size int, dataType dataType, stride int, offset int) {
	gl := c.gl
	gl.VertexAttribPointer(mgl.Attrib{Value: uint(index)}, size, mgl.Enum(dataType), false, stride, offset)
}

func (c *context) enableVertexAttribArray(p program, index int) {
	gl := c.gl
	gl.EnableVertexAttribArray(mgl.Attrib{Value: uint(index)})
}

func (c *context) disableVertexAttribArray(p program, index int) {
	gl := c.gl
	gl.DisableVertexAttribArray(mgl.Attrib{Value: uint(index)})
}

func (c *context) newArrayBuffer(size int) buffer {
	gl := c.gl
	b := gl.CreateBuffer()
	gl.BindBuffer(mgl.Enum(arrayBuffer), b)
	gl.BufferInit(mgl.Enum(arrayBuffer), size, mgl.Enum(dynamicDraw))
	return buffer(b)
}

func (c *context) newElementArrayBuffer(size int) buffer {
	gl := c.gl
	b := gl.CreateBuffer()
	gl.BindBuffer(mgl.Enum(elementArrayBuffer), b)
	gl.BufferInit(mgl.Enum(elementArrayBuffer), size, mgl.Enum(dynamicDraw))
	return buffer(b)
}

func (c *context) bindBuffer(bufferType bufferType, b buffer) {
	gl := c.gl
	gl.BindBuffer(mgl.Enum(bufferType), mgl.Buffer(b))
}

func (c *context) arrayBufferSubData(data []float32) {
	gl := c.gl
	gl.BufferSubData(mgl.Enum(arrayBuffer), 0, float32sToBytes(data))
}

func (c *context) elementArrayBufferSubData(data []uint16) {
	gl := c.gl
	gl.BufferSubData(mgl.Enum(elementArrayBuffer), 0, uint16sToBytes(data))
}

func (c *context) deleteBuffer(b buffer) {
	gl := c.gl
	gl.DeleteBuffer(mgl.Buffer(b))
}

func (c *context) drawElements(len int, offsetInBytes int) {
	gl := c.gl
	gl.DrawElements(mgl.TRIANGLES, len, mgl.UNSIGNED_SHORT, offsetInBytes)
}

func (c *context) maxTextureSizeImpl() int {
	gl := c.gl
	return gl.GetInteger(mgl.MAX_TEXTURE_SIZE)
}

func (c *context) getShaderPrecisionFormatPrecision() int {
	gl := c.gl
	_, _, p := gl.GetShaderPrecisionFormat(mgl.FRAGMENT_SHADER, mgl.HIGH_FLOAT)
	return p
}

func (c *context) flush() {
	gl := c.gl
	gl.Flush()
}

func (c *context) needsRestoring() bool {
	return true
}

func (c *context) canUsePBO() bool {
	// On Android, using PBO might slow the applications, especially when coming back from the context lost.
	// Let's not use PBO until we find a good solution.
	return false
}

func (c *context) texSubImage2D(t textureNative, width, height int, args []*driver.ReplacePixelsArgs) {
	c.bindTexture(t)
	gl := c.gl
	for _, a := range args {
		gl.TexSubImage2D(mgl.TEXTURE_2D, 0, a.X, a.Y, a.Width, a.Height, mgl.RGBA, mgl.UNSIGNED_BYTE, a.Pixels)
	}
}

func (c *context) newPixelBufferObject(width, height int) buffer {
	gl := c.gl
	b := gl.CreateBuffer()
	gl.BindBuffer(mgl.PIXEL_UNPACK_BUFFER, b)
	gl.BufferInit(mgl.PIXEL_UNPACK_BUFFER, 4*width*height, mgl.STREAM_DRAW)
	gl.BindBuffer(mgl.PIXEL_UNPACK_BUFFER, mgl.Buffer{0})
	return buffer(b)
}

func (c *context) replacePixelsWithPBO(buffer buffer, t textureNative, width, height int, args []*driver.ReplacePixelsArgs) {
	// This implementation is not used yet so far. See the comment at canUsePBO.

	c.bindTexture(t)
	gl := c.gl
	gl.BindBuffer(mgl.PIXEL_UNPACK_BUFFER, mgl.Buffer(buffer))

	stride := 4 * width
	for _, a := range args {
		offset := 4 * (a.Y*width + a.X)
		for j := 0; j < a.Height; j++ {
			gl.BufferSubData(mgl.PIXEL_UNPACK_BUFFER, offset+stride*j, a.Pixels[4*a.Width*j:4*a.Width*(j+1)])
		}
	}

	gl.TexSubImage2D(mgl.TEXTURE_2D, 0, 0, 0, width, height, mgl.RGBA, mgl.UNSIGNED_BYTE, nil)
	gl.BindBuffer(mgl.PIXEL_UNPACK_BUFFER, mgl.Buffer{0})
}
