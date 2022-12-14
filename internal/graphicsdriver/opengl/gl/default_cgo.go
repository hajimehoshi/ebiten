// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2014 Eric Woroshow

//go:build !darwin && !js && !windows

package gl

// #ifndef APIENTRY
//   #define APIENTRY
// #endif
//
// #ifndef APIENTRYP
//   #define APIENTRYP APIENTRY *
// #endif
//
// #ifndef GLAPI
//   #define GLAPI extern
// #endif
//
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
// typedef void (APIENTRY *GLDEBUGPROC)(GLenum source,GLenum type,GLuint id,GLenum severity,GLsizei length,const GLchar *message,const void *userParam);
// typedef void (APIENTRY *GLDEBUGPROCARB)(GLenum source,GLenum type,GLuint id,GLenum severity,GLsizei length,const GLchar *message,const void *userParam);
// typedef void (APIENTRY *GLDEBUGPROCKHR)(GLenum source,GLenum type,GLuint id,GLenum severity,GLsizei length,const GLchar *message,const void *userParam);
// typedef void (APIENTRY *GLDEBUGPROCAMD)(GLuint id,GLenum category,GLenum severity,GLsizei length,const GLchar *message,void *userParam);
// typedef unsigned short GLhalfNV;
// typedef GLintptr GLvdpauSurfaceNV;
//
// typedef void  (APIENTRYP GPACTIVETEXTURE)(GLenum  texture);
// typedef void  (APIENTRYP GPATTACHSHADER)(GLuint  program, GLuint  shader);
// typedef void  (APIENTRYP GPBINDATTRIBLOCATION)(GLuint  program, GLuint  index, const GLchar * name);
// typedef void  (APIENTRYP GPBINDBUFFER)(GLenum  target, GLuint  buffer);
// typedef void  (APIENTRYP GPBINDFRAMEBUFFEREXT)(GLenum  target, GLuint  framebuffer);
// typedef void  (APIENTRYP GPBINDRENDERBUFFEREXT)(GLenum  target, GLuint  renderbuffer);
// typedef void  (APIENTRYP GPBINDTEXTURE)(GLenum  target, GLuint  texture);
// typedef void  (APIENTRYP GPBLENDEQUATIONSEPARATE)(GLenum  modeRGB, GLenum  modeAlpha);
// typedef void  (APIENTRYP GPBLENDFUNCSEPARATE)(GLenum  srcRGB, GLenum  dstRGB, GLenum  srcAlpha, GLenum  dstAlpha);
// typedef void  (APIENTRYP GPBUFFERDATA)(GLenum  target, GLsizeiptr  size, const void * data, GLenum  usage);
// typedef void  (APIENTRYP GPBUFFERSUBDATA)(GLenum  target, GLintptr  offset, GLsizeiptr  size, const void * data);
// typedef GLenum  (APIENTRYP GPCHECKFRAMEBUFFERSTATUSEXT)(GLenum  target);
// typedef void  (APIENTRYP GPCLEAR)(GLbitfield  mask);
// typedef void  (APIENTRYP GPCOLORMASK)(GLboolean  red, GLboolean  green, GLboolean  blue, GLboolean  alpha);
// typedef void  (APIENTRYP GPCOMPILESHADER)(GLuint  shader);
// typedef GLuint  (APIENTRYP GPCREATEPROGRAM)();
// typedef GLuint  (APIENTRYP GPCREATESHADER)(GLenum  type);
// typedef void  (APIENTRYP GPDELETEBUFFERS)(GLsizei  n, const GLuint * buffers);
// typedef void  (APIENTRYP GPDELETEFRAMEBUFFERSEXT)(GLsizei  n, const GLuint * framebuffers);
// typedef void  (APIENTRYP GPDELETEPROGRAM)(GLuint  program);
// typedef void  (APIENTRYP GPDELETERENDERBUFFERSEXT)(GLsizei  n, const GLuint * renderbuffers);
// typedef void  (APIENTRYP GPDELETESHADER)(GLuint  shader);
// typedef void  (APIENTRYP GPDELETETEXTURES)(GLsizei  n, const GLuint * textures);
// typedef void  (APIENTRYP GPDISABLE)(GLenum  cap);
// typedef void  (APIENTRYP GPDISABLEVERTEXATTRIBARRAY)(GLuint  index);
// typedef void  (APIENTRYP GPDRAWELEMENTS)(GLenum  mode, GLsizei  count, GLenum  type, const uintptr_t indices);
// typedef void  (APIENTRYP GPENABLE)(GLenum  cap);
// typedef void  (APIENTRYP GPENABLEVERTEXATTRIBARRAY)(GLuint  index);
// typedef void  (APIENTRYP GPFLUSH)();
// typedef void  (APIENTRYP GPFRAMEBUFFERRENDERBUFFEREXT)(GLenum  target, GLenum  attachment, GLenum  renderbuffertarget, GLuint  renderbuffer);
// typedef void  (APIENTRYP GPFRAMEBUFFERTEXTURE2DEXT)(GLenum  target, GLenum  attachment, GLenum  textarget, GLuint  texture, GLint  level);
// typedef void  (APIENTRYP GPGENBUFFERS)(GLsizei  n, GLuint * buffers);
// typedef void  (APIENTRYP GPGENFRAMEBUFFERSEXT)(GLsizei  n, GLuint * framebuffers);
// typedef void  (APIENTRYP GPGENRENDERBUFFERSEXT)(GLsizei  n, GLuint * renderbuffers);
// typedef void  (APIENTRYP GPGENTEXTURES)(GLsizei  n, GLuint * textures);
// typedef GLenum  (APIENTRYP GPGETERROR)();
// typedef void  (APIENTRYP GPGETINTEGERV)(GLenum  pname, GLint * data);
// typedef void  (APIENTRYP GPGETPROGRAMINFOLOG)(GLuint  program, GLsizei  bufSize, GLsizei * length, GLchar * infoLog);
// typedef void  (APIENTRYP GPGETPROGRAMIV)(GLuint  program, GLenum  pname, GLint * params);
// typedef void  (APIENTRYP GPGETSHADERINFOLOG)(GLuint  shader, GLsizei  bufSize, GLsizei * length, GLchar * infoLog);
// typedef void  (APIENTRYP GPGETSHADERIV)(GLuint  shader, GLenum  pname, GLint * params);
// typedef GLint  (APIENTRYP GPGETUNIFORMLOCATION)(GLuint  program, const GLchar * name);
// typedef GLboolean  (APIENTRYP GPISFRAMEBUFFEREXT)(GLuint  framebuffer);
// typedef GLboolean  (APIENTRYP GPISPROGRAM)(GLuint  program);
// typedef GLboolean  (APIENTRYP GPISRENDERBUFFEREXT)(GLuint  renderbuffer);
// typedef GLboolean  (APIENTRYP GPISTEXTURE)(GLuint  texture);
// typedef void  (APIENTRYP GPLINKPROGRAM)(GLuint  program);
// typedef void  (APIENTRYP GPPIXELSTOREI)(GLenum  pname, GLint  param);
// typedef void  (APIENTRYP GPREADPIXELS)(GLint  x, GLint  y, GLsizei  width, GLsizei  height, GLenum  format, GLenum  type, void * pixels);
// typedef void  (APIENTRYP GPRENDERBUFFERSTORAGEEXT)(GLenum  target, GLenum  internalformat, GLsizei  width, GLsizei  height);
// typedef void  (APIENTRYP GPSCISSOR)(GLint  x, GLint  y, GLsizei  width, GLsizei  height);
// typedef void  (APIENTRYP GPSHADERSOURCE)(GLuint  shader, GLsizei  count, const GLchar *const* string, const GLint * length);
// typedef void  (APIENTRYP GPSTENCILFUNC)(GLenum  func, GLint  ref, GLuint  mask);
// typedef void  (APIENTRYP GPSTENCILOP)(GLenum  fail, GLenum  zfail, GLenum  zpass);
// typedef void  (APIENTRYP GPTEXIMAGE2D)(GLenum  target, GLint  level, GLint  internalformat, GLsizei  width, GLsizei  height, GLint  border, GLenum  format, GLenum  type, const void * pixels);
// typedef void  (APIENTRYP GPTEXPARAMETERI)(GLenum  target, GLenum  pname, GLint  param);
// typedef void  (APIENTRYP GPTEXSUBIMAGE2D)(GLenum  target, GLint  level, GLint  xoffset, GLint  yoffset, GLsizei  width, GLsizei  height, GLenum  format, GLenum  type, const void * pixels);
// typedef void  (APIENTRYP GPUNIFORM1FV)(GLint  location, GLsizei  count, const GLfloat * value);
// typedef void  (APIENTRYP GPUNIFORM1I)(GLint  location, GLint  v0);
// typedef void  (APIENTRYP GPUNIFORM1IV)(GLint  location, GLsizei  count, const GLint * value);
// typedef void  (APIENTRYP GPUNIFORM2FV)(GLint  location, GLsizei  count, const GLfloat * value);
// typedef void  (APIENTRYP GPUNIFORM2IV)(GLint  location, GLsizei  count, const GLint * value);
// typedef void  (APIENTRYP GPUNIFORM3FV)(GLint  location, GLsizei  count, const GLfloat * value);
// typedef void  (APIENTRYP GPUNIFORM3IV)(GLint  location, GLsizei  count, const GLint * value);
// typedef void  (APIENTRYP GPUNIFORM4FV)(GLint  location, GLsizei  count, const GLfloat * value);
// typedef void  (APIENTRYP GPUNIFORM4IV)(GLint  location, GLsizei  count, const GLint * value);
// typedef void  (APIENTRYP GPUNIFORMMATRIX2FV)(GLint  location, GLsizei  count, GLboolean  transpose, const GLfloat * value);
// typedef void  (APIENTRYP GPUNIFORMMATRIX3FV)(GLint  location, GLsizei  count, GLboolean  transpose, const GLfloat * value);
// typedef void  (APIENTRYP GPUNIFORMMATRIX4FV)(GLint  location, GLsizei  count, GLboolean  transpose, const GLfloat * value);
// typedef void  (APIENTRYP GPUSEPROGRAM)(GLuint  program);
// typedef void  (APIENTRYP GPVERTEXATTRIBPOINTER)(GLuint  index, GLint  size, GLenum  type, GLboolean  normalized, GLsizei  stride, const uintptr_t pointer);
// typedef void  (APIENTRYP GPVIEWPORT)(GLint  x, GLint  y, GLsizei  width, GLsizei  height);
//
// static void  glowActiveTexture(GPACTIVETEXTURE fnptr, GLenum  texture) {
//   (*fnptr)(texture);
// }
// static void  glowAttachShader(GPATTACHSHADER fnptr, GLuint  program, GLuint  shader) {
//   (*fnptr)(program, shader);
// }
// static void  glowBindAttribLocation(GPBINDATTRIBLOCATION fnptr, GLuint  program, GLuint  index, const GLchar * name) {
//   (*fnptr)(program, index, name);
// }
// static void  glowBindBuffer(GPBINDBUFFER fnptr, GLenum  target, GLuint  buffer) {
//   (*fnptr)(target, buffer);
// }
// static void  glowBindFramebufferEXT(GPBINDFRAMEBUFFEREXT fnptr, GLenum  target, GLuint  framebuffer) {
//   (*fnptr)(target, framebuffer);
// }
// static void  glowBindRenderbufferEXT(GPBINDRENDERBUFFEREXT fnptr, GLenum  target, GLuint  renderbuffer) {
//   (*fnptr)(target, renderbuffer);
// }
// static void  glowBindTexture(GPBINDTEXTURE fnptr, GLenum  target, GLuint  texture) {
//   (*fnptr)(target, texture);
// }
// static void  glowBlendEquationSeparate(GPBLENDEQUATIONSEPARATE fnptr, GLenum  modeRGB, GLenum  modeAlpha) {
//   (*fnptr)(modeRGB, modeAlpha);
// }
// static void  glowBlendFuncSeparate(GPBLENDFUNCSEPARATE fnptr, GLenum  srcRGB, GLenum  dstRGB, GLenum  srcAlpha, GLenum  dstAlpha) {
//   (*fnptr)(srcRGB, dstRGB, srcAlpha, dstAlpha);
// }
// static void  glowBufferData(GPBUFFERDATA fnptr, GLenum  target, GLsizeiptr  size, const void * data, GLenum  usage) {
//   (*fnptr)(target, size, data, usage);
// }
// static void  glowBufferSubData(GPBUFFERSUBDATA fnptr, GLenum  target, GLintptr  offset, GLsizeiptr  size, const void * data) {
//   (*fnptr)(target, offset, size, data);
// }
// static GLenum  glowCheckFramebufferStatusEXT(GPCHECKFRAMEBUFFERSTATUSEXT fnptr, GLenum  target) {
//   return (*fnptr)(target);
// }
// static void  glowClear(GPCLEAR fnptr, GLbitfield  mask) {
//   (*fnptr)(mask);
// }
// static void  glowColorMask(GPCOLORMASK fnptr, GLboolean  red, GLboolean  green, GLboolean  blue, GLboolean  alpha) {
//   (*fnptr)(red, green, blue, alpha);
// }
// static void  glowCompileShader(GPCOMPILESHADER fnptr, GLuint  shader) {
//   (*fnptr)(shader);
// }
// static GLuint  glowCreateProgram(GPCREATEPROGRAM fnptr) {
//   return (*fnptr)();
// }
// static GLuint  glowCreateShader(GPCREATESHADER fnptr, GLenum  type) {
//   return (*fnptr)(type);
// }
// static void  glowDeleteBuffers(GPDELETEBUFFERS fnptr, GLsizei  n, const GLuint * buffers) {
//   (*fnptr)(n, buffers);
// }
// static void  glowDeleteFramebuffersEXT(GPDELETEFRAMEBUFFERSEXT fnptr, GLsizei  n, const GLuint * framebuffers) {
//   (*fnptr)(n, framebuffers);
// }
// static void  glowDeleteProgram(GPDELETEPROGRAM fnptr, GLuint  program) {
//   (*fnptr)(program);
// }
// static void  glowDeleteRenderbuffersEXT(GPDELETERENDERBUFFERSEXT fnptr, GLsizei  n, const GLuint * renderbuffers) {
//   (*fnptr)(n, renderbuffers);
// }
// static void  glowDeleteShader(GPDELETESHADER fnptr, GLuint  shader) {
//   (*fnptr)(shader);
// }
// static void  glowDeleteTextures(GPDELETETEXTURES fnptr, GLsizei  n, const GLuint * textures) {
//   (*fnptr)(n, textures);
// }
// static void  glowDisable(GPDISABLE fnptr, GLenum  cap) {
//   (*fnptr)(cap);
// }
// static void  glowDisableVertexAttribArray(GPDISABLEVERTEXATTRIBARRAY fnptr, GLuint  index) {
//   (*fnptr)(index);
// }
// static void  glowDrawElements(GPDRAWELEMENTS fnptr, GLenum  mode, GLsizei  count, GLenum  type, const uintptr_t indices) {
//   (*fnptr)(mode, count, type, indices);
// }
// static void  glowEnable(GPENABLE fnptr, GLenum  cap) {
//   (*fnptr)(cap);
// }
// static void  glowEnableVertexAttribArray(GPENABLEVERTEXATTRIBARRAY fnptr, GLuint  index) {
//   (*fnptr)(index);
// }
// static void  glowFlush(GPFLUSH fnptr) {
//   (*fnptr)();
// }
// static void  glowFramebufferRenderbufferEXT(GPFRAMEBUFFERRENDERBUFFEREXT fnptr, GLenum  target, GLenum  attachment, GLenum  renderbuffertarget, GLuint  renderbuffer) {
//   (*fnptr)(target, attachment, renderbuffertarget, renderbuffer);
// }
// static void  glowFramebufferTexture2DEXT(GPFRAMEBUFFERTEXTURE2DEXT fnptr, GLenum  target, GLenum  attachment, GLenum  textarget, GLuint  texture, GLint  level) {
//   (*fnptr)(target, attachment, textarget, texture, level);
// }
// static void  glowGenBuffers(GPGENBUFFERS fnptr, GLsizei  n, GLuint * buffers) {
//   (*fnptr)(n, buffers);
// }
// static void  glowGenFramebuffersEXT(GPGENFRAMEBUFFERSEXT fnptr, GLsizei  n, GLuint * framebuffers) {
//   (*fnptr)(n, framebuffers);
// }
// static void  glowGenRenderbuffersEXT(GPGENRENDERBUFFERSEXT fnptr, GLsizei  n, GLuint * renderbuffers) {
//   (*fnptr)(n, renderbuffers);
// }
// static void  glowGenTextures(GPGENTEXTURES fnptr, GLsizei  n, GLuint * textures) {
//   (*fnptr)(n, textures);
// }
// static GLenum  glowGetError(GPGETERROR fnptr) {
//   return (*fnptr)();
// }
// static void  glowGetIntegerv(GPGETINTEGERV fnptr, GLenum  pname, GLint * data) {
//   (*fnptr)(pname, data);
// }
// static void  glowGetProgramInfoLog(GPGETPROGRAMINFOLOG fnptr, GLuint  program, GLsizei  bufSize, GLsizei * length, GLchar * infoLog) {
//   (*fnptr)(program, bufSize, length, infoLog);
// }
// static void  glowGetProgramiv(GPGETPROGRAMIV fnptr, GLuint  program, GLenum  pname, GLint * params) {
//   (*fnptr)(program, pname, params);
// }
// static void  glowGetShaderInfoLog(GPGETSHADERINFOLOG fnptr, GLuint  shader, GLsizei  bufSize, GLsizei * length, GLchar * infoLog) {
//   (*fnptr)(shader, bufSize, length, infoLog);
// }
// static void  glowGetShaderiv(GPGETSHADERIV fnptr, GLuint  shader, GLenum  pname, GLint * params) {
//   (*fnptr)(shader, pname, params);
// }
// static GLint  glowGetUniformLocation(GPGETUNIFORMLOCATION fnptr, GLuint  program, const GLchar * name) {
//   return (*fnptr)(program, name);
// }
// static GLboolean  glowIsFramebufferEXT(GPISFRAMEBUFFEREXT fnptr, GLuint  framebuffer) {
//   return (*fnptr)(framebuffer);
// }
// static GLboolean  glowIsProgram(GPISPROGRAM fnptr, GLuint  program) {
//   return (*fnptr)(program);
// }
// static GLboolean  glowIsRenderbufferEXT(GPISRENDERBUFFEREXT fnptr, GLuint  renderbuffer) {
//   return (*fnptr)(renderbuffer);
// }
// static GLboolean  glowIsTexture(GPISTEXTURE fnptr, GLuint  texture) {
//   return (*fnptr)(texture);
// }
// static void  glowLinkProgram(GPLINKPROGRAM fnptr, GLuint  program) {
//   (*fnptr)(program);
// }
// static void  glowPixelStorei(GPPIXELSTOREI fnptr, GLenum  pname, GLint  param) {
//   (*fnptr)(pname, param);
// }
// static void  glowReadPixels(GPREADPIXELS fnptr, GLint  x, GLint  y, GLsizei  width, GLsizei  height, GLenum  format, GLenum  type, void * pixels) {
//   (*fnptr)(x, y, width, height, format, type, pixels);
// }
// static void  glowRenderbufferStorageEXT(GPRENDERBUFFERSTORAGEEXT fnptr, GLenum  target, GLenum  internalformat, GLsizei  width, GLsizei  height) {
//   (*fnptr)(target, internalformat, width, height);
// }
// static void  glowScissor(GPSCISSOR fnptr, GLint  x, GLint  y, GLsizei  width, GLsizei  height) {
//   (*fnptr)(x, y, width, height);
// }
// static void  glowShaderSource(GPSHADERSOURCE fnptr, GLuint  shader, GLsizei  count, const GLchar *const* string, const GLint * length) {
//   (*fnptr)(shader, count, string, length);
// }
// static void  glowStencilFunc(GPSTENCILFUNC fnptr, GLenum  func, GLint  ref, GLuint  mask) {
//   (*fnptr)(func, ref, mask);
// }
// static void  glowStencilOp(GPSTENCILOP fnptr, GLenum  fail, GLenum  zfail, GLenum  zpass) {
//   (*fnptr)(fail, zfail, zpass);
// }
// static void  glowTexImage2D(GPTEXIMAGE2D fnptr, GLenum  target, GLint  level, GLint  internalformat, GLsizei  width, GLsizei  height, GLint  border, GLenum  format, GLenum  type, const void * pixels) {
//   (*fnptr)(target, level, internalformat, width, height, border, format, type, pixels);
// }
// static void  glowTexParameteri(GPTEXPARAMETERI fnptr, GLenum  target, GLenum  pname, GLint  param) {
//   (*fnptr)(target, pname, param);
// }
// static void  glowTexSubImage2D(GPTEXSUBIMAGE2D fnptr, GLenum  target, GLint  level, GLint  xoffset, GLint  yoffset, GLsizei  width, GLsizei  height, GLenum  format, GLenum  type, const void * pixels) {
//   (*fnptr)(target, level, xoffset, yoffset, width, height, format, type, pixels);
// }
// static void  glowUniform1fv(GPUNIFORM1FV fnptr, GLint  location, GLsizei  count, const GLfloat * value) {
//   (*fnptr)(location, count, value);
// }
// static void  glowUniform1i(GPUNIFORM1I fnptr, GLint  location, GLint  v0) {
//   (*fnptr)(location, v0);
// }
// static void  glowUniform1iv(GPUNIFORM1IV fnptr, GLint  location, GLsizei  count, const GLint * value) {
//   (*fnptr)(location, count, value);
// }
// static void  glowUniform2fv(GPUNIFORM2FV fnptr, GLint  location, GLsizei  count, const GLfloat * value) {
//   (*fnptr)(location, count, value);
// }
// static void  glowUniform2iv(GPUNIFORM2IV fnptr, GLint  location, GLsizei  count, const GLint * value) {
//   (*fnptr)(location, count, value);
// }
// static void  glowUniform3fv(GPUNIFORM3FV fnptr, GLint  location, GLsizei  count, const GLfloat * value) {
//   (*fnptr)(location, count, value);
// }
// static void  glowUniform3iv(GPUNIFORM3IV fnptr, GLint  location, GLsizei  count, const GLint * value) {
//   (*fnptr)(location, count, value);
// }
// static void  glowUniform4fv(GPUNIFORM4FV fnptr, GLint  location, GLsizei  count, const GLfloat * value) {
//   (*fnptr)(location, count, value);
// }
// static void  glowUniform4iv(GPUNIFORM4IV fnptr, GLint  location, GLsizei  count, const GLint * value) {
//   (*fnptr)(location, count, value);
// }
// static void  glowUniformMatrix2fv(GPUNIFORMMATRIX2FV fnptr, GLint  location, GLsizei  count, GLboolean  transpose, const GLfloat * value) {
//   (*fnptr)(location, count, transpose, value);
// }
// static void  glowUniformMatrix3fv(GPUNIFORMMATRIX3FV fnptr, GLint  location, GLsizei  count, GLboolean  transpose, const GLfloat * value) {
//   (*fnptr)(location, count, transpose, value);
// }
// static void  glowUniformMatrix4fv(GPUNIFORMMATRIX4FV fnptr, GLint  location, GLsizei  count, GLboolean  transpose, const GLfloat * value) {
//   (*fnptr)(location, count, transpose, value);
// }
// static void  glowUseProgram(GPUSEPROGRAM fnptr, GLuint  program) {
//   (*fnptr)(program);
// }
// static void  glowVertexAttribPointer(GPVERTEXATTRIBPOINTER fnptr, GLuint  index, GLint  size, GLenum  type, GLboolean  normalized, GLsizei  stride, const uintptr_t pointer) {
//   (*fnptr)(index, size, type, normalized, stride, pointer);
// }
// static void  glowViewport(GPVIEWPORT fnptr, GLint  x, GLint  y, GLsizei  width, GLsizei  height) {
//   (*fnptr)(x, y, width, height);
// }
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

