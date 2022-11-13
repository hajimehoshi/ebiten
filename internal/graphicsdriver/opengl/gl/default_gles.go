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

//go:build android || ios || opengles

package gl

// #cgo !darwin          CFLAGS:     -Dos_notdarwin
// #cgo darwin           CFLAGS:     -Dos_darwin
// #cgo !android,!darwin pkg-config: glesv2
// #cgo android          LDFLAGS:    -lGLESv2
// #cgo darwin           LDFLAGS:    -framework OpenGLES
//
// #if defined(os_darwin)
//   #define GLES_SILENCE_DEPRECATION
//   #include <OpenGLES/ES2/glext.h>
// #endif
//
// #if defined(os_notdarwin)
//   #include <GLES2/gl2.h>
// #endif
//
// #include <stdlib.h>
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

type defaultContext struct{}

func NewDefaultContext() Context {
	return defaultContext{}
}

func (defaultContext) Init() error {
	return nil
}

func (defaultContext) ActiveTexture(texture uint32) {
	C.glActiveTexture(C.GLenum(texture))
}

func (defaultContext) AttachShader(program uint32, shader uint32) {
	C.glAttachShader(C.GLuint(program), C.GLuint(shader))
}

func (defaultContext) BindAttribLocation(program uint32, index uint32, name string) {
	s := C.CString(name)
	defer C.free(unsafe.Pointer(s))
	C.glBindAttribLocation(C.GLuint(program), C.GLuint(index), (*C.GLchar)(unsafe.Pointer(s)))
}

func (defaultContext) BindBuffer(target uint32, buffer uint32) {
	C.glBindBuffer(C.GLenum(target), C.GLuint(buffer))
}

func (defaultContext) BindFramebuffer(target uint32, framebuffer uint32) {
	C.glBindFramebuffer(C.GLenum(target), C.GLuint(framebuffer))
}

func (defaultContext) BindRenderbuffer(target uint32, renderbuffer uint32) {
	C.glBindRenderbuffer(C.GLenum(target), C.GLuint(renderbuffer))
}

func (defaultContext) BindTexture(target uint32, texture uint32) {
	C.glBindTexture(C.GLenum(target), C.GLuint(texture))
}

func (defaultContext) BlendEquationSeparate(modeRGB uint32, modeAlpha uint32) {
	C.glBlendEquationSeparate(C.GLenum(modeRGB), C.GLenum(modeAlpha))
}

func (defaultContext) BlendFuncSeparate(srcRGB uint32, dstRGB uint32, srcAlpha uint32, dstAlpha uint32) {
	C.glBlendFuncSeparate(C.GLenum(srcRGB), C.GLenum(dstRGB), C.GLenum(srcAlpha), C.GLenum(dstAlpha))
}

func (defaultContext) BufferData(target uint32, size int, data []byte, usage uint32) {
	var p *byte
	if data != nil {
		p = &data[0]
	}
	C.glBufferData(C.GLenum(target), C.GLsizeiptr(size), unsafe.Pointer(p), C.GLenum(usage))
}

func (defaultContext) BufferSubData(target uint32, offset int, data []byte) {
	C.glBufferSubData(C.GLenum(target), C.GLintptr(offset), C.GLsizeiptr(len(data)), unsafe.Pointer(&data[0]))
}

func (defaultContext) CheckFramebufferStatus(target uint32) uint32 {
	return uint32(C.glCheckFramebufferStatus(C.GLenum(target)))
}

func (defaultContext) Clear(mask uint32) {
	C.glClear(C.GLbitfield(mask))
}

func (defaultContext) ColorMask(red, green, blue, alpha bool) {
	C.glColorMask(glBool(red), glBool(green), glBool(blue), glBool(alpha))
}

func (defaultContext) CompileShader(shader uint32) {
	C.glCompileShader(C.GLuint(shader))
}

func (defaultContext) CreateProgram() uint32 {
	return uint32(C.glCreateProgram())
}

func (defaultContext) CreateShader(xtype uint32) uint32 {
	return uint32(C.glCreateShader(C.GLenum(xtype)))
}

func (defaultContext) DeleteBuffers(buffers []uint32) {
	C.glDeleteBuffers(C.GLsizei(len(buffers)), (*C.GLuint)(unsafe.Pointer(&buffers[0])))
}

func (defaultContext) DeleteFramebuffers(framebuffers []uint32) {
	C.glDeleteFramebuffers(C.GLsizei(len(framebuffers)), (*C.GLuint)(unsafe.Pointer(&framebuffers[0])))
}

