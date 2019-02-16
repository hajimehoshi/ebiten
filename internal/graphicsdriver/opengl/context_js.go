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

	"github.com/hajimehoshi/ebiten/internal/graphics"
)

type (
	textureNative     js.Value
	framebufferNative js.Value
	shader            js.Value
	buffer            js.Value
	uniformLocation   js.Value

	attribLocation int
	programID      int
	program        struct {
		value js.Value
		id    programID
	}
)

var InvalidTexture = textureNative(js.Null())

func getProgramID(p program) programID {
	return p.id
}

var (
	// Accessing the prototype is rquired on Safari.
	contextPrototype = js.Global().Get("WebGLRenderingContext").Get("prototype")

	vertexShader       = shaderType(contextPrototype.Get("VERTEX_SHADER").Int())
	fragmentShader     = shaderType(contextPrototype.Get("FRAGMENT_SHADER").Int())
	arrayBuffer        = bufferType(contextPrototype.Get("ARRAY_BUFFER").Int())
	elementArrayBuffer = bufferType(contextPrototype.Get("ELEMENT_ARRAY_BUFFER").Int())
	dynamicDraw        = bufferUsage(contextPrototype.Get("DYNAMIC_DRAW").Int())
	short              = dataType(contextPrototype.Get("SHORT").Int())
	float              = dataType(contextPrototype.Get("FLOAT").Int())

	zero             = operation(contextPrototype.Get("ZERO").Int())
	one              = operation(contextPrototype.Get("ONE").Int())
	srcAlpha         = operation(contextPrototype.Get("SRC_ALPHA").Int())
	dstAlpha         = operation(contextPrototype.Get("DST_ALPHA").Int())
	oneMinusSrcAlpha = operation(contextPrototype.Get("ONE_MINUS_SRC_ALPHA").Int())
	oneMinusDstAlpha = operation(contextPrototype.Get("ONE_MINUS_DST_ALPHA").Int())

	blend               = contextPrototype.Get("BLEND")
	clampToEdge         = contextPrototype.Get("CLAMP_TO_EDGE")
	compileStatus       = contextPrototype.Get("COMPILE_STATUS")
	colorAttachment0    = contextPrototype.Get("COLOR_ATTACHMENT0")
	framebuffer_        = contextPrototype.Get("FRAMEBUFFER")
	framebufferBinding  = contextPrototype.Get("FRAMEBUFFER_BINDING")
	framebufferComplete = contextPrototype.Get("FRAMEBUFFER_COMPLETE")
	linkStatus          = contextPrototype.Get("LINK_STATUS")
	maxTextureSize      = contextPrototype.Get("MAX_TEXTURE_SIZE")
	nearest             = contextPrototype.Get("NEAREST")
	noError             = contextPrototype.Get("NO_ERROR")
	rgba                = contextPrototype.Get("RGBA")
	texture2d           = contextPrototype.Get("TEXTURE_2D")
	textureMagFilter    = contextPrototype.Get("TEXTURE_MAG_FILTER")
	textureMinFilter    = contextPrototype.Get("TEXTURE_MIN_FILTER")
	textureWrapS        = contextPrototype.Get("TEXTURE_WRAP_S")
	textureWrapT        = contextPrototype.Get("TEXTURE_WRAP_T")
	triangles           = contextPrototype.Get("TRIANGLES")
	unpackAlignment     = contextPrototype.Get("UNPACK_ALIGNMENT")
	unsignedByte        = contextPrototype.Get("UNSIGNED_BYTE")
	unsignedShort       = contextPrototype.Get("UNSIGNED_SHORT")
)

type contextImpl struct {
	gl            js.Value
	lastProgramID programID
}

func (c *context) ensureGL() {
	if c.gl != (js.Value{}) {
		return
	}

	if js.Global().Get("WebGLRenderingContext") == js.Undefined() {
		panic("opengl: WebGL is not supported")
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
			panic("opengl: getContext failed")
		}
	}

	c.gl = gl
}

func (c *context) reset() error {
	c.locationCache = newLocationCache()
	c.lastTexture = textureNative(js.Null())
	c.lastFramebuffer = framebufferNative(js.Null())
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = graphics.CompositeModeUnknown

	c.gl = js.Value{}
	c.ensureGL()
	if c.gl.Call("isContextLost").Bool() {
		return fmt.Errorf("opengl: the context is lost")
	}
	gl := c.gl
	gl.Call("enable", blend)
	c.blendFunc(graphics.CompositeModeSourceOver)
	f := gl.Call("getParameter", framebufferBinding)
	c.screenFramebuffer = framebufferNative(f)
	return nil
}

func (c *context) blendFunc(mode graphics.CompositeMode) {
	if c.lastCompositeMode == mode {
		return
	}
	c.lastCompositeMode = mode
	s, d := mode.Operations()
	s2, d2 := convertOperation(s), convertOperation(d)
	c.ensureGL()
	gl := c.gl
	gl.Call("blendFunc", int(s2), int(d2))
}

