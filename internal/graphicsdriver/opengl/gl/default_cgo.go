// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2014 Eric Woroshow
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build nintendosdk

package gl

// #include <stdint.h>
// #include <stdlib.h>
//
// typedef unsigned int GLenum;
// typedef unsigned char GLboolean;
// typedef unsigned int GLbitfield;
// typedef int GLint;
// typedef unsigned int GLuint;
// typedef int GLsizei;
// typedef float GLfloat;
// typedef char GLchar;
// typedef ptrdiff_t GLintptr;
// typedef ptrdiff_t GLsizeiptr;
//
// #cgo noescape glowActiveTexture
// #cgo nocallback glowActiveTexture
// static void glowActiveTexture(uintptr_t fnptr, GLenum texture) {
//   typedef void (*fn)(GLenum texture);
//   ((fn)(fnptr))(texture);
// }
//
// #cgo noescape glowAttachShader
// #cgo nocallback glowAttachShader
// static void glowAttachShader(uintptr_t fnptr, GLuint program, GLuint shader) {
//   typedef void (*fn)(GLuint program, GLuint shader);
//   ((fn)(fnptr))(program, shader);
// }
//
// #cgo noescape glowBindAttribLocation
// #cgo nocallback glowBindAttribLocation
// static void glowBindAttribLocation(uintptr_t fnptr, GLuint program, GLuint index, const GLchar* name) {
//   typedef void (*fn)(GLuint program, GLuint index, const GLchar* name);
//   ((fn)(fnptr))(program, index, name);
// }
//
// #cgo noescape glowBindBuffer
// #cgo nocallback glowBindBuffer
// static void glowBindBuffer(uintptr_t fnptr, GLenum target, GLuint buffer) {
//   typedef void (*fn)(GLenum target, GLuint buffer);
//   ((fn)(fnptr))(target, buffer);
// }
//
// #cgo noescape glowBindFramebuffer
// #cgo nocallback glowBindFramebuffer
// static void glowBindFramebuffer(uintptr_t fnptr, GLenum target, GLuint framebuffer) {
//   typedef void (*fn)(GLenum target, GLuint framebuffer);
//   ((fn)(fnptr))(target, framebuffer);
// }
//
// #cgo noescape glowBindRenderbuffer
// #cgo nocallback glowBindRenderbuffer
// static void glowBindRenderbuffer(uintptr_t fnptr, GLenum target, GLuint renderbuffer) {
//   typedef void (*fn)(GLenum target, GLuint renderbuffer);
//   ((fn)(fnptr))(target, renderbuffer);
// }
//
// #cgo noescape glowBindTexture
// #cgo nocallback glowBindTexture
// static void glowBindTexture(uintptr_t fnptr, GLenum target, GLuint texture) {
//   typedef void (*fn)(GLenum target, GLuint texture);
//   ((fn)(fnptr))(target, texture);
// }
//
// #cgo noescape glowBindVertexArray
// #cgo nocallback glowBindVertexArray
// static void glowBindVertexArray(uintptr_t fnptr, GLuint array) {
//   typedef void (*fn)(GLuint array);
//   ((fn)(fnptr))(array);
// }
//
// #cgo noescape glowBlendEquationSeparate
// #cgo nocallback glowBlendEquationSeparate
// static void glowBlendEquationSeparate(uintptr_t fnptr, GLenum modeRGB, GLenum modeAlpha) {
//   typedef void (*fn)(GLenum modeRGB, GLenum modeAlpha);
//   ((fn)(fnptr))(modeRGB, modeAlpha);
// }
//
// #cgo noescape glowBlendFuncSeparate
// #cgo nocallback glowBlendFuncSeparate
// static void glowBlendFuncSeparate(uintptr_t fnptr, GLenum srcRGB, GLenum dstRGB, GLenum srcAlpha, GLenum dstAlpha) {
//   typedef void (*fn)(GLenum srcRGB, GLenum dstRGB, GLenum srcAlpha, GLenum dstAlpha);
//   ((fn)(fnptr))(srcRGB, dstRGB, srcAlpha, dstAlpha);
// }
//
// #cgo noescape glowBufferData
// #cgo nocallback glowBufferData
// static void glowBufferData(uintptr_t fnptr, GLenum target, GLsizeiptr size, const void* data, GLenum usage) {
//   typedef void (*fn)(GLenum target, GLsizeiptr size, const void* data, GLenum usage);
//   ((fn)(fnptr))(target, size, data, usage);
// }
//
// #cgo noescape glowBufferSubData
// #cgo nocallback glowBufferSubData
// static void glowBufferSubData(uintptr_t fnptr, GLenum target, GLintptr  offset, GLsizeiptr size, const void* data) {
//   typedef void (*fn)(GLenum target, GLintptr  offset, GLsizeiptr size, const void* data);
//   ((fn)(fnptr))(target, offset, size, data);
// }
//
// #cgo noescape glowCheckFramebufferStatus
// #cgo nocallback glowCheckFramebufferStatus
// static GLenum glowCheckFramebufferStatus(uintptr_t fnptr, GLenum target) {
//   typedef GLenum (*fn)(GLenum target);
//   return ((fn)(fnptr))(target);
// }
//
// #cgo noescape glowClear
// #cgo nocallback glowClear
// static void glowClear(uintptr_t fnptr, GLbitfield mask) {
//   typedef void (*fn)(GLbitfield mask);
//   ((fn)(fnptr))(mask);
// }
//
// #cgo noescape glowColorMask
// #cgo nocallback glowColorMask
// static void glowColorMask(uintptr_t fnptr, GLboolean red, GLboolean green, GLboolean blue, GLboolean alpha) {
//   typedef void (*fn)(GLboolean red, GLboolean green, GLboolean blue, GLboolean alpha);
//   ((fn)(fnptr))(red, green, blue, alpha);
// }
//
// #cgo noescape glowCompileShader
// #cgo nocallback glowCompileShader
// static void glowCompileShader(uintptr_t fnptr, GLuint shader) {
//   typedef void (*fn)(GLuint shader);
//   ((fn)(fnptr))(shader);
// }
//
// #cgo noescape glowCreateProgram
// #cgo nocallback glowCreateProgram
// static GLuint glowCreateProgram(uintptr_t fnptr) {
//   typedef GLuint (*fn)();
//   return ((fn)(fnptr))();
// }
//
// #cgo noescape glowCreateShader
// #cgo nocallback glowCreateShader
// static GLuint glowCreateShader(uintptr_t fnptr, GLenum type) {
//   typedef GLuint (*fn)(GLenum type);
//   return ((fn)(fnptr))(type);
// }
//
// #cgo noescape glowDeleteBuffers
// #cgo nocallback glowDeleteBuffers
// static void glowDeleteBuffers(uintptr_t fnptr, GLsizei n, const GLuint* buffers) {
//   typedef void (*fn)(GLsizei n, const GLuint* buffers);
//   ((fn)(fnptr))(n, buffers);
// }
//
// #cgo noescape glowDeleteFramebuffers
// #cgo nocallback glowDeleteFramebuffers
// static void glowDeleteFramebuffers(uintptr_t fnptr, GLsizei n, const GLuint* framebuffers) {
//   typedef void (*fn)(GLsizei n, const GLuint* framebuffers);
//   ((fn)(fnptr))(n, framebuffers);
// }
//
// #cgo noescape glowDeleteProgram
// #cgo nocallback glowDeleteProgram
// static void glowDeleteProgram(uintptr_t fnptr, GLuint program) {
//   typedef void (*fn)(GLuint program);
//   ((fn)(fnptr))(program);
// }
//
// #cgo noescape glowDeleteRenderbuffers
// #cgo nocallback glowDeleteRenderbuffers
// static void glowDeleteRenderbuffers(uintptr_t fnptr, GLsizei n, const GLuint* renderbuffers) {
//   typedef void (*fn)(GLsizei n, const GLuint* renderbuffers);
//   ((fn)(fnptr))(n, renderbuffers);
// }
//
// #cgo noescape glowDeleteShader
// #cgo nocallback glowDeleteShader
// static void glowDeleteShader(uintptr_t fnptr, GLuint shader) {
//   typedef void (*fn)(GLuint shader);
//   ((fn)(fnptr))(shader);
// }
//
// #cgo noescape glowDeleteTextures
// #cgo nocallback glowDeleteTextures
// static void glowDeleteTextures(uintptr_t fnptr, GLsizei n, const GLuint* textures) {
//   typedef void (*fn)(GLsizei n, const GLuint* textures);
//   ((fn)(fnptr))(n, textures);
// }
//
// #cgo noescape glowDeleteVertexArrays
// #cgo nocallback glowDeleteVertexArrays
// static void glowDeleteVertexArrays(uintptr_t fnptr, GLsizei n, const GLuint* arrays) {
//   typedef void (*fn)(GLsizei n, const GLuint* arrays);
//   ((fn)(fnptr))(n, arrays);
// }
//
// #cgo noescape glowDisable
// #cgo nocallback glowDisable
// static void glowDisable(uintptr_t fnptr, GLenum cap) {
//   typedef void (*fn)(GLenum cap);
//   ((fn)(fnptr))(cap);
// }
//
// #cgo noescape glowDisableVertexAttribArray
// #cgo nocallback glowDisableVertexAttribArray
// static void glowDisableVertexAttribArray(uintptr_t fnptr, GLuint index) {
//   typedef void (*fn)(GLuint index);
//   ((fn)(fnptr))(index);
// }
//
// #cgo noescape glowDrawElements
// #cgo nocallback glowDrawElements
// static void glowDrawElements(uintptr_t fnptr, GLenum mode, GLsizei count, GLenum type, const uintptr_t indices) {
//   typedef void (*fn)(GLenum mode, GLsizei count, GLenum type, const uintptr_t indices);
//   ((fn)(fnptr))(mode, count, type, indices);
// }
//
// #cgo noescape glowEnable
// #cgo nocallback glowEnable
// static void glowEnable(uintptr_t fnptr, GLenum cap) {
//   typedef void (*fn)(GLenum cap);
//   ((fn)(fnptr))(cap);
// }
//
// #cgo noescape glowEnableVertexAttribArray
// #cgo nocallback glowEnableVertexAttribArray
// static void glowEnableVertexAttribArray(uintptr_t fnptr, GLuint index) {
//   typedef void (*fn)(GLuint index);
//   ((fn)(fnptr))(index);
// }
//
// #cgo noescape glowFlush
// #cgo nocallback glowFlush
// static void glowFlush(uintptr_t fnptr) {
//   typedef void (*fn)();
//   ((fn)(fnptr))();
// }
//
// #cgo noescape glowFramebufferRenderbuffer
// #cgo nocallback glowFramebufferRenderbuffer
// static void glowFramebufferRenderbuffer(uintptr_t fnptr, GLenum target, GLenum attachment, GLenum renderbuffertarget, GLuint renderbuffer) {
//   typedef void (*fn)(GLenum target, GLenum attachment, GLenum renderbuffertarget, GLuint renderbuffer);
//   ((fn)(fnptr))(target, attachment, renderbuffertarget, renderbuffer);
// }
//
// #cgo noescape glowFramebufferTexture2D
// #cgo nocallback glowFramebufferTexture2D
// static void glowFramebufferTexture2D(uintptr_t fnptr, GLenum target, GLenum attachment, GLenum textarget, GLuint texture, GLint level) {
//   typedef void (*fn)(GLenum target, GLenum attachment, GLenum textarget, GLuint texture, GLint level);
//   ((fn)(fnptr))(target, attachment, textarget, texture, level);
// }
//
// #cgo noescape glowGenBuffers
// #cgo nocallback glowGenBuffers
// static void glowGenBuffers(uintptr_t fnptr, GLsizei n, GLuint* buffers) {
//   typedef void (*fn)(GLsizei n, GLuint* buffers);
//   ((fn)(fnptr))(n, buffers);
// }
//
// #cgo noescape glowGenFramebuffers
// #cgo nocallback glowGenFramebuffers
// static void glowGenFramebuffers(uintptr_t fnptr, GLsizei n, GLuint* framebuffers) {
//   typedef void (*fn)(GLsizei n, GLuint* framebuffers);
//   ((fn)(fnptr))(n, framebuffers);
// }
//
// #cgo noescape glowGenRenderbuffers
// #cgo nocallback glowGenRenderbuffers
// static void glowGenRenderbuffers(uintptr_t fnptr, GLsizei n, GLuint* renderbuffers) {
//   typedef void (*fn)(GLsizei n, GLuint* renderbuffers);
//   ((fn)(fnptr))(n, renderbuffers);
// }
//
// #cgo noescape glowGenTextures
// #cgo nocallback glowGenTextures
// static void glowGenTextures(uintptr_t fnptr, GLsizei n, GLuint* textures) {
//   typedef void (*fn)(GLsizei n, GLuint* textures);
//   ((fn)(fnptr))(n, textures);
// }
//
// #cgo noescape glowGenVertexArrays
// #cgo nocallback glowGenVertexArrays
// static void glowGenVertexArrays(uintptr_t fnptr, GLsizei n, GLuint* arrays) {
//   typedef void (*fn)(GLsizei n, GLuint* arrays);
//   ((fn)(fnptr))(n, arrays);
// }
//
// #cgo noescape glowGetError
// #cgo nocallback glowGetError
// static GLenum glowGetError(uintptr_t fnptr) {
//   typedef GLenum (*fn)();
//   return ((fn)(fnptr))();
// }
//
// #cgo noescape glowGetIntegerv
// #cgo nocallback glowGetIntegerv
// static void glowGetIntegerv(uintptr_t fnptr, GLenum pname, GLint* data) {
//   typedef void (*fn)(GLenum pname, GLint* data);
//   ((fn)(fnptr))(pname, data);
// }
//
// #cgo noescape glowGetProgramInfoLog
// #cgo nocallback glowGetProgramInfoLog
// static void glowGetProgramInfoLog(uintptr_t fnptr, GLuint program, GLsizei bufSize, GLsizei* length, GLchar* infoLog) {
//   typedef void (*fn)(GLuint program, GLsizei bufSize, GLsizei* length, GLchar* infoLog);
//   ((fn)(fnptr))(program, bufSize, length, infoLog);
// }
//
// #cgo noescape glowGetProgramiv
// #cgo nocallback glowGetProgramiv
// static void glowGetProgramiv(uintptr_t fnptr, GLuint program, GLenum pname, GLint* params) {
//   typedef void (*fn)(GLuint program, GLenum pname, GLint* params);
//   ((fn)(fnptr))(program, pname, params);
// }
//
// #cgo noescape glowGetShaderInfoLog
// #cgo nocallback glowGetShaderInfoLog
// static void glowGetShaderInfoLog(uintptr_t fnptr, GLuint shader, GLsizei bufSize, GLsizei* length, GLchar* infoLog) {
//   typedef void (*fn)(GLuint shader, GLsizei bufSize, GLsizei* length, GLchar* infoLog);
//   ((fn)(fnptr))(shader, bufSize, length, infoLog);
// }
//
// #cgo noescape glowGetShaderiv
// #cgo nocallback glowGetShaderiv
// static void glowGetShaderiv(uintptr_t fnptr, GLuint shader, GLenum pname, GLint* params) {
//   typedef void (*fn)(GLuint shader, GLenum pname, GLint* params);
//   ((fn)(fnptr))(shader, pname, params);
// }
//
// #cgo noescape glowGetUniformLocation
// #cgo nocallback glowGetUniformLocation
// static GLint glowGetUniformLocation(uintptr_t fnptr, GLuint program, const GLchar* name) {
//   typedef GLint (*fn)(GLuint program, const GLchar* name);
//   return ((fn)(fnptr))(program, name);
// }
//
// #cgo noescape glowIsProgram
// #cgo nocallback glowIsProgram
// static GLboolean glowIsProgram(uintptr_t fnptr, GLuint program) {
//   typedef GLboolean (*fn)(GLuint program);
//   return ((fn)(fnptr))(program);
// }
//
// #cgo noescape glowLinkProgram
// #cgo nocallback glowLinkProgram
// static void glowLinkProgram(uintptr_t fnptr, GLuint program) {
//   typedef void (*fn)(GLuint program);
//   ((fn)(fnptr))(program);
// }
//
// #cgo noescape glowPixelStorei
// #cgo nocallback glowPixelStorei
// static void glowPixelStorei(uintptr_t fnptr, GLenum pname, GLint param) {
//   typedef void (*fn)(GLenum pname, GLint param);
//   ((fn)(fnptr))(pname, param);
// }
//
// #cgo noescape glowReadPixels
// #cgo nocallback glowReadPixels
// static void glowReadPixels(uintptr_t fnptr, GLint x, GLint y, GLsizei width, GLsizei height, GLenum format, GLenum type, void* pixels) {
//   typedef void (*fn)(GLint x, GLint y, GLsizei width, GLsizei height, GLenum format, GLenum type, void* pixels);
//   ((fn)(fnptr))(x, y, width, height, format, type, pixels);
// }
//
// #cgo noescape glowRenderbufferStorage
// #cgo nocallback glowRenderbufferStorage
// static void glowRenderbufferStorage(uintptr_t fnptr, GLenum target, GLenum internalformat, GLsizei width, GLsizei height) {
//   typedef void (*fn)(GLenum target, GLenum internalformat, GLsizei width, GLsizei height);
//   ((fn)(fnptr))(target, internalformat, width, height);
// }
//
// #cgo noescape glowScissor
// #cgo nocallback glowScissor
// static void glowScissor(uintptr_t fnptr, GLint x, GLint y, GLsizei width, GLsizei height) {
//   typedef void (*fn)(GLint x, GLint y, GLsizei width, GLsizei height);
//   ((fn)(fnptr))(x, y, width, height);
// }
//
// #cgo noescape glowShaderSource
// #cgo nocallback glowShaderSource
// static void glowShaderSource(uintptr_t fnptr, GLuint shader, GLsizei count, const GLchar*const* string, const GLint* length) {
//   typedef void (*fn)(GLuint shader, GLsizei count, const GLchar*const* string, const GLint* length);
//   ((fn)(fnptr))(shader, count, string, length);
// }
//
// #cgo noescape glowStencilFunc
// #cgo nocallback glowStencilFunc
// static void glowStencilFunc(uintptr_t fnptr, GLenum func, GLint ref, GLuint mask) {
//   typedef void (*fn)(GLenum func, GLint ref, GLuint mask);
//   ((fn)(fnptr))(func, ref, mask);
// }
//
// #cgo noescape glowStencilOpSeparate
// #cgo nocallback glowStencilOpSeparate
// static void glowStencilOpSeparate(uintptr_t fnptr, GLenum face, GLenum fail, GLenum zfail, GLenum zpass) {
//   typedef void (*fn)(GLenum face, GLenum fail, GLenum zfail, GLenum zpass);
//   ((fn)(fnptr))(face, fail, zfail, zpass);
// }
//
// #cgo noescape glowTexImage2D
// #cgo nocallback glowTexImage2D
// static void glowTexImage2D(uintptr_t fnptr, GLenum target, GLint level, GLint internalformat, GLsizei width, GLsizei height, GLint border, GLenum format, GLenum type, const void* pixels) {
//   typedef void (*fn)(GLenum target, GLint level, GLint internalformat, GLsizei width, GLsizei height, GLint border, GLenum format, GLenum type, const void* pixels);
//   ((fn)(fnptr))(target, level, internalformat, width, height, border, format, type, pixels);
// }
//
// #cgo noescape glowTexParameteri
// #cgo nocallback glowTexParameteri
// static void glowTexParameteri(uintptr_t fnptr, GLenum target, GLenum pname, GLint param) {
//   typedef void (*fn)(GLenum target, GLenum pname, GLint param);
//   ((fn)(fnptr))(target, pname, param);
// }
//
// #cgo noescape glowTexSubImage2D
// #cgo nocallback glowTexSubImage2D
// static void glowTexSubImage2D(uintptr_t fnptr, GLenum target, GLint level, GLint xoffset, GLint yoffset, GLsizei width, GLsizei height, GLenum format, GLenum type, const void* pixels) {
//   typedef void (*fn)(GLenum target, GLint level, GLint xoffset, GLint yoffset, GLsizei width, GLsizei height, GLenum format, GLenum type, const void* pixels);
//   ((fn)(fnptr))(target, level, xoffset, yoffset, width, height, format, type, pixels);
// }
//
// #cgo noescape glowUniform1fv
// #cgo nocallback glowUniform1fv
// static void glowUniform1fv(uintptr_t fnptr, GLint location, GLsizei count, const GLfloat* value) {
//   typedef void (*fn)(GLint location, GLsizei count, const GLfloat* value);
//   ((fn)(fnptr))(location, count, value);
// }
//
// #cgo noescape glowUniform1i
// #cgo nocallback glowUniform1i
// static void glowUniform1i(uintptr_t fnptr, GLint location, GLint v0) {
//   typedef void (*fn)(GLint location, GLint v0);
//   ((fn)(fnptr))(location, v0);
// }
//
// #cgo noescape glowUniform1iv
// #cgo nocallback glowUniform1iv
// static void glowUniform1iv(uintptr_t fnptr, GLint location, GLsizei count, const GLint* value) {
//   typedef void (*fn)(GLint location, GLsizei count, const GLint* value);
//   ((fn)(fnptr))(location, count, value);
// }
//
// #cgo noescape glowUniform2fv
// #cgo nocallback glowUniform2fv
// static void glowUniform2fv(uintptr_t fnptr, GLint location, GLsizei count, const GLfloat* value) {
//   typedef void (*fn)(GLint location, GLsizei count, const GLfloat* value);
//   ((fn)(fnptr))(location, count, value);
// }
//
// #cgo noescape glowUniform2iv
// #cgo nocallback glowUniform2iv
// static void glowUniform2iv(uintptr_t fnptr, GLint location, GLsizei count, const GLint* value) {
//   typedef void (*fn)(GLint location, GLsizei count, const GLint* value);
//   ((fn)(fnptr))(location, count, value);
// }
//
// #cgo noescape glowUniform3fv
// #cgo nocallback glowUniform3fv
// static void glowUniform3fv(uintptr_t fnptr, GLint location, GLsizei count, const GLfloat* value) {
//   typedef void (*fn)(GLint location, GLsizei count, const GLfloat* value);
//   ((fn)(fnptr))(location, count, value);
// }
//
// #cgo noescape glowUniform3iv
// #cgo nocallback glowUniform3iv
// static void glowUniform3iv(uintptr_t fnptr, GLint location, GLsizei count, const GLint* value) {
//   typedef void (*fn)(GLint location, GLsizei count, const GLint* value);
//   ((fn)(fnptr))(location, count, value);
// }
//
// #cgo noescape glowUniform4fv
// #cgo nocallback glowUniform4fv
// static void glowUniform4fv(uintptr_t fnptr, GLint location, GLsizei count, const GLfloat* value) {
//   typedef void (*fn)(GLint location, GLsizei count, const GLfloat* value);
//   ((fn)(fnptr))(location, count, value);
// }
//
// #cgo noescape glowUniform4iv
// #cgo nocallback glowUniform4iv
// static void glowUniform4iv(uintptr_t fnptr, GLint location, GLsizei count, const GLint* value) {
//   typedef void (*fn)(GLint location, GLsizei count, const GLint* value);
//   ((fn)(fnptr))(location, count, value);
// }
//
// #cgo noescape glowUniformMatrix2fv
// #cgo nocallback glowUniformMatrix2fv
// static void glowUniformMatrix2fv(uintptr_t fnptr, GLint location, GLsizei count, GLboolean transpose, const GLfloat* value) {
//   typedef void (*fn)(GLint location, GLsizei count, GLboolean transpose, const GLfloat* value);
//   ((fn)(fnptr))(location, count, transpose, value);
// }
//
// #cgo noescape glowUniformMatrix3fv
// #cgo nocallback glowUniformMatrix3fv
// static void glowUniformMatrix3fv(uintptr_t fnptr, GLint location, GLsizei count, GLboolean transpose, const GLfloat* value) {
//   typedef void (*fn)(GLint location, GLsizei count, GLboolean transpose, const GLfloat* value);
//   ((fn)(fnptr))(location, count, transpose, value);
// }
//
// #cgo noescape glowUniformMatrix4fv
// #cgo nocallback glowUniformMatrix4fv
// static void glowUniformMatrix4fv(uintptr_t fnptr, GLint location, GLsizei count, GLboolean transpose, const GLfloat* value) {
//   typedef void (*fn)(GLint location, GLsizei count, GLboolean transpose, const GLfloat* value);
//   ((fn)(fnptr))(location, count, transpose, value);
// }
//
// #cgo noescape glowUseProgram
// #cgo nocallback glowUseProgram
// static void glowUseProgram(uintptr_t fnptr, GLuint program) {
//   typedef void (*fn)(GLuint program);
//   ((fn)(fnptr))(program);
// }
//
// #cgo noescape glowVertexAttribPointer
// #cgo nocallback glowVertexAttribPointer
// static void glowVertexAttribPointer(uintptr_t fnptr, GLuint index, GLint size, GLenum type, GLboolean normalized, GLsizei stride, const uintptr_t pointer) {
//   typedef void (*fn)(GLuint index, GLint size, GLenum type, GLboolean normalized, GLsizei stride, const uintptr_t pointer);
//   ((fn)(fnptr))(index, size, type, normalized, stride, pointer);
// }
//
// #cgo noescape glowViewport
// #cgo nocallback glowViewport
// static void glowViewport(uintptr_t fnptr, GLint x, GLint y, GLsizei width, GLsizei height) {
//   typedef void (*fn)(GLint x, GLint y, GLsizei width, GLsizei height);
//   ((fn)(fnptr))(x, y, width, height);
// }
import "C"

