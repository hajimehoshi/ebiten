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

package opengl

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type blendFactor int

const (
	glDstAlpha         blendFactor = 0x304
	glDstColor         blendFactor = 0x306
	glOne              blendFactor = 1
	glOneMinusDstAlpha blendFactor = 0x305
	glOneMinusDstColor blendFactor = 0x307
	glOneMinusSrcAlpha blendFactor = 0x303
	glOneMinusSrcColor blendFactor = 0x301
	glSrcAlpha         blendFactor = 0x302
	glSrcAlphaSaturate blendFactor = 0x308
	glSrcColor         blendFactor = 0x300
	glZero             blendFactor = 0
)

type blendOperation int

const (
	glFuncAdd             blendOperation = 0x8006
	glFuncReverseSubtract blendOperation = 0x800b
	glFuncSubtract        blendOperation = 0x800a
)

func convertBlendFactor(f graphicsdriver.BlendFactor) blendFactor {
	switch f {
	case graphicsdriver.BlendFactorZero:
		return glZero
	case graphicsdriver.BlendFactorOne:
		return glOne
	case graphicsdriver.BlendFactorSourceColor:
		return glSrcColor
	case graphicsdriver.BlendFactorOneMinusSourceColor:
		return glOneMinusSrcColor
	case graphicsdriver.BlendFactorSourceAlpha:
		return glSrcAlpha
	case graphicsdriver.BlendFactorOneMinusSourceAlpha:
		return glOneMinusSrcAlpha
	case graphicsdriver.BlendFactorDestinationColor:
		return glDstColor
	case graphicsdriver.BlendFactorOneMinusDestinationColor:
		return glOneMinusDstColor
	case graphicsdriver.BlendFactorDestinationAlpha:
		return glDstAlpha
	case graphicsdriver.BlendFactorOneMinusDestinationAlpha:
		return glOneMinusDstAlpha
	case graphicsdriver.BlendFactorSourceAlphaSaturated:
		return glSrcAlphaSaturate
	default:
		panic(fmt.Sprintf("opengl: invalid blend factor %d", f))
	}
}

func convertBlendOperation(o graphicsdriver.BlendOperation) blendOperation {
	switch o {
	case graphicsdriver.BlendOperationAdd:
		return glFuncAdd
	case graphicsdriver.BlendOperationSubtract:
		return glFuncSubtract
	case graphicsdriver.BlendOperationReverseSubtract:
		return glFuncReverseSubtract
	default:
		panic(fmt.Sprintf("opengl: invalid blend operation %d", o))
	}
}

type context struct {
	locationCache      *locationCache
	screenFramebuffer  framebufferNative // This might not be the default frame buffer '0' (e.g. iOS).
	lastFramebuffer    framebufferNative
	lastTexture        textureNative
	lastRenderbuffer   renderbufferNative
	lastViewportWidth  int
	lastViewportHeight int
	lastBlend          graphicsdriver.Blend
	maxTextureSize     int
	maxTextureSizeOnce sync.Once
	highp              bool
	highpOnce          sync.Once

	contextImpl
}

func (c *context) bindTexture(t textureNative) {
	if c.lastTexture.equal(t) {
		return
	}
	c.bindTextureImpl(t)
	c.lastTexture = t
}

func (c *context) bindRenderbuffer(r renderbufferNative) {
	if c.lastRenderbuffer.equal(r) {
		return
	}
	c.bindRenderbufferImpl(r)
	c.lastRenderbuffer = r
}

func (c *context) bindFramebuffer(f framebufferNative) {
	if c.lastFramebuffer.equal(f) {
		return
	}
	c.bindFramebufferImpl(f)
	c.lastFramebuffer = f
}

func (c *context) setViewport(f *framebuffer) {
	c.bindFramebuffer(f.native)
	if c.lastViewportWidth != f.width || c.lastViewportHeight != f.height {
		// On some environments, viewport size must be within the framebuffer size.
		// e.g. Edge (#71), Chrome on GPD Pocket (#420), macOS Mojave (#691).
		// Use the same size of the framebuffer here.
		c.setViewportImpl(f.width, f.height)

		// glViewport must be called at least at every frame on iOS.
		// As the screen framebuffer is the last render target, next SetViewport should be
		// the first call at a frame.
		if f.native.equal(c.screenFramebuffer) {
			c.lastViewportWidth = 0
			c.lastViewportHeight = 0
		} else {
			c.lastViewportWidth = f.width
			c.lastViewportHeight = f.height
		}
	}
}

func (c *context) getScreenFramebuffer() framebufferNative {
	return c.screenFramebuffer
}

func (c *context) getMaxTextureSize() int {
	c.maxTextureSizeOnce.Do(func() {
		c.maxTextureSize = c.maxTextureSizeImpl()
	})
	return c.maxTextureSize
}
