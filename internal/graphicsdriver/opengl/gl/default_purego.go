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

//go:build (darwin || freebsd || linux || netbsd || openbsd || windows) && !nintendosdk && !playstation5

package gl

import (
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

type defaultContext struct {
	gpActiveTexture            uintptr
	gpAttachShader             uintptr
	gpBindAttribLocation       uintptr
	gpBindBuffer               uintptr
	gpBindFramebuffer          uintptr
	gpBindRenderbuffer         uintptr
	gpBindTexture              uintptr
	gpBindVertexArray          uintptr
	gpBlendEquationSeparate    uintptr
	gpBlendFuncSeparate        uintptr
	gpBufferData               uintptr
	gpBufferSubData            uintptr
	gpCheckFramebufferStatus   uintptr
	gpClear                    uintptr
	gpColorMask                uintptr
	gpCompileShader            uintptr
	gpCreateProgram            uintptr
	gpCreateShader             uintptr
	gpDeleteBuffers            uintptr
	gpDeleteFramebuffers       uintptr
	gpDeleteProgram            uintptr
	gpDeleteRenderbuffers      uintptr
	gpDeleteShader             uintptr
	gpDeleteTextures           uintptr
	gpDeleteVertexArrays       uintptr
	gpDisable                  uintptr
	gpDisableVertexAttribArray uintptr
	gpDrawElements             uintptr
	gpEnable                   uintptr
	gpEnableVertexAttribArray  uintptr
	gpFlush                    uintptr
	gpFramebufferRenderbuffer  uintptr
	gpFramebufferTexture2D     uintptr
	gpGenBuffers               uintptr
	gpGenFramebuffers          uintptr
	gpGenRenderbuffers         uintptr
	gpGenTextures              uintptr
	gpGenVertexArrays          uintptr
	gpGetError                 uintptr
	gpGetIntegerv              uintptr
	gpGetProgramInfoLog        uintptr
	gpGetProgramiv             uintptr
	gpGetShaderInfoLog         uintptr
	gpGetShaderiv              uintptr
	gpGetUniformLocation       uintptr
	gpIsProgram                uintptr
	gpLinkProgram              uintptr
	gpPixelStorei              uintptr
	gpReadPixels               uintptr
	gpRenderbufferStorage      uintptr
	gpScissor                  uintptr
	gpShaderSource             uintptr
	gpStencilFunc              uintptr
	gpStencilOpSeparate        uintptr
	gpTexImage2D               uintptr
	gpTexParameteri            uintptr
	gpTexSubImage2D            uintptr
	gpUniform1fv               uintptr
	gpUniform1i                uintptr
	gpUniform1iv               uintptr
	gpUniform2fv               uintptr
	gpUniform2iv               uintptr
	gpUniform3fv               uintptr
	gpUniform3iv               uintptr
	gpUniform4fv               uintptr
	gpUniform4iv               uintptr
	gpUniformMatrix2fv         uintptr
	gpUniformMatrix3fv         uintptr
	gpUniformMatrix4fv         uintptr
	gpUseProgram               uintptr
	gpVertexAttribPointer      uintptr
	gpViewport                 uintptr

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
	purego.SyscallN(c.gpBindFramebuffer, uintptr(target), uintptr(framebuffer))
}

func (c *defaultContext) BindRenderbuffer(target uint32, renderbuffer uint32) {
	purego.SyscallN(c.gpBindRenderbuffer, uintptr(target), uintptr(renderbuffer))
}

func (c *defaultContext) BindTexture(target uint32, texture uint32) {
	purego.SyscallN(c.gpBindTexture, uintptr(target), uintptr(texture))
}

func (c *defaultContext) BindVertexArray(array uint32) {
	purego.SyscallN(c.gpBindVertexArray, uintptr(array))
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
	ret, _, _ := purego.SyscallN(c.gpCheckFramebufferStatus, uintptr(target))
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
	purego.SyscallN(c.gpGenFramebuffers, 1, uintptr(unsafe.Pointer(&framebuffer)))
	return framebuffer
}

func (c *defaultContext) CreateProgram() uint32 {
	ret, _, _ := purego.SyscallN(c.gpCreateProgram)
	return uint32(ret)
}

func (c *defaultContext) CreateRenderbuffer() uint32 {
	var renderbuffer uint32
	purego.SyscallN(c.gpGenRenderbuffers, 1, uintptr(unsafe.Pointer(&renderbuffer)))
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

func (c *defaultContext) CreateVertexArray() uint32 {
	var array uint32
	purego.SyscallN(c.gpGenVertexArrays, 1, uintptr(unsafe.Pointer(&array)))
	return array
}

func (c *defaultContext) DeleteBuffer(buffer uint32) {
	purego.SyscallN(c.gpDeleteBuffers, 1, uintptr(unsafe.Pointer(&buffer)))
}

func (c *defaultContext) DeleteFramebuffer(framebuffer uint32) {
	purego.SyscallN(c.gpDeleteFramebuffers, 1, uintptr(unsafe.Pointer(&framebuffer)))
}

func (c *defaultContext) DeleteProgram(program uint32) {
	purego.SyscallN(c.gpDeleteProgram, uintptr(program))
}

func (c *defaultContext) DeleteRenderbuffer(renderbuffer uint32) {
	purego.SyscallN(c.gpDeleteRenderbuffers, 1, uintptr(unsafe.Pointer(&renderbuffer)))
}

func (c *defaultContext) DeleteShader(shader uint32) {
	purego.SyscallN(c.gpDeleteShader, uintptr(shader))
}

func (c *defaultContext) DeleteTexture(texture uint32) {
	purego.SyscallN(c.gpDeleteTextures, 1, uintptr(unsafe.Pointer(&texture)))
}

func (c *defaultContext) DeleteVertexArray(array uint32) {
	purego.SyscallN(c.gpDeleteVertexArrays, 1, uintptr(unsafe.Pointer(&array)))
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
	purego.SyscallN(c.gpFramebufferRenderbuffer, uintptr(target), uintptr(attachment), uintptr(renderbuffertarget), uintptr(renderbuffer))
}

func (c *defaultContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	purego.SyscallN(c.gpFramebufferTexture2D, uintptr(target), uintptr(attachment), uintptr(textarget), uintptr(texture), uintptr(level))
}

func (c *defaultContext) GetError() uint32 {
	ret, _, _ := purego.SyscallN(c.gpGetError)
	return uint32(ret)
}

func (c *defaultContext) GetExtension(name string) any {
	return nil
}

func (c *defaultContext) GetInteger(pname uint32) int {
	var dst int32
	purego.SyscallN(c.gpGetIntegerv, uintptr(pname), uintptr(unsafe.Pointer(&dst)))
	return int(dst)
}

func (c *defaultContext) GetProgramInfoLog(program uint32) string {
	bufSize := c.GetProgrami(program, INFO_LOG_LENGTH)
	if bufSize == 0 {
		return ""
	}
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
	if bufSize == 0 {
		return ""
	}
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

func (c *defaultContext) IsProgram(program uint32) bool {
	ret, _, _ := purego.SyscallN(c.gpIsProgram, uintptr(program))
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
	purego.SyscallN(c.gpRenderbufferStorage, uintptr(target), uintptr(internalformat), uintptr(width), uintptr(height))
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

func (c *defaultContext) StencilOpSeparate(face uint32, fail uint32, zfail uint32, zpass uint32) {
	purego.SyscallN(c.gpStencilOpSeparate, uintptr(face), uintptr(fail), uintptr(zfail), uintptr(zpass))
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
	g := procAddressGetter{ctx: c}

	c.gpActiveTexture = g.get("glActiveTexture")
	c.gpAttachShader = g.get("glAttachShader")
	c.gpBindAttribLocation = g.get("glBindAttribLocation")
	c.gpBindBuffer = g.get("glBindBuffer")
	c.gpBindFramebuffer = g.get("glBindFramebuffer")
	c.gpBindRenderbuffer = g.get("glBindRenderbuffer")
	c.gpBindTexture = g.get("glBindTexture")
	c.gpBindVertexArray = g.get("glBindVertexArray")
	c.gpBlendEquationSeparate = g.get("glBlendEquationSeparate")
	c.gpBlendFuncSeparate = g.get("glBlendFuncSeparate")
	c.gpBufferData = g.get("glBufferData")
	c.gpBufferSubData = g.get("glBufferSubData")
	c.gpCheckFramebufferStatus = g.get("glCheckFramebufferStatus")
	c.gpClear = g.get("glClear")
	c.gpColorMask = g.get("glColorMask")
	c.gpCompileShader = g.get("glCompileShader")
	c.gpCreateProgram = g.get("glCreateProgram")
	c.gpCreateShader = g.get("glCreateShader")
	c.gpDeleteBuffers = g.get("glDeleteBuffers")
	c.gpDeleteFramebuffers = g.get("glDeleteFramebuffers")
	c.gpDeleteProgram = g.get("glDeleteProgram")
	c.gpDeleteRenderbuffers = g.get("glDeleteRenderbuffers")
	c.gpDeleteShader = g.get("glDeleteShader")
	c.gpDeleteTextures = g.get("glDeleteTextures")
	c.gpDeleteVertexArrays = g.get("glDeleteVertexArrays")
	c.gpDisable = g.get("glDisable")
	c.gpDisableVertexAttribArray = g.get("glDisableVertexAttribArray")
	c.gpDrawElements = g.get("glDrawElements")
	c.gpEnable = g.get("glEnable")
	c.gpEnableVertexAttribArray = g.get("glEnableVertexAttribArray")
	c.gpFlush = g.get("glFlush")
	c.gpFramebufferRenderbuffer = g.get("glFramebufferRenderbuffer")
	c.gpFramebufferTexture2D = g.get("glFramebufferTexture2D")
	c.gpGenBuffers = g.get("glGenBuffers")
	c.gpGenFramebuffers = g.get("glGenFramebuffers")
	c.gpGenRenderbuffers = g.get("glGenRenderbuffers")
	c.gpGenTextures = g.get("glGenTextures")
	c.gpGenVertexArrays = g.get("glGenVertexArrays")
	c.gpGetError = g.get("glGetError")
	c.gpGetIntegerv = g.get("glGetIntegerv")
	c.gpGetProgramInfoLog = g.get("glGetProgramInfoLog")
	c.gpGetProgramiv = g.get("glGetProgramiv")
	c.gpGetShaderInfoLog = g.get("glGetShaderInfoLog")
	c.gpGetShaderiv = g.get("glGetShaderiv")
	c.gpGetUniformLocation = g.get("glGetUniformLocation")
	c.gpIsProgram = g.get("glIsProgram")
	c.gpLinkProgram = g.get("glLinkProgram")
	c.gpPixelStorei = g.get("glPixelStorei")
	c.gpReadPixels = g.get("glReadPixels")
	c.gpRenderbufferStorage = g.get("glRenderbufferStorage")
	c.gpScissor = g.get("glScissor")
	c.gpShaderSource = g.get("glShaderSource")
	c.gpStencilFunc = g.get("glStencilFunc")
	c.gpStencilOpSeparate = g.get("glStencilOpSeparate")
	c.gpTexImage2D = g.get("glTexImage2D")
	c.gpTexParameteri = g.get("glTexParameteri")
	c.gpTexSubImage2D = g.get("glTexSubImage2D")
	c.gpUniform1fv = g.get("glUniform1fv")
	c.gpUniform1i = g.get("glUniform1i")
	c.gpUniform1iv = g.get("glUniform1iv")
	c.gpUniform2fv = g.get("glUniform2fv")
	c.gpUniform2iv = g.get("glUniform2iv")
	c.gpUniform3fv = g.get("glUniform3fv")
	c.gpUniform3iv = g.get("glUniform3iv")
	c.gpUniform4fv = g.get("glUniform4fv")
	c.gpUniform4iv = g.get("glUniform4iv")
	c.gpUniformMatrix2fv = g.get("glUniformMatrix2fv")
	c.gpUniformMatrix3fv = g.get("glUniformMatrix3fv")
	c.gpUniformMatrix4fv = g.get("glUniformMatrix4fv")
	c.gpUseProgram = g.get("glUseProgram")
	c.gpVertexAttribPointer = g.get("glVertexAttribPointer")
	c.gpViewport = g.get("glViewport")

	return g.error()
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