import (
	"runtime"
	"unsafe"
)

type defaultContext struct {
	gpActiveTexture            C.uintptr_t
	gpAttachShader             C.uintptr_t
	gpBindAttribLocation       C.uintptr_t
	gpBindBuffer               C.uintptr_t
	gpBindFramebuffer          C.uintptr_t
	gpBindRenderbuffer         C.uintptr_t
	gpBindTexture              C.uintptr_t
	gpBindVertexArray          C.uintptr_t
	gpBlendEquationSeparate    C.uintptr_t
	gpBlendFuncSeparate        C.uintptr_t
	gpBufferData               C.uintptr_t
	gpBufferSubData            C.uintptr_t
	gpCheckFramebufferStatus   C.uintptr_t
	gpClear                    C.uintptr_t
	gpColorMask                C.uintptr_t
	gpCompileShader            C.uintptr_t
	gpCreateProgram            C.uintptr_t
	gpCreateShader             C.uintptr_t
	gpDeleteBuffers            C.uintptr_t
	gpDeleteFramebuffers       C.uintptr_t
	gpDeleteProgram            C.uintptr_t
	gpDeleteRenderbuffers      C.uintptr_t
	gpDeleteShader             C.uintptr_t
	gpDeleteTextures           C.uintptr_t
	gpDeleteVertexArrays       C.uintptr_t
	gpDisable                  C.uintptr_t
	gpDisableVertexAttribArray C.uintptr_t
	gpDrawElements             C.uintptr_t
	gpEnable                   C.uintptr_t
	gpEnableVertexAttribArray  C.uintptr_t
	gpFlush                    C.uintptr_t
	gpFramebufferRenderbuffer  C.uintptr_t
	gpFramebufferTexture2D     C.uintptr_t
	gpGenBuffers               C.uintptr_t
	gpGenFramebuffers          C.uintptr_t
	gpGenRenderbuffers         C.uintptr_t
	gpGenTextures              C.uintptr_t
	gpGenVertexArrays          C.uintptr_t
	gpGetError                 C.uintptr_t
	gpGetIntegerv              C.uintptr_t
	gpGetProgramInfoLog        C.uintptr_t
	gpGetProgramiv             C.uintptr_t
	gpGetShaderInfoLog         C.uintptr_t
	gpGetShaderiv              C.uintptr_t
	gpGetUniformLocation       C.uintptr_t
	gpIsProgram                C.uintptr_t
	gpLinkProgram              C.uintptr_t
	gpPixelStorei              C.uintptr_t
	gpReadPixels               C.uintptr_t
	gpRenderbufferStorage      C.uintptr_t
	gpScissor                  C.uintptr_t
	gpShaderSource             C.uintptr_t
	gpStencilFunc              C.uintptr_t
	gpStencilOpSeparate        C.uintptr_t
	gpTexImage2D               C.uintptr_t
	gpTexParameteri            C.uintptr_t
	gpTexSubImage2D            C.uintptr_t
	gpUniform1fv               C.uintptr_t
	gpUniform1i                C.uintptr_t
	gpUniform1iv               C.uintptr_t
	gpUniform2fv               C.uintptr_t
	gpUniform2iv               C.uintptr_t
	gpUniform3fv               C.uintptr_t
	gpUniform3iv               C.uintptr_t
	gpUniform4fv               C.uintptr_t
	gpUniform4iv               C.uintptr_t
	gpUniformMatrix2fv         C.uintptr_t
	gpUniformMatrix3fv         C.uintptr_t
	gpUniformMatrix4fv         C.uintptr_t
	gpUseProgram               C.uintptr_t
	gpVertexAttribPointer      C.uintptr_t
	gpViewport                 C.uintptr_t

	isES bool
}