type defaultContext struct {
	gpActiveTexture              C.GPACTIVETEXTURE
	gpAttachShader               C.GPATTACHSHADER
	gpBindAttribLocation         C.GPBINDATTRIBLOCATION
	gpBindBuffer                 C.GPBINDBUFFER
	gpBindFramebufferEXT         C.GPBINDFRAMEBUFFEREXT
	gpBindRenderbufferEXT        C.GPBINDRENDERBUFFEREXT
	gpBindTexture                C.GPBINDTEXTURE
	gpBlendEquationSeparate      C.GPBLENDEQUATIONSEPARATE
	gpBlendFuncSeparate          C.GPBLENDFUNCSEPARATE
	gpBufferData                 C.GPBUFFERDATA
	gpBufferSubData              C.GPBUFFERSUBDATA
	gpCheckFramebufferStatusEXT  C.GPCHECKFRAMEBUFFERSTATUSEXT
	gpClear                      C.GPCLEAR
	gpColorMask                  C.GPCOLORMASK
	gpCompileShader              C.GPCOMPILESHADER
	gpCreateProgram              C.GPCREATEPROGRAM
	gpCreateShader               C.GPCREATESHADER
	gpDeleteBuffers              C.GPDELETEBUFFERS
	gpDeleteFramebuffersEXT      C.GPDELETEFRAMEBUFFERSEXT
	gpDeleteProgram              C.GPDELETEPROGRAM
	gpDeleteRenderbuffersEXT     C.GPDELETERENDERBUFFERSEXT
	gpDeleteShader               C.GPDELETESHADER
	gpDeleteTextures             C.GPDELETETEXTURES
	gpDisable                    C.GPDISABLE
	gpDisableVertexAttribArray   C.GPDISABLEVERTEXATTRIBARRAY
	gpDrawElements               C.GPDRAWELEMENTS
	gpEnable                     C.GPENABLE
	gpEnableVertexAttribArray    C.GPENABLEVERTEXATTRIBARRAY
	gpFlush                      C.GPFLUSH
	gpFramebufferRenderbufferEXT C.GPFRAMEBUFFERRENDERBUFFEREXT
	gpFramebufferTexture2DEXT    C.GPFRAMEBUFFERTEXTURE2DEXT
	gpGenBuffers                 C.GPGENBUFFERS
	gpGenFramebuffersEXT         C.GPGENFRAMEBUFFERSEXT
	gpGenRenderbuffersEXT        C.GPGENRENDERBUFFERSEXT
	gpGenTextures                C.GPGENTEXTURES
	gpGetError                   C.GPGETERROR
	gpGetIntegerv                C.GPGETINTEGERV
	gpGetProgramInfoLog          C.GPGETPROGRAMINFOLOG
	gpGetProgramiv               C.GPGETPROGRAMIV
	gpGetShaderInfoLog           C.GPGETSHADERINFOLOG
	gpGetShaderiv                C.GPGETSHADERIV
	gpGetUniformLocation         C.GPGETUNIFORMLOCATION
	gpIsFramebufferEXT           C.GPISFRAMEBUFFEREXT
	gpIsProgram                  C.GPISPROGRAM
	gpIsRenderbufferEXT          C.GPISRENDERBUFFEREXT
	gpIsTexture                  C.GPISTEXTURE
	gpLinkProgram                C.GPLINKPROGRAM
	gpPixelStorei                C.GPPIXELSTOREI
	gpReadPixels                 C.GPREADPIXELS
	gpRenderbufferStorageEXT     C.GPRENDERBUFFERSTORAGEEXT
	gpScissor                    C.GPSCISSOR
	gpShaderSource               C.GPSHADERSOURCE
	gpStencilFunc                C.GPSTENCILFUNC
	gpStencilOp                  C.GPSTENCILOP
	gpTexImage2D                 C.GPTEXIMAGE2D
	gpTexParameteri              C.GPTEXPARAMETERI
	gpTexSubImage2D              C.GPTEXSUBIMAGE2D
	gpUniform1fv                 C.GPUNIFORM1FV
	gpUniform1i                  C.GPUNIFORM1I
	gpUniform1iv                 C.GPUNIFORM1IV
	gpUniform2fv                 C.GPUNIFORM2FV
	gpUniform2iv                 C.GPUNIFORM2IV
	gpUniform3fv                 C.GPUNIFORM3FV
	gpUniform3iv                 C.GPUNIFORM3IV
	gpUniform4fv                 C.GPUNIFORM4FV
	gpUniform4iv                 C.GPUNIFORM4IV
	gpUniformMatrix2fv           C.GPUNIFORMMATRIX2FV
	gpUniformMatrix3fv           C.GPUNIFORMMATRIX3FV
	gpUniformMatrix4fv           C.GPUNIFORMMATRIX4FV
	gpUseProgram                 C.GPUSEPROGRAM
	gpVertexAttribPointer        C.GPVERTEXATTRIBPOINTER
	gpViewport                   C.GPVIEWPORT

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
	C.glowActiveTexture(c.gpActiveTexture, (C.GLenum)(texture))
}