func (defaultContext) DeleteProgram(program uint32) {
	C.glDeleteProgram(C.GLuint(program))
}

func (defaultContext) DeleteRenderbuffers(renderbuffers []uint32) {
	C.glDeleteRenderbuffers(C.GLsizei(len(renderbuffers)), (*C.GLuint)(unsafe.Pointer(&renderbuffers[0])))
}

func (defaultContext) DeleteShader(shader uint32) {
	C.glDeleteShader(C.GLuint(shader))
}

func (defaultContext) DeleteTextures(textures []uint32) {
	C.glDeleteTextures(C.GLsizei(len(textures)), (*C.GLuint)(unsafe.Pointer(&textures[0])))
}

func (defaultContext) Disable(cap uint32) {
	C.glDisable(C.GLenum(cap))
}

func (defaultContext) DisableVertexAttribArray(index uint32) {
	C.glDisableVertexAttribArray(C.GLuint(index))
}

func (defaultContext) DrawElements(mode uint32, count int32, xtype uint32, offset int) {
	C.glDrawElements(C.GLenum(mode), C.GLsizei(count), C.GLenum(xtype), unsafe.Pointer(uintptr(offset)))
}

func (defaultContext) Enable(cap uint32) {
	C.glEnable(C.GLenum(cap))
}

func (defaultContext) EnableVertexAttribArray(index uint32) {
	C.glEnableVertexAttribArray(C.GLuint(index))
}

func (defaultContext) Flush() {
	C.glFlush()
}

func (defaultContext) FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	C.glFramebufferRenderbuffer(C.GLenum(target), C.GLenum(attachment), C.GLenum(renderbuffertarget), C.GLuint(renderbuffer))
}

func (defaultContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	C.glFramebufferTexture2D(C.GLenum(target), C.GLenum(attachment), C.GLenum(textarget), C.GLuint(texture), C.GLint(level))
}

func (defaultContext) GenBuffers(n int32) []uint32 {
	buffers := make([]uint32, n)
	C.glGenBuffers(C.GLsizei(n), (*C.GLuint)(unsafe.Pointer(&buffers[0])))
	return buffers
}

func (defaultContext) GenFramebuffers(n int32) []uint32 {
	framebuffers := make([]uint32, n)
	C.glGenFramebuffers(C.GLsizei(n), (*C.GLuint)(unsafe.Pointer(&framebuffers[0])))
	return framebuffers
}

func (defaultContext) GenRenderbuffers(n int32) []uint32 {
	renderbuffers := make([]uint32, n)
	C.glGenRenderbuffers(C.GLsizei(n), (*C.GLuint)(unsafe.Pointer(&renderbuffers[0])))
	return renderbuffers
}

func (defaultContext) GenTextures(n int32) []uint32 {
	textures := make([]uint32, n)
	C.glGenTextures(C.GLsizei(n), (*C.GLuint)(unsafe.Pointer(&textures[0])))
	return textures
}

func (defaultContext) GetError() uint32 {
	return uint32(C.glGetError())
}

func (defaultContext) GetIntegerv(dst []int32, pname uint32) {
	C.glGetIntegerv(C.GLenum(pname), (*C.GLint)(unsafe.Pointer(&dst[0])))
}

func (d defaultContext) GetProgramInfoLog(program uint32) string {
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

func (defaultContext) GetProgramiv(dst []int32, program uint32, pname uint32) {
	C.glGetProgramiv(C.GLuint(program), C.GLenum(pname), (*C.GLint)(unsafe.Pointer(&dst[0])))
}

func (d defaultContext) GetShaderInfoLog(shader uint32) string {
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

func (defaultContext) GetShaderiv(dst []int32, shader uint32, pname uint32) {
	C.glGetShaderiv(C.GLuint(shader), C.GLenum(pname), (*C.GLint)(unsafe.Pointer(&dst[0])))
}

func (defaultContext) GetUniformLocation(program uint32, name string) int32 {
	s := C.CString(name)
	defer C.free(unsafe.Pointer(s))
	return int32(C.glGetUniformLocation(C.GLuint(program), (*C.GLchar)(unsafe.Pointer(s))))
}

func (defaultContext) IsFramebuffer(framebuffer uint32) bool {
	return C.glIsFramebuffer(C.GLuint(framebuffer)) != FALSE
}

func (defaultContext) IsProgram(program uint32) bool {
	return C.glIsProgram(C.GLuint(program)) != FALSE
}

func (defaultContext) IsRenderbuffer(renderbuffer uint32) bool {
	return C.glIsRenderbuffer(C.GLuint(renderbuffer)) != FALSE
}

func (defaultContext) IsTexture(texture uint32) bool {
	return C.glIsTexture(C.GLuint(texture)) != FALSE
}

func (defaultContext) LinkProgram(program uint32) {
	C.glLinkProgram(C.GLuint(program))
}

func (defaultContext) PixelStorei(pname uint32, param int32) {
	C.glPixelStorei(C.GLenum(pname), C.GLint(param))
}

func (defaultContext) ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32) {
	C.glReadPixels(C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height), C.GLenum(format), C.GLenum(xtype), unsafe.Pointer(&dst[0]))
}