func (c *context) newTexture(width, height int) (textureNative, error) {
	c.ensureGL()
	gl := c.gl
	t := gl.Call("createTexture")
	if t == js.Null() {
		return textureNative(js.Null()), errors.New("opengl: glGenTexture failed")
	}
	gl.Call("pixelStorei", unpackAlignment, 4)
	c.bindTexture(textureNative(t))

	gl.Call("texParameteri", texture2d, textureMagFilter, nearest)
	gl.Call("texParameteri", texture2d, textureMinFilter, nearest)
	gl.Call("texParameteri", texture2d, textureWrapS, clampToEdge)
	gl.Call("texParameteri", texture2d, textureWrapT, clampToEdge)

	// Firefox warns the usage of textures without specifying pixels (#629)
	//
	//     Error: WebGL warning: drawElements: This operation requires zeroing texture data. This is slow.
	//
	// In Ebiten, textures are filled with pixels laster by the filter that ignores destination, so it is fine
	// to leave textures as uninitialized here. Rather, extra memory allocating for initialization should be
	// avoided.
	gl.Call("texImage2D", texture2d, 0, rgba, width, height, 0, rgba, unsignedByte, nil)

	return textureNative(t), nil
}

func (c *context) bindFramebufferImpl(f framebufferNative) {
	c.ensureGL()
	gl := c.gl
	gl.Call("bindFramebuffer", framebuffer_, js.Value(f))
}

func (c *context) framebufferPixels(f *framebuffer, width, height int) ([]byte, error) {
	c.ensureGL()
	gl := c.gl

	c.bindFramebuffer(f.native)

	pixels := make([]byte, 4*width*height)
	p := js.TypedArrayOf(pixels)
	gl.Call("readPixels", 0, 0, width, height, rgba, unsignedByte, p)
	p.Release()
	return pixels, nil
}

func (c *context) bindTextureImpl(t textureNative) {
	c.ensureGL()
	gl := c.gl
	gl.Call("bindTexture", texture2d, js.Value(t))
}

func (c *context) deleteTexture(t textureNative) {
	c.ensureGL()
	gl := c.gl
	if !gl.Call("isTexture", js.Value(t)).Bool() {
		return
	}
	if c.lastTexture == t {
		c.lastTexture = textureNative(js.Null())
	}
	gl.Call("deleteTexture", js.Value(t))
}

func (c *context) isTexture(t textureNative) bool {
	c.ensureGL()
	gl := c.gl
	return gl.Call("isTexture", js.Value(t)).Bool()
}

func (c *context) texSubImage2D(t textureNative, pixels []byte, x, y, width, height int) {
	c.bindTexture(t)
	c.ensureGL()
	gl := c.gl
	// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
	//                    GLsizei width, GLsizei height,
	//                    GLenum format, GLenum type, ArrayBufferView? pixels);
	p := js.TypedArrayOf(pixels)
	gl.Call("texSubImage2D", texture2d, 0, x, y, width, height, rgba, unsignedByte, p)
	p.Release()
}

func (c *context) newFramebuffer(t textureNative) (framebufferNative, error) {
	c.ensureGL()
	gl := c.gl
	f := gl.Call("createFramebuffer")
	c.bindFramebuffer(framebufferNative(f))

	gl.Call("framebufferTexture2D", framebuffer_, colorAttachment0, texture2d, js.Value(t), 0)
	if s := gl.Call("checkFramebufferStatus", framebuffer_); s.Int() != framebufferComplete.Int() {
		return framebufferNative(js.Null()), errors.New(fmt.Sprintf("opengl: creating framebuffer failed: %d", s.Int()))
	}

	return framebufferNative(f), nil
}

func (c *context) setViewportImpl(width, height int) {
	c.ensureGL()
	gl := c.gl
	gl.Call("viewport", 0, 0, width, height)
}

func (c *context) deleteFramebuffer(f framebufferNative) {
	c.ensureGL()
	gl := c.gl
	if !gl.Call("isFramebuffer", js.Value(f)).Bool() {
		return
	}
	// If a framebuffer to be deleted is bound, a newly bound framebuffer
	// will be a default framebuffer.
	// https://www.khronos.org/opengles/sdk/docs/man/xhtml/glDeleteFramebuffers.xml
	if c.lastFramebuffer == f {
		c.lastFramebuffer = framebufferNative(js.Null())
		c.lastViewportWidth = 0
		c.lastViewportHeight = 0
	}
	gl.Call("deleteFramebuffer", js.Value(f))
}

func (c *context) newShader(shaderType shaderType, source string) (shader, error) {
	c.ensureGL()
	gl := c.gl
	s := gl.Call("createShader", int(shaderType))
	if s == js.Null() {
		return shader(js.Null()), fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
	}

	gl.Call("shaderSource", js.Value(s), source)
	gl.Call("compileShader", js.Value(s))

	if !gl.Call("getShaderParameter", js.Value(s), compileStatus).Bool() {
		log := gl.Call("getShaderInfoLog", js.Value(s))
		return shader(js.Null()), fmt.Errorf("opengl: shader compile failed: %s", log)
	}
	return shader(s), nil
}

