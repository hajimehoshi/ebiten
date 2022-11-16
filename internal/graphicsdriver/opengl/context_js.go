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
	"os"
	"strings"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
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

type webGLVersion int

const (
	webGLVersionUnknown webGLVersion = iota
	webGLVersion1
	webGLVersion2
)

func webGL2MightBeAvailable() bool {
	env := os.Getenv("EBITENGINE_OPENGL")
	for _, t := range strings.Split(env, ",") {
		switch strings.TrimSpace(t) {
		case "webgl1":
			return false
		}
	}
	return js.Global().Get("WebGL2RenderingContext").Truthy()
}

type contextImpl struct {
	gl            *jsGL
	canvas        js.Value
	lastProgramID programID
	webGLVersion  webGLVersion
}

func (c *context) usesWebGL2() bool {
	return c.webGLVersion == webGLVersion2
}

func (c *context) initGL() error {
	c.webGLVersion = webGLVersionUnknown

	var gl js.Value

	if doc := js.Global().Get("document"); doc.Truthy() {
		canvas := c.canvas
		attr := js.Global().Get("Object").New()
		attr.Set("alpha", true)
		attr.Set("premultipliedAlpha", true)
		attr.Set("stencil", true)

		if webGL2MightBeAvailable() {
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
			return fmt.Errorf("opengl: getContext failed")
		}
	}

	c.gl = c.newJSGL(gl)
	return nil
}

func (c *context) reset() error {
	c.locationCache = newLocationCache()
	c.lastTexture = textureNative(js.Null())
	c.lastFramebuffer = framebufferNative(js.Null())
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastBlend = graphicsdriver.Blend{}

	if err := c.initGL(); err != nil {
		return err
	}

	if c.gl.isContextLost.Invoke().Bool() {
		return graphicsdriver.GraphicsNotReady
	}
	c.gl.enable.Invoke(gl.BLEND)
	c.gl.enable.Invoke(gl.SCISSOR_TEST)
	c.blend(graphicsdriver.BlendSourceOver)
	f := c.gl.getParameter.Invoke(gl.FRAMEBUFFER_BINDING)
	c.screenFramebuffer = framebufferNative(f)

	if !c.usesWebGL2() {
		c.gl.getExtension.Invoke("OES_standard_derivatives")
	}
	return nil
}

func (c *context) blend(blend graphicsdriver.Blend) {
	if c.lastBlend == blend {
		return
	}
	c.lastBlend = blend
	c.gl.blendFuncSeparate.Invoke(
		int(convertBlendFactor(blend.BlendFactorSourceRGB)),
		int(convertBlendFactor(blend.BlendFactorDestinationRGB)),
		int(convertBlendFactor(blend.BlendFactorSourceAlpha)),
		int(convertBlendFactor(blend.BlendFactorDestinationAlpha)),
	)
	c.gl.blendEquationSeparate.Invoke(
		int(convertBlendOperation(blend.BlendOperationRGB)),
		int(convertBlendOperation(blend.BlendOperationAlpha)),
	)
}

func (c *context) scissor(x, y, width, height int) {
	c.gl.scissor.Invoke(x, y, width, height)
}

