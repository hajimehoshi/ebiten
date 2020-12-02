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

package opengl

import (
	"errors"
	"fmt"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gles"
	"github.com/hajimehoshi/ebiten/v2/internal/jsutil"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/web"
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

func (t textureNative) equal(rhs textureNative) bool {
	return jsutil.Equal(js.Value(t), js.Value(rhs))
}

func (f framebufferNative) equal(rhs framebufferNative) bool {
	return jsutil.Equal(js.Value(f), js.Value(rhs))
}

func (s shader) equal(rhs shader) bool {
	return jsutil.Equal(js.Value(s), js.Value(rhs))
}

func (b buffer) equal(rhs buffer) bool {
	return jsutil.Equal(js.Value(b), js.Value(rhs))
}

func (u uniformLocation) equal(rhs uniformLocation) bool {
	return jsutil.Equal(js.Value(u), js.Value(rhs))
}

func (p program) equal(rhs program) bool {
	return jsutil.Equal(p.value, rhs.value) && p.id == rhs.id
}

var InvalidTexture = textureNative(js.Null())

var invalidUniform = uniformLocation(js.Null())

func getProgramID(p program) programID {
	return p.id
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

var (
	isWebGL2Available = !forceWebGL1 && js.Global().Get("WebGL2RenderingContext").Truthy()
	needsRestoring_   = !web.IsMobileBrowser() && !js.Global().Get("go2cpp").Truthy()
)

type contextImpl struct {
	gl            js.Value
	lastProgramID programID
}

func (c *context) initGL() {
	c.gl = js.Value{}

	var gl js.Value

	// TODO: Define id?
	if doc := js.Global().Get("document"); doc.Truthy() {
		canvas := doc.Call("querySelector", "canvas")
		attr := js.Global().Get("Object").New()
		attr.Set("alpha", true)
		attr.Set("premultipliedAlpha", true)

		if isWebGL2Available {
			gl = canvas.Call("getContext", "webgl2", attr)
		} else {
			gl = canvas.Call("getContext", "webgl", attr)
			if jsutil.Equal(gl, js.Null()) {
				gl = canvas.Call("getContext", "experimental-webgl", attr)
				if jsutil.Equal(gl, js.Null()) {
					panic("opengl: getContext failed")
				}
			}
		}
	} else if go2cpp := js.Global().Get("go2cpp"); go2cpp.Truthy() {
		gl = go2cpp.Get("gl")
	}

	c.gl = gl
}

func (c *context) reset() error {
	c.locationCache = newLocationCache()
	c.lastTexture = textureNative(js.Null())
	c.lastFramebuffer = framebufferNative(js.Null())
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = driver.CompositeModeUnknown

	c.initGL()

	if c.gl.Call("isContextLost").Bool() {
		return driver.GraphicsNotReady
	}
	gl := c.gl
	gl.Call("enable", gles.BLEND)
	gl.Call("enable", gles.SCISSOR_TEST)
	c.blendFunc(driver.CompositeModeSourceOver)
	f := gl.Call("getParameter", gles.FRAMEBUFFER_BINDING)
	c.screenFramebuffer = framebufferNative(f)

	if !isWebGL2Available {
		gl.Call("getExtension", "OES_standard_derivatives")
	}
	return nil
}

func (c *context) blendFunc(mode driver.CompositeMode) {
	if c.lastCompositeMode == mode {
		return
	}
	c.lastCompositeMode = mode
	s, d := mode.Operations()
	s2, d2 := convertOperation(s), convertOperation(d)
	gl := c.gl
	gl.Call("blendFunc", int(s2), int(d2))
}

func (c *context) scissor(x, y, width, height int) {
	gl := c.gl
	gl.Call("scissor", x, y, width, height)
}

func (c *context) newTexture(width, height int) (textureNative, error) {
	gl := c.gl
	t := gl.Call("createTexture")
	if jsutil.Equal(t, js.Null()) {
		return textureNative(js.Null()), errors.New("opengl: glGenTexture failed")
	}
	gl.Call("pixelStorei", gles.UNPACK_ALIGNMENT, 4)
	c.bindTexture(textureNative(t))

	gl.Call("texParameteri", gles.TEXTURE_2D, gles.TEXTURE_MAG_FILTER, gles.NEAREST)
	gl.Call("texParameteri", gles.TEXTURE_2D, gles.TEXTURE_MIN_FILTER, gles.NEAREST)
	gl.Call("texParameteri", gles.TEXTURE_2D, gles.TEXTURE_WRAP_S, gles.CLAMP_TO_EDGE)
	gl.Call("texParameteri", gles.TEXTURE_2D, gles.TEXTURE_WRAP_T, gles.CLAMP_TO_EDGE)

	// Firefox warns the usage of textures without specifying pixels (#629)
	//
	//     Error: WebGL warning: drawElements: This operation requires zeroing texture data. This is slow.
	//
	// In Ebiten, textures are filled with pixels laster by the filter that ignores destination, so it is fine
	// to leave textures as uninitialized here. Rather, extra memory allocating for initialization should be
	// avoided.
	gl.Call("texImage2D", gles.TEXTURE_2D, 0, gles.RGBA, width, height, 0, gles.RGBA, gles.UNSIGNED_BYTE, nil)

	return textureNative(t), nil
}

func (c *context) bindFramebufferImpl(f framebufferNative) {
	gl := c.gl
	gl.Call("bindFramebuffer", gles.FRAMEBUFFER, js.Value(f))
}

func (c *context) framebufferPixels(f *framebuffer, width, height int) []byte {
	gl := c.gl

	c.bindFramebuffer(f.native)

	p := jsutil.TemporaryUint8Array(4 * width * height)
	gl.Call("readPixels", 0, 0, width, height, gles.RGBA, gles.UNSIGNED_BYTE, p)

	return jsutil.Uint8ArrayToSlice(p)
}

func (c *context) framebufferPixelsToBuffer(f *framebuffer, buffer buffer, width, height int) {
	gl := c.gl

	c.bindFramebuffer(f.native)
	gl.Call("bindBuffer", gles.PIXEL_PACK_BUFFER, js.Value(buffer))
	// void gl.readPixels(x, y, width, height, format, type, GLintptr offset);
	gl.Call("readPixels", 0, 0, width, height, gles.RGBA, gles.UNSIGNED_BYTE, 0)
	gl.Call("bindBuffer", gles.PIXEL_PACK_BUFFER, nil)
}

func (c *context) activeTexture(idx int) {
	gl := c.gl
	gl.Call("activeTexture", gles.TEXTURE0+idx)
}

func (c *context) bindTextureImpl(t textureNative) {
	gl := c.gl
	gl.Call("bindTexture", gles.TEXTURE_2D, js.Value(t))
}

func (c *context) deleteTexture(t textureNative) {
	gl := c.gl
	if !gl.Call("isTexture", js.Value(t)).Bool() {
		return
	}
	if c.lastTexture.equal(t) {
		c.lastTexture = textureNative(js.Null())
	}
	gl.Call("deleteTexture", js.Value(t))
}

func (c *context) isTexture(t textureNative) bool {
	// isTexture should not be called to detect context-lost since this performance is not good (#1175).
	panic("opengl: isTexture is not implemented")
}

func (c *context) newFramebuffer(t textureNative) (framebufferNative, error) {
	gl := c.gl
	f := gl.Call("createFramebuffer")
	c.bindFramebuffer(framebufferNative(f))

	gl.Call("framebufferTexture2D", gles.FRAMEBUFFER, gles.COLOR_ATTACHMENT0, gles.TEXTURE_2D, js.Value(t), 0)
	if s := gl.Call("checkFramebufferStatus", gles.FRAMEBUFFER); s.Int() != gles.FRAMEBUFFER_COMPLETE {
		return framebufferNative(js.Null()), errors.New(fmt.Sprintf("opengl: creating framebuffer failed: %d", s.Int()))
	}

	return framebufferNative(f), nil
}

func (c *context) setViewportImpl(width, height int) {
	gl := c.gl
	gl.Call("viewport", 0, 0, width, height)
}

func (c *context) deleteFramebuffer(f framebufferNative) {
	gl := c.gl
	if !gl.Call("isFramebuffer", js.Value(f)).Bool() {
		return
	}
	// If a framebuffer to be deleted is bound, a newly bound framebuffer
	// will be a default framebuffer.
	// https://www.khronos.org/opengles/sdk/docs/man/xhtml/glDeleteFramebuffers.xml
	if c.lastFramebuffer.equal(f) {
		c.lastFramebuffer = framebufferNative(js.Null())
		c.lastViewportWidth = 0
		c.lastViewportHeight = 0
	}
	gl.Call("deleteFramebuffer", js.Value(f))
}

func (c *context) newVertexShader(source string) (shader, error) {
	return c.newShader(gles.VERTEX_SHADER, source)
}

func (c *context) newFragmentShader(source string) (shader, error) {
	return c.newShader(gles.FRAGMENT_SHADER, source)
}

func (c *context) newShader(shaderType int, source string) (shader, error) {
	gl := c.gl
	s := gl.Call("createShader", int(shaderType))
	if jsutil.Equal(s, js.Null()) {
		return shader(js.Null()), fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
	}

	gl.Call("shaderSource", js.Value(s), source)
	gl.Call("compileShader", js.Value(s))

	if !gl.Call("getShaderParameter", js.Value(s), gles.COMPILE_STATUS).Bool() {
		log := gl.Call("getShaderInfoLog", js.Value(s))
		return shader(js.Null()), fmt.Errorf("opengl: shader compile failed: %s", log)
	}
	return shader(s), nil
}

func (c *context) deleteShader(s shader) {
	gl := c.gl
	gl.Call("deleteShader", js.Value(s))
}

func (c *context) newProgram(shaders []shader, attributes []string) (program, error) {
	gl := c.gl
	v := gl.Call("createProgram")
	if jsutil.Equal(v, js.Null()) {
		return program{}, errors.New("opengl: glCreateProgram failed")
	}

	for _, shader := range shaders {
		gl.Call("attachShader", v, js.Value(shader))
	}

	for i, name := range attributes {
		gl.Call("bindAttribLocation", v, i, name)
	}

	gl.Call("linkProgram", v)
	if !gl.Call("getProgramParameter", v, gles.LINK_STATUS).Bool() {
		info := gl.Call("getProgramInfoLog", v).String()
		return program{}, fmt.Errorf("opengl: program error: %s", info)
	}

	id := c.lastProgramID
	c.lastProgramID++
	return program{
		value: v,
		id:    id,
	}, nil
}

func (c *context) useProgram(p program) {
	gl := c.gl
	gl.Call("useProgram", p.value)
}

func (c *context) deleteProgram(p program) {
	gl := c.gl
	if !gl.Call("isProgram", p.value).Bool() {
		return
	}
	gl.Call("deleteProgram", p.value)
}

func (c *context) getUniformLocationImpl(p program, location string) uniformLocation {
	gl := c.gl
	return uniformLocation(gl.Call("getUniformLocation", p.value, location))
}

func (c *context) uniformInt(p program, location string, v int) bool {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l.equal(invalidUniform) {
		return false
	}
	gl.Call("uniform1i", js.Value(l), v)
	return true
}

func (c *context) uniformFloat(p program, location string, v float32) bool {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l.equal(invalidUniform) {
		return false
	}
	gl.Call("uniform1f", js.Value(l), v)
	return true
}

func (c *context) uniformFloats(p program, location string, v []float32, typ shaderir.Type) bool {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l.equal(invalidUniform) {
		return false
	}

	base := typ.Main
	if base == shaderir.Array {
		base = typ.Sub[0].Main
	}

	arr8 := jsutil.TemporaryUint8Array(len(v) * 4)
	arr := js.Global().Get("Float32Array").New(arr8.Get("buffer"), arr8.Get("byteOffset"), len(v))
	jsutil.CopySliceToJS(arr, v)

	switch base {
	case shaderir.Float:
		gl.Call("uniform1fv", js.Value(l), arr)
	case shaderir.Vec2:
		gl.Call("uniform2fv", js.Value(l), arr)
	case shaderir.Vec3:
		gl.Call("uniform3fv", js.Value(l), arr)
	case shaderir.Vec4:
		gl.Call("uniform4fv", js.Value(l), arr)
	case shaderir.Mat2:
		gl.Call("uniformMatrix2fv", js.Value(l), false, arr)
	case shaderir.Mat3:
		gl.Call("uniformMatrix3fv", js.Value(l), false, arr)
	case shaderir.Mat4:
		gl.Call("uniformMatrix4fv", js.Value(l), false, arr)
	default:
		panic(fmt.Sprintf("opengl: unexpected type: %s", typ.String()))
	}

	return true
}

func (c *context) vertexAttribPointer(index int, size int, stride int, offset int) {
	gl := c.gl
	gl.Call("vertexAttribPointer", index, size, gles.FLOAT, false, stride, offset)
}

func (c *context) enableVertexAttribArray(index int) {
	gl := c.gl
	gl.Call("enableVertexAttribArray", index)
}

func (c *context) disableVertexAttribArray(index int) {
	gl := c.gl
	gl.Call("disableVertexAttribArray", index)
}

func (c *context) newArrayBuffer(size int) buffer {
	gl := c.gl
	b := gl.Call("createBuffer")
	gl.Call("bindBuffer", gles.ARRAY_BUFFER, js.Value(b))
	gl.Call("bufferData", gles.ARRAY_BUFFER, size, gles.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) newElementArrayBuffer(size int) buffer {
	gl := c.gl
	b := gl.Call("createBuffer")
	gl.Call("bindBuffer", gles.ELEMENT_ARRAY_BUFFER, js.Value(b))
	gl.Call("bufferData", gles.ELEMENT_ARRAY_BUFFER, size, gles.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) bindArrayBuffer(b buffer) {
	gl := c.gl
	gl.Call("bindBuffer", gles.ARRAY_BUFFER, js.Value(b))
}

func (c *context) bindElementArrayBuffer(b buffer) {
	gl := c.gl
	gl.Call("bindBuffer", gles.ELEMENT_ARRAY_BUFFER, js.Value(b))
}

func (c *context) arrayBufferSubData(data []float32) {
	gl := c.gl
	arr := jsutil.TemporaryUint8Array(len(data) * 4)
	jsutil.CopySliceToJS(arr, data)
	gl.Call("bufferSubData", gles.ARRAY_BUFFER, 0, arr)
}

func (c *context) elementArrayBufferSubData(data []uint16) {
	gl := c.gl
	arr := jsutil.TemporaryUint8Array(len(data) * 2)
	jsutil.CopySliceToJS(arr, data)
	gl.Call("bufferSubData", gles.ELEMENT_ARRAY_BUFFER, 0, arr)
}

func (c *context) deleteBuffer(b buffer) {
	gl := c.gl
	gl.Call("deleteBuffer", js.Value(b))
}

func (c *context) drawElements(len int, offsetInBytes int) {
	gl := c.gl
	gl.Call("drawElements", gles.TRIANGLES, len, gles.UNSIGNED_SHORT, offsetInBytes)
}

func (c *context) maxTextureSizeImpl() int {
	gl := c.gl
	return gl.Call("getParameter", gles.MAX_TEXTURE_SIZE).Int()
}

func (c *context) getShaderPrecisionFormatPrecision() int {
	gl := c.gl
	return gl.Call("getShaderPrecisionFormat", gles.FRAGMENT_SHADER, gles.HIGH_FLOAT).Get("precision").Int()
}

func (c *context) flush() {
	gl := c.gl
	gl.Call("flush")
}

func (c *context) needsRestoring() bool {
	return needsRestoring_
}

func (c *context) canUsePBO() bool {
	return isWebGL2Available
}

func (c *context) texSubImage2D(t textureNative, width, height int, args []*driver.ReplacePixelsArgs) {
	c.bindTexture(t)
	gl := c.gl
	// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
	//                    GLsizei width, GLsizei height,
	//                    GLenum format, GLenum type, ArrayBufferView? pixels);
	for _, a := range args {
		arr := jsutil.TemporaryUint8Array(len(a.Pixels))
		jsutil.CopySliceToJS(arr, a.Pixels)
		gl.Call("texSubImage2D", gles.TEXTURE_2D, 0, a.X, a.Y, a.Width, a.Height, gles.RGBA, gles.UNSIGNED_BYTE, arr)
	}
}

func (c *context) newPixelBufferObject(width, height int) buffer {
	gl := c.gl
	b := gl.Call("createBuffer")
	gl.Call("bindBuffer", gles.PIXEL_UNPACK_BUFFER, js.Value(b))
	gl.Call("bufferData", gles.PIXEL_UNPACK_BUFFER, 4*width*height, gles.STREAM_DRAW)
	gl.Call("bindBuffer", gles.PIXEL_UNPACK_BUFFER, nil)
	return buffer(b)
}

func (c *context) replacePixelsWithPBO(buffer buffer, t textureNative, width, height int, args []*driver.ReplacePixelsArgs) {
	c.bindTexture(t)
	gl := c.gl
	gl.Call("bindBuffer", gles.PIXEL_UNPACK_BUFFER, js.Value(buffer))

	stride := 4 * width
	for _, a := range args {
		arr := jsutil.TemporaryUint8Array(len(a.Pixels))
		jsutil.CopySliceToJS(arr, a.Pixels)
		offset := 4 * (a.Y*width + a.X)
		for j := 0; j < a.Height; j++ {
			gl.Call("bufferSubData", gles.PIXEL_UNPACK_BUFFER, offset+stride*j, arr, 4*a.Width*j, 4*a.Width)
		}
	}

	// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
	//                    GLsizei width, GLsizei height,
	//                    GLenum format, GLenum type, GLintptr offset);
	gl.Call("texSubImage2D", gles.TEXTURE_2D, 0, 0, 0, width, height, gles.RGBA, gles.UNSIGNED_BYTE, 0)
	gl.Call("bindBuffer", gles.PIXEL_UNPACK_BUFFER, nil)
}

func (c *context) getBufferSubData(buffer buffer, width, height int) []byte {
	gl := c.gl
	gl.Call("bindBuffer", gles.PIXEL_UNPACK_BUFFER, buffer)
	arr := jsutil.TemporaryUint8Array(4 * width * height)
	gl.Call("getBufferSubData", gles.PIXEL_UNPACK_BUFFER, 0, arr)
	gl.Call("bindBuffer", gles.PIXEL_UNPACK_BUFFER, 0)
	return jsutil.Uint8ArrayToSlice(arr)
}