func NewDefaultContext() (Context, error) {
	ctx := &defaultContext{}
	if err := ctx.init(); err != nil {
		return nil, err
	}
	return ctx, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (c *defaultContext) IsES() bool {
	return c.isES
}

func (c *defaultContext) ActiveTexture(texture uint32) {
	C.glowActiveTexture(c.gpActiveTexture, C.GLenum(texture))
}

func (c *defaultContext) AttachShader(program uint32, shader uint32) {
	C.glowAttachShader(c.gpAttachShader, C.GLuint(program), C.GLuint(shader))
}

func (c *defaultContext) BindAttribLocation(program uint32, index uint32, name string) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	C.glowBindAttribLocation(c.gpBindAttribLocation, C.GLuint(program), C.GLuint(index), (*C.GLchar)(unsafe.Pointer(cname)))
}

func (c *defaultContext) BindBuffer(target uint32, buffer uint32) {
	C.glowBindBuffer(c.gpBindBuffer, C.GLenum(target), C.GLuint(buffer))
}

func (c *defaultContext) BindFramebuffer(target uint32, framebuffer uint32) {
	C.glowBindFramebuffer(c.gpBindFramebuffer, C.GLenum(target), C.GLuint(framebuffer))
}

func (c *defaultContext) BindRenderbuffer(target uint32, renderbuffer uint32) {
	C.glowBindRenderbuffer(c.gpBindRenderbuffer, C.GLenum(target), C.GLuint(renderbuffer))
}

