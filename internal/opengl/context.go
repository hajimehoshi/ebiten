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
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

var (
	vertexShader       shaderType
	fragmentShader     shaderType
	arrayBuffer        bufferType
	elementArrayBuffer bufferType
	dynamicDraw        bufferUsage
	staticDraw         bufferUsage
	short              dataType
	float              dataType

	zero             operation
	one              operation
	srcAlpha         operation
	dstAlpha         operation
	oneMinusSrcAlpha operation
	oneMinusDstAlpha operation
)

func convertOperation(op graphics.Operation) operation {
	switch op {
	case graphics.Zero:
		return zero
	case graphics.One:
		return one
	case graphics.SrcAlpha:
		return srcAlpha
	case graphics.DstAlpha:
		return dstAlpha
	case graphics.OneMinusSrcAlpha:
		return oneMinusSrcAlpha
	case graphics.OneMinusDstAlpha:
		return oneMinusDstAlpha
	default:
		panic("not reached")
	}
}

// TODO: Unexport this
type Context struct {
	locationCache      *locationCache
	screenFramebuffer  framebufferNative // This might not be the default frame buffer '0' (e.g. iOS).
	lastFramebuffer    framebufferNative
	lastTexture        textureNative
	lastViewportWidth  int
	lastViewportHeight int
	lastCompositeMode  graphics.CompositeMode
	maxTextureSize     int
	context
}

var theContext Context

func (c *Context) bindTexture(t textureNative) {
	if c.lastTexture == t {
		return
	}
	c.bindTextureImpl(t)
	c.lastTexture = t
}

func (c *Context) bindFramebuffer(f framebufferNative) {
	if c.lastFramebuffer == f {
		return
	}
	c.bindFramebufferImpl(f)
	c.lastFramebuffer = f
}

func (c *Context) setViewport(f *framebuffer) {
	c.bindFramebuffer(f.native)
	if c.lastViewportWidth != f.width || c.lastViewportHeight != f.height {
		c.setViewportImpl(f.width, f.height)
		// glViewport must be called at least at every frame on iOS.
		// As the screen framebuffer is the last render target, next SetViewport should be
		// the first call at a frame.
		if f.native == c.screenFramebuffer {
			c.lastViewportWidth = 0
			c.lastViewportHeight = 0
		} else {
			c.lastViewportWidth = f.width
			c.lastViewportHeight = f.height
		}
	}
}

func (c *Context) getScreenFramebuffer() framebufferNative {
	return c.screenFramebuffer
}

func (c *Context) getMaxTextureSize() int {
	if c.maxTextureSize == 0 {
		c.maxTextureSize = c.maxTextureSizeImpl()
	}
	return c.maxTextureSize
}
