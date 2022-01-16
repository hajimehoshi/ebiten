// Copyright 2020 The Ebiten Authors
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

//go:build android || ios
// +build android ios

package gles

import (
	"golang.org/x/mobile/gl"
)

type GomobileContext struct {
	ctx gl.Context
}

func gmProgram(program uint32) gl.Program {
	return gl.Program{
		Init:  true,
		Value: program,
	}
}

func NewGomobileContext(ctx gl.Context) *GomobileContext {
	return &GomobileContext{ctx}
}

func (g *GomobileContext) ActiveTexture(texture uint32) {
	g.ctx.ActiveTexture(gl.Enum(texture))
}

func (g *GomobileContext) AttachShader(program uint32, shader uint32) {
	g.ctx.AttachShader(gmProgram(program), gl.Shader{Value: shader})
}

func (g *GomobileContext) BindAttribLocation(program uint32, index uint32, name string) {
	g.ctx.BindAttribLocation(gmProgram(program), gl.Attrib{Value: uint(index)}, name)
}

func (g *GomobileContext) BindBuffer(target uint32, buffer uint32) {
	g.ctx.BindBuffer(gl.Enum(target), gl.Buffer{Value: buffer})
}

func (g *GomobileContext) BindFramebuffer(target uint32, framebuffer uint32) {
	g.ctx.BindFramebuffer(gl.Enum(target), gl.Framebuffer{Value: framebuffer})
}

func (g *GomobileContext) BindRenderbuffer(target uint32, renderbuffer uint32) {
	g.ctx.BindRenderbuffer(gl.Enum(target), gl.Renderbuffer{Value: renderbuffer})
}

func (g *GomobileContext) BindTexture(target uint32, texture uint32) {
	g.ctx.BindTexture(gl.Enum(target), gl.Texture{Value: texture})
}

func (g *GomobileContext) BlendFunc(sfactor uint32, dfactor uint32) {
	g.ctx.BlendFunc(gl.Enum(sfactor), gl.Enum(dfactor))
}

func (g *GomobileContext) BufferData(target uint32, size int, data []byte, usage uint32) {
	if data == nil {
		g.ctx.BufferInit(gl.Enum(target), size, gl.Enum(usage))
	} else {
		if size != len(data) {
			panic("gles: size and len(data) must be same at BufferData")
		}
		g.ctx.BufferData(gl.Enum(target), data, gl.Enum(usage))
	}
}

func (g *GomobileContext) BufferSubData(target uint32, offset int, data []byte) {
	g.ctx.BufferSubData(gl.Enum(target), offset, data)
}

func (g *GomobileContext) CheckFramebufferStatus(target uint32) uint32 {
	return uint32(g.ctx.CheckFramebufferStatus(gl.Enum(target)))
}

func (g *GomobileContext) Clear(mask uint32) {
	g.ctx.Clear(gl.Enum(mask))
}

func (g *GomobileContext) ColorMask(red, green, blue, alpha bool) {
	g.ctx.ColorMask(red, green, blue, alpha)
}

func (g *GomobileContext) CompileShader(shader uint32) {
	g.ctx.CompileShader(gl.Shader{Value: shader})
}

func (g *GomobileContext) CreateProgram() uint32 {
	return g.ctx.CreateProgram().Value
}

func (g *GomobileContext) CreateShader(xtype uint32) uint32 {
	return g.ctx.CreateShader(gl.Enum(xtype)).Value
}

func (g *GomobileContext) DeleteBuffers(buffers []uint32) {
	for _, b := range buffers {
		g.ctx.DeleteBuffer(gl.Buffer{Value: b})
	}
}

func (g *GomobileContext) DeleteFramebuffers(framebuffers []uint32) {
	for _, b := range framebuffers {
		g.ctx.DeleteFramebuffer(gl.Framebuffer{Value: b})
	}
}

func (g *GomobileContext) DeleteProgram(program uint32) {
	g.ctx.DeleteProgram(gmProgram(program))
}

func (g *GomobileContext) DeleteRenderbuffers(renderbuffers []uint32) {
	for _, r := range renderbuffers {
		g.ctx.DeleteRenderbuffer(gl.Renderbuffer{Value: r})
	}
}

func (g *GomobileContext) DeleteShader(shader uint32) {
	g.ctx.DeleteShader(gl.Shader{Value: shader})
}

