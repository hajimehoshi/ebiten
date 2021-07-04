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

// #cgo android CFLAGS:  -Dos_android
// #cgo android LDFLAGS: -lGLESv2
// #cgo ios     CFLAGS:  -Dos_ios
// #cgo ios     LDFLAGS: -framework OpenGLES
//
// #if defined(os_android)
//   #include <GLES2/gl2.h>
// #endif
//
// #if defined(os_ios)
//   #include <OpenGLES/ES2/glext.h>
// #endif
import "C"

import (
	"unsafe"
)

func glBool(x bool) C.GLboolean {
	if x {
		return TRUE
	}
	return FALSE
}

type DefaultContext struct{}

func (DefaultContext) ActiveTexture(texture uint32) {
	C.glActiveTexture(C.GLenum(texture))
}

func (DefaultContext) AttachShader(program uint32, shader uint32) {
	C.glAttachShader(C.GLuint(program), C.GLuint(shader))
}

func (DefaultContext) BindAttribLocation(program uint32, index uint32, name string) {
	s, free := cString(name)
	defer free()
	C.glBindAttribLocation(C.GLuint(program), C.GLuint(index), (*C.GLchar)(unsafe.Pointer(s)))
}

func (DefaultContext) BindBuffer(target uint32, buffer uint32) {
	C.glBindBuffer(C.GLenum(target), C.GLuint(buffer))
}

func (DefaultContext) BindFramebuffer(target uint32, framebuffer uint32) {
	C.glBindFramebuffer(C.GLenum(target), C.GLuint(framebuffer))
}

func (DefaultContext) BindRenderbuffer(target uint32, renderbuffer uint32) {
	C.glBindRenderbuffer(C.GLenum(target), C.GLuint(renderbuffer))
}

func (DefaultContext) BindTexture(target uint32, texture uint32) {
	C.glBindTexture(C.GLenum(target), C.GLuint(texture))
}

func (DefaultContext) BlendFunc(sfactor uint32, dfactor uint32) {
	C.glBlendFunc(C.GLenum(sfactor), C.GLenum(dfactor))
}

func (DefaultContext) BufferData(target uint32, size int, data []byte, usage uint32) {
	var p *byte
	if data != nil {
		p = &data[0]
	}
	C.glBufferData(C.GLenum(target), C.GLsizeiptr(size), unsafe.Pointer(p), C.GLenum(usage))
}

func (DefaultContext) BufferSubData(target uint32, offset int, data []byte) {
	C.glBufferSubData(C.GLenum(target), C.GLintptr(offset), C.GLsizeiptr(len(data)), unsafe.Pointer(&data[0]))
}

func (DefaultContext) CheckFramebufferStatus(target uint32) uint32 {
	return uint32(C.glCheckFramebufferStatus(C.GLenum(target)))
}

func (DefaultContext) Clear(mask uint32) {
	C.glClear(C.GLbitfield(mask))
}

func (DefaultContext) ColorMask(red, green, blue, alpha bool) {
	C.glColorMask(glBool(red), glBool(green), glBool(blue), glBool(alpha))
}

func (DefaultContext) CompileShader(shader uint32) {
	C.glCompileShader(C.GLuint(shader))
}

func (DefaultContext) CreateProgram() uint32 {
	return uint32(C.glCreateProgram())
}

func (DefaultContext) CreateShader(xtype uint32) uint32 {
	return uint32(C.glCreateShader(C.GLenum(xtype)))
}

func (DefaultContext) DeleteBuffers(buffers []uint32) {
	C.glDeleteBuffers(C.GLsizei(len(buffers)), (*C.GLuint)(unsafe.Pointer(&buffers[0])))
}

func (DefaultContext) DeleteFramebuffers(framebuffers []uint32) {
	C.glDeleteFramebuffers(C.GLsizei(len(framebuffers)), (*C.GLuint)(unsafe.Pointer(&framebuffers[0])))
}

func (DefaultContext) DeleteProgram(program uint32) {
	C.glDeleteProgram(C.GLuint(program))
}

func (DefaultContext) DeleteRenderbuffers(renderbuffers []uint32) {
	C.glDeleteRenderbuffers(C.GLsizei(len(renderbuffers)), (*C.GLuint)(unsafe.Pointer(&renderbuffers[0])))
}

func (DefaultContext) DeleteShader(shader uint32) {
	C.glDeleteShader(C.GLuint(shader))
}

func (DefaultContext) DeleteTextures(textures []uint32) {
	C.glDeleteTextures(C.GLsizei(len(textures)), (*C.GLuint)(unsafe.Pointer(&textures[0])))
}

func (DefaultContext) Disable(cap uint32) {
	C.glDisable(C.GLenum(cap))
}

func (DefaultContext) DisableVertexAttribArray(index uint32) {
	C.glDisableVertexAttribArray(C.GLuint(index))
}

