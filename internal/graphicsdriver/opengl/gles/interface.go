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

type Context interface {
	ActiveTexture(texture uint32)
	AttachShader(program, shader uint32)
	BindAttribLocation(program, index uint32, name string)
	BindBuffer(target, buffer uint32)
	BindFramebuffer(target, framebuffer uint32)
	BindRenderbuffer(target, renderbuffer uint32)
	BindTexture(target, texture uint32)
	BlendFunc(sfactor, dfactor uint32)
	BufferData(target uint32, size int, data []byte, usage uint32)
	BufferSubData(target uint32, offset int, data []byte)
	CheckFramebufferStatus(target uint32) uint32
	Clear(mask uint32)
	ColorMask(red, green, blue, alpha bool)
	CompileShader(shader uint32)
	CreateProgram() uint32
	CreateShader(xtype uint32) uint32
	DeleteBuffers(buffers []uint32)
	DeleteFramebuffers(framebuffers []uint32)
	DeleteProgram(program uint32)
	DeleteRenderbuffers(renderbuffer []uint32)
	DeleteShader(shader uint32)
	DeleteTextures(textures []uint32)
	Disable(cap uint32)
	DisableVertexAttribArray(index uint32)
	DrawElements(mode uint32, count int32, xtype uint32, offset int)
	Enable(cap uint32)
	EnableVertexAttribArray(index uint32)
	Flush()
	FramebufferRenderbuffer(target, attachment, renderbuffertarget, renderbuffer uint32)
	FramebufferTexture2D(target, attachment, textarget, texture uint32, level int32)
	GenBuffers(n int32) []uint32
	GenFramebuffers(n int32) []uint32
	GenRenderbuffers(n int32) []uint32
	GenTextures(n int32) []uint32
	GetError() uint32
	GetIntegerv(dst []int32, pname uint32)
	GetProgramiv(dst []int32, program, pname uint32)
	GetProgramInfoLog(program uint32) string
	GetShaderiv(dst []int32, shader, pname uint32)
	GetShaderInfoLog(shader uint32) string
	GetShaderPrecisionFormat(shadertype, precisiontype uint32) (rangeLow, rangeHigh, precision int)
	GetUniformLocation(program uint32, name string) int32
	IsFramebuffer(framebuffer uint32) bool
	IsProgram(program uint32) bool
	IsRenderbuffer(renderbuffer uint32) bool
	IsTexture(texture uint32) bool
	LinkProgram(program uint32)
	PixelStorei(pname uint32, param int32)
	ReadPixels(dst []byte, x, y, width, height int32, format, xtype uint32)
	RenderbufferStorage(target, internalFormat uint32, width, height int32)
	Scissor(x, y, width, height int32)
	ShaderSource(shader uint32, xstring string)
	StencilFunc(func_ uint32, ref int32, mask uint32)
	StencilOp(sfail, dpfail, dppass uint32)
	TexImage2D(target uint32, level, internalformat, width, height int32, format, xtype uint32, pixels []byte)
	TexParameteri(target, pname uint32, param int32)
	TexSubImage2D(target uint32, level, xoffset, yoffset, width, height int32, format, xtype uint32, pixels []byte)
	Uniform1f(location int32, v0 float32)
	Uniform1fv(location int32, value []float32)
	Uniform1i(location, v0 int32)
	Uniform2fv(location int32, value []float32)
	Uniform3fv(location int32, value []float32)
	Uniform4fv(location int32, value []float32)
	UniformMatrix2fv(location int32, transpose bool, value []float32)
	UniformMatrix3fv(location int32, transpose bool, value []float32)
	UniformMatrix4fv(location int32, transpose bool, value []float32)
	UseProgram(program uint32)
	VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int)
	Viewport(x, y, width, height int32)
}