func (c *defaultContext) BindTexture(target uint32, texture uint32) {
	C.glowBindTexture(c.gpBindTexture, C.GLenum(target), C.GLuint(texture))
}

func (c *defaultContext) BindVertexArray(array uint32) {
	C.glowBindVertexArray(c.gpBindVertexArray, C.GLuint(array))
}

func (c *defaultContext) BlendEquationSeparate(modeRGB uint32, modeAlpha uint32) {
	C.glowBlendEquationSeparate(c.gpBlendEquationSeparate, C.GLenum(modeRGB), C.GLenum(modeAlpha))
}

func (c *defaultContext) BlendFuncSeparate(srcRGB uint32, dstRGB uint32, srcAlpha uint32, dstAlpha uint32) {
	C.glowBlendFuncSeparate(c.gpBlendFuncSeparate, C.GLenum(srcRGB), C.GLenum(dstRGB), C.GLenum(srcAlpha), C.GLenum(dstAlpha))
}

func (c *defaultContext) BufferInit(target uint32, size int, usage uint32) {
	C.glowBufferData(c.gpBufferData, C.GLenum(target), C.GLsizeiptr(size), nil, C.GLenum(usage))
}

func (c *defaultContext) BufferSubData(target uint32, offset int, data []byte) {
	C.glowBufferSubData(c.gpBufferSubData, C.GLenum(target), C.GLintptr(offset), C.GLsizeiptr(len(data)), unsafe.Pointer(&data[0]))
	runtime.KeepAlive(data)
}

