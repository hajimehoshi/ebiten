// Copyright 2016 Hajime Hoshi
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

//go:build !playstation5

package opengl

import (
	"errors"
	"fmt"
	"image"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/glsl"
)

type blendFactor int

type blendOperation int

func convertBlendFactor(f graphicsdriver.BlendFactor) blendFactor {
	switch f {
	case graphicsdriver.BlendFactorZero:
		return gl.ZERO
	case graphicsdriver.BlendFactorOne:
		return gl.ONE
	case graphicsdriver.BlendFactorSourceColor:
		return gl.SRC_COLOR
	case graphicsdriver.BlendFactorOneMinusSourceColor:
		return gl.ONE_MINUS_SRC_COLOR
	case graphicsdriver.BlendFactorSourceAlpha:
		return gl.SRC_ALPHA
	case graphicsdriver.BlendFactorOneMinusSourceAlpha:
		return gl.ONE_MINUS_SRC_ALPHA
	case graphicsdriver.BlendFactorDestinationColor:
		return gl.DST_COLOR
	case graphicsdriver.BlendFactorOneMinusDestinationColor:
		return gl.ONE_MINUS_DST_COLOR
	case graphicsdriver.BlendFactorDestinationAlpha:
		return gl.DST_ALPHA
	case graphicsdriver.BlendFactorOneMinusDestinationAlpha:
		return gl.ONE_MINUS_DST_ALPHA
	case graphicsdriver.BlendFactorSourceAlphaSaturated:
		return gl.SRC_ALPHA_SATURATE
	default:
		panic(fmt.Sprintf("opengl: invalid blend factor %d", f))
	}
}

func convertBlendOperation(o graphicsdriver.BlendOperation) blendOperation {
	switch o {
	case graphicsdriver.BlendOperationAdd:
		return gl.FUNC_ADD
	case graphicsdriver.BlendOperationSubtract:
		return gl.FUNC_SUBTRACT
	case graphicsdriver.BlendOperationReverseSubtract:
		return gl.FUNC_REVERSE_SUBTRACT
	case graphicsdriver.BlendOperationMin:
		return gl.MIN
	case graphicsdriver.BlendOperationMax:
		return gl.MAX
	default:
		panic(fmt.Sprintf("opengl: invalid blend operation %d", o))
	}
}

type (
	textureNative      uint32
	renderbufferNative uint32
	framebufferNative  uint32
	shader             uint32
	program            uint32
	buffer             uint32
)

type (
	uniformLocation int32
)

const (
	invalidFramebuffer = (1 << 32) - 1
	invalidUniform     = -1
)

type context struct {
	ctx gl.Context

	locationCache                   *locationCache
	screenFramebuffer               framebufferNative // This might not be the default frame buffer '0' (e.g. iOS).
	lastFramebuffer                 framebufferNative
	lastTexture                     textureNative
	lastRenderbuffer                renderbufferNative
	lastViewportWidth               int
	lastViewportHeight              int
	lastBlend                       graphicsdriver.Blend
	maxTextureSize                  int
	maxTextureSizeOnce              sync.Once
	initOnce                        sync.Once
	hasKHRParallelShaderCompile     bool
	hasKHRParallelShaderCompileOnce sync.Once
}

func (c *context) bindTexture(t textureNative) {
	if c.lastTexture == t {
		return
	}
	c.ctx.BindTexture(gl.TEXTURE_2D, uint32(t))
	c.lastTexture = t
}

func (c *context) bindRenderbuffer(r renderbufferNative) {
	if c.lastRenderbuffer == r {
		return
	}
	c.ctx.BindRenderbuffer(gl.RENDERBUFFER, uint32(r))
	c.lastRenderbuffer = r
}

func (c *context) bindFramebuffer(f framebufferNative) {
	if c.lastFramebuffer == f {
		return
	}
	c.ctx.BindFramebuffer(gl.FRAMEBUFFER, uint32(f))
	c.lastFramebuffer = f
}

