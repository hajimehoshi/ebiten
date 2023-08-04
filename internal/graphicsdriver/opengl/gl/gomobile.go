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

package gl

import (
	"golang.org/x/mobile/gl"
)

type gomobileContext struct {
	ctx gl.Context
}

func gmProgram(program uint32) gl.Program {
	return gl.Program{
		Init:  true,
		Value: program,
	}
}

func NewGomobileContext(ctx gl.Context) Context {
	return &gomobileContext{ctx}
}

func (g *gomobileContext) LoadFunctions() error {
	return nil
}

func (g *gomobileContext) IsES() bool {
	return true
}

func (g *gomobileContext) ActiveTexture(texture uint32) {
	g.ctx.ActiveTexture(gl.Enum(texture))
}

func (g *gomobileContext) AttachShader(program uint32, shader uint32) {
	g.ctx.AttachShader(gmProgram(program), gl.Shader{Value: shader})
}

func (g *gomobileContext) BindAttribLocation(program uint32, index uint32, name string) {
	g.ctx.BindAttribLocation(gmProgram(program), gl.Attrib{Value: uint(index)}, name)
}

func (g *gomobileContext) BindBuffer(target uint32, buffer uint32) {
	g.ctx.BindBuffer(gl.Enum(target), gl.Buffer{Value: buffer})
}

func (g *gomobileContext) BindFramebuffer(target uint32, framebuffer uint32) {
	g.ctx.BindFramebuffer(gl.Enum(target), gl.Framebuffer{Value: framebuffer})
}

func (g *gomobileContext) BindRenderbuffer(target uint32, renderbuffer uint32) {
	g.ctx.BindRenderbuffer(gl.Enum(target), gl.Renderbuffer{Value: renderbuffer})
}

func (g *gomobileContext) BindTexture(target uint32, texture uint32) {
	g.ctx.BindTexture(gl.Enum(target), gl.Texture{Value: texture})
}

func (g *gomobileContext) BindVertexArray(array uint32) {
	g.ctx.BindVertexArray(gl.VertexArray{Value: array})
}

func (g *gomobileContext) BlendEquationSeparate(modeRGB uint32, modeAlpha uint32) {
	g.ctx.BlendEquationSeparate(gl.Enum(modeRGB), gl.Enum(modeAlpha))
}

func (g *gomobileContext) BlendFuncSeparate(srcRGB uint32, dstRGB uint32, srcAlpha uint32, dstAlpha uint32) {
	g.ctx.BlendFuncSeparate(gl.Enum(srcRGB), gl.Enum(dstRGB), gl.Enum(srcAlpha), gl.Enum(dstAlpha))
}

func (g *gomobileContext) BufferInit(target uint32, size int, usage uint32) {
	g.ctx.BufferInit(gl.Enum(target), size, gl.Enum(usage))
}

func (g *gomobileContext) BufferSubData(target uint32, offset int, data []byte) {
	g.ctx.BufferSubData(gl.Enum(target), offset, data)
}

func (g *gomobileContext) CheckFramebufferStatus(target uint32) uint32 {
	return uint32(g.ctx.CheckFramebufferStatus(gl.Enum(target)))
}

func (g *gomobileContext) Clear(mask uint32) {
	g.ctx.Clear(gl.Enum(mask))
}

func (g *gomobileContext) ColorMask(red, green, blue, alpha bool) {
	g.ctx.ColorMask(red, green, blue, alpha)
}

func (g *gomobileContext) CompileShader(shader uint32) {
	g.ctx.CompileShader(gl.Shader{Value: shader})
}

func (g *gomobileContext) CreateBuffer() uint32 {
	return g.ctx.CreateBuffer().Value
}

func (g *gomobileContext) CreateFramebuffer() uint32 {
	return g.ctx.CreateFramebuffer().Value
}

func (g *gomobileContext) CreateProgram() uint32 {
	return g.ctx.CreateProgram().Value
}

func (g *gomobileContext) CreateRenderbuffer() uint32 {
	return g.ctx.CreateRenderbuffer().Value
}