func (defaultContext) RenderbufferStorage(target uint32, internalFormat uint32, width int32, height int32) {
	C.glRenderbufferStorage(C.GLenum(target), C.GLenum(internalFormat), C.GLsizei(width), C.GLsizei(height))
}

func (defaultContext) Scissor(x, y, width, height int32) {
	C.glScissor(C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height))
}

func (defaultContext) ShaderSource(shader uint32, xstring string) {
	s, free := cStringPtr(xstring)
	defer free()
	C.glShaderSource(C.GLuint(shader), 1, (**C.GLchar)(s), nil)
}

func (defaultContext) StencilFunc(func_ uint32, ref int32, mask uint32) {
	C.glStencilFunc(C.GLenum(func_), C.GLint(ref), C.GLuint(mask))
}

func (defaultContext) StencilOp(sfail, dpfail, dppass uint32) {
	C.glStencilOp(C.GLenum(sfail), C.GLenum(dpfail), C.GLenum(dppass))
}

func (defaultContext) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	var p *byte
	if pixels != nil {
		p = &pixels[0]
	}
	C.glTexImage2D(C.GLenum(target), C.GLint(level), C.GLint(internalformat), C.GLsizei(width), C.GLsizei(height), 0 /* border */, C.GLenum(format), C.GLenum(xtype), unsafe.Pointer(p))
}

func (defaultContext) TexParameteri(target uint32, pname uint32, param int32) {
	C.glTexParameteri(C.GLenum(target), C.GLenum(pname), C.GLint(param))
}

func (defaultContext) TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	C.glTexSubImage2D(C.GLenum(target), C.GLint(level), C.GLint(xoffset), C.GLint(yoffset), C.GLsizei(width), C.GLsizei(height), C.GLenum(format), C.GLenum(xtype), unsafe.Pointer(&pixels[0]))
}

func (defaultContext) Uniform1fv(location int32, value []float32) {
	C.glUniform1fv(C.GLint(location), C.GLsizei(len(value)), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (defaultContext) Uniform1i(location int32, v0 int32) {
	C.glUniform1i(C.GLint(location), C.GLint(v0))
}

func (defaultContext) Uniform1iv(location int32, value []int32) {
	C.glUniform1iv(C.GLint(location), C.GLsizei(len(value)), (*C.GLint)(unsafe.Pointer(&value[0])))
}

func (defaultContext) Uniform2fv(location int32, value []float32) {
	C.glUniform2fv(C.GLint(location), C.GLsizei(len(value)/2), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (defaultContext) Uniform3fv(location int32, value []float32) {
	C.glUniform3fv(C.GLint(location), C.GLsizei(len(value)/3), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (defaultContext) Uniform4fv(location int32, value []float32) {
	C.glUniform4fv(C.GLint(location), C.GLsizei(len(value)/4), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (defaultContext) UniformMatrix2fv(location int32, transpose bool, value []float32) {
	C.glUniformMatrix2fv(C.GLint(location), C.GLsizei(len(value)/4), glBool(transpose), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (defaultContext) UniformMatrix3fv(location int32, transpose bool, value []float32) {
	C.glUniformMatrix3fv(C.GLint(location), C.GLsizei(len(value)/9), glBool(transpose), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (defaultContext) UniformMatrix4fv(location int32, transpose bool, value []float32) {
	C.glUniformMatrix4fv(C.GLint(location), C.GLsizei(len(value)/16), glBool(transpose), (*C.GLfloat)(unsafe.Pointer(&value[0])))
}

func (defaultContext) UseProgram(program uint32) {
	C.glUseProgram(C.GLuint(program))
}

func (defaultContext) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int) {
	C.glVertexAttribPointer(C.GLuint(index), C.GLint(size), C.GLenum(xtype), glBool(normalized), C.GLsizei(stride), unsafe.Pointer(uintptr(offset)))
}

func (defaultContext) Viewport(x int32, y int32, width int32, height int32) {
	C.glViewport(C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height))
}