func (c *context) setViewport(f *framebuffer) {
	c.bindFramebuffer(f.native)
	if c.lastViewportWidth == f.viewportWidth && c.lastViewportHeight == f.viewportHeight {
		return
	}

	// On some environments, viewport size must be within the framebuffer size.
	// e.g. Edge (#71), Chrome on GPD Pocket (#420), macOS Mojave (#691).
	// Use the same size of the framebuffer here.
	c.ctx.Viewport(0, 0, int32(f.viewportWidth), int32(f.viewportHeight))

	// glViewport must be called at least at every frame on iOS.
	// As the screen framebuffer is the last render target, next SetViewport should be
	// the first call at a frame.
	if f.native == c.screenFramebuffer {
		c.lastViewportWidth = 0
		c.lastViewportHeight = 0
	} else {
		c.lastViewportWidth = f.viewportWidth
		c.lastViewportHeight = f.viewportHeight
	}
}

func (c *context) newScreenFramebuffer(width, height int) *framebuffer {
	return &framebuffer{
		native:         c.screenFramebuffer,
		viewportWidth:  width,
		viewportHeight: height,
	}
}

func (c *context) getMaxTextureSize() int {
	c.maxTextureSizeOnce.Do(func() {
		c.maxTextureSize = c.ctx.GetInteger(gl.MAX_TEXTURE_SIZE)
	})
	return c.maxTextureSize
}

func (c *context) reset() error {
	var err1 error
	c.initOnce.Do(func() {
		// Load OpenGL functions after WGL is initialized especially for Windows (#2452).
		if err := c.ctx.LoadFunctions(); err != nil {
			err1 = err
			return
		}
	})
	if err1 != nil {
		return err1
	}

	c.locationCache = newLocationCache()
	c.lastTexture = 0
	c.lastFramebuffer = invalidFramebuffer
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastBlend = graphicsdriver.Blend{}

	c.ctx.Enable(gl.BLEND)
	c.ctx.Enable(gl.SCISSOR_TEST)
	c.blend(graphicsdriver.BlendSourceOver)
	c.screenFramebuffer = framebufferNative(c.ctx.GetInteger(gl.FRAMEBUFFER_BINDING))
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

func (c *context) newTexture(width, height int) (textureNative, error) {
	t := c.ctx.CreateTexture()
	if t <= 0 {
		return 0, errors.New("opengl: creating texture failed")
	}
	c.bindTexture(textureNative(t))

	c.ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	c.ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	c.ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	c.ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	c.ctx.PixelStorei(gl.UNPACK_ALIGNMENT, 4)

	// Firefox warns the usage of textures without specifying pixels (#629, #2077)
	//
	//     Error: WebGL warning: drawElements: This operation requires zeroing texture data. This is slow.
	//
	// In Ebitengine, textures are filled with pixels later by the filter that ignores destination, so it is fine
	// to leave textures as uninitialized here. Rather, extra memory allocating for initialization should be
	// avoided.
	//
	// See also https://stackoverflow.com/questions/57734645.
	c.ctx.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, nil)

	return textureNative(t), nil
}

func (c *context) framebufferPixels(buf []byte, f *framebuffer, region image.Rectangle) error {
	if got, want := len(buf), 4*region.Dx()*region.Dy(); got != want {
		return fmt.Errorf("opengl: len(buf) must be %d but was %d at framebufferPixels", got, want)
	}

	c.ctx.Flush()
	c.bindFramebuffer(f.native)
	x := int32(region.Min.X)
	y := int32(region.Min.Y)
	width := int32(region.Dx())
	height := int32(region.Dy())
	c.ctx.ReadPixels(buf, x, y, width, height, gl.RGBA, gl.UNSIGNED_BYTE)
	return nil
}