func (c *defaultContext) CheckFramebufferStatus(target uint32) uint32 {
	ret := C.glowCheckFramebufferStatus(c.gpCheckFramebufferStatus, C.GLenum(target))
	return uint32(ret)
}

func (c *defaultContext) Clear(mask uint32) {
	C.glowClear(c.gpClear, C.GLbitfield(mask))
}

func (c *defaultContext) ColorMask(red bool, green bool, blue bool, alpha bool) {
	C.glowColorMask(c.gpColorMask, C.GLboolean(boolToInt(red)), C.GLboolean(boolToInt(green)), C.GLboolean(boolToInt(blue)), C.GLboolean(boolToInt(alpha)))
}

func (c *defaultContext) CompileShader(shader uint32) {
	C.glowCompileShader(c.gpCompileShader, C.GLuint(shader))
}

func (c *defaultContext) CreateBuffer() uint32 {
	var buffer uint32
	C.glowGenBuffers(c.gpGenBuffers, 1, (*C.GLuint)(unsafe.Pointer(&buffer)))
	return buffer
}

func (c *defaultContext) CreateFramebuffer() uint32 {
	var framebuffer uint32
	C.glowGenFramebuffers(c.gpGenFramebuffers, 1, (*C.GLuint)(unsafe.Pointer(&framebuffer)))
	return framebuffer
}