func (c *defaultContext) AttachShader(program uint32, shader uint32) {
	C.glowAttachShader(c.gpAttachShader, (C.GLuint)(program), (C.GLuint)(shader))
}

func (c *defaultContext) BindAttribLocation(program uint32, index uint32, name string) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	C.glowBindAttribLocation(c.gpBindAttribLocation, (C.GLuint)(program), (C.GLuint)(index), (*C.GLchar)(unsafe.Pointer(cname)))
}

func (c *defaultContext) BindBuffer(target uint32, buffer uint32) {
	C.glowBindBuffer(c.gpBindBuffer, (C.GLenum)(target), (C.GLuint)(buffer))
}

func (c *defaultContext) BindFramebuffer(target uint32, framebuffer uint32) {
	C.glowBindFramebufferEXT(c.gpBindFramebufferEXT, (C.GLenum)(target), (C.GLuint)(framebuffer))
}

func (c *defaultContext) BindRenderbuffer(target uint32, renderbuffer uint32) {
	C.glowBindRenderbufferEXT(c.gpBindRenderbufferEXT, (C.GLenum)(target), (C.GLuint)(renderbuffer))
}

func (c *defaultContext) BindTexture(target uint32, texture uint32) {
	C.glowBindTexture(c.gpBindTexture, (C.GLenum)(target), (C.GLuint)(texture))
}

