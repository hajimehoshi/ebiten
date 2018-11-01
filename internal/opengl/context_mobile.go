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

	"github.com/hajimehoshi/ebiten/internal/graphics"
)

type (
	Texture     mgl.Texture
	Framebuffer mgl.Framebuffer
	shader      mgl.Shader
	program     mgl.Program
	buffer      mgl.Buffer
)

var InvalidTexture Texture

type (
	uniformLocation mgl.Uniform
	attribLocation  mgl.Attrib
)

type programID uint32

var (
	invalidTexture     = Texture(mgl.Texture{})
	invalidFramebuffer = Framebuffer(mgl.Framebuffer{(1 << 32) - 1})
)

func getProgramID(p program) programID {
	return programID(p.Value)
}

func init() {
	vertexShader = mgl.VERTEX_SHADER
	fragmentShader = mgl.FRAGMENT_SHADER
	arrayBuffer = mgl.ARRAY_BUFFER
	elementArrayBuffer = mgl.ELEMENT_ARRAY_BUFFER
	dynamicDraw = mgl.DYNAMIC_DRAW
	Short = mgl.SHORT
	Float = mgl.FLOAT

	zero = mgl.ZERO
	one = mgl.ONE
	srcAlpha = mgl.SRC_ALPHA
	dstAlpha = mgl.DST_ALPHA
	oneMinusSrcAlpha = mgl.ONE_MINUS_SRC_ALPHA
	oneMinusDstAlpha = mgl.ONE_MINUS_DST_ALPHA

	initializeArrayBuferLayout()
}

type context struct {
	gl     mgl.Context
	worker mgl.Worker
}

func Init() {
	c := &Context{}
	c.gl, c.worker = mgl.NewContext()
	theContext = c
}

func InitWithContext(context mgl.Context) {
	c := &Context{}
	c.gl = context
	theContext = c
}

func (c *Context) DoWork(chError <-chan error, chDone <-chan struct{}) error {
	if c.worker == nil {
		panic("not reached")
	}
	// TODO: Check this is called on the rendering thread
	workAvailable := c.worker.WorkAvailable()
loop:
	for {
		select {
		case err := <-chError:
			return err
		case <-workAvailable:
			c.worker.DoWork()
		case <-chDone:
			break loop
		}
	}
	return nil
}

func (c *Context) reset() error {
	c.locationCache = newLocationCache()
	c.lastTexture = invalidTexture
	c.lastFramebuffer = invalidFramebuffer
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = graphics.CompositeModeUnknown
	c.gl.Enable(mgl.BLEND)
	c.BlendFunc(graphics.CompositeModeSourceOver)
	f := c.gl.GetInteger(mgl.FRAMEBUFFER_BINDING)
	c.screenFramebuffer = Framebuffer(mgl.Framebuffer{uint32(f)})
	// TODO: Need to update screenFramebufferWidth/Height?
	return nil
}

func (c *Context) BlendFunc(mode graphics.CompositeMode) {
	gl := c.gl
	if c.lastCompositeMode == mode {
		return
	}
	c.lastCompositeMode = mode
	s, d := mode.Operations()
	s2, d2 := convertOperation(s), convertOperation(d)
	gl.BlendFunc(mgl.Enum(s2), mgl.Enum(d2))
}

func (c *Context) NewTexture(width, height int) (Texture, error) {
	gl := c.gl
	t := gl.CreateTexture()
	if t.Value <= 0 {
		return Texture{}, errors.New("opengl: creating texture failed")
	}
	gl.PixelStorei(mgl.UNPACK_ALIGNMENT, 4)
	c.bindTexture(Texture(t))

	gl.TexParameteri(mgl.TEXTURE_2D, mgl.TEXTURE_MAG_FILTER, mgl.NEAREST)
	gl.TexParameteri(mgl.TEXTURE_2D, mgl.TEXTURE_MIN_FILTER, mgl.NEAREST)
	gl.TexParameteri(mgl.TEXTURE_2D, mgl.TEXTURE_WRAP_S, mgl.CLAMP_TO_EDGE)
	gl.TexParameteri(mgl.TEXTURE_2D, mgl.TEXTURE_WRAP_T, mgl.CLAMP_TO_EDGE)
	gl.TexImage2D(mgl.TEXTURE_2D, 0, mgl.RGBA, width, height, mgl.RGBA, mgl.UNSIGNED_BYTE, nil)

	return Texture(t), nil
}

func (c *Context) bindFramebufferImpl(f Framebuffer) {
	gl := c.gl
	gl.BindFramebuffer(mgl.FRAMEBUFFER, mgl.Framebuffer(f))
}

func (c *Context) FramebufferPixels(f Framebuffer, width, height int) ([]byte, error) {
	gl := c.gl
	gl.Flush()

	c.bindFramebuffer(f)

	pixels := make([]byte, 4*width*height)
	gl.ReadPixels(pixels, 0, 0, width, height, mgl.RGBA, mgl.UNSIGNED_BYTE)
	if e := gl.GetError(); e != mgl.NO_ERROR {
		return nil, fmt.Errorf("opengl: glReadPixels: %d", e)
	}
	return pixels, nil
}

func (c *Context) bindTextureImpl(t Texture) {
	gl := c.gl
	gl.BindTexture(mgl.TEXTURE_2D, mgl.Texture(t))
}

func (c *Context) DeleteTexture(t Texture) {
	gl := c.gl
	if !gl.IsTexture(mgl.Texture(t)) {
		return
	}
	if c.lastTexture == t {
		c.lastTexture = invalidTexture
	}
	gl.DeleteTexture(mgl.Texture(t))
}