func (c *defaultContext) CreateProgram() uint32 {
	ret := C.glowCreateProgram(c.gpCreateProgram)
	return uint32(ret)
}

func (c *defaultContext) CreateRenderbuffer() uint32 {
	var renderbuffer uint32
	C.glowGenRenderbuffers(c.gpGenRenderbuffers, 1, (*C.GLuint)(unsafe.Pointer(&renderbuffer)))
	return renderbuffer
}

func (c *defaultContext) CreateShader(xtype uint32) uint32 {
	ret := C.glowCreateShader(c.gpCreateShader, C.GLenum(xtype))
	return uint32(ret)
}

func (c *defaultContext) CreateTexture() uint32 {
	var texture uint32
	C.glowGenTextures(c.gpGenTextures, 1, (*C.GLuint)(unsafe.Pointer(&texture)))
	return texture
}

func (c *defaultContext) CreateVertexArray() uint32 {
	var array uint32
	C.glowGenVertexArrays(c.gpGenVertexArrays, 1, (*C.GLuint)(unsafe.Pointer(&array)))
	return array
}

func (c *defaultContext) DeleteBuffer(buffer uint32) {
	C.glowDeleteBuffers(c.gpDeleteBuffers, 1, (*C.GLuint)(unsafe.Pointer(&buffer)))
}

func (c *defaultContext) DeleteFramebuffer(framebuffer uint32) {
	C.glowDeleteFramebuffers(c.gpDeleteFramebuffers, 1, (*C.GLuint)(unsafe.Pointer(&framebuffer)))
}

func (c *defaultContext) DeleteProgram(program uint32) {
	C.glowDeleteProgram(c.gpDeleteProgram, C.GLuint(program))
}

func (c *defaultContext) DeleteRenderbuffer(renderbuffer uint32) {
	C.glowDeleteRenderbuffers(c.gpDeleteRenderbuffers, 1, (*C.GLuint)(unsafe.Pointer(&renderbuffer)))
}

func (c *defaultContext) DeleteShader(shader uint32) {
	C.glowDeleteShader(c.gpDeleteShader, C.GLuint(shader))
}

func (c *defaultContext) DeleteTexture(texture uint32) {
	C.glowDeleteTextures(c.gpDeleteTextures, 1, (*C.GLuint)(unsafe.Pointer(&texture)))
}

func (c *defaultContext) DeleteVertexArray(array uint32) {
	C.glowDeleteVertexArrays(c.gpDeleteVertexArrays, 1, (*C.GLuint)(unsafe.Pointer(&array)))
}

func (c *defaultContext) Disable(cap uint32) {
	C.glowDisable(c.gpDisable, C.GLenum(cap))
}

func (c *defaultContext) DisableVertexAttribArray(index uint32) {
	C.glowDisableVertexAttribArray(c.gpDisableVertexAttribArray, C.GLuint(index))
}

func (c *defaultContext) DrawElements(mode uint32, count int32, xtype uint32, offset int) {
	C.glowDrawElements(c.gpDrawElements, C.GLenum(mode), C.GLsizei(count), C.GLenum(xtype), C.uintptr_t(offset))
}

func (c *defaultContext) Enable(cap uint32) {
	C.glowEnable(c.gpEnable, C.GLenum(cap))
}

func (c *defaultContext) EnableVertexAttribArray(index uint32) {
	C.glowEnableVertexAttribArray(c.gpEnableVertexAttribArray, C.GLuint(index))
}

func (c *defaultContext) Flush() {
	C.glowFlush(c.gpFlush)
}

func (c *defaultContext) FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	C.glowFramebufferRenderbuffer(c.gpFramebufferRenderbuffer, C.GLenum(target), C.GLenum(attachment), C.GLenum(renderbuffertarget), C.GLuint(renderbuffer))
}

func (c *defaultContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	C.glowFramebufferTexture2D(c.gpFramebufferTexture2D, C.GLenum(target), C.GLenum(attachment), C.GLenum(textarget), C.GLuint(texture), C.GLint(level))
}

func (c *defaultContext) GetError() uint32 {
	ret := C.glowGetError(c.gpGetError)
	return uint32(ret)
}

func (c *defaultContext) GetExtension(name string) any {
	return nil
}

func (c *defaultContext) GetInteger(pname uint32) int {
	var dst int32
	C.glowGetIntegerv(c.gpGetIntegerv, C.GLenum(pname), (*C.GLint)(unsafe.Pointer(&dst)))
	return int(dst)
}

func (c *defaultContext) GetProgramInfoLog(program uint32) string {
	bufSize := c.GetProgrami(program, INFO_LOG_LENGTH)
	if bufSize == 0 {
		return ""
	}
	infoLog := make([]byte, bufSize)
	C.glowGetProgramInfoLog(c.gpGetProgramInfoLog, C.GLuint(program), C.GLsizei(bufSize), nil, (*C.GLchar)(unsafe.Pointer(&infoLog[0])))
	return string(infoLog)
}

func (c *defaultContext) GetProgrami(program uint32, pname uint32) int {
	var dst int32
	C.glowGetProgramiv(c.gpGetProgramiv, C.GLuint(program), C.GLenum(pname), (*C.GLint)(unsafe.Pointer(&dst)))
	return int(dst)
}

func (c *defaultContext) GetShaderInfoLog(shader uint32) string {
	bufSize := c.GetShaderi(shader, INFO_LOG_LENGTH)
	if bufSize == 0 {
		return ""
	}
	infoLog := make([]byte, bufSize)
	C.glowGetShaderInfoLog(c.gpGetShaderInfoLog, C.GLuint(shader), C.GLsizei(bufSize), nil, (*C.GLchar)(unsafe.Pointer(&infoLog[0])))
	return string(infoLog)
}

