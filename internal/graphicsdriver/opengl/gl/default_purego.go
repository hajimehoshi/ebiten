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

//go:build (darwin || freebsd || linux || netbsd || windows) && !nintendosdk && !playstation5

package gl

import (
	"math/bits"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

type defaultContext struct {
	gpActiveTexture           uintptr
	gpAttachShader            uintptr
	gpBindAttribLocation      uintptr
	gpBindBuffer              uintptr
	gpBindFramebuffer         uintptr
	gpBindTexture             uintptr
	gpBindVertexArray         uintptr
	gpBlendEquationSeparate   uintptr
	gpBlendFuncSeparate       uintptr
	gpBufferData              uintptr
	gpBufferSubData           uintptr
	gpCheckFramebufferStatus  uintptr
	gpClientWaitSync          uintptr
	gpCompileShader           uintptr
	gpCreateProgram           uintptr
	gpCreateShader            uintptr
	gpDeleteBuffers           uintptr
	gpDeleteFramebuffers      uintptr
	gpDeleteProgram           uintptr
	gpDeleteShader            uintptr
	gpDeleteSync              uintptr
	gpDeleteTextures          uintptr
	gpDeleteVertexArrays      uintptr
	gpDrawElements            uintptr
	gpEnable                  uintptr
	gpEnableVertexAttribArray uintptr
	gpFenceSync               uintptr
	gpFinish                  uintptr
	gpFlush                   uintptr
	gpFramebufferTexture2D    uintptr
	gpGenBuffers              uintptr
	gpGenFramebuffers         uintptr
	gpGenTextures             uintptr
	gpGenVertexArrays         uintptr
	gpGetError                uintptr
	gpGetIntegerv             uintptr
	gpGetProgramInfoLog       uintptr
	gpGetProgramiv            uintptr
	gpGetShaderInfoLog        uintptr
	gpGetShaderiv             uintptr
	gpGetUniformLocation      uintptr
	gpIsProgram               uintptr
	gpLinkProgram             uintptr
	// gpMapBufferRange is registered by purego.RegisterFunc so that it returns the mapped memory as an
	// unsafe.Pointer. Converting the uintptr result of call to unsafe.Pointer is reported by go vet.
	gpMapBufferRange      func(target uint32, offset, length int, access uint32) unsafe.Pointer
	gpPixelStorei         uintptr
	gpReadPixels          uintptr
	gpScissor             uintptr
	gpShaderSource        uintptr
	gpTexImage2D          uintptr
	gpTexParameteri       uintptr
	gpTexSubImage2D       uintptr
	gpUniform1fv          uintptr
	gpUniform1i           uintptr
	gpUniform1iv          uintptr
	gpUniform2fv          uintptr
	gpUniform2iv          uintptr
	gpUniform3fv          uintptr
	gpUniform3iv          uintptr
	gpUniform4fv          uintptr
	gpUniform4iv          uintptr
	gpUniformMatrix2fv    uintptr
	gpUniformMatrix3fv    uintptr
	gpUniformMatrix4fv    uintptr
	gpUnmapBuffer         uintptr
	gpUseProgram          uintptr
	gpVertexAttribPointer uintptr
	gpViewport            uintptr

	// args holds the arguments of call. args, pinner, nameBuf, intBuf, and strBuf are shared by
	// every call, so defaultContext must not be used concurrently.
	args [15]uintptr

	// pinner pins the Go memory passed to call during the call.
	pinner runtime.Pinner

	// nameBuf holds the object name of a glGen* or glDelete* call.
	nameBuf uint32

	// intBuf holds the result of a glGet*iv call.
	intBuf int32

	// strBuf holds a NUL-terminated copy of the string argument of a call. strAddr holds the
	// address of strBuf for ShaderSource, which takes an array of string addresses.
	strBuf  []byte
	strAddr uintptr

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

// call calls fn with args. args must not hold Go pointers unless their objects are pinned.
func (c *defaultContext) call(fn uintptr, args ...uintptr) (r1, r2, err uintptr) {
	// An existing slice keeps purego.SyscallN from allocating its variadic arguments.
	n := len(args)
	copy(c.args[:n], args)
	return purego.SyscallN(fn, c.args[:n]...)
}

// genName calls fn, a glGen* function, for one object and returns its name.
func (c *defaultContext) genName(fn uintptr) uint32 {
	c.pinner.Pin(&c.nameBuf)
	defer c.pinner.Unpin()
	c.call(fn, 1, uintptr(unsafe.Pointer(&c.nameBuf)))
	return c.nameBuf
}

// deleteName calls fn, a glDelete* function, for the object name.
func (c *defaultContext) deleteName(fn uintptr, name uint32) {
	c.nameBuf = name
	c.pinner.Pin(&c.nameBuf)
	defer c.pinner.Unpin()
	c.call(fn, 1, uintptr(unsafe.Pointer(&c.nameBuf)))
}

// cString returns a NUL-terminated copy of str. The copy is valid until the next call.
func (c *defaultContext) cString(str string) *byte {
	c.strBuf = append(c.strBuf[:0], str...)
	c.strBuf = append(c.strBuf, 0)
	return &c.strBuf[0]
}

func (c *defaultContext) IsES() bool {
	return c.isES
}

func (c *defaultContext) ActiveTexture(texture uint32) {
	c.call(c.gpActiveTexture, uintptr(texture))
}

func (c *defaultContext) AttachShader(program uint32, shader uint32) {
	c.call(c.gpAttachShader, uintptr(program), uintptr(shader))
}

func (c *defaultContext) BindAttribLocation(program uint32, index uint32, name string) {
	cname := c.cString(name)
	c.pinner.Pin(cname)
	defer c.pinner.Unpin()
	c.call(c.gpBindAttribLocation, uintptr(program), uintptr(index), uintptr(unsafe.Pointer(cname)))
}

func (c *defaultContext) BindBuffer(target uint32, buffer uint32) {
	c.call(c.gpBindBuffer, uintptr(target), uintptr(buffer))
}

func (c *defaultContext) BindFramebuffer(target uint32, framebuffer uint32) {
	c.call(c.gpBindFramebuffer, uintptr(target), uintptr(framebuffer))
}

func (c *defaultContext) BindTexture(target uint32, texture uint32) {
	c.call(c.gpBindTexture, uintptr(target), uintptr(texture))
}

func (c *defaultContext) BindVertexArray(array uint32) {
	c.call(c.gpBindVertexArray, uintptr(array))
}

func (c *defaultContext) BlendEquationSeparate(modeRGB uint32, modeAlpha uint32) {
	c.call(c.gpBlendEquationSeparate, uintptr(modeRGB), uintptr(modeAlpha))
}

func (c *defaultContext) BlendFuncSeparate(srcRGB uint32, dstRGB uint32, srcAlpha uint32, dstAlpha uint32) {
	c.call(c.gpBlendFuncSeparate, uintptr(srcRGB), uintptr(dstRGB), uintptr(srcAlpha), uintptr(dstAlpha))
}

func (c *defaultContext) BufferInit(target uint32, size int, usage uint32) {
	c.call(c.gpBufferData, uintptr(target), uintptr(size), 0, uintptr(usage))
}

func (c *defaultContext) BufferSubData(target uint32, offset int, data []byte) {
	c.pinner.Pin(&data[0])
	defer c.pinner.Unpin()
	c.call(c.gpBufferSubData, uintptr(target), uintptr(offset), uintptr(len(data)), uintptr(unsafe.Pointer(&data[0])))
}

func (c *defaultContext) CheckFramebufferStatus(target uint32) uint32 {
	ret, _, _ := c.call(c.gpCheckFramebufferStatus, uintptr(target))
	return uint32(ret)
}

func (c *defaultContext) ClientWaitSync(sync uintptr, flags uint32, timeout uint64) uint32 {
	// GLuint64 occupies two argument words on 32-bit platforms.
	if bits.UintSize == 32 {
		ret, _, _ := c.call(c.gpClientWaitSync, sync, uintptr(flags), uintptr(uint32(timeout)), uintptr(timeout>>32))
		return uint32(ret)
	}
	ret, _, _ := c.call(c.gpClientWaitSync, sync, uintptr(flags), uintptr(timeout))
	return uint32(ret)
}

func (c *defaultContext) CompileShader(shader uint32) {
	c.call(c.gpCompileShader, uintptr(shader))
}

func (c *defaultContext) CreateBuffer() uint32 {
	return c.genName(c.gpGenBuffers)
}

func (c *defaultContext) CreateFramebuffer() uint32 {
	return c.genName(c.gpGenFramebuffers)
}

func (c *defaultContext) CreateProgram() uint32 {
	ret, _, _ := c.call(c.gpCreateProgram)
	return uint32(ret)
}

func (c *defaultContext) CreateShader(xtype uint32) uint32 {
	ret, _, _ := c.call(c.gpCreateShader, uintptr(xtype))
	return uint32(ret)
}

func (c *defaultContext) CreateTexture() uint32 {
	return c.genName(c.gpGenTextures)
}

func (c *defaultContext) CreateVertexArray() uint32 {
	return c.genName(c.gpGenVertexArrays)
}

func (c *defaultContext) DeleteBuffer(buffer uint32) {
	c.deleteName(c.gpDeleteBuffers, buffer)
}

func (c *defaultContext) DeleteFramebuffer(framebuffer uint32) {
	c.deleteName(c.gpDeleteFramebuffers, framebuffer)
}

func (c *defaultContext) DeleteProgram(program uint32) {
	// A program is no longer valid after a context loss and must not be deleted.
	if !c.IsProgram(program) {
		return
	}
	c.call(c.gpDeleteProgram, uintptr(program))
}

func (c *defaultContext) DeleteShader(shader uint32) {
	c.call(c.gpDeleteShader, uintptr(shader))
}

func (c *defaultContext) DeleteSync(sync uintptr) {
	c.call(c.gpDeleteSync, sync)
}

func (c *defaultContext) DeleteTexture(texture uint32) {
	c.deleteName(c.gpDeleteTextures, texture)
}

func (c *defaultContext) DeleteVertexArray(array uint32) {
	c.deleteName(c.gpDeleteVertexArrays, array)
}

func (c *defaultContext) DrawElements(mode uint32, count int32, xtype uint32, offset int) {
	c.call(c.gpDrawElements, uintptr(mode), uintptr(count), uintptr(xtype), uintptr(offset))
}

func (c *defaultContext) Enable(cap uint32) {
	c.call(c.gpEnable, uintptr(cap))
}

func (c *defaultContext) EnableVertexAttribArray(index uint32) {
	c.call(c.gpEnableVertexAttribArray, uintptr(index))
}

func (c *defaultContext) FenceSync(condition uint32, flags uint32) uintptr {
	ret, _, _ := c.call(c.gpFenceSync, uintptr(condition), uintptr(flags))
	return ret
}

func (c *defaultContext) Finish() {
	c.call(c.gpFinish)
}

func (c *defaultContext) Flush() {
	c.call(c.gpFlush)
}

func (c *defaultContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	c.call(c.gpFramebufferTexture2D, uintptr(target), uintptr(attachment), uintptr(textarget), uintptr(texture), uintptr(level))
}

func (c *defaultContext) GetBufferSubData(target uint32, offset int, data []byte) {
	panic("gl: GetBufferSubData is not implemented")
}

func (c *defaultContext) GetError() uint32 {
	ret, _, _ := c.call(c.gpGetError)
	return uint32(ret)
}

func (c *defaultContext) GetExtension(name string) any {
	return nil
}

func (c *defaultContext) GetInteger(pname uint32) int {
	c.pinner.Pin(&c.intBuf)
	defer c.pinner.Unpin()
	c.call(c.gpGetIntegerv, uintptr(pname), uintptr(unsafe.Pointer(&c.intBuf)))
	return int(c.intBuf)
}

func (c *defaultContext) GetProgramInfoLog(program uint32) string {
	bufSize := c.GetProgrami(program, INFO_LOG_LENGTH)
	if bufSize == 0 {
		return ""
	}
	infoLog := make([]byte, bufSize)
	c.pinner.Pin(&infoLog[0])
	defer c.pinner.Unpin()
	c.call(c.gpGetProgramInfoLog, uintptr(program), uintptr(bufSize), 0, uintptr(unsafe.Pointer(&infoLog[0])))
	return string(infoLog)
}

func (c *defaultContext) GetProgrami(program uint32, pname uint32) int {
	c.pinner.Pin(&c.intBuf)
	defer c.pinner.Unpin()
	c.call(c.gpGetProgramiv, uintptr(program), uintptr(pname), uintptr(unsafe.Pointer(&c.intBuf)))
	return int(c.intBuf)
}

func (c *defaultContext) GetShaderInfoLog(shader uint32) string {
	bufSize := c.GetShaderi(shader, INFO_LOG_LENGTH)
	if bufSize == 0 {
		return ""
	}
	infoLog := make([]byte, bufSize)
	c.pinner.Pin(&infoLog[0])
	defer c.pinner.Unpin()
	c.call(c.gpGetShaderInfoLog, uintptr(shader), uintptr(bufSize), 0, uintptr(unsafe.Pointer(&infoLog[0])))
	return string(infoLog)
}

func (c *defaultContext) GetShaderi(shader uint32, pname uint32) int {
	c.pinner.Pin(&c.intBuf)
	defer c.pinner.Unpin()
	c.call(c.gpGetShaderiv, uintptr(shader), uintptr(pname), uintptr(unsafe.Pointer(&c.intBuf)))
	return int(c.intBuf)
}

func (c *defaultContext) GetUniformLocation(program uint32, name string) int32 {
	cname := c.cString(name)
	c.pinner.Pin(cname)
	defer c.pinner.Unpin()
	ret, _, _ := c.call(c.gpGetUniformLocation, uintptr(program), uintptr(unsafe.Pointer(cname)))
	return int32(ret)
}

func (c *defaultContext) IsProgram(program uint32) bool {
	ret, _, _ := c.call(c.gpIsProgram, uintptr(program))
	return byte(ret) != 0
}

func (c *defaultContext) LinkProgram(program uint32) {
	c.call(c.gpLinkProgram, uintptr(program))
}

func (c *defaultContext) MapBufferRange(target uint32, offset int, length int, access uint32) []byte {
	p := c.gpMapBufferRange(target, offset, length, access)
	if p == nil {
		return nil
	}
	return unsafe.Slice((*byte)(p), length)
}

func (c *defaultContext) PixelStorei(pname uint32, param int32) {
	c.call(c.gpPixelStorei, uintptr(pname), uintptr(param))
}

func (c *defaultContext) ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32) {
	if dst == nil {
		c.call(c.gpReadPixels, uintptr(x), uintptr(y), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), 0)
		return
	}
	c.pinner.Pin(&dst[0])
	defer c.pinner.Unpin()
	c.call(c.gpReadPixels, uintptr(x), uintptr(y), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(unsafe.Pointer(&dst[0])))
}

