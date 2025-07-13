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

//go:build !playstation5

package gl

//go:generate go run gen.go
//go:generate gofmt -s -w .

// Context is a context for OpenGL (ES) functions.
//
// Context is basically the same as gomobile's gl.Context.
// See https://pkg.go.dev/github.com/ebitengine/gomobile/gl#Context
type Context interface {
	LoadFunctions() error
	IsES() bool

	ActiveTexture(texture uint32)
	AttachShader(program uint32, shader uint32)
	BindAttribLocation(program uint32, index uint32, name string)
	BindBuffer(target uint32, buffer uint32)
	BindFramebuffer(target uint32, framebuffer uint32)
	BindRenderbuffer(target uint32, renderbuffer uint32)
	BindTexture(target uint32, texture uint32)
	BindVertexArray(array uint32)
	BlendEquationSeparate(modeRGB uint32, modeAlpha uint32)
	BlendFuncSeparate(srcRGB uint32, dstRGB uint32, srcAlpha uint32, dstAlpha uint32)
	BufferInit(target uint32, size int, usage uint32)
	BufferSubData(target uint32, offset int, data []byte)
	CheckFramebufferStatus(target uint32) uint32
	Clear(mask uint32)
	ColorMask(red, green, blue, alpha bool)
	CompileShader(shader uint32)
	CreateBuffer() uint32
	CreateFramebuffer() uint32
	CreateProgram() uint32
	CreateRenderbuffer() uint32
	CreateShader(xtype uint32) uint32
	CreateTexture() uint32
	CreateVertexArray() uint32
	DeleteBuffer(buffer uint32)
	DeleteFramebuffer(framebuffer uint32)
	DeleteProgram(program uint32)
	DeleteRenderbuffer(renderbuffer uint32)
	DeleteShader(shader uint32)
	DeleteTexture(texture uint32)
	DeleteVertexArray(array uint32)
	Disable(cap uint32)
	DisableVertexAttribArray(index uint32)
	DrawElements(mode uint32, count int32, xtype uint32, offset int)
	Enable(cap uint32)
	EnableVertexAttribArray(index uint32)
	Flush()
	FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32)
	FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32)
	GetError() uint32
	GetExtension(name string) any
	GetInteger(pname uint32) int
	GetProgramInfoLog(program uint32) string
	GetProgrami(program uint32, pname uint32) int
	GetShaderInfoLog(shader uint32) string
	GetShaderi(shader uint32, pname uint32) int
	GetUniformLocation(program uint32, name string) int32
	IsProgram(program uint32) bool
	LinkProgram(program uint32)
	PixelStorei(pname uint32, param int32)
	ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32)
	RenderbufferStorage(target uint32, internalFormat uint32, width int32, height int32)
	Scissor(x, y, width, height int32)
	ShaderSource(shader uint32, xstring string)
	StencilFunc(func_ uint32, ref int32, mask uint32)
	StencilOpSeparate(face, sfail, dpfail, dppass uint32)
	TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels []byte)
	TexParameteri(target uint32, pname uint32, param int32)
	TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte)
	Uniform1fv(location int32, value []float32)
	Uniform1i(location int32, v0 int32)
	Uniform1iv(location int32, value []int32)
	Uniform2fv(location int32, value []float32)
	Uniform2iv(location int32, value []int32)
	Uniform3fv(location int32, value []float32)
	Uniform3iv(location int32, value []int32)
	Uniform4fv(location int32, value []float32)
	Uniform4iv(location int32, value []int32)
	UniformMatrix2fv(location int32, value []float32)
	UniformMatrix3fv(location int32, value []float32)
	UniformMatrix4fv(location int32, value []float32)
	UseProgram(program uint32)
	VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int)
	Viewport(x int32, y int32, width int32, height int32)
}