func (c *context) framebufferPixelsToBuffer(f *framebuffer, buffer buffer, width, height int) {
	c.ctx.Flush()

	c.bindFramebuffer(f.native)

	c.ctx.BindBuffer(gl.PIXEL_PACK_BUFFER, uint32(buffer))
	c.ctx.ReadPixels(nil, 0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE)
	c.ctx.BindBuffer(gl.PIXEL_PACK_BUFFER, 0)
}

func (c *context) deleteTexture(t textureNative) {
	if c.lastTexture == t {
		c.lastTexture = 0
	}
	c.ctx.DeleteTexture(uint32(t))
}

func (c *context) newRenderbuffer(width, height int) (renderbufferNative, error) {
	r := c.ctx.CreateRenderbuffer()
	if r <= 0 {
		return 0, errors.New("opengl: creating renderbuffer failed")
	}

	renderbuffer := renderbufferNative(r)
	c.bindRenderbuffer(renderbuffer)

	var stencilFormat uint32
	if c.ctx.IsES() {
		// https://docs.gl/es2/glRenderbufferStorage
		// > Must be one of the following symbolic constants: GL_RGBA4, GL_RGB565, GL_RGB5_A1,
		// > GL_DEPTH_COMPONENT16, or GL_STENCIL_INDEX8.
		//
		// https://developer.mozilla.org/en-US/docs/Web/API/WebGLRenderingContext/renderbufferStorage
		// > A GLenum specifying the internal format of the renderbuffer. Possible values:
		// > * gl.RGBA4: 4 red bits, 4 green bits, 4 blue bits 4 alpha bits.
		// > * gl.RGB565: 5 red bits, 6 green bits, 5 blue bits.
		// > * gl.RGB5_A1: 5 red bits, 5 green bits, 5 blue bits, 1 alpha bit.
		// > * gl.DEPTH_COMPONENT16: 16 depth bits.
		// > * gl.STENCIL_INDEX8: 8 stencil bits.
		// > * gl.DEPTH_STENCIL
		stencilFormat = gl.STENCIL_INDEX8
	} else {
		// GL_STENCIL_INDEX8 might not be available with OpenGL 2.1.
		// https://www.khronos.org/opengl/wiki/Image_Format
		// > There are only 2 depth/stencil formats, each providing 8 stencil bits: GL_DEPTH24_STENCIL8 and GL_DEPTH32F_STENCIL8.
		// > [...]
		// > Stencil formats can only be used for Textures if OpenGL 4.4 or ARB_texture_stencil8 is available.
		stencilFormat = gl.DEPTH24_STENCIL8
	}
	c.ctx.RenderbufferStorage(gl.RENDERBUFFER, stencilFormat, int32(width), int32(height))

	return renderbuffer, nil
}

func (c *context) deleteRenderbuffer(r renderbufferNative) {
	if c.lastRenderbuffer == r {
		c.lastRenderbuffer = 0
	}
	c.ctx.DeleteRenderbuffer(uint32(r))
}

func (c *context) newFramebuffer(texture textureNative, width, height int) (*framebuffer, error) {
	f := c.ctx.CreateFramebuffer()
	if f <= 0 {
		return nil, fmt.Errorf("opengl: creating framebuffer failed: the returned value is not positive but %d", f)
	}
	c.bindFramebuffer(framebufferNative(f))

	c.ctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, uint32(texture), 0)

	if shouldCheckFramebufferStatus() {
		if s := c.ctx.CheckFramebufferStatus(gl.FRAMEBUFFER); s != gl.FRAMEBUFFER_COMPLETE {
			if s != 0 {
				return nil, fmt.Errorf("opengl: creating framebuffer failed: %v", s)
			}
			if e := c.ctx.GetError(); e != gl.NO_ERROR {
				return nil, fmt.Errorf("opengl: creating framebuffer failed: (glGetError) %d", e)
			}
			return nil, fmt.Errorf("opengl: creating framebuffer failed: unknown error")
		}
	}

	return &framebuffer{
		native:         framebufferNative(f),
		viewportWidth:  width,
		viewportHeight: height,
	}, nil
}