func (DefaultContext) DrawElements(mode uint32, count int32, xtype uint32, offset int) {
	C.glDrawElements(C.GLenum(mode), C.GLsizei(count), C.GLenum(xtype), unsafe.Pointer(uintptr(offset)))
}

func (DefaultContext) Enable(cap uint32) {
	C.glEnable(C.GLenum(cap))
}

func (DefaultContext) EnableVertexAttribArray(index uint32) {
	C.glEnableVertexAttribArray(C.GLuint(index))
}

func (DefaultContext) Flush() {
	C.glFlush()
}

func (DefaultContext) FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	C.glFramebufferRenderbuffer(C.GLenum(target), C.GLenum(attachment), C.GLenum(renderbuffertarget), C.GLuint(renderbuffer))
}

func (DefaultContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	C.glFramebufferTexture2D(C.GLenum(target), C.GLenum(attachment), C.GLenum(textarget), C.GLuint(texture), C.GLint(level))
}

func (DefaultContext) GenBuffers(n int32) []uint32 {
	buffers := make([]uint32, n)
	C.glGenBuffers(C.GLsizei(n), (*C.GLuint)(unsafe.Pointer(&buffers[0])))
	return buffers
}

func (DefaultContext) GenFramebuffers(n int32) []uint32 {
	framebuffers := make([]uint32, n)
	C.glGenFramebuffers(C.GLsizei(n), (*C.GLuint)(unsafe.Pointer(&framebuffers[0])))
	return framebuffers
}

func (DefaultContext) GenRenderbuffers(n int32) []uint32 {
	renderbuffers := make([]uint32, n)
	C.glGenRenderbuffers(C.GLsizei(n), (*C.GLuint)(unsafe.Pointer(&renderbuffers[0])))
	return renderbuffers
}

func (DefaultContext) GenTextures(n int32) []uint32 {
	textures := make([]uint32, n)
	C.glGenTextures(C.GLsizei(n), (*C.GLuint)(unsafe.Pointer(&textures[0])))
	return textures
}

func (DefaultContext) GetError() uint32 {
	return uint32(C.glGetError())
}

func (DefaultContext) GetIntegerv(dst []int32, pname uint32) {
	C.glGetIntegerv(C.GLenum(pname), (*C.GLint)(unsafe.Pointer(&dst[0])))
}

func (DefaultContext) GetProgramiv(dst []int32, program uint32, pname uint32) {
	C.glGetProgramiv(C.GLuint(program), C.GLenum(pname), (*C.GLint)(unsafe.Pointer(&dst[0])))
}

func (d DefaultContext) GetProgramInfoLog(program uint32) string {
	buflens := make([]int32, 1)
	d.GetProgramiv(buflens, program, INFO_LOG_LENGTH)
	buflen := buflens[0]
	if buflen == 0 {
		return ""
	}
	buf := make([]byte, buflen)
	var length int32
	C.glGetProgramInfoLog(C.GLuint(program), C.GLsizei(buflen), (*C.GLsizei)(unsafe.Pointer(&length)), (*C.GLchar)(unsafe.Pointer(&buf[0])))
	return string(buf[:length])
}

func (DefaultContext) GetShaderiv(dst []int32, shader uint32, pname uint32) {
	C.glGetShaderiv(C.GLuint(shader), C.GLenum(pname), (*C.GLint)(unsafe.Pointer(&dst[0])))
}

func (d DefaultContext) GetShaderInfoLog(shader uint32) string {
	buflens := make([]int32, 1)
	d.GetShaderiv(buflens, shader, INFO_LOG_LENGTH)
	buflen := buflens[0]
	if buflen == 0 {
		return ""
	}
	buf := make([]byte, buflen)
	var length int32
	C.glGetShaderInfoLog(C.GLuint(shader), C.GLsizei(buflen), (*C.GLsizei)(unsafe.Pointer(&length)), (*C.GLchar)(unsafe.Pointer(&buf[0])))
	return string(buf[:length])
}

func (DefaultContext) GetShaderPrecisionFormat(shadertype uint32, precisiontype uint32) (rangeLow, rangeHigh, precision int) {
	var r [2]int32
	var p int32
	C.glGetShaderPrecisionFormat(C.GLenum(shadertype), C.GLenum(precisiontype), (*C.GLint)(unsafe.Pointer(&r[0])), (*C.GLint)(unsafe.Pointer(&p)))
	return int(r[0]), int(r[1]), int(p)
}

func (DefaultContext) GetUniformLocation(program uint32, name string) int32 {
	s, free := cString(name)
	defer free()
	return int32(C.glGetUniformLocation(C.GLuint(program), (*C.GLchar)(unsafe.Pointer(s))))
}

func (DefaultContext) IsFramebuffer(framebuffer uint32) bool {
	return C.glIsFramebuffer(C.GLuint(framebuffer)) != FALSE
}

