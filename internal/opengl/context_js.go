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

// +build js

package opengl

import (
	"errors"
	"fmt"

	"github.com/gopherjs/gopherwasm/js"
)

type (
	Texture         js.Value
	Framebuffer     js.Value
	Shader          js.Value
	Buffer          js.Value
	uniformLocation js.Value

	attribLocation int
	programID      int
	Program        struct {
		value js.Value
		id    programID
	}
)

var InvalidTexture = Texture(js.Null())

func getProgramID(p Program) programID {
	return p.id
}

var (
	blend               js.Value
	clampToEdge         js.Value
	colorAttachment0    js.Value
	compileStatus       js.Value
	framebuffer         js.Value
	framebufferBinding  js.Value
	framebufferComplete js.Value
	linkStatus          js.Value
	maxTextureSize      js.Value
	nearest             js.Value
	noError             js.Value
	texture2d           js.Value
	textureMagFilter    js.Value
	textureMinFilter    js.Value
	textureWrapS        js.Value
	textureWrapT        js.Value
	rgba                js.Value
	unpackAlignment     js.Value
	unsignedByte        js.Value
	unsignedShort       js.Value
)

func init() {
	// Accessing the prototype is rquired on Safari.
	c := js.Global().Get("WebGLRenderingContext").Get("prototype")
	VertexShader = ShaderType(c.Get("VERTEX_SHADER").Int())
	FragmentShader = ShaderType(c.Get("FRAGMENT_SHADER").Int())
	ArrayBuffer = BufferType(c.Get("ARRAY_BUFFER").Int())
	ElementArrayBuffer = BufferType(c.Get("ELEMENT_ARRAY_BUFFER").Int())
	DynamicDraw = BufferUsage(c.Get("DYNAMIC_DRAW").Int())
	Triangles = Mode(c.Get("TRIANGLES").Int())
	Lines = Mode(c.Get("LINES").Int())
	Short = DataType(c.Get("SHORT").Int())
	Float = DataType(c.Get("FLOAT").Int())

	zero = operation(c.Get("ZERO").Int())
	one = operation(c.Get("ONE").Int())
	srcAlpha = operation(c.Get("SRC_ALPHA").Int())
	dstAlpha = operation(c.Get("DST_ALPHA").Int())
	oneMinusSrcAlpha = operation(c.Get("ONE_MINUS_SRC_ALPHA").Int())
	oneMinusDstAlpha = operation(c.Get("ONE_MINUS_DST_ALPHA").Int())

	blend = c.Get("BLEND")
	clampToEdge = c.Get("CLAMP_TO_EDGE")
	compileStatus = c.Get("COMPILE_STATUS")
	colorAttachment0 = c.Get("COLOR_ATTACHMENT0")
	framebuffer = c.Get("FRAMEBUFFER")
	framebufferBinding = c.Get("FRAMEBUFFER_BINDING")
	framebufferComplete = c.Get("FRAMEBUFFER_COMPLETE")
	linkStatus = c.Get("LINK_STATUS")
	maxTextureSize = c.Get("MAX_TEXTURE_SIZE")
	nearest = c.Get("NEAREST")
	noError = c.Get("NO_ERROR")
	rgba = c.Get("RGBA")
	texture2d = c.Get("TEXTURE_2D")
	textureMagFilter = c.Get("TEXTURE_MAG_FILTER")
	textureMinFilter = c.Get("TEXTURE_MIN_FILTER")
	textureWrapS = c.Get("TEXTURE_WRAP_S")
	textureWrapT = c.Get("TEXTURE_WRAP_T")
	unpackAlignment = c.Get("UNPACK_ALIGNMENT")
	unsignedByte = c.Get("UNSIGNED_BYTE")
	unsignedShort = c.Get("UNSIGNED_SHORT")
}

type context struct {
	gl            js.Value
	loseContext   js.Value
	lastProgramID programID
}

func Init() error {
	if js.Global().Get("WebGLRenderingContext") == js.Undefined() {
		return fmt.Errorf("opengl: WebGL is not supported")
	}

	// TODO: Define id?
	canvas := js.Global().Get("document").Call("querySelector", "canvas")
	attr := js.Global().Get("Object").New()
	attr.Set("alpha", true)
	attr.Set("premultipliedAlpha", true)
	gl := canvas.Call("getContext", "webgl", attr)
	if gl == js.Null() {
		gl = canvas.Call("getContext", "experimental-webgl", attr)
		if gl == js.Null() {
			return fmt.Errorf("opengl: getContext failed")
		}
	}
	c := &Context{}
	c.gl = gl

	// Getting an extension might fail after the context is lost, so
	// it is required to get the extension here.
	c.loseContext = gl.Call("getExtension", "WEBGL_lose_context")
	if c.loseContext != js.Null() {
		// This testing function name is temporary.
		js.Global().Set("_ebiten_loseContextForTesting", js.NewCallback(func([]js.Value) {
			c.loseContext.Call("loseContext")
		}))
	}
	theContext = c
	return nil
}