func (g *GomobileContext) DeleteTextures(textures []uint32) {
	for _, t := range textures {
		g.ctx.DeleteTexture(gl.Texture{Value: t})
	}
}

func (g *GomobileContext) Disable(cap uint32) {
	g.ctx.Disable(gl.Enum(cap))
}

func (g *GomobileContext) DisableVertexAttribArray(index uint32) {
	g.ctx.DisableVertexAttribArray(gl.Attrib{Value: uint(index)})
}

func (g *GomobileContext) DrawElements(mode uint32, count int32, xtype uint32, offset int) {
	g.ctx.DrawElements(gl.Enum(mode), int(count), gl.Enum(xtype), offset)
}

func (g *GomobileContext) Enable(cap uint32) {
	g.ctx.Enable(gl.Enum(cap))
}

func (g *GomobileContext) EnableVertexAttribArray(index uint32) {
	g.ctx.EnableVertexAttribArray(gl.Attrib{Value: uint(index)})
}

func (g *GomobileContext) Flush() {
	g.ctx.Flush()
}

func (g *GomobileContext) FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	g.ctx.FramebufferRenderbuffer(gl.Enum(target), gl.Enum(attachment), gl.Enum(renderbuffertarget), gl.Renderbuffer{Value: renderbuffer})
}

func (g *GomobileContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	g.ctx.FramebufferTexture2D(gl.Enum(target), gl.Enum(attachment), gl.Enum(textarget), gl.Texture{Value: texture}, int(level))
}

func (g *GomobileContext) GenBuffers(n int32) []uint32 {
	buffers := make([]uint32, n)
	for i := range buffers {
		buffers[i] = g.ctx.CreateBuffer().Value
	}
	return buffers
}

func (g *GomobileContext) GenFramebuffers(n int32) []uint32 {
	framebuffers := make([]uint32, n)
	for i := range framebuffers {
		framebuffers[i] = g.ctx.CreateFramebuffer().Value
	}
	return framebuffers
}

func (g *GomobileContext) GenRenderbuffers(n int32) []uint32 {
	renderbuffers := make([]uint32, n)
	for i := range renderbuffers {
		renderbuffers[i] = g.ctx.CreateRenderbuffer().Value
	}
	return renderbuffers
}

func (g *GomobileContext) GenTextures(n int32) []uint32 {
	textures := make([]uint32, n)
	for i := range textures {
		textures[i] = g.ctx.CreateTexture().Value
	}
	return textures
}

func (g *GomobileContext) GetError() uint32 {
	return uint32(g.ctx.GetError())
}

func (g *GomobileContext) GetIntegerv(dst []int32, pname uint32) {
	g.ctx.GetIntegerv(dst, gl.Enum(pname))
}

func (g *GomobileContext) GetProgramiv(dst []int32, program uint32, pname uint32) {
	dst[0] = int32(g.ctx.GetProgrami(gmProgram(program), gl.Enum(pname)))
}

func (g *GomobileContext) GetProgramInfoLog(program uint32) string {
	return g.ctx.GetProgramInfoLog(gmProgram(program))
}

func (g *GomobileContext) GetShaderiv(dst []int32, shader uint32, pname uint32) {
	dst[0] = int32(g.ctx.GetShaderi(gl.Shader{Value: shader}, gl.Enum(pname)))
}

func (g *GomobileContext) GetShaderInfoLog(shader uint32) string {
	return g.ctx.GetShaderInfoLog(gl.Shader{Value: shader})
}

func (g *GomobileContext) GetShaderPrecisionFormat(shadertype uint32, precisiontype uint32) (rangeLow, rangeHigh, precision int) {
	return g.ctx.GetShaderPrecisionFormat(gl.Enum(shadertype), gl.Enum(precisiontype))
}

func (g *GomobileContext) GetUniformLocation(program uint32, name string) int32 {
	return g.ctx.GetUniformLocation(gmProgram(program), name).Value
}

func (g *GomobileContext) IsFramebuffer(framebuffer uint32) bool {
	return g.ctx.IsFramebuffer(gl.Framebuffer{Value: framebuffer})
}

func (g *GomobileContext) IsProgram(program uint32) bool {
	return g.ctx.IsProgram(gmProgram(program))
}

func (g *GomobileContext) IsRenderbuffer(renderbuffer uint32) bool {
	return g.ctx.IsRenderbuffer(gl.Renderbuffer{Value: renderbuffer})
}

