// Copyright 2022 The Ebitengine Authors
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

//go:build darwin || windows

package gl

import (
	"errors"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

type defaultContext struct {
	gpActiveTexture              uintptr
	gpAttachShader               uintptr
	gpBindAttribLocation         uintptr
	gpBindBuffer                 uintptr
	gpBindFramebufferEXT         uintptr
	gpBindRenderbufferEXT        uintptr
	gpBindTexture                uintptr
	gpBlendEquationSeparate      uintptr
	gpBlendFuncSeparate          uintptr
	gpBufferData                 uintptr
	gpBufferSubData              uintptr
	gpCheckFramebufferStatusEXT  uintptr
	gpClear                      uintptr
	gpColorMask                  uintptr
	gpCompileShader              uintptr
	gpCreateProgram              uintptr
	gpCreateShader               uintptr
	gpDeleteBuffers              uintptr
	gpDeleteFramebuffersEXT      uintptr
	gpDeleteProgram              uintptr
	gpDeleteRenderbuffersEXT     uintptr
	gpDeleteShader               uintptr
	gpDeleteTextures             uintptr
	gpDisable                    uintptr
	gpDisableVertexAttribArray   uintptr
	gpDrawElements               uintptr
	gpEnable                     uintptr
	gpEnableVertexAttribArray    uintptr
	gpFlush                      uintptr
	gpFramebufferRenderbufferEXT uintptr
	gpFramebufferTexture2DEXT    uintptr
	gpGenBuffers                 uintptr
	gpGenFramebuffersEXT         uintptr
	gpGenRenderbuffersEXT        uintptr
	gpGenTextures                uintptr
	gpGetError                   uintptr
	gpGetIntegerv                uintptr
	gpGetProgramInfoLog          uintptr
	gpGetProgramiv               uintptr
	gpGetShaderInfoLog           uintptr
	gpGetShaderiv                uintptr
	gpGetUniformLocation         uintptr
	gpIsFramebufferEXT           uintptr
	gpIsProgram                  uintptr
	gpIsRenderbufferEXT          uintptr
	gpIsTexture                  uintptr
	gpLinkProgram                uintptr
	gpPixelStorei                uintptr
	gpReadPixels                 uintptr
	gpRenderbufferStorageEXT     uintptr
	gpScissor                    uintptr
	gpShaderSource               uintptr
	gpStencilFunc                uintptr
	gpStencilOp                  uintptr
	gpTexImage2D                 uintptr
	gpTexParameteri              uintptr
	gpTexSubImage2D              uintptr
	gpUniform1fv                 uintptr
	gpUniform1i                  uintptr
	gpUniform1iv                 uintptr
	gpUniform2fv                 uintptr
	gpUniform2iv                 uintptr
	gpUniform3fv                 uintptr
	gpUniform3iv                 uintptr
	gpUniform4fv                 uintptr
	gpUniform4iv                 uintptr
	gpUniformMatrix2fv           uintptr
	gpUniformMatrix3fv           uintptr
	gpUniformMatrix4fv           uintptr
	gpUseProgram                 uintptr
	gpVertexAttribPointer        uintptr
	gpViewport                   uintptr

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
	purego.SyscallN(c.gpActiveTexture, uintptr(texture))
}

func (c *defaultContext) AttachShader(program uint32, shader uint32) {
	purego.SyscallN(c.gpAttachShader, uintptr(program), uintptr(shader))
}

func (c *defaultContext) BindAttribLocation(program uint32, index uint32, name string) {
	cname, free := cStr(name)
	defer free()
	purego.SyscallN(c.gpBindAttribLocation, uintptr(program), uintptr(index), uintptr(unsafe.Pointer(cname)))
}

func (c *defaultContext) BindBuffer(target uint32, buffer uint32) {
	purego.SyscallN(c.gpBindBuffer, uintptr(target), uintptr(buffer))
}

func (c *defaultContext) BindFramebuffer(target uint32, framebuffer uint32) {
	purego.SyscallN(c.gpBindFramebufferEXT, uintptr(target), uintptr(framebuffer))
}

func (c *defaultContext) BindRenderbuffer(target uint32, renderbuffer uint32) {
	purego.SyscallN(c.gpBindRenderbufferEXT, uintptr(target), uintptr(renderbuffer))
}

func (c *defaultContext) BindTexture(target uint32, texture uint32) {
	purego.SyscallN(c.gpBindTexture, uintptr(target), uintptr(texture))
}

func (c *defaultContext) BlendEquationSeparate(modeRGB uint32, modeAlpha uint32) {
	purego.SyscallN(c.gpBlendEquationSeparate, uintptr(modeRGB), uintptr(modeAlpha))
}

func (c *defaultContext) BlendFuncSeparate(srcRGB uint32, dstRGB uint32, srcAlpha uint32, dstAlpha uint32) {
	purego.SyscallN(c.gpBlendFuncSeparate, uintptr(srcRGB), uintptr(dstRGB), uintptr(srcAlpha), uintptr(dstAlpha))
}

func (c *defaultContext) BufferInit(target uint32, size int, usage uint32) {
	purego.SyscallN(c.gpBufferData, uintptr(target), uintptr(size), 0, uintptr(usage))
}

func (c *defaultContext) BufferSubData(target uint32, offset int, data []byte) {
	purego.SyscallN(c.gpBufferSubData, uintptr(target), uintptr(offset), uintptr(len(data)), uintptr(unsafe.Pointer(&data[0])))
	runtime.KeepAlive(data)
}

func (c *defaultContext) CheckFramebufferStatus(target uint32) uint32 {
	ret, _, _ := purego.SyscallN(c.gpCheckFramebufferStatusEXT, uintptr(target))
	return uint32(ret)
}

func (c *defaultContext) Clear(mask uint32) {
	purego.SyscallN(c.gpClear, uintptr(mask))
}

func (c *defaultContext) ColorMask(red bool, green bool, blue bool, alpha bool) {
	purego.SyscallN(c.gpColorMask, uintptr(boolToInt(red)), uintptr(boolToInt(green)), uintptr(boolToInt(blue)), uintptr(boolToInt(alpha)))
}

func (c *defaultContext) CompileShader(shader uint32) {
	purego.SyscallN(c.gpCompileShader, uintptr(shader))
}

func (c *defaultContext) CreateBuffer() uint32 {
	var buffer uint32
	purego.SyscallN(c.gpGenBuffers, 1, uintptr(unsafe.Pointer(&buffer)))
	return buffer
}

func (c *defaultContext) CreateFramebuffer() uint32 {
	var framebuffer uint32
	purego.SyscallN(c.gpGenFramebuffersEXT, 1, uintptr(unsafe.Pointer(&framebuffer)))
	return framebuffer
}

func (c *defaultContext) CreateProgram() uint32 {
	ret, _, _ := purego.SyscallN(c.gpCreateProgram)
	return uint32(ret)
}

func (c *defaultContext) CreateRenderbuffer() uint32 {
	var renderbuffer uint32
	purego.SyscallN(c.gpGenRenderbuffersEXT, 1, uintptr(unsafe.Pointer(&renderbuffer)))
	return renderbuffer
}

func (c *defaultContext) CreateShader(xtype uint32) uint32 {
	ret, _, _ := purego.SyscallN(c.gpCreateShader, uintptr(xtype))
	return uint32(ret)
}

func (c *defaultContext) CreateTexture() uint32 {
	var texture uint32
	purego.SyscallN(c.gpGenTextures, 1, uintptr(unsafe.Pointer(&texture)))
	return texture
}

func (c *defaultContext) DeleteBuffer(buffer uint32) {
	purego.SyscallN(c.gpDeleteBuffers, 1, uintptr(unsafe.Pointer(&buffer)))
}

func (c *defaultContext) DeleteFramebuffer(framebuffer uint32) {
	purego.SyscallN(c.gpDeleteFramebuffersEXT, 1, uintptr(unsafe.Pointer(&framebuffer)))
}

func (c *defaultContext) DeleteProgram(program uint32) {
	purego.SyscallN(c.gpDeleteProgram, uintptr(program))
}

func (c *defaultContext) DeleteRenderbuffer(renderbuffer uint32) {
	purego.SyscallN(c.gpDeleteRenderbuffersEXT, 1, uintptr(unsafe.Pointer(&renderbuffer)))
}

func (c *defaultContext) DeleteShader(shader uint32) {
	purego.SyscallN(c.gpDeleteShader, uintptr(shader))
}

func (c *defaultContext) DeleteTexture(texture uint32) {
	purego.SyscallN(c.gpDeleteTextures, 1, uintptr(unsafe.Pointer(&texture)))
}

func (c *defaultContext) Disable(cap uint32) {
	purego.SyscallN(c.gpDisable, uintptr(cap))
}

func (c *defaultContext) DisableVertexAttribArray(index uint32) {
	purego.SyscallN(c.gpDisableVertexAttribArray, uintptr(index))
}

func (c *defaultContext) DrawElements(mode uint32, count int32, xtype uint32, offset int) {
	purego.SyscallN(c.gpDrawElements, uintptr(mode), uintptr(count), uintptr(xtype), uintptr(offset))
}

func (c *defaultContext) Enable(cap uint32) {
	purego.SyscallN(c.gpEnable, uintptr(cap))
}

func (c *defaultContext) EnableVertexAttribArray(index uint32) {
	purego.SyscallN(c.gpEnableVertexAttribArray, uintptr(index))
}

func (c *defaultContext) Flush() {
	purego.SyscallN(c.gpFlush)
}

func (c *defaultContext) FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	purego.SyscallN(c.gpFramebufferRenderbufferEXT, uintptr(target), uintptr(attachment), uintptr(renderbuffertarget), uintptr(renderbuffer))
}