func (c *defaultContext) GetShaderi(shader uint32, pname uint32) int {
	var dst int32
	C.glowGetShaderiv(c.gpGetShaderiv, C.GLuint(shader), C.GLenum(pname), (*C.GLint)(unsafe.Pointer(&dst)))
	return int(dst)
}

func (c *defaultContext) GetUniformLocation(program uint32, name string) int32 {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	ret := C.glowGetUniformLocation(c.gpGetUniformLocation, C.GLuint(program), (*C.GLchar)(unsafe.Pointer(cname)))
	return int32(ret)
}

func (c *defaultContext) IsProgram(program uint32) bool {
	ret := C.glowIsProgram(c.gpIsProgram, C.GLuint(program))
	return ret == TRUE
}

func (c *defaultContext) LinkProgram(program uint32) {
	C.glowLinkProgram(c.gpLinkProgram, C.GLuint(program))
}

func (c *defaultContext) PixelStorei(pname uint32, param int32) {
	C.glowPixelStorei(c.gpPixelStorei, C.GLenum(pname), C.GLint(param))
}

func (c *defaultContext) ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32) {
	C.glowReadPixels(c.gpReadPixels, C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height), C.GLenum(format), C.GLenum(xtype), unsafe.Pointer(&dst[0]))
}

func (c *defaultContext) RenderbufferStorage(target uint32, internalformat uint32, width int32, height int32) {
	C.glowRenderbufferStorage(c.gpRenderbufferStorage, C.GLenum(target), C.GLenum(internalformat), C.GLsizei(width), C.GLsizei(height))
}

func (c *defaultContext) Scissor(x int32, y int32, width int32, height int32) {
	C.glowScissor(c.gpScissor, C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height))
}

func (c *defaultContext) ShaderSource(shader uint32, xstring string) {
	cstring := C.CString(xstring)
	defer C.free(unsafe.Pointer(cstring))
	C.glowShaderSource(c.gpShaderSource, C.GLuint(shader), 1, (**C.GLchar)(unsafe.Pointer(&cstring)), nil)
}

func (c *defaultContext) StencilFunc(xfunc uint32, ref int32, mask uint32) {
	C.glowStencilFunc(c.gpStencilFunc, C.GLenum(xfunc), C.GLint(ref), C.GLuint(mask))
}

func (c *defaultContext) StencilOpSeparate(face uint32, fail uint32, zfail uint32, zpass uint32) {
	C.glowStencilOpSeparate(c.gpStencilOpSeparate, C.GLenum(face), C.GLenum(fail), C.GLenum(zfail), C.GLenum(zpass))
}

func (c *defaultContext) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	var ptr *byte
	if len(pixels) > 0 {
		ptr = &pixels[0]
	}
	C.glowTexImage2D(c.gpTexImage2D, C.GLenum(target), C.GLint(level), C.GLint(internalformat), C.GLsizei(width), C.GLsizei(height), 0, C.GLenum(format), C.GLenum(xtype), unsafe.Pointer(ptr))
	runtime.KeepAlive(pixels)
}

func (c *defaultContext) TexParameteri(target uint32, pname uint32, param int32) {
	C.glowTexParameteri(c.gpTexParameteri, C.GLenum(target), C.GLenum(pname), C.GLint(param))
}

func (c *defaultContext) TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	C.glowTexSubImage2D(c.gpTexSubImage2D, C.GLenum(target), C.GLint(level), C.GLint(xoffset), C.GLint(yoffset), C.GLsizei(width), C.GLsizei(height), C.GLenum(format), C.GLenum(xtype), unsafe.Pointer(&pixels[0]))
	runtime.KeepAlive(pixels)
}