func (g *GomobileContext) IsTexture(texture uint32) bool {
	return g.ctx.IsTexture(gl.Texture{Value: texture})
}

func (g *GomobileContext) LinkProgram(program uint32) {
	g.ctx.LinkProgram(gmProgram(program))
}

func (g *GomobileContext) PixelStorei(pname uint32, param int32) {
	g.ctx.PixelStorei(gl.Enum(pname), param)
}

func (g *GomobileContext) ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32) {
	g.ctx.ReadPixels(dst, int(x), int(y), int(width), int(height), gl.Enum(format), gl.Enum(xtype))
}

func (g *GomobileContext) RenderbufferStorage(target uint32, internalFormat uint32, width int32, height int32) {
	g.ctx.RenderbufferStorage(gl.Enum(target), gl.Enum(internalFormat), int(width), int(height))
}

func (g *GomobileContext) Scissor(x, y, width, height int32) {
	g.ctx.Scissor(x, y, width, height)
}

func (g *GomobileContext) ShaderSource(shader uint32, xstring string) {
	g.ctx.ShaderSource(gl.Shader{Value: shader}, xstring)
}

func (g *GomobileContext) StencilFunc(func_ uint32, ref int32, mask uint32) {
	g.ctx.StencilFunc(gl.Enum(func_), int(ref), mask)
}

func (g *GomobileContext) StencilOp(sfail, dpfail, dppass uint32) {
	g.ctx.StencilOp(gl.Enum(sfail), gl.Enum(dpfail), gl.Enum(dppass))
}

func (g *GomobileContext) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	g.ctx.TexImage2D(gl.Enum(target), int(level), int(internalformat), int(width), int(height), gl.Enum(format), gl.Enum(xtype), pixels)
}

func (g *GomobileContext) TexParameteri(target uint32, pname uint32, param int32) {
	g.ctx.TexParameteri(gl.Enum(target), gl.Enum(pname), int(param))
}

func (g *GomobileContext) TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	g.ctx.TexSubImage2D(gl.Enum(target), int(level), int(xoffset), int(yoffset), int(width), int(height), gl.Enum(format), gl.Enum(xtype), pixels)
}

func (g *GomobileContext) Uniform1f(location int32, v0 float32) {
	g.ctx.Uniform1f(gl.Uniform{Value: location}, v0)
}

func (g *GomobileContext) Uniform1fv(location int32, value []float32) {
	g.ctx.Uniform1fv(gl.Uniform{Value: location}, value)
}

func (g *GomobileContext) Uniform1i(location int32, v0 int32) {
	g.ctx.Uniform1i(gl.Uniform{Value: location}, int(v0))
}

func (g *GomobileContext) Uniform2fv(location int32, value []float32) {
	g.ctx.Uniform2fv(gl.Uniform{Value: location}, value)
}

func (g *GomobileContext) Uniform3fv(location int32, value []float32) {
	g.ctx.Uniform3fv(gl.Uniform{Value: location}, value)
}

func (g *GomobileContext) Uniform4fv(location int32, value []float32) {
	g.ctx.Uniform4fv(gl.Uniform{Value: location}, value)
}

func (g *GomobileContext) UniformMatrix2fv(location int32, transpose bool, value []float32) {
	if transpose {
		panic("gles: UniformMatrix2fv with transpose is not implemented")
	}
	g.ctx.UniformMatrix2fv(gl.Uniform{Value: location}, value)
}

func (g *GomobileContext) UniformMatrix3fv(location int32, transpose bool, value []float32) {
	if transpose {
		panic("gles: UniformMatrix3fv with transpose is not implemented")
	}
	g.ctx.UniformMatrix3fv(gl.Uniform{Value: location}, value)
}

func (g *GomobileContext) UniformMatrix4fv(location int32, transpose bool, value []float32) {
	if transpose {
		panic("gles: UniformMatrix4fv with transpose is not implemented")
	}
	g.ctx.UniformMatrix4fv(gl.Uniform{Value: location}, value)
}

func (g *GomobileContext) UseProgram(program uint32) {
	g.ctx.UseProgram(gmProgram(program))
}

func (g *GomobileContext) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int) {
	g.ctx.VertexAttribPointer(gl.Attrib{Value: uint(index)}, int(size), gl.Enum(xtype), normalized, int(stride), int(offset))
}

func (g *GomobileContext) Viewport(x int32, y int32, width int32, height int32) {
	g.ctx.Viewport(int(x), int(y), int(width), int(height))
}