func (c *defaultContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	purego.SyscallN(c.gpFramebufferTexture2DEXT, uintptr(target), uintptr(attachment), uintptr(textarget), uintptr(texture), uintptr(level))
}

func (c *defaultContext) GetError() uint32 {
	ret, _, _ := purego.SyscallN(c.gpGetError)
	return uint32(ret)
}

func (c *defaultContext) GetInteger(pname uint32) int {
	var dst int32
	purego.SyscallN(c.gpGetIntegerv, uintptr(pname), uintptr(unsafe.Pointer(&dst)))
	return int(dst)
}

func (c *defaultContext) GetProgramInfoLog(program uint32) string {
	bufSize := c.GetProgrami(program, INFO_LOG_LENGTH)
	infoLog := make([]byte, bufSize)
	purego.SyscallN(c.gpGetProgramInfoLog, uintptr(program), uintptr(bufSize), 0, uintptr(unsafe.Pointer(&infoLog[0])))
	return string(infoLog)
}

func (c *defaultContext) GetProgrami(program uint32, pname uint32) int {
	var dst int32
	purego.SyscallN(c.gpGetProgramiv, uintptr(program), uintptr(pname), uintptr(unsafe.Pointer(&dst)))
	return int(dst)
}

func (c *defaultContext) GetShaderInfoLog(shader uint32) string {
	bufSize := c.GetShaderi(shader, INFO_LOG_LENGTH)
	infoLog := make([]byte, bufSize)
	purego.SyscallN(c.gpGetShaderInfoLog, uintptr(shader), uintptr(bufSize), 0, uintptr(unsafe.Pointer(&infoLog[0])))
	return string(infoLog)
}