func (c *defaultContext) BlendEquationSeparate(modeRGB uint32, modeAlpha uint32) {
	C.glowBlendEquationSeparate(c.gpBlendEquationSeparate, (C.GLenum)(modeRGB), (C.GLenum)(modeAlpha))
}

func (c *defaultContext) BlendFuncSeparate(srcRGB uint32, dstRGB uint32, srcAlpha uint32, dstAlpha uint32) {
	C.glowBlendFuncSeparate(c.gpBlendFuncSeparate, (C.GLenum)(srcRGB), (C.GLenum)(dstRGB), (C.GLenum)(srcAlpha), (C.GLenum)(dstAlpha))
}

func (c *defaultContext) BufferInit(target uint32, size int, usage uint32) {
	C.glowBufferData(c.gpBufferData, (C.GLenum)(target), (C.GLsizeiptr)(size), nil, (C.GLenum)(usage))
}

func (c *defaultContext) BufferSubData(target uint32, offset int, data []byte) {
	C.glowBufferSubData(c.gpBufferSubData, (C.GLenum)(target), (C.GLintptr)(offset), (C.GLsizeiptr)(len(data)), unsafe.Pointer(&data[0]))
	runtime.KeepAlive(data)
}

func (c *defaultContext) CheckFramebufferStatus(target uint32) uint32 {
	ret := C.glowCheckFramebufferStatusEXT(c.gpCheckFramebufferStatusEXT, (C.GLenum)(target))
	return uint32(ret)
}