func (c *context) deleteShader(s shader) {
	c.ensureGL()
	gl := c.gl
	gl.Call("deleteShader", js.Value(s))
}

func (c *context) newProgram(shaders []shader, attributes []string) (program, error) {
	c.ensureGL()
	gl := c.gl
	v := gl.Call("createProgram")
	if v == js.Null() {
		return program{}, errors.New("opengl: glCreateProgram failed")
	}

	for _, shader := range shaders {
		gl.Call("attachShader", v, js.Value(shader))
	}

	for i, name := range attributes {
		gl.Call("bindAttribLocation", v, i, name)
	}

	gl.Call("linkProgram", v)
	if !gl.Call("getProgramParameter", v, linkStatus).Bool() {
		return program{}, errors.New("opengl: program error")
	}

	id := c.lastProgramID
	c.lastProgramID++
	return program{
		value: v,
		id:    id,
	}, nil
}

func (c *context) useProgram(p program) {
	c.ensureGL()
	gl := c.gl
	gl.Call("useProgram", p.value)
}

func (c *context) deleteProgram(p program) {
	c.ensureGL()
	gl := c.gl
	if !gl.Call("isProgram", p.value).Bool() {
		return
	}
	gl.Call("deleteProgram", p.value)
}

func (c *context) getUniformLocationImpl(p program, location string) uniformLocation {
	c.ensureGL()
	gl := c.gl
	return uniformLocation(gl.Call("getUniformLocation", p.value, location))
}

func (c *context) uniformInt(p program, location string, v int) {
	c.ensureGL()
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	gl.Call("uniform1i", js.Value(l), v)
}

func (c *context) uniformFloat(p program, location string, v float32) {
	c.ensureGL()
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	gl.Call("uniform1f", js.Value(l), v)
}

var (
	float32Array = js.Global().Get("Float32Array")
)

func (c *context) uniformFloats(p program, location string, v []float32) {
	c.ensureGL()
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	switch len(v) {
	case 2:
		gl.Call("uniform2f", js.Value(l), v[0], v[1])
	case 4:
		gl.Call("uniform4f", js.Value(l), v[0], v[1], v[2], v[3])
	case 16:
		arr := js.TypedArrayOf(v)
		gl.Call("uniformMatrix4fv", js.Value(l), false, arr)
		arr.Release()
	default:
		panic(fmt.Sprintf("opengl: invalid uniform floats num: %d", len(v)))
	}
}

func (c *context) vertexAttribPointer(p program, index int, size int, dataType dataType, stride int, offset int) {
	c.ensureGL()
	gl := c.gl
	gl.Call("vertexAttribPointer", index, size, int(dataType), false, stride, offset)
}

func (c *context) enableVertexAttribArray(p program, index int) {
	c.ensureGL()
	gl := c.gl
	gl.Call("enableVertexAttribArray", index)
}

func (c *context) disableVertexAttribArray(p program, index int) {
	c.ensureGL()
	gl := c.gl
	gl.Call("disableVertexAttribArray", index)
}

func (c *context) newArrayBuffer(size int) buffer {
	c.ensureGL()
	gl := c.gl
	b := gl.Call("createBuffer")
	gl.Call("bindBuffer", int(arrayBuffer), js.Value(b))
	gl.Call("bufferData", int(arrayBuffer), size, int(dynamicDraw))
	return buffer(b)
}

func (c *context) newElementArrayBuffer(size int) buffer {
	c.ensureGL()
	gl := c.gl
	b := gl.Call("createBuffer")
	gl.Call("bindBuffer", int(elementArrayBuffer), js.Value(b))
	gl.Call("bufferData", int(elementArrayBuffer), size, int(dynamicDraw))
	return buffer(b)
}

func (c *context) bindBuffer(bufferType bufferType, b buffer) {
	c.ensureGL()
	gl := c.gl
	gl.Call("bindBuffer", int(bufferType), js.Value(b))
}

func (c *context) arrayBufferSubData(data []float32) {
	c.ensureGL()
	gl := c.gl
	arr := js.TypedArrayOf(data)
	gl.Call("bufferSubData", int(arrayBuffer), 0, arr)
	arr.Release()
}

func (c *context) elementArrayBufferSubData(data []uint16) {
	c.ensureGL()
	gl := c.gl
	arr := js.TypedArrayOf(data)
	gl.Call("bufferSubData", int(elementArrayBuffer), 0, arr)
	arr.Release()
}

func (c *context) deleteBuffer(b buffer) {
	c.ensureGL()
	gl := c.gl
	gl.Call("deleteBuffer", js.Value(b))
}

func (c *context) drawElements(len int, offsetInBytes int) {
	c.ensureGL()
	gl := c.gl
	gl.Call("drawElements", triangles, len, unsignedShort, offsetInBytes)
}

func (c *context) maxTextureSizeImpl() int {
	c.ensureGL()
	gl := c.gl
	return gl.Call("getParameter", maxTextureSize).Int()
}

func (c *context) flush() {
	c.ensureGL()
	gl := c.gl
	gl.Call("flush")
}