func (c *defaultContext) GetShaderi(shader uint32, pname uint32) int {
	var dst int32
	purego.SyscallN(c.gpGetShaderiv, uintptr(shader), uintptr(pname), uintptr(unsafe.Pointer(&dst)))
	return int(dst)
}

func (c *defaultContext) GetUniformLocation(program uint32, name string) int32 {
	cname, free := cStr(name)
	defer free()
	ret, _, _ := purego.SyscallN(c.gpGetUniformLocation, uintptr(program), uintptr(unsafe.Pointer(cname)))
	return int32(ret)
}

func (c *defaultContext) IsFramebuffer(framebuffer uint32) bool {
	ret, _, _ := purego.SyscallN(c.gpIsFramebufferEXT, uintptr(framebuffer))
	return byte(ret) != 0
}

func (c *defaultContext) IsProgram(program uint32) bool {
	ret, _, _ := purego.SyscallN(c.gpIsProgram, uintptr(program))
	return byte(ret) != 0
}

func (c *defaultContext) IsRenderbuffer(renderbuffer uint32) bool {
	ret, _, _ := purego.SyscallN(c.gpIsRenderbufferEXT, uintptr(renderbuffer))
	return byte(ret) != 0
}

func (c *defaultContext) IsTexture(texture uint32) bool {
	ret, _, _ := purego.SyscallN(c.gpIsTexture, uintptr(texture))
	return byte(ret) != 0
}