func (g *gomobileContext) CreateShader(xtype uint32) uint32 {
	return g.ctx.CreateShader(gl.Enum(xtype)).Value
}

func (g *gomobileContext) CreateTexture() uint32 {
	return g.ctx.CreateTexture().Value
}

func (g *gomobileContext) CreateVertexArray() uint32 {
	return g.ctx.CreateVertexArray().Value
}

func (g *gomobileContext) DeleteBuffer(buffer uint32) {
	g.ctx.DeleteBuffer(gl.Buffer{Value: buffer})
}

func (g *gomobileContext) DeleteFramebuffer(framebuffer uint32) {
	g.ctx.DeleteFramebuffer(gl.Framebuffer{Value: framebuffer})
}

func (g *gomobileContext) DeleteProgram(program uint32) {
	g.ctx.DeleteProgram(gmProgram(program))
}

func (g *gomobileContext) DeleteRenderbuffer(renderbuffer uint32) {
	g.ctx.DeleteRenderbuffer(gl.Renderbuffer{Value: renderbuffer})
}

func (g *gomobileContext) DeleteShader(shader uint32) {
	g.ctx.DeleteShader(gl.Shader{Value: shader})
}

func (g *gomobileContext) DeleteTexture(texture uint32) {
	g.ctx.DeleteTexture(gl.Texture{Value: texture})
}

func (g *gomobileContext) DeleteVertexArray(texture uint32) {
	g.ctx.DeleteVertexArray(gl.VertexArray{Value: texture})
}

func (g *gomobileContext) Disable(cap uint32) {
	g.ctx.Disable(gl.Enum(cap))
}

func (g *gomobileContext) DisableVertexAttribArray(index uint32) {
	g.ctx.DisableVertexAttribArray(gl.Attrib{Value: uint(index)})
}

func (g *gomobileContext) DrawElements(mode uint32, count int32, xtype uint32, offset int) {
	g.ctx.DrawElements(gl.Enum(mode), int(count), gl.Enum(xtype), offset)
}

func (g *gomobileContext) Enable(cap uint32) {
	g.ctx.Enable(gl.Enum(cap))
}

func (g *gomobileContext) EnableVertexAttribArray(index uint32) {
	g.ctx.EnableVertexAttribArray(gl.Attrib{Value: uint(index)})
}

func (g *gomobileContext) Flush() {
	g.ctx.Flush()
}

func (g *gomobileContext) FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	g.ctx.FramebufferRenderbuffer(gl.Enum(target), gl.Enum(attachment), gl.Enum(renderbuffertarget), gl.Renderbuffer{Value: renderbuffer})
}

func (g *gomobileContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	g.ctx.FramebufferTexture2D(gl.Enum(target), gl.Enum(attachment), gl.Enum(textarget), gl.Texture{Value: texture}, int(level))
}

func (g *gomobileContext) GetError() uint32 {
	return uint32(g.ctx.GetError())
}

func (g *gomobileContext) GetInteger(pname uint32) int {
	return g.ctx.GetInteger(gl.Enum(pname))
}

func (g *gomobileContext) GetProgramInfoLog(program uint32) string {
	return g.ctx.GetProgramInfoLog(gmProgram(program))
}

func (g *gomobileContext) GetProgrami(program uint32, pname uint32) int {
	return g.ctx.GetProgrami(gmProgram(program), gl.Enum(pname))
}

func (g *gomobileContext) GetShaderInfoLog(shader uint32) string {
	return g.ctx.GetShaderInfoLog(gl.Shader{Value: shader})
}

func (g *gomobileContext) GetShaderi(shader uint32, pname uint32) int {
	return g.ctx.GetShaderi(gl.Shader{Value: shader}, gl.Enum(pname))
}

func (g *gomobileContext) GetUniformLocation(program uint32, name string) int32 {
	return g.ctx.GetUniformLocation(gmProgram(program), name).Value
}

func (g *gomobileContext) IsFramebuffer(framebuffer uint32) bool {
	return g.ctx.IsFramebuffer(gl.Framebuffer{Value: framebuffer})
}

func (g *gomobileContext) IsProgram(program uint32) bool {
	return g.ctx.IsProgram(gmProgram(program))
}