func (c *defaultContext) Clear(mask uint32) {
	C.glowClear(c.gpClear, (C.GLbitfield)(mask))
}

func (c *defaultContext) ColorMask(red bool, green bool, blue bool, alpha bool) {
	C.glowColorMask(c.gpColorMask, (C.GLboolean)(boolToInt(red)), (C.GLboolean)(boolToInt(green)), (C.GLboolean)(boolToInt(blue)), (C.GLboolean)(boolToInt(alpha)))
}

func (c *defaultContext) CompileShader(shader uint32) {
	C.glowCompileShader(c.gpCompileShader, (C.GLuint)(shader))
}

func (c *defaultContext) CreateBuffer() uint32 {
	var buffer uint32
	C.glowGenBuffers(c.gpGenBuffers, 1, (*C.GLuint)(unsafe.Pointer(&buffer)))
	return buffer
}

func (c *defaultContext) CreateFramebuffer() uint32 {
	var framebuffer uint32
	C.glowGenFramebuffersEXT(c.gpGenFramebuffersEXT, 1, (*C.GLuint)(unsafe.Pointer(&framebuffer)))
	return framebuffer
}

func (c *defaultContext) CreateProgram() uint32 {
	ret := C.glowCreateProgram(c.gpCreateProgram)
	return uint32(ret)
}

func (c *defaultContext) CreateRenderbuffer() uint32 {
	var renderbuffer uint32
	C.glowGenRenderbuffersEXT(c.gpGenRenderbuffersEXT, 1, (*C.GLuint)(unsafe.Pointer(&renderbuffer)))
	return renderbuffer
}

func (c *defaultContext) CreateShader(xtype uint32) uint32 {
	ret := C.glowCreateShader(c.gpCreateShader, (C.GLenum)(xtype))
	return uint32(ret)
}

func (c *defaultContext) CreateTexture() uint32 {
	var texture uint32
	C.glowGenTextures(c.gpGenTextures, 1, (*C.GLuint)(unsafe.Pointer(&texture)))
	return texture
}

func (c *defaultContext) DeleteBuffer(buffer uint32) {
	C.glowDeleteBuffers(c.gpDeleteBuffers, 1, (*C.GLuint)(unsafe.Pointer(&buffer)))
}

func (c *defaultContext) DeleteFramebuffer(framebuffer uint32) {
	C.glowDeleteFramebuffersEXT(c.gpDeleteFramebuffersEXT, 1, (*C.GLuint)(unsafe.Pointer(&framebuffer)))
}

func (c *defaultContext) DeleteProgram(program uint32) {
	C.glowDeleteProgram(c.gpDeleteProgram, (C.GLuint)(program))
}

func (c *defaultContext) DeleteRenderbuffer(renderbuffer uint32) {
	C.glowDeleteRenderbuffersEXT(c.gpDeleteRenderbuffersEXT, 1, (*C.GLuint)(unsafe.Pointer(&renderbuffer)))
}

func (c *defaultContext) DeleteShader(shader uint32) {
	C.glowDeleteShader(c.gpDeleteShader, (C.GLuint)(shader))
}

func (c *defaultContext) DeleteTexture(texture uint32) {
	C.glowDeleteTextures(c.gpDeleteTextures, 1, (*C.GLuint)(unsafe.Pointer(&texture)))
}

func (c *defaultContext) Disable(cap uint32) {
	C.glowDisable(c.gpDisable, (C.GLenum)(cap))
}

func (c *defaultContext) DisableVertexAttribArray(index uint32) {
	C.glowDisableVertexAttribArray(c.gpDisableVertexAttribArray, (C.GLuint)(index))
}

func (c *defaultContext) DrawElements(mode uint32, count int32, xtype uint32, offset int) {
	C.glowDrawElements(c.gpDrawElements, (C.GLenum)(mode), (C.GLsizei)(count), (C.GLenum)(xtype), C.uintptr_t(offset))
}

func (c *defaultContext) Enable(cap uint32) {
	C.glowEnable(c.gpEnable, (C.GLenum)(cap))
}

func (c *defaultContext) EnableVertexAttribArray(index uint32) {
	C.glowEnableVertexAttribArray(c.gpEnableVertexAttribArray, (C.GLuint)(index))
}

func (c *defaultContext) Flush() {
	C.glowFlush(c.gpFlush)
}

func (c *defaultContext) FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	C.glowFramebufferRenderbufferEXT(c.gpFramebufferRenderbufferEXT, (C.GLenum)(target), (C.GLenum)(attachment), (C.GLenum)(renderbuffertarget), (C.GLuint)(renderbuffer))
}

func (c *defaultContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	C.glowFramebufferTexture2DEXT(c.gpFramebufferTexture2DEXT, (C.GLenum)(target), (C.GLenum)(attachment), (C.GLenum)(textarget), (C.GLuint)(texture), (C.GLint)(level))
}

func (c *defaultContext) GetError() uint32 {
	ret := C.glowGetError(c.gpGetError)
	return uint32(ret)
}

func (c *defaultContext) GetInteger(pname uint32) int {
	var dst int32
	C.glowGetIntegerv(c.gpGetIntegerv, (C.GLenum)(pname), (*C.GLint)(unsafe.Pointer(&dst)))
	return int(dst)
}

func (c *defaultContext) GetProgramInfoLog(program uint32) string {
	bufSize := c.GetProgrami(program, INFO_LOG_LENGTH)
	infoLog := make([]byte, bufSize)
	C.glowGetProgramInfoLog(c.gpGetProgramInfoLog, (C.GLuint)(program), (C.GLsizei)(bufSize), nil, (*C.GLchar)(unsafe.Pointer(&infoLog[0])))
	return string(infoLog)
}

func (c *defaultContext) GetProgrami(program uint32, pname uint32) int {
	var dst int32
	C.glowGetProgramiv(c.gpGetProgramiv, (C.GLuint)(program), (C.GLenum)(pname), (*C.GLint)(unsafe.Pointer(&dst)))
	return int(dst)
}

func (c *defaultContext) GetShaderInfoLog(shader uint32) string {
	bufSize := c.GetShaderi(shader, INFO_LOG_LENGTH)
	infoLog := make([]byte, bufSize)
	C.glowGetShaderInfoLog(c.gpGetShaderInfoLog, (C.GLuint)(shader), (C.GLsizei)(bufSize), nil, (*C.GLchar)(unsafe.Pointer(&infoLog[0])))
	return string(infoLog)
}