func (c *defaultContext) LinkProgram(program uint32) {
	purego.SyscallN(c.gpLinkProgram, uintptr(program))
}

func (c *defaultContext) PixelStorei(pname uint32, param int32) {
	purego.SyscallN(c.gpPixelStorei, uintptr(pname), uintptr(param))
}

func (c *defaultContext) ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32) {
	purego.SyscallN(c.gpReadPixels, uintptr(x), uintptr(y), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(unsafe.Pointer(&dst[0])))
}

func (c *defaultContext) RenderbufferStorage(target uint32, internalformat uint32, width int32, height int32) {
	purego.SyscallN(c.gpRenderbufferStorageEXT, uintptr(target), uintptr(internalformat), uintptr(width), uintptr(height))
}

func (c *defaultContext) Scissor(x int32, y int32, width int32, height int32) {
	purego.SyscallN(c.gpScissor, uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

func (c *defaultContext) ShaderSource(shader uint32, xstring string) {
	cstring, free := cStr(xstring)
	defer free()
	purego.SyscallN(c.gpShaderSource, uintptr(shader), 1, uintptr(unsafe.Pointer(&cstring)), 0)
}

func (c *defaultContext) StencilFunc(xfunc uint32, ref int32, mask uint32) {
	purego.SyscallN(c.gpStencilFunc, uintptr(xfunc), uintptr(ref), uintptr(mask))
}

func (c *defaultContext) StencilOp(fail uint32, zfail uint32, zpass uint32) {
	purego.SyscallN(c.gpStencilOp, uintptr(fail), uintptr(zfail), uintptr(zpass))
}

func (c *defaultContext) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	var ptr *byte
	if len(pixels) > 0 {
		ptr = &pixels[0]
	}
	purego.SyscallN(c.gpTexImage2D, uintptr(target), uintptr(level), uintptr(internalformat), uintptr(width), uintptr(height), 0, uintptr(format), uintptr(xtype), uintptr(unsafe.Pointer(ptr)))
	runtime.KeepAlive(pixels)
}

func (c *defaultContext) TexParameteri(target uint32, pname uint32, param int32) {
	purego.SyscallN(c.gpTexParameteri, uintptr(target), uintptr(pname), uintptr(param))
}

func (c *defaultContext) TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	purego.SyscallN(c.gpTexSubImage2D, uintptr(target), uintptr(level), uintptr(xoffset), uintptr(yoffset), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(unsafe.Pointer(&pixels[0])))
	runtime.KeepAlive(pixels)
}