func (g *gomobileContext) IsRenderbuffer(renderbuffer uint32) bool {
	return g.ctx.IsRenderbuffer(gl.Renderbuffer{Value: renderbuffer})
}

func (g *gomobileContext) IsTexture(texture uint32) bool {
	return g.ctx.IsTexture(gl.Texture{Value: texture})
}

func (g *gomobileContext) LinkProgram(program uint32) {
	g.ctx.LinkProgram(gmProgram(program))
}

func (g *gomobileContext) PixelStorei(pname uint32, param int32) {
	g.ctx.PixelStorei(gl.Enum(pname), param)
}

func (g *gomobileContext) ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32) {
	g.ctx.ReadPixels(dst, int(x), int(y), int(width), int(height), gl.Enum(format), gl.Enum(xtype))
}

func (g *gomobileContext) RenderbufferStorage(target uint32, internalFormat uint32, width int32, height int32) {
	g.ctx.RenderbufferStorage(gl.Enum(target), gl.Enum(internalFormat), int(width), int(height))
}

func (g *gomobileContext) Scissor(x, y, width, height int32) {
	g.ctx.Scissor(x, y, width, height)
}

func (g *gomobileContext) ShaderSource(shader uint32, xstring string) {
	g.ctx.ShaderSource(gl.Shader{Value: shader}, xstring)
}

func (g *gomobileContext) StencilFunc(func_ uint32, ref int32, mask uint32) {
	g.ctx.StencilFunc(gl.Enum(func_), int(ref), mask)
}

func (g *gomobileContext) StencilOp(sfail, dpfail, dppass uint32) {
	g.ctx.StencilOp(gl.Enum(sfail), gl.Enum(dpfail), gl.Enum(dppass))
}

func (g *gomobileContext) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	g.ctx.TexImage2D(gl.Enum(target), int(level), int(internalformat), int(width), int(height), gl.Enum(format), gl.Enum(xtype), pixels)
}

func (g *gomobileContext) TexParameteri(target uint32, pname uint32, param int32) {
	g.ctx.TexParameteri(gl.Enum(target), gl.Enum(pname), int(param))
}

func (g *gomobileContext) TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	g.ctx.TexSubImage2D(gl.Enum(target), int(level), int(xoffset), int(yoffset), int(width), int(height), gl.Enum(format), gl.Enum(xtype), pixels)
}

func (g *gomobileContext) Uniform1fv(location int32, value []float32) {
	g.ctx.Uniform1fv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) Uniform1i(location int32, v0 int32) {
	g.ctx.Uniform1i(gl.Uniform{Value: location}, int(v0))
}

func (g *gomobileContext) Uniform1iv(location int32, value []int32) {
	g.ctx.Uniform1iv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) Uniform2fv(location int32, value []float32) {
	g.ctx.Uniform2fv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) Uniform2iv(location int32, value []int32) {
	g.ctx.Uniform2iv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) Uniform3fv(location int32, value []float32) {
	g.ctx.Uniform3fv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) Uniform3iv(location int32, value []int32) {
	g.ctx.Uniform3iv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) Uniform4fv(location int32, value []float32) {
	g.ctx.Uniform4fv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) Uniform4iv(location int32, value []int32) {
	g.ctx.Uniform4iv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) UniformMatrix2fv(location int32, value []float32) {
	g.ctx.UniformMatrix2fv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) UniformMatrix3fv(location int32, value []float32) {
	g.ctx.UniformMatrix3fv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) UniformMatrix4fv(location int32, value []float32) {
	g.ctx.UniformMatrix4fv(gl.Uniform{Value: location}, value)
}

func (g *gomobileContext) UseProgram(program uint32) {
	g.ctx.UseProgram(gmProgram(program))
}

func (g *gomobileContext) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int) {
	g.ctx.VertexAttribPointer(gl.Attrib{Value: uint(index)}, int(size), gl.Enum(xtype), normalized, int(stride), int(offset))
}

func (g *gomobileContext) Viewport(x int32, y int32, width int32, height int32) {
	g.ctx.Viewport(int(x), int(y), int(width), int(height))
}