func (c *context) bindStencilBuffer(f framebufferNative, r renderbufferNative) error {
	c.bindFramebuffer(f)

	c.ctx.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.STENCIL_ATTACHMENT, gl.RENDERBUFFER, uint32(r))

	if shouldCheckFramebufferStatus() {
		if s := c.ctx.CheckFramebufferStatus(gl.FRAMEBUFFER); s != gl.FRAMEBUFFER_COMPLETE {
			return fmt.Errorf("opengl: glFramebufferRenderbuffer failed: %d", s)
		}
	}

	return nil
}

func (c *context) deleteFramebuffer(f framebufferNative) {
	if f == c.screenFramebuffer {
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
	c.ctx.DeleteFramebuffer(uint32(f))
}

func (c *context) newShader(shaderType uint32, source string) (shader, error) {
	s := c.ctx.CreateShader(shaderType)
	if s == 0 {
		return 0, fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
	}

	c.ctx.ShaderSource(s, source)
	c.ctx.CompileShader(s)

	return shader(s), nil
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
	return program(p), nil
}

func (c *context) deleteProgram(p program) {
	c.locationCache.deleteProgram(p)

	if !c.ctx.IsProgram(uint32(p)) {
		return
	}
	c.ctx.DeleteProgram(uint32(p))
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
	case shaderir.Bool:
		c.ctx.Uniform1iv(int32(l), uint32sToInt32s(v))
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
	case shaderir.IVec2:
		c.ctx.Uniform2iv(int32(l), uint32sToInt32s(v))
	case shaderir.IVec3:
		c.ctx.Uniform3iv(int32(l), uint32sToInt32s(v))
	case shaderir.IVec4:
		c.ctx.Uniform4iv(int32(l), uint32sToInt32s(v))
	case shaderir.Mat2:
		c.ctx.UniformMatrix2fv(int32(l), uint32sToFloat32s(v))
	case shaderir.Mat3:
		c.ctx.UniformMatrix3fv(int32(l), uint32sToFloat32s(v))
	case shaderir.Mat4:
		c.ctx.UniformMatrix4fv(int32(l), uint32sToFloat32s(v))
	default:
		panic(fmt.Sprintf("opengl: unexpected type: %s", typ.String()))
	}
	return true
}

func (c *context) newArrayBuffer(size int) buffer {
	b := c.ctx.CreateBuffer()
	c.ctx.BindBuffer(gl.ARRAY_BUFFER, b)
	c.ctx.BufferInit(gl.ARRAY_BUFFER, size, gl.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) newElementArrayBuffer(size int) buffer {
	b := c.ctx.CreateBuffer()
	c.ctx.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, b)
	c.ctx.BufferInit(gl.ELEMENT_ARRAY_BUFFER, size, gl.DYNAMIC_DRAW)
	return buffer(b)
}

func (c *context) glslVersion() glsl.GLSLVersion {
	if c.ctx.IsES() {
		return glsl.GLSLVersionES300
	}
	return glsl.GLSLVersionDefault
}

func shouldCheckFramebufferStatus() bool {
	// CheckFramebufferStatus is slow and should be avoided especially in browsers.
	// See https://developer.mozilla.org/en-US/docs/Web/API/WebGL_API/WebGL_best_practices#avoid_blocking_api_calls_in_production
	//
	// TODO: Should this be avoided in all environments?
	return runtime.GOOS != "js"
}

func (c *context) hasParallelShaderCompile() bool {
	c.hasKHRParallelShaderCompileOnce.Do(func() {
		if runtime.GOOS != "js" {
			return
		}
		ext := c.ctx.GetExtension("KHR_parallel_shader_compile")
		c.hasKHRParallelShaderCompile = ext != nil
	})
	return c.hasKHRParallelShaderCompile
}