func (c *defaultContext) Uniform1fv(location int32, value []float32) {
	purego.SyscallN(c.gpUniform1fv, uintptr(location), uintptr(len(value)), uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform1i(location int32, v0 int32) {
	purego.SyscallN(c.gpUniform1i, uintptr(location), uintptr(v0))
}

func (c *defaultContext) Uniform1iv(location int32, value []int32) {
	purego.SyscallN(c.gpUniform1iv, uintptr(location), uintptr(len(value)), uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform2fv(location int32, value []float32) {
	purego.SyscallN(c.gpUniform2fv, uintptr(location), uintptr(len(value)/2), uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform2iv(location int32, value []int32) {
	purego.SyscallN(c.gpUniform2iv, uintptr(location), uintptr(len(value)/2), uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform3fv(location int32, value []float32) {
	purego.SyscallN(c.gpUniform3fv, uintptr(location), uintptr(len(value)/3), uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform3iv(location int32, value []int32) {
	purego.SyscallN(c.gpUniform3iv, uintptr(location), uintptr(len(value)/3), uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform4fv(location int32, value []float32) {
	purego.SyscallN(c.gpUniform4fv, uintptr(location), uintptr(len(value)/4), uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) Uniform4iv(location int32, value []int32) {
	purego.SyscallN(c.gpUniform4iv, uintptr(location), uintptr(len(value)/4), uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UniformMatrix2fv(location int32, value []float32) {
	purego.SyscallN(c.gpUniformMatrix2fv, uintptr(location), uintptr(len(value)/4), 0, uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UniformMatrix3fv(location int32, value []float32) {
	purego.SyscallN(c.gpUniformMatrix3fv, uintptr(location), uintptr(len(value)/9), 0, uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UniformMatrix4fv(location int32, value []float32) {
	purego.SyscallN(c.gpUniformMatrix4fv, uintptr(location), uintptr(len(value)/16), 0, uintptr(unsafe.Pointer(&value[0])))
	runtime.KeepAlive(value)
}

func (c *defaultContext) UseProgram(program uint32) {
	purego.SyscallN(c.gpUseProgram, uintptr(program))
}

func (c *defaultContext) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int) {
	purego.SyscallN(c.gpVertexAttribPointer, uintptr(index), uintptr(size), uintptr(xtype), uintptr(boolToInt(normalized)), uintptr(stride), uintptr(offset))
}

func (c *defaultContext) Viewport(x int32, y int32, width int32, height int32) {
	purego.SyscallN(c.gpViewport, uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

func (c *defaultContext) LoadFunctions() error {
	c.gpActiveTexture = c.getProcAddress("glActiveTexture")
	if c.gpActiveTexture == 0 {
		return errors.New("gl: glActiveTexture is missing")
	}
	c.gpAttachShader = c.getProcAddress("glAttachShader")
	if c.gpAttachShader == 0 {
		return errors.New("gl: glAttachShader is missing")
	}
	c.gpBindAttribLocation = c.getProcAddress("glBindAttribLocation")
	if c.gpBindAttribLocation == 0 {
		return errors.New("gl: glBindAttribLocation is missing")
	}
	c.gpBindBuffer = c.getProcAddress("glBindBuffer")
	if c.gpBindBuffer == 0 {
		return errors.New("gl: glBindBuffer is missing")
	}
	c.gpBindFramebufferEXT = c.getProcAddress("glBindFramebufferEXT")
	c.gpBindRenderbufferEXT = c.getProcAddress("glBindRenderbufferEXT")
	c.gpBindTexture = c.getProcAddress("glBindTexture")
	if c.gpBindTexture == 0 {
		return errors.New("gl: glBindTexture is missing")
	}
	c.gpBlendEquationSeparate = c.getProcAddress("glBlendEquationSeparate")
	if c.gpBlendEquationSeparate == 0 {
		return errors.New("gl: glBlendEquationSeparate is missing")
	}
	c.gpBlendFuncSeparate = c.getProcAddress("glBlendFuncSeparate")
	if c.gpBlendFuncSeparate == 0 {
		return errors.New("gl: glBlendFuncSeparate is missing")
	}
	c.gpBufferData = c.getProcAddress("glBufferData")
	if c.gpBufferData == 0 {
		return errors.New("gl: glBufferData is missing")
	}
	c.gpBufferSubData = c.getProcAddress("glBufferSubData")
	if c.gpBufferSubData == 0 {
		return errors.New("gl: glBufferSubData is missing")
	}
	c.gpCheckFramebufferStatusEXT = c.getProcAddress("glCheckFramebufferStatusEXT")
	c.gpClear = c.getProcAddress("glClear")
	if c.gpClear == 0 {
		return errors.New("gl: glClear is missing")
	}
	c.gpColorMask = c.getProcAddress("glColorMask")
	if c.gpColorMask == 0 {
		return errors.New("gl: glColorMask is missing")
	}
	c.gpCompileShader = c.getProcAddress("glCompileShader")
	if c.gpCompileShader == 0 {
		return errors.New("gl: glCompileShader is missing")
	}
	c.gpCreateProgram = c.getProcAddress("glCreateProgram")
	if c.gpCreateProgram == 0 {
		return errors.New("gl: glCreateProgram is missing")
	}
	c.gpCreateShader = c.getProcAddress("glCreateShader")
	if c.gpCreateShader == 0 {
		return errors.New("gl: glCreateShader is missing")
	}
	c.gpDeleteBuffers = c.getProcAddress("glDeleteBuffers")
	if c.gpDeleteBuffers == 0 {
		return errors.New("gl: glDeleteBuffers is missing")
	}
	c.gpDeleteFramebuffersEXT = c.getProcAddress("glDeleteFramebuffersEXT")
	c.gpDeleteProgram = c.getProcAddress("glDeleteProgram")
	if c.gpDeleteProgram == 0 {
		return errors.New("gl: glDeleteProgram is missing")
	}
	c.gpDeleteRenderbuffersEXT = c.getProcAddress("glDeleteRenderbuffersEXT")
	c.gpDeleteShader = c.getProcAddress("glDeleteShader")
	if c.gpDeleteShader == 0 {
		return errors.New("gl: glDeleteShader is missing")
	}
	c.gpDeleteTextures = c.getProcAddress("glDeleteTextures")
	if c.gpDeleteTextures == 0 {
		return errors.New("gl: glDeleteTextures is missing")
	}
	c.gpDisable = c.getProcAddress("glDisable")
	if c.gpDisable == 0 {
		return errors.New("gl: glDisable is missing")
	}
	c.gpDisableVertexAttribArray = c.getProcAddress("glDisableVertexAttribArray")
	if c.gpDisableVertexAttribArray == 0 {
		return errors.New("gl: glDisableVertexAttribArray is missing")
	}
	c.gpDrawElements = c.getProcAddress("glDrawElements")
	if c.gpDrawElements == 0 {
		return errors.New("gl: glDrawElements is missing")
	}
	c.gpEnable = c.getProcAddress("glEnable")
	if c.gpEnable == 0 {
		return errors.New("gl: glEnable is missing")
	}
	c.gpEnableVertexAttribArray = c.getProcAddress("glEnableVertexAttribArray")
	if c.gpEnableVertexAttribArray == 0 {
		return errors.New("gl: glEnableVertexAttribArray is missing")
	}
	c.gpFlush = c.getProcAddress("glFlush")
	if c.gpFlush == 0 {
		return errors.New("gl: glFlush is missing")
	}
	c.gpFramebufferRenderbufferEXT = c.getProcAddress("glFramebufferRenderbufferEXT")
	c.gpFramebufferTexture2DEXT = c.getProcAddress("glFramebufferTexture2DEXT")
	c.gpGenBuffers = c.getProcAddress("glGenBuffers")
	if c.gpGenBuffers == 0 {
		return errors.New("gl: glGenBuffers is missing")
	}
	c.gpGenFramebuffersEXT = c.getProcAddress("glGenFramebuffersEXT")
	c.gpGenRenderbuffersEXT = c.getProcAddress("glGenRenderbuffersEXT")
	c.gpGenTextures = c.getProcAddress("glGenTextures")
	if c.gpGenTextures == 0 {
		return errors.New("gl: glGenTextures is missing")
	}
	c.gpGetError = c.getProcAddress("glGetError")
	if c.gpGetError == 0 {
		return errors.New("gl: glGetError is missing")
	}
	c.gpGetIntegerv = c.getProcAddress("glGetIntegerv")
	if c.gpGetIntegerv == 0 {
		return errors.New("gl: glGetIntegerv is missing")
	}
	c.gpGetProgramInfoLog = c.getProcAddress("glGetProgramInfoLog")
	if c.gpGetProgramInfoLog == 0 {
		return errors.New("gl: glGetProgramInfoLog is missing")
	}
	c.gpGetProgramiv = c.getProcAddress("glGetProgramiv")
	if c.gpGetProgramiv == 0 {
		return errors.New("gl: glGetProgramiv is missing")
	}
	c.gpGetShaderInfoLog = c.getProcAddress("glGetShaderInfoLog")
	if c.gpGetShaderInfoLog == 0 {
		return errors.New("gl: glGetShaderInfoLog is missing")
	}
	c.gpGetShaderiv = c.getProcAddress("glGetShaderiv")
	if c.gpGetShaderiv == 0 {
		return errors.New("gl: glGetShaderiv is missing")
	}
	c.gpGetUniformLocation = c.getProcAddress("glGetUniformLocation")
	if c.gpGetUniformLocation == 0 {
		return errors.New("gl: glGetUniformLocation is missing")
	}
	c.gpIsFramebufferEXT = c.getProcAddress("glIsFramebufferEXT")
	c.gpIsProgram = c.getProcAddress("glIsProgram")
	if c.gpIsProgram == 0 {
		return errors.New("gl: glIsProgram is missing")
	}
	c.gpIsRenderbufferEXT = c.getProcAddress("glIsRenderbufferEXT")
	c.gpIsTexture = c.getProcAddress("glIsTexture")
	if c.gpIsTexture == 0 {
		return errors.New("gl: glIsTexture is missing")
	}
	c.gpLinkProgram = c.getProcAddress("glLinkProgram")
	if c.gpLinkProgram == 0 {
		return errors.New("gl: glLinkProgram is missing")
	}
	c.gpPixelStorei = c.getProcAddress("glPixelStorei")
	if c.gpPixelStorei == 0 {
		return errors.New("gl: glPixelStorei is missing")
	}
	c.gpReadPixels = c.getProcAddress("glReadPixels")
	if c.gpReadPixels == 0 {
		return errors.New("gl: glReadPixels is missing")
	}
	c.gpRenderbufferStorageEXT = c.getProcAddress("glRenderbufferStorageEXT")
	c.gpScissor = c.getProcAddress("glScissor")
	if c.gpScissor == 0 {
		return errors.New("gl: glScissor is missing")
	}
	c.gpShaderSource = c.getProcAddress("glShaderSource")
	if c.gpShaderSource == 0 {
		return errors.New("gl: glShaderSource is missing")
	}
	c.gpStencilFunc = c.getProcAddress("glStencilFunc")
	if c.gpStencilFunc == 0 {
		return errors.New("gl: glStencilFunc is missing")
	}
	c.gpStencilOp = c.getProcAddress("glStencilOp")
	if c.gpStencilOp == 0 {
		return errors.New("gl: glStencilOp is missing")
	}
	c.gpTexImage2D = c.getProcAddress("glTexImage2D")
	if c.gpTexImage2D == 0 {
		return errors.New("gl: glTexImage2D is missing")
	}
	c.gpTexParameteri = c.getProcAddress("glTexParameteri")
	if c.gpTexParameteri == 0 {
		return errors.New("gl: glTexParameteri is missing")
	}
	c.gpTexSubImage2D = c.getProcAddress("glTexSubImage2D")
	if c.gpTexSubImage2D == 0 {
		return errors.New("gl: glTexSubImage2D is missing")
	}
	c.gpUniform1fv = c.getProcAddress("glUniform1fv")
	if c.gpUniform1fv == 0 {
		return errors.New("gl: glUniform1fv is missing")
	}
	c.gpUniform1i = c.getProcAddress("glUniform1i")
	if c.gpUniform1i == 0 {
		return errors.New("gl: glUniform1i is missing")
	}
	c.gpUniform1iv = c.getProcAddress("glUniform1iv")
	if c.gpUniform1iv == 0 {
		return errors.New("gl: glUniform1iv is missing")
	}
	c.gpUniform2fv = c.getProcAddress("glUniform2fv")
	if c.gpUniform2fv == 0 {
		return errors.New("gl: glUniform2fv is missing")
	}
	c.gpUniform2iv = c.getProcAddress("glUniform2iv")
	if c.gpUniform2iv == 0 {
		return errors.New("gl: glUniform2iv is missing")
	}
	c.gpUniform3fv = c.getProcAddress("glUniform3fv")
	if c.gpUniform3fv == 0 {
		return errors.New("gl: glUniform3fv is missing")
	}
	c.gpUniform3iv = c.getProcAddress("glUniform3iv")
	if c.gpUniform3iv == 0 {
		return errors.New("gl: glUniform3iv is missing")
	}
	c.gpUniform4fv = c.getProcAddress("glUniform4fv")
	if c.gpUniform4fv == 0 {
		return errors.New("gl: glUniform4fv is missing")
	}
	c.gpUniform4iv = c.getProcAddress("glUniform4iv")
	if c.gpUniform4iv == 0 {
		return errors.New("gl: glUniform4iv is missing")
	}
	c.gpUniformMatrix2fv = c.getProcAddress("glUniformMatrix2fv")
	if c.gpUniformMatrix2fv == 0 {
		return errors.New("gl: glUniformMatrix2fv is missing")
	}
	c.gpUniformMatrix3fv = c.getProcAddress("glUniformMatrix3fv")
	if c.gpUniformMatrix3fv == 0 {
		return errors.New("gl: glUniformMatrix3fv is missing")
	}
	c.gpUniformMatrix4fv = c.getProcAddress("glUniformMatrix4fv")
	if c.gpUniformMatrix4fv == 0 {
		return errors.New("gl: glUniformMatrix4fv is missing")
	}
	c.gpUseProgram = c.getProcAddress("glUseProgram")
	if c.gpUseProgram == 0 {
		return errors.New("gl: glUseProgram is missing")
	}
	c.gpVertexAttribPointer = c.getProcAddress("glVertexAttribPointer")
	if c.gpVertexAttribPointer == 0 {
		return errors.New("gl: glVertexAttribPointer is missing")
	}
	c.gpViewport = c.getProcAddress("glViewport")
	if c.gpViewport == 0 {
		return errors.New("gl: glViewport is missing")
	}
	return nil
}

// cStr takes a Go string (with or without null-termination)
// and returns the C counterpart.
//
// The returned free function must be called once you are done using the string
// in order to free the memory.
func cStr(str string) (cstr *byte, free func()) {
	bs := []byte(str)
	if len(bs) == 0 || bs[len(bs)-1] != 0 {
		bs = append(bs, 0)
	}
	return &bs[0], func() {
		runtime.KeepAlive(bs)
		bs = nil
	}
}