func (c *defaultContext) GetShaderi(shader uint32, pname uint32) int {
	var dst int32
	C.glowGetShaderiv(c.gpGetShaderiv, (C.GLuint)(shader), (C.GLenum)(pname), (*C.GLint)(unsafe.Pointer(&dst)))
	return int(dst)
}

func (c *defaultContext) GetUniformLocation(program uint32, name string) int32 {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	ret := C.glowGetUniformLocation(c.gpGetUniformLocation, (C.GLuint)(program), (*C.GLchar)(unsafe.Pointer(cname)))
	return int32(ret)
}

func (c *defaultContext) IsFramebuffer(framebuffer uint32) bool {
	ret := C.glowIsFramebufferEXT(c.gpIsFramebufferEXT, (C.GLuint)(framebuffer))
	return ret == TRUE
}

func (c *defaultContext) IsProgram(program uint32) bool {
	ret := C.glowIsProgram(c.gpIsProgram, (C.GLuint)(program))
	return ret == TRUE
}

func (c *defaultContext) IsRenderbuffer(renderbuffer uint32) bool {
	ret := C.glowIsRenderbufferEXT(c.gpIsRenderbufferEXT, (C.GLuint)(renderbuffer))
	return ret == TRUE
}

func (c *defaultContext) IsTexture(texture uint32) bool {
	ret := C.glowIsTexture(c.gpIsTexture, (C.GLuint)(texture))
	return ret == TRUE
}

func (c *defaultContext) LinkProgram(program uint32) {
	C.glowLinkProgram(c.gpLinkProgram, (C.GLuint)(program))
}

func (c *defaultContext) PixelStorei(pname uint32, param int32) {
	C.glowPixelStorei(c.gpPixelStorei, (C.GLenum)(pname), (C.GLint)(param))
}

func (c *defaultContext) ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32) {
	C.glowReadPixels(c.gpReadPixels, (C.GLint)(x), (C.GLint)(y), (C.GLsizei)(width), (C.GLsizei)(height), (C.GLenum)(format), (C.GLenum)(xtype), unsafe.Pointer(&dst[0]))
}

func (c *defaultContext) RenderbufferStorage(target uint32, internalformat uint32, width int32, height int32) {
	C.glowRenderbufferStorageEXT(c.gpRenderbufferStorageEXT, (C.GLenum)(target), (C.GLenum)(internalformat), (C.GLsizei)(width), (C.GLsizei)(height))
}

func (c *defaultContext) Scissor(x int32, y int32, width int32, height int32) {
	C.glowScissor(c.gpScissor, (C.GLint)(x), (C.GLint)(y), (C.GLsizei)(width), (C.GLsizei)(height))
}

func (c *defaultContext) ShaderSource(shader uint32, xstring string) {
	cstring := C.CString(xstring)
	defer C.free(unsafe.Pointer(cstring))
	C.glowShaderSource(c.gpShaderSource, (C.GLuint)(shader), 1, (**C.GLchar)(unsafe.Pointer(&cstring)), nil)
}

func (c *defaultContext) StencilFunc(xfunc uint32, ref int32, mask uint32) {
	C.glowStencilFunc(c.gpStencilFunc, (C.GLenum)(xfunc), (C.GLint)(ref), (C.GLuint)(mask))
}

func (c *defaultContext) StencilOp(fail uint32, zfail uint32, zpass uint32) {
	C.glowStencilOp(c.gpStencilOp, (C.GLenum)(fail), (C.GLenum)(zfail), (C.GLenum)(zpass))
}

func (c *defaultContext) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	var ptr *byte
	if len(pixels) > 0 {
		ptr = &pixels[0]
	}
	C.glowTexImage2D(c.gpTexImage2D, (C.GLenum)(target), (C.GLint)(level), (C.GLint)(internalformat), (C.GLsizei)(width), (C.GLsizei)(height), 0, (C.GLenum)(format), (C.GLenum)(xtype), unsafe.Pointer(ptr))
	runtime.KeepAlive(pixels)
}

func (c *defaultContext) TexParameteri(target uint32, pname uint32, param int32) {
	C.glowTexParameteri(c.gpTexParameteri, (C.GLenum)(target), (C.GLenum)(pname), (C.GLint)(param))
}

func (c *defaultContext) TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	C.glowTexSubImage2D(c.gpTexSubImage2D, (C.GLenum)(target), (C.GLint)(level), (C.GLint)(xoffset), (C.GLint)(yoffset), (C.GLsizei)(width), (C.GLsizei)(height), (C.GLenum)(format), (C.GLenum)(xtype), unsafe.Pointer(&pixels[0]))
	runtime.KeepAlive(pixels)
}