func (c *defaultContext) Scissor(x int32, y int32, width int32, height int32) {
	c.call(c.gpScissor, uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

func (c *defaultContext) ShaderSource(shader uint32, xstring string) {
	cstring := c.cString(xstring)
	c.strAddr = uintptr(unsafe.Pointer(cstring))
	c.pinner.Pin(cstring)
	c.pinner.Pin(&c.strAddr)
	defer c.pinner.Unpin()
	c.call(c.gpShaderSource, uintptr(shader), 1, uintptr(unsafe.Pointer(&c.strAddr)), 0)
}

func (c *defaultContext) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	var ptr *byte
	if len(pixels) > 0 {
		ptr = &pixels[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpTexImage2D, uintptr(target), uintptr(level), uintptr(internalformat), uintptr(width), uintptr(height), 0, uintptr(format), uintptr(xtype), uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) TexParameteri(target uint32, pname uint32, param int32) {
	c.call(c.gpTexParameteri, uintptr(target), uintptr(pname), uintptr(param))
}

func (c *defaultContext) TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	c.pinner.Pin(&pixels[0])
	defer c.pinner.Unpin()
	c.call(c.gpTexSubImage2D, uintptr(target), uintptr(level), uintptr(xoffset), uintptr(yoffset), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(unsafe.Pointer(&pixels[0])))
}

func (c *defaultContext) Uniform1fv(location int32, value []float32) {
	var ptr *float32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniform1fv, uintptr(location), uintptr(len(value)), uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) Uniform1i(location int32, v0 int32) {
	c.call(c.gpUniform1i, uintptr(location), uintptr(v0))
}

func (c *defaultContext) Uniform1iv(location int32, value []int32) {
	var ptr *int32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniform1iv, uintptr(location), uintptr(len(value)), uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) Uniform2fv(location int32, value []float32) {
	var ptr *float32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniform2fv, uintptr(location), uintptr(len(value)/2), uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) Uniform2iv(location int32, value []int32) {
	var ptr *int32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniform2iv, uintptr(location), uintptr(len(value)/2), uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) Uniform3fv(location int32, value []float32) {
	var ptr *float32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniform3fv, uintptr(location), uintptr(len(value)/3), uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) Uniform3iv(location int32, value []int32) {
	var ptr *int32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniform3iv, uintptr(location), uintptr(len(value)/3), uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) Uniform4fv(location int32, value []float32) {
	var ptr *float32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniform4fv, uintptr(location), uintptr(len(value)/4), uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) Uniform4iv(location int32, value []int32) {
	var ptr *int32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniform4iv, uintptr(location), uintptr(len(value)/4), uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) UniformMatrix2fv(location int32, value []float32) {
	var ptr *float32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniformMatrix2fv, uintptr(location), uintptr(len(value)/4), 0, uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) UniformMatrix3fv(location int32, value []float32) {
	var ptr *float32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniformMatrix3fv, uintptr(location), uintptr(len(value)/9), 0, uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) UniformMatrix4fv(location int32, value []float32) {
	var ptr *float32
	if len(value) > 0 {
		ptr = &value[0]
		c.pinner.Pin(ptr)
		defer c.pinner.Unpin()
	}
	c.call(c.gpUniformMatrix4fv, uintptr(location), uintptr(len(value)/16), 0, uintptr(unsafe.Pointer(ptr)))
}

func (c *defaultContext) UnmapBuffer(target uint32) bool {
	r, _, _ := c.call(c.gpUnmapBuffer, uintptr(target))
	return r != 0
}

func (c *defaultContext) UseProgram(program uint32) {
	c.call(c.gpUseProgram, uintptr(program))
}

func (c *defaultContext) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int) {
	c.call(c.gpVertexAttribPointer, uintptr(index), uintptr(size), uintptr(xtype), uintptr(boolToInt(normalized)), uintptr(stride), uintptr(offset))
}

func (c *defaultContext) Viewport(x int32, y int32, width int32, height int32) {
	c.call(c.gpViewport, uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

func (c *defaultContext) LoadFunctions() error {
	g := procAddressGetter{ctx: c}

	c.gpActiveTexture = g.get("glActiveTexture")
	c.gpAttachShader = g.get("glAttachShader")
	c.gpBindAttribLocation = g.get("glBindAttribLocation")
	c.gpBindBuffer = g.get("glBindBuffer")
	c.gpBindFramebuffer = g.get("glBindFramebuffer")
	c.gpBindTexture = g.get("glBindTexture")
	c.gpBindVertexArray = g.get("glBindVertexArray")
	c.gpBlendEquationSeparate = g.get("glBlendEquationSeparate")
	c.gpBlendFuncSeparate = g.get("glBlendFuncSeparate")
	c.gpBufferData = g.get("glBufferData")
	c.gpBufferSubData = g.get("glBufferSubData")
	c.gpCheckFramebufferStatus = g.get("glCheckFramebufferStatus")
	c.gpClientWaitSync = g.get("glClientWaitSync")
	c.gpCompileShader = g.get("glCompileShader")
	c.gpCreateProgram = g.get("glCreateProgram")
	c.gpCreateShader = g.get("glCreateShader")
	c.gpDeleteBuffers = g.get("glDeleteBuffers")
	c.gpDeleteFramebuffers = g.get("glDeleteFramebuffers")
	c.gpDeleteProgram = g.get("glDeleteProgram")
	c.gpDeleteShader = g.get("glDeleteShader")
	c.gpDeleteSync = g.get("glDeleteSync")
	c.gpDeleteTextures = g.get("glDeleteTextures")
	c.gpDeleteVertexArrays = g.get("glDeleteVertexArrays")
	c.gpDrawElements = g.get("glDrawElements")
	c.gpEnable = g.get("glEnable")
	c.gpEnableVertexAttribArray = g.get("glEnableVertexAttribArray")
	c.gpFenceSync = g.get("glFenceSync")
	c.gpFinish = g.get("glFinish")
	c.gpFlush = g.get("glFlush")
	c.gpFramebufferTexture2D = g.get("glFramebufferTexture2D")
	c.gpGenBuffers = g.get("glGenBuffers")
	c.gpGenFramebuffers = g.get("glGenFramebuffers")
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
	if p := g.get("glMapBufferRange"); p != 0 {
		purego.RegisterFunc(&c.gpMapBufferRange, p)
	}
	c.gpPixelStorei = g.get("glPixelStorei")
	c.gpReadPixels = g.get("glReadPixels")
	c.gpScissor = g.get("glScissor")
	c.gpShaderSource = g.get("glShaderSource")
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
	c.gpUnmapBuffer = g.get("glUnmapBuffer")
	c.gpUseProgram = g.get("glUseProgram")
	c.gpVertexAttribPointer = g.get("glVertexAttribPointer")
	c.gpViewport = g.get("glViewport")

	return g.error()
}