func (c *Context) Reset() error {
	c.locationCache = newLocationCache()
	c.lastTexture = Texture(js.Null())
	c.lastFramebuffer = Framebuffer(js.Null())
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = CompositeModeUnknown
	gl := c.gl
	gl.Call("enable", blend)
	c.BlendFunc(CompositeModeSourceOver)
	f := gl.Call("getParameter", framebufferBinding)
	c.screenFramebuffer = Framebuffer(f)
	return nil
}

func (c *Context) BlendFunc(mode CompositeMode) {
	if c.lastCompositeMode == mode {
		return
	}
	c.lastCompositeMode = mode
	s, d := operations(mode)
	gl := c.gl
	gl.Call("blendFunc", int(s), int(d))
}

func (c *Context) NewTexture(width, height int) (Texture, error) {
	gl := c.gl
	t := gl.Call("createTexture")
	if t == js.Null() {
		return Texture(js.Null()), errors.New("opengl: glGenTexture failed")
	}
	gl.Call("pixelStorei", unpackAlignment, 4)
	c.BindTexture(Texture(t))

	gl.Call("texParameteri", texture2d, textureMagFilter, nearest)
	gl.Call("texParameteri", texture2d, textureMinFilter, nearest)
	gl.Call("texParameteri", texture2d, textureWrapS, clampToEdge)
	gl.Call("texParameteri", texture2d, textureWrapT, clampToEdge)

	// void texImage2D(GLenum target, GLint level, GLenum internalformat,
	//     GLsizei width, GLsizei height, GLint border, GLenum format,
	//     GLenum type, ArrayBufferView? pixels);
	gl.Call("texImage2D", texture2d, 0, rgba, width, height, 0, rgba, unsignedByte, nil)

	return Texture(t), nil
}

func (c *Context) bindFramebufferImpl(f Framebuffer) {
	gl := c.gl
	gl.Call("bindFramebuffer", framebuffer, js.Value(f))
}

func (c *Context) FramebufferPixels(f Framebuffer, width, height int) ([]byte, error) {
	gl := c.gl

	c.bindFramebuffer(f)

	pixels := make([]byte, 4*width*height)
	gl.Call("readPixels", 0, 0, width, height, rgba, unsignedByte, pixels)
	if e := gl.Call("getError"); e.Int() != noError.Int() {
		return nil, errors.New(fmt.Sprintf("opengl: error: %d", e))
	}
	return pixels, nil
}

func (c *Context) bindTextureImpl(t Texture) {
	gl := c.gl
	gl.Call("bindTexture", texture2d, js.Value(t))
}

func (c *Context) DeleteTexture(t Texture) {
	gl := c.gl
	if !gl.Call("isTexture", js.Value(t)).Bool() {
		return
	}
	if c.lastTexture == t {
		c.lastTexture = Texture(js.Null())
	}
	gl.Call("deleteTexture", js.Value(t))
}

func (c *Context) IsTexture(t Texture) bool {
	gl := c.gl
	return gl.Call("isTexture", js.Value(t)).Bool()
}

func (c *Context) TexSubImage2D(p []byte, x, y, width, height int) {
	gl := c.gl
	// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
	//                    GLsizei width, GLsizei height,
	//                    GLenum format, GLenum type, ArrayBufferView? pixels);
	gl.Call("texSubImage2D", texture2d, 0, x, y, width, height, rgba, unsignedByte, p)
}

func (c *Context) NewFramebuffer(t Texture) (Framebuffer, error) {
	gl := c.gl
	f := gl.Call("createFramebuffer")
	c.bindFramebuffer(Framebuffer(f))

	gl.Call("framebufferTexture2D", framebuffer, colorAttachment0, texture2d, js.Value(t), 0)
	if s := gl.Call("checkFramebufferStatus", framebuffer); s.Int() != framebufferComplete.Int() {
		return Framebuffer(js.Null()), errors.New(fmt.Sprintf("opengl: creating framebuffer failed: %d", s.Int()))
	}

	return Framebuffer(f), nil
}

func (c *Context) setViewportImpl(width, height int) {
	gl := c.gl
	gl.Call("viewport", 0, 0, width, height)
}

func (c *Context) DeleteFramebuffer(f Framebuffer) {
	gl := c.gl
	if !gl.Call("isFramebuffer", js.Value(f)).Bool() {
		return
	}
	// If a framebuffer to be deleted is bound, a newly bound framebuffer
	// will be a default framebuffer.
	// https://www.khronos.org/opengles/sdk/docs/man/xhtml/glDeleteFramebuffers.xml
	if c.lastFramebuffer == f {
		c.lastFramebuffer = Framebuffer(js.Null())
		c.lastViewportWidth = 0
		c.lastViewportHeight = 0
	}
	gl.Call("deleteFramebuffer", js.Value(f))
}

func (c *Context) NewShader(shaderType ShaderType, source string) (Shader, error) {
	gl := c.gl
	s := gl.Call("createShader", int(shaderType))
	if s == js.Null() {
		return Shader(js.Null()), fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
	}

	gl.Call("shaderSource", js.Value(s), source)
	gl.Call("compileShader", js.Value(s))

	if !gl.Call("getShaderParameter", js.Value(s), compileStatus).Bool() {
		log := gl.Call("getShaderInfoLog", js.Value(s))
		return Shader(js.Null()), fmt.Errorf("opengl: shader compile failed: %s", log)
	}
	return Shader(s), nil
}