func (c *context) newTexture(width, height int) (textureNative, error) {
	t := c.gl.createTexture.Invoke()
	if !t.Truthy() {
		return textureNative(js.Null()), errors.New("opengl: createTexture failed")
	}
	c.bindTexture(textureNative(t))

	c.gl.texParameteri.Invoke(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	c.gl.texParameteri.Invoke(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	c.gl.texParameteri.Invoke(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	c.gl.texParameteri.Invoke(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	c.gl.pixelStorei.Invoke(gl.UNPACK_ALIGNMENT, 4)
	// Firefox warns the usage of textures without specifying pixels (#629)
	//
	//     Error: WebGL warning: drawElements: This operation requires zeroing texture data. This is slow.
	//
	// In Ebitengine, textures are filled with pixels laster by the filter that ignores destination, so it is fine
	// to leave textures as uninitialized here. Rather, extra memory allocating for initialization should be
	// avoided.
	c.gl.texImage2D.Invoke(gl.TEXTURE_2D, 0, gl.RGBA, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)

	return textureNative(t), nil
}

func (c *context) bindFramebufferImpl(f framebufferNative) {
	c.gl.bindFramebuffer.Invoke(gl.FRAMEBUFFER, js.Value(f))
}

func (c *context) framebufferPixels(buf []byte, f *framebuffer, x, y, width, height int) {
	c.bindFramebuffer(f.native)

	if got, want := len(buf), 4*width*height; got != want {
		panic(fmt.Sprintf("opengl: len(buf) must be %d but %d", got, want))
	}

	p := jsutil.TemporaryUint8ArrayFromUint8Slice(len(buf), nil)
	c.gl.readPixels.Invoke(x, y, width, height, gl.RGBA, gl.UNSIGNED_BYTE, p)
	js.CopyBytesToGo(buf, p)
}

func (c *context) framebufferPixelsToBuffer(f *framebuffer, buffer buffer, width, height int) {
	c.bindFramebuffer(f.native)
	c.gl.bindBuffer.Invoke(gl.PIXEL_PACK_BUFFER, js.Value(buffer))
	// void gl.readPixels(x, y, width, height, format, type, GLintptr offset);
	c.gl.readPixels.Invoke(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, 0)
	c.gl.bindBuffer.Invoke(gl.PIXEL_PACK_BUFFER, nil)
}

func (c *context) activeTexture(idx int) {
	c.gl.activeTexture.Invoke(gl.TEXTURE0 + idx)
}

func (c *context) bindTextureImpl(t textureNative) {
	c.gl.bindTexture.Invoke(gl.TEXTURE_2D, js.Value(t))
}

func (c *context) deleteTexture(t textureNative) {
	if !c.gl.isTexture.Invoke(js.Value(t)).Bool() {
		return
	}
	if c.lastTexture.equal(t) {
		c.lastTexture = textureNative(js.Null())
	}
	c.gl.deleteTexture.Invoke(js.Value(t))
}

func (c *context) isTexture(t textureNative) bool {
	// isTexture should not be called to detect context-lost since this performance is not good (#1175).
	panic("opengl: isTexture is not implemented")
}

func (c *context) newRenderbuffer(width, height int) (renderbufferNative, error) {
	r := c.gl.createRenderbuffer.Invoke()
	if !r.Truthy() {
		return renderbufferNative(js.Null()), errors.New("opengl: createRenderbuffer failed")
	}

	c.bindRenderbuffer(renderbufferNative(r))
	// TODO: Is STENCIL_INDEX8 portable?
	// https://stackoverflow.com/questions/11084961/binding-a-stencil-render-buffer-to-a-frame-buffer-in-opengl
	c.gl.renderbufferStorage.Invoke(gl.RENDERBUFFER, gl.STENCIL_INDEX8, width, height)

	return renderbufferNative(r), nil
}

func (c *context) bindRenderbufferImpl(r renderbufferNative) {
	c.gl.bindRenderbuffer.Invoke(gl.RENDERBUFFER, js.Value(r))
}

func (c *context) deleteRenderbuffer(r renderbufferNative) {
	if !c.gl.isRenderbuffer.Invoke(js.Value(r)).Bool() {
		return
	}
	if c.lastRenderbuffer.equal(r) {
		c.lastRenderbuffer = renderbufferNative(js.Null())
	}
	c.gl.deleteRenderbuffer.Invoke(js.Value(r))
}

func (c *context) newFramebuffer(t textureNative) (framebufferNative, error) {
	f := c.gl.createFramebuffer.Invoke()
	c.bindFramebuffer(framebufferNative(f))

	c.gl.framebufferTexture2D.Invoke(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, js.Value(t), 0)
	if s := c.gl.checkFramebufferStatus.Invoke(gl.FRAMEBUFFER); s.Int() != gl.FRAMEBUFFER_COMPLETE {
		return framebufferNative(js.Null()), errors.New(fmt.Sprintf("opengl: creating framebuffer failed: %d", s.Int()))
	}

	return framebufferNative(f), nil
}

func (c *context) bindStencilBuffer(f framebufferNative, r renderbufferNative) error {
	c.bindFramebuffer(f)

	c.gl.framebufferRenderbuffer.Invoke(gl.FRAMEBUFFER, gl.STENCIL_ATTACHMENT, gl.RENDERBUFFER, js.Value(r))
	if s := c.gl.checkFramebufferStatus.Invoke(gl.FRAMEBUFFER); s.Int() != gl.FRAMEBUFFER_COMPLETE {
		return errors.New(fmt.Sprintf("opengl: framebufferRenderbuffer failed: %d", s.Int()))
	}
	return nil
}

func (c *context) setViewportImpl(width, height int) {
	c.gl.viewport.Invoke(0, 0, width, height)
}

func (c *context) deleteFramebuffer(f framebufferNative) {
	if !c.gl.isFramebuffer.Invoke(js.Value(f)).Bool() {
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
	c.gl.deleteFramebuffer.Invoke(js.Value(f))
}

func (c *context) newVertexShader(source string) (shader, error) {
	return c.newShader(gl.VERTEX_SHADER, source)
}

func (c *context) newFragmentShader(source string) (shader, error) {
	return c.newShader(gl.FRAGMENT_SHADER, source)
}

func (c *context) newShader(shaderType int, source string) (shader, error) {
	s := c.gl.createShader.Invoke(int(shaderType))
	if !s.Truthy() {
		return shader(js.Null()), fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
	}

	c.gl.shaderSource.Invoke(js.Value(s), source)
	c.gl.compileShader.Invoke(js.Value(s))

	if !c.gl.getShaderParameter.Invoke(js.Value(s), gl.COMPILE_STATUS).Bool() {
		log := c.gl.getShaderInfoLog.Invoke(js.Value(s))
		return shader(js.Null()), fmt.Errorf("opengl: shader compile failed: %s", log)
	}
	return shader(s), nil
}

func (c *context) deleteShader(s shader) {
	c.gl.deleteShader.Invoke(js.Value(s))
}

func (c *context) newProgram(shaders []shader, attributes []string) (program, error) {
	v := c.gl.createProgram.Invoke()
	if !v.Truthy() {
		return program{}, errors.New("opengl: glCreateProgram failed")
	}

	for _, shader := range shaders {
		c.gl.attachShader.Invoke(v, js.Value(shader))
	}

	for i, name := range attributes {
		c.gl.bindAttribLocation.Invoke(v, i, name)
	}

	c.gl.linkProgram.Invoke(v)
	if !c.gl.getProgramParameter.Invoke(v, gl.LINK_STATUS).Bool() {
		info := c.gl.getProgramInfoLog.Invoke(v).String()
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
	c.gl.useProgram.Invoke(p.value)
}

func (c *context) deleteProgram(p program) {
	c.locationCache.deleteProgram(p)

	if !c.gl.isProgram.Invoke(p.value).Bool() {
		return
	}
	c.gl.deleteProgram.Invoke(p.value)
}

func (c *context) getUniformLocationImpl(p program, location string) uniformLocation {
	return uniformLocation(c.gl.getUniformLocation.Invoke(p.value, location))
}

func (c *context) uniformInt(p program, location string, v int) bool {
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l.equal(invalidUniform) {
		return false
	}
	c.gl.uniform1i.Invoke(js.Value(l), v)
	return true
}

func (c *context) uniforms(p program, location string, v []uint32, typ shaderir.Type) bool {
	l := c.locationCache.GetUniformLocation(c, p, location)
	if l.equal(invalidUniform) {
		return false
	}

	base := typ.Main
	if base == shaderir.Array {
		base = typ.Sub[0].Main
	}

	switch base {
	case shaderir.Float:
		arr := jsutil.TemporaryFloat32Array(len(v), uint32sToFloat32s(v))
		if c.usesWebGL2() {
			c.gl.uniform1fv.Invoke(js.Value(l), arr, 0, len(v))
		} else {
			c.gl.uniform1fv.Invoke(js.Value(l), arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Int:
		arr := jsutil.TemporaryInt32Array(len(v), uint32sToInt32s(v))
		if c.usesWebGL2() {
			c.gl.uniform1iv.Invoke(js.Value(l), arr, 0, len(v))
		} else {
			c.gl.uniform1iv.Invoke(js.Value(l), arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Vec2:
		arr := jsutil.TemporaryFloat32Array(len(v), uint32sToFloat32s(v))
		if c.usesWebGL2() {
			c.gl.uniform2fv.Invoke(js.Value(l), arr, 0, len(v))
		} else {
			c.gl.uniform2fv.Invoke(js.Value(l), arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Vec3:
		arr := jsutil.TemporaryFloat32Array(len(v), uint32sToFloat32s(v))
		if c.usesWebGL2() {
			c.gl.uniform3fv.Invoke(js.Value(l), arr, 0, len(v))
		} else {
			c.gl.uniform3fv.Invoke(js.Value(l), arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Vec4:
		arr := jsutil.TemporaryFloat32Array(len(v), uint32sToFloat32s(v))
		if c.usesWebGL2() {
			c.gl.uniform4fv.Invoke(js.Value(l), arr, 0, len(v))
		} else {
			c.gl.uniform4fv.Invoke(js.Value(l), arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Mat2:
		arr := jsutil.TemporaryFloat32Array(len(v), uint32sToFloat32s(v))
		if c.usesWebGL2() {
			c.gl.uniformMatrix2fv.Invoke(js.Value(l), false, arr, 0, len(v))
		} else {
			c.gl.uniformMatrix2fv.Invoke(js.Value(l), false, arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Mat3:
		arr := jsutil.TemporaryFloat32Array(len(v), uint32sToFloat32s(v))
		if c.usesWebGL2() {
			c.gl.uniformMatrix3fv.Invoke(js.Value(l), false, arr, 0, len(v))
		} else {
			c.gl.uniformMatrix3fv.Invoke(js.Value(l), false, arr.Call("subarray", 0, len(v)))
		}
	case shaderir.Mat4:
		arr := jsutil.TemporaryFloat32Array(len(v), uint32sToFloat32s(v))
		if c.usesWebGL2() {
			c.gl.uniformMatrix4fv.Invoke(js.Value(l), false, arr, 0, len(v))
		} else {
			c.gl.uniformMatrix4fv.Invoke(js.Value(l), false, arr.Call("subarray", 0, len(v)))
		}
	default:
		panic(fmt.Sprintf("opengl: unexpected type: %s", typ.String()))
	}

	return true
}

func (c *context) vertexAttribPointer(index int, size int, stride int, offset int) {
	c.gl.vertexAttribPointer.Invoke(index, size, gl.FLOAT, false, stride, offset)
}

func (c *context) enableVertexAttribArray(index int) {
	c.gl.enableVertexAttribArray.Invoke(index)
}

func (c *context) disableVertexAttribArray(index int) {
	c.gl.disableVertexAttribArray.Invoke(index)
}

func (c *context) newArrayBuffer(size int) buffer {
	b := c.gl.createBuffer.Invoke()
	c.gl.bindBuffer.Invoke(gl.ARRAY_BUFFER, js.Value(b))
	c.gl.bufferData.Invoke(gl.ARRAY_BUFFER, size, gl.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) newElementArrayBuffer(size int) buffer {
	b := c.gl.createBuffer.Invoke()
	c.gl.bindBuffer.Invoke(gl.ELEMENT_ARRAY_BUFFER, js.Value(b))
	c.gl.bufferData.Invoke(gl.ELEMENT_ARRAY_BUFFER, size, gl.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) bindArrayBuffer(b buffer) {
	c.gl.bindBuffer.Invoke(gl.ARRAY_BUFFER, js.Value(b))
}

func (c *context) bindElementArrayBuffer(b buffer) {
	c.gl.bindBuffer.Invoke(gl.ELEMENT_ARRAY_BUFFER, js.Value(b))
}

func (c *context) arrayBufferSubData(data []float32) {
	l := len(data) * 4
	arr := jsutil.TemporaryUint8ArrayFromFloat32Slice(l, data)
	if c.usesWebGL2() {
		c.gl.bufferSubData.Invoke(gl.ARRAY_BUFFER, 0, arr, 0, l)
	} else {
		c.gl.bufferSubData.Invoke(gl.ARRAY_BUFFER, 0, arr.Call("subarray", 0, l))
	}
}

func (c *context) elementArrayBufferSubData(data []uint16) {
	l := len(data) * 2
	arr := jsutil.TemporaryUint8ArrayFromUint16Slice(l, data)
	if c.usesWebGL2() {
		c.gl.bufferSubData.Invoke(gl.ELEMENT_ARRAY_BUFFER, 0, arr, 0, l)
	} else {
		c.gl.bufferSubData.Invoke(gl.ELEMENT_ARRAY_BUFFER, 0, arr.Call("subarray", 0, l))
	}
}

func (c *context) deleteBuffer(b buffer) {
	c.gl.deleteBuffer.Invoke(js.Value(b))
}

func (c *context) drawElements(len int, offsetInBytes int) {
	c.gl.drawElements.Invoke(gl.TRIANGLES, len, gl.UNSIGNED_SHORT, offsetInBytes)
}

func (c *context) maxTextureSizeImpl() int {
	return c.gl.getParameter.Invoke(gl.MAX_TEXTURE_SIZE).Int()
}

func (c *context) flush() {
	c.gl.flush.Invoke()
}

func (c *context) texSubImage2D(t textureNative, args []*graphicsdriver.WritePixelsArgs) {
	c.bindTexture(t)
	for _, a := range args {
		arr := jsutil.TemporaryUint8ArrayFromUint8Slice(len(a.Pixels), a.Pixels)
		if c.usesWebGL2() {
			// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
			//                    GLsizei width, GLsizei height,
			//                    GLenum format, GLenum type, ArrayBufferView pixels, srcOffset);
			c.gl.texSubImage2D.Invoke(gl.TEXTURE_2D, 0, a.X, a.Y, a.Width, a.Height, gl.RGBA, gl.UNSIGNED_BYTE, arr, 0)
		} else {
			// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
			//                    GLsizei width, GLsizei height,
			//                    GLenum format, GLenum type, ArrayBufferView? pixels);
			c.gl.texSubImage2D.Invoke(gl.TEXTURE_2D, 0, a.X, a.Y, a.Width, a.Height, gl.RGBA, gl.UNSIGNED_BYTE, arr)
		}
	}
}

func (c *context) enableStencilTest() {
	c.gl.enable.Invoke(gl.STENCIL_TEST)
}

func (c *context) disableStencilTest() {
	c.gl.disable.Invoke(gl.STENCIL_TEST)
}

func (c *context) beginStencilWithEvenOddRule() {
	c.gl.clear.Invoke(gl.STENCIL_BUFFER_BIT)
	c.gl.stencilFunc.Invoke(gl.ALWAYS, 0x00, 0xff)
	c.gl.stencilOp.Invoke(gl.KEEP, gl.KEEP, gl.INVERT)
	c.gl.colorMask.Invoke(false, false, false, false)
}

func (c *context) endStencilWithEvenOddRule() {
	c.gl.stencilFunc.Invoke(gl.NOTEQUAL, 0x00, 0xff)
	c.gl.stencilOp.Invoke(gl.KEEP, gl.KEEP, gl.KEEP)
	c.gl.colorMask.Invoke(true, true, true, true)
}

func (c *context) isES() bool {
	// WebGL is compatible with GLES.
	return true
}
