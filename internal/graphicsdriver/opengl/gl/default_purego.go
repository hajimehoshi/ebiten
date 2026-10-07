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
	gpCompileShader           uintptr
	gpCreateProgram           uintptr
	gpCreateShader            uintptr
	gpDeleteBuffers           uintptr
	gpDeleteFramebuffers      uintptr
	gpDeleteProgram           uintptr
	gpDeleteShader            uintptr
	gpDeleteTextures          uintptr
	gpDeleteVertexArrays      uintptr
	gpDrawElements            uintptr
	gpEnable                  uintptr
	gpEnableVertexAttribArray uintptr
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
	gpPixelStorei             uintptr
	gpReadPixels              uintptr
	gpScissor                 uintptr
	gpShaderSource            uintptr
	gpTexImage2D              uintptr
	gpTexParameteri           uintptr
	gpTexSubImage2D           uintptr
	gpUniform1fv              uintptr
	gpUniform1i               uintptr
	gpUniform1iv              uintptr
	gpUniform2fv              uintptr
	gpUniform2iv              uintptr
	gpUniform3fv              uintptr
	gpUniform3iv              uintptr
	gpUniform4fv              uintptr
	gpUniform4iv              uintptr
	gpUniformMatrix2fv        uintptr
	gpUniformMatrix3fv        uintptr
	gpUniformMatrix4fv        uintptr
	gpUseProgram              uintptr
	gpVertexAttribPointer     uintptr
	gpViewport                uintptr

	// args holds the arguments of call. args, pinner, and uniformBuf are shared by every call, so
	// defaultContext must not be used concurrently.
	args [15]uintptr

	// pinner pins the data of BufferSubData.
	pinner runtime.Pinner

	// uniformBuf holds a copy of the value of a Uniform*v call. uniformPinner keeps it pinned, so
	// defaultContext must never be garbage-collected.
	uniformBuf    []uint32
	uniformPinner runtime.Pinner

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

// uniformBuffer returns the address of a pinned buffer with room for n values. The address is
// valid until the next call.
func (c *defaultContext) uniformBuffer(n int) unsafe.Pointer {
	if len(c.uniformBuf) < n || c.uniformBuf == nil {
		c.uniformPinner.Unpin()
		c.uniformBuf = make([]uint32, max(n, 2*len(c.uniformBuf), 256))
		c.uniformPinner.Pin(&c.uniformBuf[0])
	}
	return unsafe.Pointer(&c.uniformBuf[0])
}

func (c *defaultContext) float32s(value []float32) uintptr {
	p := c.uniformBuffer(len(value))
	copy(unsafe.Slice((*float32)(p), len(value)), value)
	return uintptr(p)
}