func (c *defaultContext) Uniform1fv(location int32, value []float32) {
	C.glowUniform1fv(c.gpUniform1fv, (C.GLint)(location), (C.GLsizei)(len(value)), (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform1i(location int32, v0 int32) {
	C.glowUniform1i(c.gpUniform1i, (C.GLint)(location), (C.GLint)(v0))
}

func (c *defaultContext) Uniform1iv(location int32, value []int32) {
	C.glowUniform1iv(c.gpUniform1iv, (C.GLint)(location), (C.GLsizei)(len(value)), (*C.GLint)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform2fv(location int32, value []float32) {
	C.glowUniform2fv(c.gpUniform2fv, (C.GLint)(location), (C.GLsizei)(len(value)/2), (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform2iv(location int32, value []int32) {
	C.glowUniform2iv(c.gpUniform2iv, (C.GLint)(location), (C.GLsizei)(len(value)/2), (*C.GLint)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform3fv(location int32, value []float32) {
	C.glowUniform3fv(c.gpUniform3fv, (C.GLint)(location), (C.GLsizei)(len(value)/3), (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform3iv(location int32, value []int32) {
	C.glowUniform3iv(c.gpUniform3iv, (C.GLint)(location), (C.GLsizei)(len(value)/3), (*C.GLint)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform4fv(location int32, value []float32) {
	C.glowUniform4fv(c.gpUniform4fv, (C.GLint)(location), (C.GLsizei)(len(value)/4), (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform4iv(location int32, value []int32) {
	C.glowUniform4iv(c.gpUniform4iv, (C.GLint)(location), (C.GLsizei)(len(value)/4), (*C.GLint)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UniformMatrix2fv(location int32, value []float32) {
	C.glowUniformMatrix2fv(c.gpUniformMatrix2fv, (C.GLint)(location), (C.GLsizei)(len(value)/4), 0, (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UniformMatrix3fv(location int32, value []float32) {
	C.glowUniformMatrix3fv(c.gpUniformMatrix3fv, (C.GLint)(location), (C.GLsizei)(len(value)/9), 0, (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UniformMatrix4fv(location int32, value []float32) {
	C.glowUniformMatrix4fv(c.gpUniformMatrix4fv, (C.GLint)(location), (C.GLsizei)(len(value)/16), 0, (*C.GLfloat)(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UseProgram(program uint32) {
	C.glowUseProgram(c.gpUseProgram, (C.GLuint)(program))
}

func (c *defaultContext) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int) {
	C.glowVertexAttribPointer(c.gpVertexAttribPointer, (C.GLuint)(index), (C.GLint)(size), (C.GLenum)(xtype), (C.GLboolean)(boolToInt(normalized)), (C.GLsizei)(stride), C.uintptr_t(offset))
}

func (c *defaultContext) Viewport(x int32, y int32, width int32, height int32) {
	C.glowViewport(c.gpViewport, (C.GLint)(x), (C.GLint)(y), (C.GLsizei)(width), (C.GLsizei)(height))
}

func (c *defaultContext) LoadFunctions() error {
	c.gpActiveTexture = (C.GPACTIVETEXTURE)(c.getProcAddress("glActiveTexture"))
	if c.gpActiveTexture == nil {
		return errors.New("gl: glActiveTexture is missing")
	}
	c.gpAttachShader = (C.GPATTACHSHADER)(c.getProcAddress("glAttachShader"))
	if c.gpAttachShader == nil {
		return errors.New("gl: glAttachShader is missing")
	}
	c.gpBindAttribLocation = (C.GPBINDATTRIBLOCATION)(c.getProcAddress("glBindAttribLocation"))
	if c.gpBindAttribLocation == nil {
		return errors.New("gl: glBindAttribLocation is missing")
	}
	c.gpBindBuffer = (C.GPBINDBUFFER)(c.getProcAddress("glBindBuffer"))
	if c.gpBindBuffer == nil {
		return errors.New("gl: glBindBuffer is missing")
	}
	c.gpBindFramebufferEXT = (C.GPBINDFRAMEBUFFEREXT)(c.getProcAddress("glBindFramebufferEXT"))
	c.gpBindRenderbufferEXT = (C.GPBINDRENDERBUFFEREXT)(c.getProcAddress("glBindRenderbufferEXT"))
	c.gpBindTexture = (C.GPBINDTEXTURE)(c.getProcAddress("glBindTexture"))
	if c.gpBindTexture == nil {
		return errors.New("gl: glBindTexture is missing")
	}
	c.gpBlendEquationSeparate = (C.GPBLENDEQUATIONSEPARATE)(c.getProcAddress("glBlendEquationSeparate"))
	if c.gpBlendEquationSeparate == nil {
		return errors.New("gl: glBlendEquationSeparate is missing")
	}
	c.gpBlendFuncSeparate = (C.GPBLENDFUNCSEPARATE)(c.getProcAddress("glBlendFuncSeparate"))
	if c.gpBlendFuncSeparate == nil {
		return errors.New("gl: glBlendFuncSeparate is missing")
	}
	c.gpBufferData = (C.GPBUFFERDATA)(c.getProcAddress("glBufferData"))
	if c.gpBufferData == nil {
		return errors.New("gl: glBufferData is missing")
	}
	c.gpBufferSubData = (C.GPBUFFERSUBDATA)(c.getProcAddress("glBufferSubData"))
	if c.gpBufferSubData == nil {
		return errors.New("gl: glBufferSubData is missing")
	}
	c.gpCheckFramebufferStatusEXT = (C.GPCHECKFRAMEBUFFERSTATUSEXT)(c.getProcAddress("glCheckFramebufferStatusEXT"))
	c.gpClear = (C.GPCLEAR)(c.getProcAddress("glClear"))
	if c.gpClear == nil {
		return errors.New("gl: glClear is missing")
	}
	c.gpColorMask = (C.GPCOLORMASK)(c.getProcAddress("glColorMask"))
	if c.gpColorMask == nil {
		return errors.New("gl: glColorMask is missing")
	}
	c.gpCompileShader = (C.GPCOMPILESHADER)(c.getProcAddress("glCompileShader"))
	if c.gpCompileShader == nil {
		return errors.New("gl: glCompileShader is missing")
	}
	c.gpCreateProgram = (C.GPCREATEPROGRAM)(c.getProcAddress("glCreateProgram"))
	if c.gpCreateProgram == nil {
		return errors.New("gl: glCreateProgram is missing")
	}
	c.gpCreateShader = (C.GPCREATESHADER)(c.getProcAddress("glCreateShader"))
	if c.gpCreateShader == nil {
		return errors.New("gl: glCreateShader is missing")
	}
	c.gpDeleteBuffers = (C.GPDELETEBUFFERS)(c.getProcAddress("glDeleteBuffers"))
	if c.gpDeleteBuffers == nil {
		return errors.New("gl: glDeleteBuffers is missing")
	}
	c.gpDeleteFramebuffersEXT = (C.GPDELETEFRAMEBUFFERSEXT)(c.getProcAddress("glDeleteFramebuffersEXT"))
	c.gpDeleteProgram = (C.GPDELETEPROGRAM)(c.getProcAddress("glDeleteProgram"))
	if c.gpDeleteProgram == nil {
		return errors.New("gl: glDeleteProgram is missing")
	}
	c.gpDeleteRenderbuffersEXT = (C.GPDELETERENDERBUFFERSEXT)(c.getProcAddress("glDeleteRenderbuffersEXT"))
	c.gpDeleteShader = (C.GPDELETESHADER)(c.getProcAddress("glDeleteShader"))
	if c.gpDeleteShader == nil {
		return errors.New("gl: glDeleteShader is missing")
	}
	c.gpDeleteTextures = (C.GPDELETETEXTURES)(c.getProcAddress("glDeleteTextures"))
	if c.gpDeleteTextures == nil {
		return errors.New("gl: glDeleteTextures is missing")
	}
	c.gpDisable = (C.GPDISABLE)(c.getProcAddress("glDisable"))
	if c.gpDisable == nil {
		return errors.New("gl: glDisable is missing")
	}
	c.gpDisableVertexAttribArray = (C.GPDISABLEVERTEXATTRIBARRAY)(c.getProcAddress("glDisableVertexAttribArray"))
	if c.gpDisableVertexAttribArray == nil {
		return errors.New("gl: glDisableVertexAttribArray is missing")
	}
	c.gpDrawElements = (C.GPDRAWELEMENTS)(c.getProcAddress("glDrawElements"))
	if c.gpDrawElements == nil {
		return errors.New("gl: glDrawElements is missing")
	}
	c.gpEnable = (C.GPENABLE)(c.getProcAddress("glEnable"))
	if c.gpEnable == nil {
		return errors.New("gl: glEnable is missing")
	}
	c.gpEnableVertexAttribArray = (C.GPENABLEVERTEXATTRIBARRAY)(c.getProcAddress("glEnableVertexAttribArray"))
	if c.gpEnableVertexAttribArray == nil {
		return errors.New("gl: glEnableVertexAttribArray is missing")
	}
	c.gpFlush = (C.GPFLUSH)(c.getProcAddress("glFlush"))
	if c.gpFlush == nil {
		return errors.New("gl: glFlush is missing")
	}
	c.gpFramebufferRenderbufferEXT = (C.GPFRAMEBUFFERRENDERBUFFEREXT)(c.getProcAddress("glFramebufferRenderbufferEXT"))
	c.gpFramebufferTexture2DEXT = (C.GPFRAMEBUFFERTEXTURE2DEXT)(c.getProcAddress("glFramebufferTexture2DEXT"))
	c.gpGenBuffers = (C.GPGENBUFFERS)(c.getProcAddress("glGenBuffers"))
	if c.gpGenBuffers == nil {
		return errors.New("gl: glGenBuffers is missing")
	}
	c.gpGenFramebuffersEXT = (C.GPGENFRAMEBUFFERSEXT)(c.getProcAddress("glGenFramebuffersEXT"))
	c.gpGenRenderbuffersEXT = (C.GPGENRENDERBUFFERSEXT)(c.getProcAddress("glGenRenderbuffersEXT"))
	c.gpGenTextures = (C.GPGENTEXTURES)(c.getProcAddress("glGenTextures"))
	if c.gpGenTextures == nil {
		return errors.New("gl: glGenTextures is missing")
	}
	c.gpGetError = (C.GPGETERROR)(c.getProcAddress("glGetError"))
	if c.gpGetError == nil {
		return errors.New("gl: glGetError is missing")
	}
	c.gpGetIntegerv = (C.GPGETINTEGERV)(c.getProcAddress("glGetIntegerv"))
	if c.gpGetIntegerv == nil {
		return errors.New("gl: glGetIntegerv is missing")
	}
	c.gpGetProgramInfoLog = (C.GPGETPROGRAMINFOLOG)(c.getProcAddress("glGetProgramInfoLog"))
	if c.gpGetProgramInfoLog == nil {
		return errors.New("gl: glGetProgramInfoLog is missing")
	}
	c.gpGetProgramiv = (C.GPGETPROGRAMIV)(c.getProcAddress("glGetProgramiv"))
	if c.gpGetProgramiv == nil {
		return errors.New("gl: glGetProgramiv is missing")
	}
	c.gpGetShaderInfoLog = (C.GPGETSHADERINFOLOG)(c.getProcAddress("glGetShaderInfoLog"))
	if c.gpGetShaderInfoLog == nil {
		return errors.New("gl: glGetShaderInfoLog is missing")
	}
	c.gpGetShaderiv = (C.GPGETSHADERIV)(c.getProcAddress("glGetShaderiv"))
	if c.gpGetShaderiv == nil {
		return errors.New("gl: glGetShaderiv is missing")
	}
	c.gpGetUniformLocation = (C.GPGETUNIFORMLOCATION)(c.getProcAddress("glGetUniformLocation"))
	if c.gpGetUniformLocation == nil {
		return errors.New("gl: glGetUniformLocation is missing")
	}
	c.gpIsFramebufferEXT = (C.GPISFRAMEBUFFEREXT)(c.getProcAddress("glIsFramebufferEXT"))
	c.gpIsProgram = (C.GPISPROGRAM)(c.getProcAddress("glIsProgram"))
	if c.gpIsProgram == nil {
		return errors.New("gl: glIsProgram is missing")
	}
	c.gpIsRenderbufferEXT = (C.GPISRENDERBUFFEREXT)(c.getProcAddress("glIsRenderbufferEXT"))
	c.gpIsTexture = (C.GPISTEXTURE)(c.getProcAddress("glIsTexture"))
	if c.gpIsTexture == nil {
		return errors.New("gl: glIsTexture is missing")
	}
	c.gpLinkProgram = (C.GPLINKPROGRAM)(c.getProcAddress("glLinkProgram"))
	if c.gpLinkProgram == nil {
		return errors.New("gl: glLinkProgram is missing")
	}
	c.gpPixelStorei = (C.GPPIXELSTOREI)(c.getProcAddress("glPixelStorei"))
	if c.gpPixelStorei == nil {
		return errors.New("gl: glPixelStorei is missing")
	}
	c.gpReadPixels = (C.GPREADPIXELS)(c.getProcAddress("glReadPixels"))
	if c.gpReadPixels == nil {
		return errors.New("gl: glReadPixels is missing")
	}
	c.gpRenderbufferStorageEXT = (C.GPRENDERBUFFERSTORAGEEXT)(c.getProcAddress("glRenderbufferStorageEXT"))
	c.gpScissor = (C.GPSCISSOR)(c.getProcAddress("glScissor"))
	if c.gpScissor == nil {
		return errors.New("gl: glScissor is missing")
	}
	c.gpShaderSource = (C.GPSHADERSOURCE)(c.getProcAddress("glShaderSource"))
	if c.gpShaderSource == nil {
		return errors.New("gl: glShaderSource is missing")
	}
	c.gpStencilFunc = (C.GPSTENCILFUNC)(c.getProcAddress("glStencilFunc"))
	if c.gpStencilFunc == nil {
		return errors.New("gl: glStencilFunc is missing")
	}
	c.gpStencilOp = (C.GPSTENCILOP)(c.getProcAddress("glStencilOp"))
	if c.gpStencilOp == nil {
		return errors.New("gl: glStencilOp is missing")
	}
	c.gpTexImage2D = (C.GPTEXIMAGE2D)(c.getProcAddress("glTexImage2D"))
	if c.gpTexImage2D == nil {
		return errors.New("gl: glTexImage2D is missing")
	}
	c.gpTexParameteri = (C.GPTEXPARAMETERI)(c.getProcAddress("glTexParameteri"))
	if c.gpTexParameteri == nil {
		return errors.New("gl: glTexParameteri is missing")
	}
	c.gpTexSubImage2D = (C.GPTEXSUBIMAGE2D)(c.getProcAddress("glTexSubImage2D"))
	if c.gpTexSubImage2D == nil {
		return errors.New("gl: glTexSubImage2D is missing")
	}
	c.gpUniform1fv = (C.GPUNIFORM1FV)(c.getProcAddress("glUniform1fv"))
	if c.gpUniform1fv == nil {
		return errors.New("gl: glUniform1fv is missing")
	}
	c.gpUniform1i = (C.GPUNIFORM1I)(c.getProcAddress("glUniform1i"))
	if c.gpUniform1i == nil {
		return errors.New("gl: glUniform1i is missing")
	}
	c.gpUniform1iv = (C.GPUNIFORM1IV)(c.getProcAddress("glUniform1iv"))
	if c.gpUniform1iv == nil {
		return errors.New("gl: glUniform1iv is missing")
	}
	c.gpUniform2fv = (C.GPUNIFORM2FV)(c.getProcAddress("glUniform2fv"))
	if c.gpUniform2fv == nil {
		return errors.New("gl: glUniform2fv is missing")
	}
	c.gpUniform2iv = (C.GPUNIFORM2IV)(c.getProcAddress("glUniform2iv"))
	if c.gpUniform2iv == nil {
		return errors.New("gl: glUniform2iv is missing")
	}
	c.gpUniform3fv = (C.GPUNIFORM3FV)(c.getProcAddress("glUniform3fv"))
	if c.gpUniform3fv == nil {
		return errors.New("gl: glUniform3fv is missing")
	}
	c.gpUniform3iv = (C.GPUNIFORM3IV)(c.getProcAddress("glUniform3iv"))
	if c.gpUniform3iv == nil {
		return errors.New("gl: glUniform3iv is missing")
	}
	c.gpUniform4fv = (C.GPUNIFORM4FV)(c.getProcAddress("glUniform4fv"))
	if c.gpUniform4fv == nil {
		return errors.New("gl: glUniform4fv is missing")
	}
	c.gpUniform4iv = (C.GPUNIFORM4IV)(c.getProcAddress("glUniform4iv"))
	if c.gpUniform4iv == nil {
		return errors.New("gl: glUniform4iv is missing")
	}
	c.gpUniformMatrix2fv = (C.GPUNIFORMMATRIX2FV)(c.getProcAddress("glUniformMatrix2fv"))
	if c.gpUniformMatrix2fv == nil {
		return errors.New("gl: glUniformMatrix2fv is missing")
	}
	c.gpUniformMatrix3fv = (C.GPUNIFORMMATRIX3FV)(c.getProcAddress("glUniformMatrix3fv"))
	if c.gpUniformMatrix3fv == nil {
		return errors.New("gl: glUniformMatrix3fv is missing")
	}
	c.gpUniformMatrix4fv = (C.GPUNIFORMMATRIX4FV)(c.getProcAddress("glUniformMatrix4fv"))
	if c.gpUniformMatrix4fv == nil {
		return errors.New("gl: glUniformMatrix4fv is missing")
	}
	c.gpUseProgram = (C.GPUSEPROGRAM)(c.getProcAddress("glUseProgram"))
	if c.gpUseProgram == nil {
		return errors.New("gl: glUseProgram is missing")
	}
	c.gpVertexAttribPointer = (C.GPVERTEXATTRIBPOINTER)(c.getProcAddress("glVertexAttribPointer"))
	if c.gpVertexAttribPointer == nil {
		return errors.New("gl: glVertexAttribPointer is missing")
	}
	c.gpViewport = (C.GPVIEWPORT)(c.getProcAddress("glViewport"))
	if c.gpViewport == nil {
		return errors.New("gl: glViewport is missing")
	}
	return nil
}