func (c *Context) IsTexture(t Texture) bool {
	gl := c.gl
	return gl.IsTexture(mgl.Texture(t))
}

func (c *Context) TexSubImage2D(t Texture, p []byte, x, y, width, height int) {
	c.bindTexture(t)
	gl := c.gl
	gl.TexSubImage2D(mgl.TEXTURE_2D, 0, x, y, width, height, mgl.RGBA, mgl.UNSIGNED_BYTE, p)
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

func (c *Context) setViewportImpl(width, height int) {
	gl := c.gl
	gl.Viewport(0, 0, width, height)
}

func (c *Context) DeleteFramebuffer(f Framebuffer) {
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

func (c *Context) newShader(shaderType shaderType, source string) (shader, error) {
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

func (c *Context) deleteShader(s shader) {
	gl := c.gl
	gl.DeleteShader(mgl.Shader(s))
}

func (c *Context) newProgram(shaders []shader) (program, error) {
	gl := c.gl
	p := gl.CreateProgram()
	if p.Value == 0 {
		return program{}, errors.New("opengl: glCreateProgram failed")
	}

	for _, shader := range shaders {
		gl.AttachShader(p, mgl.Shader(shader))
	}
	gl.LinkProgram(p)
	v := gl.GetProgrami(p, mgl.LINK_STATUS)
	if v == mgl.FALSE {
		return program{}, errors.New("opengl: program error")
	}
	return program(p), nil
}

func (c *Context) useProgram(p program) {
	gl := c.gl
	gl.UseProgram(mgl.Program(p))
}

func (c *Context) deleteProgram(p program) {
	gl := c.gl
	if !gl.IsProgram(mgl.Program(p)) {
		return
	}
	gl.DeleteProgram(mgl.Program(p))
}

func (c *Context) getUniformLocationImpl(p program, location string) uniformLocation {
	gl := c.gl
	u := uniformLocation(gl.GetUniformLocation(mgl.Program(p), location))
	if u.Value == -1 {
		panic("invalid uniform location: " + location)
	}
	return u
}

func (c *Context) uniformInt(p program, location string, v int) {
	gl := c.gl
	gl.Uniform1i(mgl.Uniform(c.locationCache.GetUniformLocation(c, p, location)), v)
}

func (c *Context) uniformFloat(p program, location string, v float32) {
	gl := c.gl
	gl.Uniform1f(mgl.Uniform(c.locationCache.GetUniformLocation(c, p, location)), v)
}

func (c *Context) uniformFloats(p program, location string, v []float32) {
	gl := c.gl
	l := mgl.Uniform(c.locationCache.GetUniformLocation(c, p, location))
	switch len(v) {
	case 2:
		gl.Uniform2fv(l, v)
	case 4:
		gl.Uniform4fv(l, v)
	case 16:
		gl.UniformMatrix4fv(l, v)
	default:
		panic("not reached")
	}
}

func (c *Context) getAttribLocationImpl(p program, location string) attribLocation {
	gl := c.gl
	a := attribLocation(gl.GetAttribLocation(mgl.Program(p), location))
	if a.Value == ^uint(0) {
		panic("invalid attrib location: " + location)
	}
	return a
}

func (c *Context) vertexAttribPointer(p program, location string, size int, dataType DataType, stride int, offset int) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.VertexAttribPointer(mgl.Attrib(l), size, mgl.Enum(dataType), false, stride, offset)
}

func (c *Context) enableVertexAttribArray(p program, location string) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.EnableVertexAttribArray(mgl.Attrib(l))
}

func (c *Context) disableVertexAttribArray(p program, location string) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.DisableVertexAttribArray(mgl.Attrib(l))
}

func (c *Context) newArrayBuffer(size int) buffer {
	gl := c.gl
	b := gl.CreateBuffer()
	gl.BindBuffer(mgl.Enum(arrayBuffer), b)
	gl.BufferInit(mgl.Enum(arrayBuffer), size, mgl.Enum(dynamicDraw))
	return buffer(b)
}

func (c *Context) newElementArrayBuffer(size int) buffer {
	gl := c.gl
	b := gl.CreateBuffer()
	gl.BindBuffer(mgl.Enum(elementArrayBuffer), b)
	gl.BufferInit(mgl.Enum(elementArrayBuffer), size, mgl.Enum(dynamicDraw))
	return buffer(b)
}

func (c *Context) bindBuffer(bufferType bufferType, b buffer) {
	gl := c.gl
	gl.BindBuffer(mgl.Enum(bufferType), mgl.Buffer(b))
}

func (c *Context) arrayBufferSubData(data []float32) {
	gl := c.gl
	gl.BufferSubData(mgl.Enum(arrayBuffer), 0, float32sToBytes(data))
}

func (c *Context) elementArrayBufferSubData(data []uint16) {
	gl := c.gl
	gl.BufferSubData(mgl.Enum(elementArrayBuffer), 0, uint16sToBytes(data))
}

func (c *Context) deleteBuffer(b buffer) {
	gl := c.gl
	gl.DeleteBuffer(mgl.Buffer(b))
}

func (c *Context) DrawElements(len int, offsetInBytes int) {
	gl := c.gl
	gl.DrawElements(mgl.TRIANGLES, len, mgl.UNSIGNED_SHORT, offsetInBytes)
}

func (c *Context) maxTextureSizeImpl() int {
	gl := c.gl
	return gl.GetInteger(mgl.MAX_TEXTURE_SIZE)
}

func (c *Context) Flush() {
	gl := c.gl
	gl.Flush()
}