func (c *defaultContext) int32s(value []int32) uintptr {
	p := c.uniformBuffer(len(value))
	copy(unsafe.Slice((*int32)(p), len(value)), value)
	return uintptr(p)
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
	cname, free := cStr(name)
	defer free()
	purego.SyscallN(c.gpBindAttribLocation, uintptr(program), uintptr(index), uintptr(unsafe.Pointer(cname)))
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

func (c *defaultContext) CompileShader(shader uint32) {
	c.call(c.gpCompileShader, uintptr(shader))
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
	ret, _, _ := c.call(c.gpCreateProgram)
	return uint32(ret)
}

func (c *defaultContext) CreateShader(xtype uint32) uint32 {
	ret, _, _ := c.call(c.gpCreateShader, uintptr(xtype))
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
	// A program is no longer valid after a context loss and must not be deleted.
	if !c.IsProgram(program) {
		return
	}
	c.call(c.gpDeleteProgram, uintptr(program))
}

func (c *defaultContext) DeleteShader(shader uint32) {
	c.call(c.gpDeleteShader, uintptr(shader))
}

func (c *defaultContext) DeleteTexture(texture uint32) {
	purego.SyscallN(c.gpDeleteTextures, 1, uintptr(unsafe.Pointer(&texture)))
}

func (c *defaultContext) DeleteVertexArray(array uint32) {
	purego.SyscallN(c.gpDeleteVertexArrays, 1, uintptr(unsafe.Pointer(&array)))
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

func (c *defaultContext) Finish() {
	c.call(c.gpFinish)
}

func (c *defaultContext) Flush() {
	c.call(c.gpFlush)
}

func (c *defaultContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	c.call(c.gpFramebufferTexture2D, uintptr(target), uintptr(attachment), uintptr(textarget), uintptr(texture), uintptr(level))
}

func (c *defaultContext) GetError() uint32 {
	ret, _, _ := c.call(c.gpGetError)
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
	ret, _, _ := c.call(c.gpIsProgram, uintptr(program))
	return byte(ret) != 0
}

func (c *defaultContext) LinkProgram(program uint32) {
	c.call(c.gpLinkProgram, uintptr(program))
}

func (c *defaultContext) PixelStorei(pname uint32, param int32) {
	c.call(c.gpPixelStorei, uintptr(pname), uintptr(param))
}

func (c *defaultContext) ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32) {
	purego.SyscallN(c.gpReadPixels, uintptr(x), uintptr(y), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(unsafe.Pointer(&dst[0])))
	runtime.KeepAlive(dst)
}

func (c *defaultContext) Scissor(x int32, y int32, width int32, height int32) {
	c.call(c.gpScissor, uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

func (c *defaultContext) ShaderSource(shader uint32, xstring string) {
	cstring, free := cStr(xstring)
	defer free()
	purego.SyscallN(c.gpShaderSource, uintptr(shader), 1, uintptr(unsafe.Pointer(&cstring)), 0)
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
	c.call(c.gpTexParameteri, uintptr(target), uintptr(pname), uintptr(param))
}

func (c *defaultContext) TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	purego.SyscallN(c.gpTexSubImage2D, uintptr(target), uintptr(level), uintptr(xoffset), uintptr(yoffset), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(unsafe.Pointer(&pixels[0])))
	runtime.KeepAlive(pixels)
}

func (c *defaultContext) Uniform1fv(location int32, value []float32) {
	c.call(c.gpUniform1fv, uintptr(location), uintptr(len(value)), c.float32s(value))
}

func (c *defaultContext) Uniform1i(location int32, v0 int32) {
	c.call(c.gpUniform1i, uintptr(location), uintptr(v0))
}

func (c *defaultContext) Uniform1iv(location int32, value []int32) {
	c.call(c.gpUniform1iv, uintptr(location), uintptr(len(value)), c.int32s(value))
}

func (c *defaultContext) Uniform2fv(location int32, value []float32) {
	c.call(c.gpUniform2fv, uintptr(location), uintptr(len(value)/2), c.float32s(value))
}

func (c *defaultContext) Uniform2iv(location int32, value []int32) {
	c.call(c.gpUniform2iv, uintptr(location), uintptr(len(value)/2), c.int32s(value))
}

func (c *defaultContext) Uniform3fv(location int32, value []float32) {
	c.call(c.gpUniform3fv, uintptr(location), uintptr(len(value)/3), c.float32s(value))
}

func (c *defaultContext) Uniform3iv(location int32, value []int32) {
	c.call(c.gpUniform3iv, uintptr(location), uintptr(len(value)/3), c.int32s(value))
}

func (c *defaultContext) Uniform4fv(location int32, value []float32) {
	c.call(c.gpUniform4fv, uintptr(location), uintptr(len(value)/4), c.float32s(value))
}

func (c *defaultContext) Uniform4iv(location int32, value []int32) {
	c.call(c.gpUniform4iv, uintptr(location), uintptr(len(value)/4), c.int32s(value))
}

func (c *defaultContext) UniformMatrix2fv(location int32, value []float32) {
	c.call(c.gpUniformMatrix2fv, uintptr(location), uintptr(len(value)/4), 0, c.float32s(value))
}

func (c *defaultContext) UniformMatrix3fv(location int32, value []float32) {
	c.call(c.gpUniformMatrix3fv, uintptr(location), uintptr(len(value)/9), 0, c.float32s(value))
}

func (c *defaultContext) UniformMatrix4fv(location int32, value []float32) {
	c.call(c.gpUniformMatrix4fv, uintptr(location), uintptr(len(value)/16), 0, c.float32s(value))
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
	c.gpCompileShader = g.get("glCompileShader")
	c.gpCreateProgram = g.get("glCreateProgram")
	c.gpCreateShader = g.get("glCreateShader")
	c.gpDeleteBuffers = g.get("glDeleteBuffers")
	c.gpDeleteFramebuffers = g.get("glDeleteFramebuffers")
	c.gpDeleteProgram = g.get("glDeleteProgram")
	c.gpDeleteShader = g.get("glDeleteShader")
	c.gpDeleteTextures = g.get("glDeleteTextures")
	c.gpDeleteVertexArrays = g.get("glDeleteVertexArrays")
	c.gpDrawElements = g.get("glDrawElements")
	c.gpEnable = g.get("glEnable")
	c.gpEnableVertexAttribArray = g.get("glEnableVertexAttribArray")
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
	c.gpUseProgram = g.get("glUseProgram")
	c.gpVertexAttribPointer = g.get("glVertexAttribPointer")
	c.gpViewport = g.get("glViewport")

	return g.error()
}

// cStr takes a Go string (with or without null-termination)
// and returns the C counterpart.
//
// The bytes are Go-managed memory, so the returned function frees nothing. It
// must be called once the string is no longer used, as it keeps it alive until
// then.
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