func (c *defaultContext) Uniform1fv(location int32, value []float32) {
	C.glowUniform1fv(c.gpUniform1fv, C.GLint(location), C.GLsizei(len(value)), (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform1i(location int32, v0 int32) {
	C.glowUniform1i(c.gpUniform1i, C.GLint(location), C.GLint(v0))
}

func (c *defaultContext) Uniform1iv(location int32, value []int32) {
	C.glowUniform1iv(c.gpUniform1iv, C.GLint(location), C.GLsizei(len(value)), (*C.GLint)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform2fv(location int32, value []float32) {
	C.glowUniform2fv(c.gpUniform2fv, C.GLint(location), C.GLsizei(len(value)/2), (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform2iv(location int32, value []int32) {
	C.glowUniform2iv(c.gpUniform2iv, C.GLint(location), C.GLsizei(len(value)/2), (*C.GLint)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform3fv(location int32, value []float32) {
	C.glowUniform3fv(c.gpUniform3fv, C.GLint(location), C.GLsizei(len(value)/3), (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform3iv(location int32, value []int32) {
	C.glowUniform3iv(c.gpUniform3iv, C.GLint(location), C.GLsizei(len(value)/3), (*C.GLint)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform4fv(location int32, value []float32) {
	C.glowUniform4fv(c.gpUniform4fv, C.GLint(location), C.GLsizei(len(value)/4), (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform4iv(location int32, value []int32) {
	C.glowUniform4iv(c.gpUniform4iv, C.GLint(location), C.GLsizei(len(value)/4), (*C.GLint)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UniformMatrix2fv(location int32, value []float32) {
	C.glowUniformMatrix2fv(c.gpUniformMatrix2fv, C.GLint(location), C.GLsizei(len(value)/4), 0, (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UniformMatrix3fv(location int32, value []float32) {
	C.glowUniformMatrix3fv(c.gpUniformMatrix3fv, C.GLint(location), C.GLsizei(len(value)/9), 0, (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UniformMatrix4fv(location int32, value []float32) {
	C.glowUniformMatrix4fv(c.gpUniformMatrix4fv, C.GLint(location), C.GLsizei(len(value)/16), 0, (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UseProgram(program uint32) {
	C.glowUseProgram(c.gpUseProgram, C.GLuint(program))
}

func (c *defaultContext) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int) {
	C.glowVertexAttribPointer(c.gpVertexAttribPointer, C.GLuint(index), C.GLint(size), C.GLenum(xtype), C.GLboolean(boolToInt(normalized)), C.GLsizei(stride), C.uintptr_t(offset))
}

func (c *defaultContext) Viewport(x int32, y int32, width int32, height int32) {
	C.glowViewport(c.gpViewport, C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height))
}

func (c *defaultContext) LoadFunctions() error {
	g := procAddressGetter{ctx: c}

	c.gpActiveTexture = C.uintptr_t(g.get("glActiveTexture"))
	c.gpAttachShader = C.uintptr_t(g.get("glAttachShader"))
	c.gpBindAttribLocation = C.uintptr_t(g.get("glBindAttribLocation"))
	c.gpBindBuffer = C.uintptr_t(g.get("glBindBuffer"))
	c.gpBindFramebuffer = C.uintptr_t(g.get("glBindFramebuffer"))
	c.gpBindRenderbuffer = C.uintptr_t(g.get("glBindRenderbuffer"))
	c.gpBindTexture = C.uintptr_t(g.get("glBindTexture"))
	c.gpBindVertexArray = C.uintptr_t(g.get("glBindVertexArray"))
	c.gpBlendEquationSeparate = C.uintptr_t(g.get("glBlendEquationSeparate"))
	c.gpBlendFuncSeparate = C.uintptr_t(g.get("glBlendFuncSeparate"))
	c.gpBufferData = C.uintptr_t(g.get("glBufferData"))
	c.gpBufferSubData = C.uintptr_t(g.get("glBufferSubData"))
	c.gpCheckFramebufferStatus = C.uintptr_t(g.get("glCheckFramebufferStatus"))
	c.gpClear = C.uintptr_t(g.get("glClear"))
	c.gpColorMask = C.uintptr_t(g.get("glColorMask"))
	c.gpCompileShader = C.uintptr_t(g.get("glCompileShader"))
	c.gpCreateProgram = C.uintptr_t(g.get("glCreateProgram"))
	c.gpCreateShader = C.uintptr_t(g.get("glCreateShader"))
	c.gpDeleteBuffers = C.uintptr_t(g.get("glDeleteBuffers"))
	c.gpDeleteFramebuffers = C.uintptr_t(g.get("glDeleteFramebuffers"))
	c.gpDeleteProgram = C.uintptr_t(g.get("glDeleteProgram"))
	c.gpDeleteRenderbuffers = C.uintptr_t(g.get("glDeleteRenderbuffers"))
	c.gpDeleteShader = C.uintptr_t(g.get("glDeleteShader"))
	c.gpDeleteTextures = C.uintptr_t(g.get("glDeleteTextures"))
	c.gpDeleteVertexArrays = C.uintptr_t(g.get("glDeleteVertexArrays"))
	c.gpDisable = C.uintptr_t(g.get("glDisable"))
	c.gpDisableVertexAttribArray = C.uintptr_t(g.get("glDisableVertexAttribArray"))
	c.gpDrawElements = C.uintptr_t(g.get("glDrawElements"))
	c.gpEnable = C.uintptr_t(g.get("glEnable"))
	c.gpEnableVertexAttribArray = C.uintptr_t(g.get("glEnableVertexAttribArray"))
	c.gpFlush = C.uintptr_t(g.get("glFlush"))
	c.gpFramebufferRenderbuffer = C.uintptr_t(g.get("glFramebufferRenderbuffer"))
	c.gpFramebufferTexture2D = C.uintptr_t(g.get("glFramebufferTexture2D"))
	c.gpGenBuffers = C.uintptr_t(g.get("glGenBuffers"))
	c.gpGenFramebuffers = C.uintptr_t(g.get("glGenFramebuffers"))
	c.gpGenRenderbuffers = C.uintptr_t(g.get("glGenRenderbuffers"))
	c.gpGenTextures = C.uintptr_t(g.get("glGenTextures"))
	c.gpGenVertexArrays = C.uintptr_t(g.get("glGenVertexArrays"))
	c.gpGetError = C.uintptr_t(g.get("glGetError"))
	c.gpGetIntegerv = C.uintptr_t(g.get("glGetIntegerv"))
	c.gpGetProgramInfoLog = C.uintptr_t(g.get("glGetProgramInfoLog"))
	c.gpGetProgramiv = C.uintptr_t(g.get("glGetProgramiv"))
	c.gpGetShaderInfoLog = C.uintptr_t(g.get("glGetShaderInfoLog"))
	c.gpGetShaderiv = C.uintptr_t(g.get("glGetShaderiv"))
	c.gpGetUniformLocation = C.uintptr_t(g.get("glGetUniformLocation"))
	c.gpIsProgram = C.uintptr_t(g.get("glIsProgram"))
	c.gpLinkProgram = C.uintptr_t(g.get("glLinkProgram"))
	c.gpPixelStorei = C.uintptr_t(g.get("glPixelStorei"))
	c.gpReadPixels = C.uintptr_t(g.get("glReadPixels"))
	c.gpRenderbufferStorage = C.uintptr_t(g.get("glRenderbufferStorage"))
	c.gpScissor = C.uintptr_t(g.get("glScissor"))
	c.gpShaderSource = C.uintptr_t(g.get("glShaderSource"))
	c.gpStencilFunc = C.uintptr_t(g.get("glStencilFunc"))
	c.gpStencilOpSeparate = C.uintptr_t(g.get("glStencilOpSeparate"))
	c.gpTexImage2D = C.uintptr_t(g.get("glTexImage2D"))
	c.gpTexParameteri = C.uintptr_t(g.get("glTexParameteri"))
	c.gpTexSubImage2D = C.uintptr_t(g.get("glTexSubImage2D"))
	c.gpUniform1fv = C.uintptr_t(g.get("glUniform1fv"))
	c.gpUniform1i = C.uintptr_t(g.get("glUniform1i"))
	c.gpUniform1iv = C.uintptr_t(g.get("glUniform1iv"))
	c.gpUniform2fv = C.uintptr_t(g.get("glUniform2fv"))
	c.gpUniform2iv = C.uintptr_t(g.get("glUniform2iv"))
	c.gpUniform3fv = C.uintptr_t(g.get("glUniform3fv"))
	c.gpUniform3iv = C.uintptr_t(g.get("glUniform3iv"))
	c.gpUniform4fv = C.uintptr_t(g.get("glUniform4fv"))
	c.gpUniform4iv = C.uintptr_t(g.get("glUniform4iv"))
	c.gpUniformMatrix2fv = C.uintptr_t(g.get("glUniformMatrix2fv"))
	c.gpUniformMatrix3fv = C.uintptr_t(g.get("glUniformMatrix3fv"))
	c.gpUniformMatrix4fv = C.uintptr_t(g.get("glUniformMatrix4fv"))
	c.gpUseProgram = C.uintptr_t(g.get("glUseProgram"))
	c.gpVertexAttribPointer = C.uintptr_t(g.get("glVertexAttribPointer"))
	c.gpViewport = C.uintptr_t(g.get("glViewport"))

	return g.error()
}