func (DefaultContext) IsProgram(program uint32) bool {
	return C.glIsProgram(C.GLuint(program)) != FALSE
}

func (DefaultContext) IsRenderbuffer(renderbuffer uint32) bool {
	return C.glIsRenderbuffer(C.GLuint(renderbuffer)) != FALSE
}

func (DefaultContext) IsTexture(texture uint32) bool {
	return C.glIsTexture(C.GLuint(texture)) != FALSE
}

func (DefaultContext) LinkProgram(program uint32) {
	C.glLinkProgram(C.GLuint(program))
}

func (DefaultContext) PixelStorei(pname uint32, param int32) {
	C.glPixelStorei(C.GLenum(pname), C.GLint(param))
}

func (DefaultContext) ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32) {
	C.glReadPixels(C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height), C.GLenum(format), C.GLenum(xtype), unsafe.Pointer(&dst[0]))
}

func (DefaultContext) RenderbufferStorage(target uint32, internalFormat uint32, width int32, height int32) {
	C.glRenderbufferStorage(C.GLenum(target), C.GLenum(internalFormat), C.GLsizei(width), C.GLsizei(height))
}

func (DefaultContext) Scissor(x, y, width, height int32) {
	C.glScissor(C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height))
}

func (DefaultContext) ShaderSource(shader uint32, xstring string) {
	s, free := cStringPtr(xstring)
	defer free()
	C.glShaderSource(C.GLuint(shader), 1, (**C.GLchar)(unsafe.Pointer(s)), nil)
}

func (DefaultContext) StencilFunc(func_ uint32, ref int32, mask uint32) {
	C.glStencilFunc(C.GLenum(func_), C.GLint(ref), C.GLuint(mask))
}

func (DefaultContext) StencilOp(sfail, dpfail, dppass uint32) {
	C.glStencilOp(C.GLenum(sfail), C.GLenum(dpfail), C.GLenum(dppass))
}

func (DefaultContext) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	var p *byte
	if pixels != nil {
		p = &pixels[0]
	}
	C.glTexImage2D(C.GLenum(target), C.GLint(level), C.GLint(internalformat), C.GLsizei(width), C.GLsizei(height), 0 /* border */, C.GLenum(format), C.GLenum(xtype), unsafe.Pointer(p))
}

func (DefaultContext) TexParameteri(target uint32, pname uint32, param int32) {
	C.glTexParameteri(C.GLenum(target), C.GLenum(pname), C.GLint(param))
}

func (DefaultContext) TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	C.glTexSubImage2D(C.GLenum(target), C.GLint(level), C.GLint(xoffset), C.GLint(yoffset), C.GLsizei(width), C.GLsizei(height), C.GLenum(format), C.GLenum(xtype), unsafe.Pointer(&pixels[0]))
}

func (DefaultContext) Uniform1f(location int32, v0 float32) {
	C.glUniform1f(C.GLint(location), C.GLfloat(v0))
}

func (DefaultContext) Uniform1fv(location int32, value []float32) {
	C.glUniform1fv(C.GLint(location), C.GLsizei(len(value)), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (DefaultContext) Uniform1i(location int32, v0 int32) {
	C.glUniform1i(C.GLint(location), C.GLint(v0))
}

func (DefaultContext) Uniform2fv(location int32, value []float32) {
	C.glUniform2fv(C.GLint(location), C.GLsizei(len(value)/2), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (DefaultContext) Uniform3fv(location int32, value []float32) {
	C.glUniform3fv(C.GLint(location), C.GLsizei(len(value)/3), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (DefaultContext) Uniform4fv(location int32, value []float32) {
	C.glUniform4fv(C.GLint(location), C.GLsizei(len(value)/4), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (DefaultContext) UniformMatrix2fv(location int32, transpose bool, value []float32) {
	C.glUniformMatrix2fv(C.GLint(location), C.GLsizei(len(value)/4), glBool(transpose), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (DefaultContext) UniformMatrix3fv(location int32, transpose bool, value []float32) {
	C.glUniformMatrix3fv(C.GLint(location), C.GLsizei(len(value)/9), glBool(transpose), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (DefaultContext) UniformMatrix4fv(location int32, transpose bool, value []float32) {
	C.glUniformMatrix4fv(C.GLint(location), C.GLsizei(len(value)/16), glBool(transpose), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (DefaultContext) UseProgram(program uint32) {
	C.glUseProgram(C.GLuint(program))
}

func (DefaultContext) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int) {
	C.glVertexAttribPointer(C.GLuint(index), C.GLint(size), C.GLenum(xtype), glBool(normalized), C.GLsizei(stride), unsafe.Pointer(uintptr(offset)))
}

func (DefaultContext) Viewport(x int32, y int32, width int32, height int32) {
	C.glViewport(C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height))
}