func (c *Context) DeleteShader(s Shader) {
	gl := c.gl
	gl.Call("deleteShader", js.Value(s))
}

func (c *Context) NewProgram(shaders []Shader) (Program, error) {
	gl := c.gl
	v := gl.Call("createProgram")
	if v == js.Null() {
		return Program{}, errors.New("opengl: glCreateProgram failed")
	}

	for _, shader := range shaders {
		gl.Call("attachShader", v, js.Value(shader))
	}
	gl.Call("linkProgram", v)
	if !gl.Call("getProgramParameter", v, linkStatus).Bool() {
		return Program{}, errors.New("opengl: program error")
	}

	id := c.lastProgramID
	c.lastProgramID++
	return Program{
		value: v,
		id:    id,
	}, nil
}

func (c *Context) UseProgram(p Program) {
	gl := c.gl
	gl.Call("useProgram", p.value)
}

func (c *Context) DeleteProgram(p Program) {
	gl := c.gl
	if !gl.Call("isProgram", p).Bool() {
		return
	}
	gl.Call("deleteProgram", p)
}

func (c *Context) getUniformLocationImpl(p Program, location string) uniformLocation {
	gl := c.gl
	return uniformLocation(gl.Call("getUniformLocation", p.value, location))
}

func (c *Context) UniformInt(p Program, location string, v int) {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	gl.Call("uniform1i", js.Value(l), v)
}

func (c *Context) UniformFloat(p Program, location string, v float32) {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	gl.Call("uniform1f", js.Value(l), v)
}

var (
	float32Array = js.Global().Get("Float32Array")
)

func (c *Context) UniformFloats(p Program, location string, v []float32) {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	switch len(v) {
	case 2:
		gl.Call("uniform2f", js.Value(l), v[0], v[1])
	case 4:
		gl.Call("uniform4f", js.Value(l), v[0], v[1], v[2], v[3])
	case 16:
		gl.Call("uniformMatrix4fv", js.Value(l), false, js.ValueOf(v))
	default:
		panic("not reached")
	}
}

func (c *Context) getAttribLocationImpl(p Program, location string) attribLocation {
	gl := c.gl
	return attribLocation(gl.Call("getAttribLocation", p.value, location).Int())
}

func (c *Context) VertexAttribPointer(p Program, location string, size int, dataType DataType, stride int, offset int) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.Call("vertexAttribPointer", int(l), size, int(dataType), false, stride, offset)
}

func (c *Context) EnableVertexAttribArray(p Program, location string) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.Call("enableVertexAttribArray", int(l))
}

func (c *Context) DisableVertexAttribArray(p Program, location string) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.Call("disableVertexAttribArray", int(l))
}

func (c *Context) NewArrayBuffer(size int) Buffer {
	gl := c.gl
	b := gl.Call("createBuffer")
	gl.Call("bindBuffer", int(ArrayBuffer), js.Value(b))
	gl.Call("bufferData", int(ArrayBuffer), size, int(DynamicDraw))
	return Buffer(b)
}

func (c *Context) NewElementArrayBuffer(size int) Buffer {
	gl := c.gl
	b := gl.Call("createBuffer")
	gl.Call("bindBuffer", int(ElementArrayBuffer), js.Value(b))
	gl.Call("bufferData", int(ElementArrayBuffer), size, int(DynamicDraw))
	return Buffer(b)
}

func (c *Context) BindBuffer(bufferType BufferType, b Buffer) {
	gl := c.gl
	gl.Call("bindBuffer", int(bufferType), js.Value(b))
}

func (c *Context) ArrayBufferSubData(data []float32) {
	gl := c.gl
	gl.Call("bufferSubData", int(ArrayBuffer), 0, js.ValueOf(data))
}

func (c *Context) ElementArrayBufferSubData(data []uint16) {
	gl := c.gl
	gl.Call("bufferSubData", int(ElementArrayBuffer), 0, js.ValueOf(data))
}

func (c *Context) DeleteBuffer(b Buffer) {
	gl := c.gl
	gl.Call("deleteBuffer", js.Value(b))
}

func (c *Context) DrawElements(mode Mode, len int, offsetInBytes int) {
	gl := c.gl
	gl.Call("drawElements", int(mode), len, unsignedShort, offsetInBytes)
}

func (c *Context) maxTextureSizeImpl() int {
	gl := c.gl
	return gl.Call("getParameter", maxTextureSize).Int()
}

func (c *Context) Flush() {
	gl := c.gl
	gl.Call("flush")
}

func (c *Context) IsContextLost() bool {
	gl := c.gl
	return gl.Call("isContextLost").Bool()
}

func (c *Context) RestoreContext() {
	if c.loseContext != js.Null() {
		c.loseContext.Call("restoreContext")
	}
}
