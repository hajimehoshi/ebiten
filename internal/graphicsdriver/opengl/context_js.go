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
)

type (
	textureNative      js.Value
	renderbufferNative js.Value
	framebufferNative  js.Value
	shader             js.Value
	buffer             js.Value
	uniformLocation    js.Value

	attribLocation int
	programID      int
	program        struct {
		value js.Value
		id    programID
	}
)

func (t textureNative) equal(rhs textureNative) bool {
	return js.Value(t).Equal(js.Value(rhs))
}

func (r renderbufferNative) equal(rhs renderbufferNative) bool {
	return js.Value(r).Equal(js.Value(rhs))
}

func (f framebufferNative) equal(rhs framebufferNative) bool {
	return js.Value(f).Equal(js.Value(rhs))
}

func (s shader) equal(rhs shader) bool {
	return js.Value(s).Equal(js.Value(rhs))
}

func (b buffer) equal(rhs buffer) bool {
	return js.Value(b).Equal(js.Value(rhs))
}

func (u uniformLocation) equal(rhs uniformLocation) bool {
	return js.Value(u).Equal(js.Value(rhs))
}

func (p program) equal(rhs program) bool {
	return p.value.Equal(rhs.value) && p.id == rhs.id
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

type webGLVersion int

const (
	webGLVersionUnknown webGLVersion = iota
	webGLVersion1
	webGLVersion2
)

var (
	webGL2MightBeAvailable = !forceWebGL1 && (js.Global().Get("WebGL2RenderingContext").Truthy() || js.Global().Get("go2cpp").Truthy())
)

type contextImpl struct {
	gl            *gl
	lastProgramID programID
	webGLVersion  webGLVersion
}

func (c *context) usesWebGL2() bool {
	return c.webGLVersion == webGLVersion2
}

func (c *context) initGL() {
	c.webGLVersion = webGLVersionUnknown

	var gl js.Value

	// TODO: Define id?
	if doc := js.Global().Get("document"); doc.Truthy() {
		canvas := doc.Call("querySelector", "canvas")
		attr := js.Global().Get("Object").New()
		attr.Set("alpha", true)
		attr.Set("premultipliedAlpha", true)
		attr.Set("stencil", true)

		if webGL2MightBeAvailable {
			gl = canvas.Call("getContext", "webgl2", attr)
			if gl.Truthy() {
				c.webGLVersion = webGLVersion2
			}
		}

		// Even though WebGL2RenderingContext exists, getting a webgl2 context might fail (#1738).
		if !gl.Truthy() {
			gl = canvas.Call("getContext", "webgl", attr)
			if !gl.Truthy() {
				gl = canvas.Call("getContext", "experimental-webgl", attr)
			}
			if gl.Truthy() {
				c.webGLVersion = webGLVersion1
			}
		}

		if !gl.Truthy() {
			panic("opengl: getContext failed")
		}
	} else if go2cpp := js.Global().Get("go2cpp"); go2cpp.Truthy() {
		gl = go2cpp.Get("gl")
		c.webGLVersion = webGLVersion2
	}

	c.gl = c.newGL(gl)
}

func (c *context) reset() error {
	c.locationCache = newLocationCache()
	c.lastTexture = textureNative(js.Null())
	c.lastFramebuffer = framebufferNative(js.Null())
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = driver.CompositeModeUnknown

	c.initGL()

	if c.gl.isContextLost.Invoke().Bool() {
		return driver.GraphicsNotReady
	}
	gl := c.gl
	gl.enable.Invoke(gles.BLEND)
	gl.enable.Invoke(gles.SCISSOR_TEST)
	c.blendFunc(driver.CompositeModeSourceOver)
	f := gl.getParameter.Invoke(gles.FRAMEBUFFER_BINDING)
	c.screenFramebuffer = framebufferNative(f)

	if !c.usesWebGL2() {
		gl.getExtension.Invoke("OES_standard_derivatives")
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
	gl.blendFunc.Invoke(int(s2), int(d2))
}

func (c *context) scissor(x, y, width, height int) {
	gl := c.gl
	gl.scissor.Invoke(x, y, width, height)
}

func (c *context) newTexture(width, height int) (textureNative, error) {
	gl := c.gl
	t := gl.createTexture.Invoke()
	if !t.Truthy() {
		return textureNative(js.Null()), errors.New("opengl: createTexture failed")
	}
	c.bindTexture(textureNative(t))

	gl.texParameteri.Invoke(gles.TEXTURE_2D, gles.TEXTURE_MAG_FILTER, gles.NEAREST)
	gl.texParameteri.Invoke(gles.TEXTURE_2D, gles.TEXTURE_MIN_FILTER, gles.NEAREST)
	gl.texParameteri.Invoke(gles.TEXTURE_2D, gles.TEXTURE_WRAP_S, gles.CLAMP_TO_EDGE)
	gl.texParameteri.Invoke(gles.TEXTURE_2D, gles.TEXTURE_WRAP_T, gles.CLAMP_TO_EDGE)

	gl.pixelStorei.Invoke(gles.UNPACK_ALIGNMENT, 4)
	// Firefox warns the usage of textures without specifying pixels (#629)
	//
	//     Error: WebGL warning: drawElements: This operation requires zeroing texture data. This is slow.
	//
	// In Ebiten, textures are filled with pixels laster by the filter that ignores destination, so it is fine
	// to leave textures as uninitialized here. Rather, extra memory allocating for initialization should be
	// avoided.
	gl.texImage2D.Invoke(gles.TEXTURE_2D, 0, gles.RGBA, width, height, 0, gles.RGBA, gles.UNSIGNED_BYTE, nil)

	return textureNative(t), nil
}

func (c *context) bindFramebufferImpl(f framebufferNative) {
	gl := c.gl
	gl.bindFramebuffer.Invoke(gles.FRAMEBUFFER, js.Value(f))
}

func (c *context) framebufferPixels(f *framebuffer, width, height int) []byte {
	gl := c.gl

	c.bindFramebuffer(f.native)

	l := 4 * width * height
	p := jsutil.TemporaryUint8Array(l, nil)
	gl.readPixels.Invoke(0, 0, width, height, gles.RGBA, gles.UNSIGNED_BYTE, p)

	return jsutil.Uint8ArrayToSlice(p, l)
}

func (c *context) framebufferPixelsToBuffer(f *framebuffer, buffer buffer, width, height int) {
	gl := c.gl

	c.bindFramebuffer(f.native)
	gl.bindBuffer.Invoke(gles.PIXEL_PACK_BUFFER, js.Value(buffer))
	// void gl.readPixels(x, y, width, height, format, type, GLintptr offset);
	gl.readPixels.Invoke(0, 0, width, height, gles.RGBA, gles.UNSIGNED_BYTE, 0)
	gl.bindBuffer.Invoke(gles.PIXEL_PACK_BUFFER, nil)
}

func (c *context) activeTexture(idx int) {
	gl := c.gl
	gl.activeTexture.Invoke(gles.TEXTURE0 + idx)
}

func (c *context) bindTextureImpl(t textureNative) {
	gl := c.gl
	gl.bindTexture.Invoke(gles.TEXTURE_2D, js.Value(t))
}

func (c *context) deleteTexture(t textureNative) {
	gl := c.gl
	if !gl.isTexture.Invoke(js.Value(t)).Bool() {
		return
	}
	if c.lastTexture.equal(t) {
		c.lastTexture = textureNative(js.Null())
	}
	gl.deleteTexture.Invoke(js.Value(t))
}

func (c *context) isTexture(t textureNative) bool {
	// isTexture should not be called to detect context-lost since this performance is not good (#1175).
	panic("opengl: isTexture is not implemented")
}

func (c *context) newRenderbuffer(width, height int) (renderbufferNative, error) {
	gl := c.gl
	r := gl.createRenderbuffer.Invoke()
	if !r.Truthy() {
		return renderbufferNative(js.Null()), errors.New("opengl: createRenderbuffer failed")
	}

	c.bindRenderbuffer(renderbufferNative(r))
	// TODO: Is STENCIL_INDEX8 portable?
	// https://stackoverflow.com/questions/11084961/binding-a-stencil-render-buffer-to-a-frame-buffer-in-opengl
	gl.renderbufferStorage.Invoke(gles.RENDERBUFFER, gles.STENCIL_INDEX8, width, height)

	return renderbufferNative(r), nil
}

func (c *context) bindRenderbufferImpl(r renderbufferNative) {
	gl := c.gl
	gl.bindRenderbuffer.Invoke(gles.RENDERBUFFER, js.Value(r))
}

func (c *context) deleteRenderbuffer(r renderbufferNative) {
	gl := c.gl
	if !gl.isRenderbuffer.Invoke(js.Value(r)).Bool() {
		return
	}
	if c.lastRenderbuffer.equal(r) {
		c.lastRenderbuffer = renderbufferNative(js.Null())
	}
	gl.deleteRenderbuffer.Invoke(js.Value(r))
}

func (c *context) newFramebuffer(t textureNative) (framebufferNative, error) {
	gl := c.gl
	f := gl.createFramebuffer.Invoke()
	c.bindFramebuffer(framebufferNative(f))

	gl.framebufferTexture2D.Invoke(gles.FRAMEBUFFER, gles.COLOR_ATTACHMENT0, gles.TEXTURE_2D, js.Value(t), 0)
	if s := gl.checkFramebufferStatus.Invoke(gles.FRAMEBUFFER); s.Int() != gles.FRAMEBUFFER_COMPLETE {
		return framebufferNative(js.Null()), errors.New(fmt.Sprintf("opengl: creating framebuffer failed: %d", s.Int()))
	}

	return framebufferNative(f), nil
}

func (c *context) bindStencilBuffer(f framebufferNative, r renderbufferNative) error {
	gl := c.gl
	c.bindFramebuffer(f)

	gl.framebufferRenderbuffer.Invoke(gles.FRAMEBUFFER, gles.STENCIL_ATTACHMENT, gles.RENDERBUFFER, js.Value(r))
	if s := gl.checkFramebufferStatus.Invoke(gles.FRAMEBUFFER); s.Int() != gles.FRAMEBUFFER_COMPLETE {
		return errors.New(fmt.Sprintf("opengl: framebufferRenderbuffer failed: %d", s.Int()))
	}
	return nil
}

func (c *context) setViewportImpl(width, height int) {
	gl := c.gl
	gl.viewport.Invoke(0, 0, width, height)
}

func (c *context) deleteFramebuffer(f framebufferNative) {
	gl := c.gl
	if !gl.isFramebuffer.Invoke(js.Value(f)).Bool() {
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
	gl.deleteFramebuffer.Invoke(js.Value(f))
}

func (c *context) newVertexShader(source string) (shader, error) {
	return c.newShader(gles.VERTEX_SHADER, source)
}

func (c *context) newFragmentShader(source string) (shader, error) {
	return c.newShader(gles.FRAGMENT_SHADER, source)
}

func (c *context) newShader(shaderType int, source string) (shader, error) {
	gl := c.gl
	s := gl.createShader.Invoke(int(shaderType))
	if !s.Truthy() {
		return shader(js.Null()), fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
	}

	gl.shaderSource.Invoke(js.Value(s), source)
	gl.compileShader.Invoke(js.Value(s))

	if !gl.getShaderParameter.Invoke(js.Value(s), gles.COMPILE_STATUS).Bool() {
		log := gl.getShaderInfoLog.Invoke(js.Value(s))
		return shader(js.Null()), fmt.Errorf("opengl: shader compile failed: %s", log)
	}
	return shader(s), nil
}

func (c *context) deleteShader(s shader) {
	gl := c.gl
	gl.deleteShader.Invoke(js.Value(s))
}

func (c *context) newProgram(shaders []shader, attributes []string) (program, error) {
	gl := c.gl
	v := gl.createProgram.Invoke()
	if !v.Truthy() {
		return program{}, errors.New("opengl: glCreateProgram failed")
	}

	for _, shader := range shaders {
		gl.attachShader.Invoke(v, js.Value(shader))
	}

	for i, name := range attributes {
		gl.bindAttribLocation.Invoke(v, i, name)
	}

	gl.linkProgram.Invoke(v)
	if !gl.getProgramParameter.Invoke(v, gles.LINK_STATUS).Bool() {
		info := gl.getProgramInfoLog.Invoke(v).String()
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
	gl.useProgram.Invoke(p.value)
}

func (c *context) deleteProgram(p program) {
	c.locationCache.deleteProgram(p)

	gl := c.gl
	if !gl.isProgram.Invoke(p.value).Bool() {
		return
	}
	gl.deleteProgram.Invoke(p.value)
}

func (c *context) getUniformLocationImpl(p program, location string) uniformLocation {
	gl := c.gl
	return uniformLocation(gl.getUniformLocation.Invoke(p.value, location))
}

func (c *context) uniformInt(p program, location string, v int) bool {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l.equal(invalidUniform) {
		return false
	}
	gl.uniform1i.Invoke(js.Value(l), v)
	return true
}

func (c *context) uniformFloat(p program, location string, v float32) bool {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l.equal(invalidUniform) {
		return false
	}
	gl.uniform1f.Invoke(js.Value(l), v)
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

	arr := jsutil.TemporaryFloat32Array(len(v), v)

	switch base {
	case shaderir.Float:
		if c.usesWebGL2() {
			gl.uniform1fv.Invoke(js.Value(l), arr, 0, len(v))
		} else {
			gl.uniform1fv.Invoke(js.Value(l), arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Vec2:
		if c.usesWebGL2() {
			gl.uniform2fv.Invoke(js.Value(l), arr, 0, len(v))
		} else {
			gl.uniform2fv.Invoke(js.Value(l), arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Vec3:
		if c.usesWebGL2() {
			gl.uniform3fv.Invoke(js.Value(l), arr, 0, len(v))
		} else {
			gl.uniform3fv.Invoke(js.Value(l), arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Vec4:
		if c.usesWebGL2() {
			gl.uniform4fv.Invoke(js.Value(l), arr, 0, len(v))
		} else {
			gl.uniform4fv.Invoke(js.Value(l), arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Mat2:
		if c.usesWebGL2() {
			gl.uniformMatrix2fv.Invoke(js.Value(l), false, arr, 0, len(v))
		} else {
			gl.uniformMatrix2fv.Invoke(js.Value(l), false, arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Mat3:
		if c.usesWebGL2() {
			gl.uniformMatrix3fv.Invoke(js.Value(l), false, arr, 0, len(v))
		} else {
			gl.uniformMatrix3fv.Invoke(js.Value(l), false, arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Mat4:
		if c.usesWebGL2() {
			gl.uniformMatrix4fv.Invoke(js.Value(l), false, arr, 0, len(v))
		} else {
			gl.uniformMatrix4fv.Invoke(js.Value(l), false, arr.Call("subarray", 0, len(v)))
		}
	default:
		panic(fmt.Sprintf("opengl: unexpected type: %s", typ.String()))
	}

	return true
}

func (c *context) vertexAttribPointer(index int, size int, stride int, offset int) {
	gl := c.gl
	gl.vertexAttribPointer.Invoke(index, size, gles.FLOAT, false, stride, offset)
}

func (c *context) enableVertexAttribArray(index int) {
	gl := c.gl
	gl.enableVertexAttribArray.Invoke(index)
}

func (c *context) disableVertexAttribArray(index int) {
	gl := c.gl
	gl.disableVertexAttribArray.Invoke(index)
}

func (c *context) newArrayBuffer(size int) buffer {
	gl := c.gl
	b := gl.createBuffer.Invoke()
	gl.bindBuffer.Invoke(gles.ARRAY_BUFFER, js.Value(b))
	gl.bufferData.Invoke(gles.ARRAY_BUFFER, size, gles.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) newElementArrayBuffer(size int) buffer {
	gl := c.gl
	b := gl.createBuffer.Invoke()
	gl.bindBuffer.Invoke(gles.ELEMENT_ARRAY_BUFFER, js.Value(b))
	gl.bufferData.Invoke(gles.ELEMENT_ARRAY_BUFFER, size, gles.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) bindArrayBuffer(b buffer) {
	gl := c.gl
	gl.bindBuffer.Invoke(gles.ARRAY_BUFFER, js.Value(b))
}

func (c *context) bindElementArrayBuffer(b buffer) {
	gl := c.gl
	gl.bindBuffer.Invoke(gles.ELEMENT_ARRAY_BUFFER, js.Value(b))
}

func (c *context) arrayBufferSubData(data []float32) {
	gl := c.gl
	l := len(data) * 4
	arr := jsutil.TemporaryUint8Array(l, data)
	if c.usesWebGL2() {
		gl.bufferSubData.Invoke(gles.ARRAY_BUFFER, 0, arr, 0, l)
	} else {
		gl.bufferSubData.Invoke(gles.ARRAY_BUFFER, 0, arr.Call("subarray", 0, l))
	}
}

func (c *context) elementArrayBufferSubData(data []uint16) {
	gl := c.gl
	l := len(data) * 2
	arr := jsutil.TemporaryUint8Array(l, data)
	if c.usesWebGL2() {
		gl.bufferSubData.Invoke(gles.ELEMENT_ARRAY_BUFFER, 0, arr, 0, l)
	} else {
		gl.bufferSubData.Invoke(gles.ELEMENT_ARRAY_BUFFER, 0, arr.Call("subarray", 0, l))
	}
}

func (c *context) deleteBuffer(b buffer) {
	gl := c.gl
	gl.deleteBuffer.Invoke(js.Value(b))
}

func (c *context) drawElements(len int, offsetInBytes int) {
	gl := c.gl
	gl.drawElements.Invoke(gles.TRIANGLES, len, gles.UNSIGNED_SHORT, offsetInBytes)
}

func (c *context) maxTextureSizeImpl() int {
	gl := c.gl
	return gl.getParameter.Invoke(gles.MAX_TEXTURE_SIZE).Int()
}

func (c *context) getShaderPrecisionFormatPrecision() int {
	gl := c.gl
	return gl.getShaderPrecisionFormat.Invoke(gles.FRAGMENT_SHADER, gles.HIGH_FLOAT).Get("precision").Int()
}

func (c *context) flush() {
	gl := c.gl
	gl.flush.Invoke()
}

func (c *context) needsRestoring() bool {
	// Though it is possible to have a logic to restore the graphics data for GPU, do not use it for performance (#1603).
	return false
}

func (c *context) canUsePBO() bool {
	return false
}

func (c *context) texSubImage2D(t textureNative, args []*driver.ReplacePixelsArgs) {
	c.bindTexture(t)
	gl := c.gl
	for _, a := range args {
		arr := jsutil.TemporaryUint8Array(len(a.Pixels), a.Pixels)
		if c.usesWebGL2() {
			// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
			//                    GLsizei width, GLsizei height,
			//                    GLenum format, GLenum type, ArrayBufferView pixels, srcOffset);
			gl.texSubImage2D.Invoke(gles.TEXTURE_2D, 0, a.X, a.Y, a.Width, a.Height, gles.RGBA, gles.UNSIGNED_BYTE, arr, 0)
		} else {
			// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
			//                    GLsizei width, GLsizei height,
			//                    GLenum format, GLenum type, ArrayBufferView? pixels);
			gl.texSubImage2D.Invoke(gles.TEXTURE_2D, 0, a.X, a.Y, a.Width, a.Height, gles.RGBA, gles.UNSIGNED_BYTE, arr)
		}
	}
}

func (c *context) enableStencilTest() {
	gl := c.gl
	gl.enable.Invoke(gles.STENCIL_TEST)
}

func (c *context) disableStencilTest() {
	gl := c.gl
	gl.disable.Invoke(gles.STENCIL_TEST)
}

func (c *context) beginStencilWithEvenOddRule() {
	gl := c.gl
	gl.clear.Invoke(gles.STENCIL_BUFFER_BIT)
	gl.stencilFunc.Invoke(gles.ALWAYS, 0x00, 0xff)
	gl.stencilOp.Invoke(gles.KEEP, gles.KEEP, gles.INVERT)
	gl.colorMask.Invoke(false, false, false, false)
}

func (c *context) endStencilWithEvenOddRule() {
	gl := c.gl
	gl.stencilFunc.Invoke(gles.NOTEQUAL, 0x00, 0xff)
	gl.stencilOp.Invoke(gles.KEEP, gles.KEEP, gles.KEEP)
	gl.colorMask.Invoke(true, true, true, true)
}
