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

type Context struct {
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
	zero               operation
	one                operation
	srcAlpha           operation
	dstAlpha           operation
	oneMinusSrcAlpha   operation
	oneMinusDstAlpha   operation
	locationCache      *locationCache
	lastFramebuffer    Framebuffer
	lastCompositeMode  CompositeMode
	context
}

func (c *Context) bindFramebuffer(f Framebuffer) {
	if c.lastFramebuffer != f {
		c.bindFramebufferImpl(f)
		c.lastFramebuffer = f
	}
}
