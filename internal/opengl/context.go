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

var (
	Nearest            Filter
	Linear             Filter
	VertexShader       ShaderType
	FragmentShader     ShaderType
	ArrayBuffer        BufferType
	ElementArrayBuffer BufferType
	DynamicDraw        BufferUsage
	StaticDraw         BufferUsage
	Triangles          Mode
	Lines              Mode
	Short              DataType
	Float              DataType

	zero             operation
	one              operation
	srcAlpha         operation
	dstAlpha         operation
	oneMinusSrcAlpha operation
	oneMinusDstAlpha operation
)

type Context struct {
	locationCache      *locationCache
	screenFramebuffer  Framebuffer // This might not be the default frame buffer '0' (e.g. iOS).
	lastFramebuffer    Framebuffer
	lastTexture        Texture
	lastViewportWidth  int
	lastViewportHeight int
	lastCompositeMode  CompositeMode
	context
}

var theContext *Context

func GetContext() *Context {
	return theContext
}

func (c *Context) BindTexture(t Texture) error {
	if c.lastTexture == t {
		return nil
	}
	if err := c.bindTextureImpl(t); err != nil {
		return err
	}
	c.lastTexture = t
	return nil
}

func (c *Context) bindFramebuffer(f Framebuffer) error {
	if c.lastFramebuffer == f {
		return nil
	}
	if err := c.bindFramebufferImpl(f); err != nil {
		return err
	}
	c.lastFramebuffer = f
	return nil
}

func (c *Context) SetViewport(f Framebuffer, width, height int) error {
	if err := c.bindFramebuffer(f); err != nil {
		return err
	}
	if c.lastViewportWidth != width || c.lastViewportHeight != height {
		if err := c.setViewportImpl(width, height); err != nil {
			return nil
		}
		c.lastViewportWidth = width
		c.lastViewportHeight = height
	}
	return nil
}

func (c *Context) ScreenFramebuffer() Framebuffer {
	return c.screenFramebuffer
}

func (c *Context) ResetViewportSize() {
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
}
