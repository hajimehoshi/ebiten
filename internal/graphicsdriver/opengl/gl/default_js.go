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

package gl

import (
	"fmt"
	"syscall/js"
)

type defaultContext struct {
	fnActiveTexture            js.Value
	fnAttachShader             js.Value
	fnBindAttribLocation       js.Value
	fnBindBuffer               js.Value
	fnBindFramebuffer          js.Value
	fnBindRenderbuffer         js.Value
	fnBindTexture              js.Value
	fnBindVertexArray          js.Value
	fnBlendEquationSeparate    js.Value
	fnBlendFuncSeparate        js.Value
	fnBufferData               js.Value
	fnBufferSubData            js.Value
	fnCheckFramebufferStatus   js.Value
	fnClear                    js.Value
	fnColorMask                js.Value
	fnCompileShader            js.Value
	fnCreateBuffer             js.Value
	fnCreateFramebuffer        js.Value
	fnCreateProgram            js.Value
	fnCreateRenderbuffer       js.Value
	fnCreateShader             js.Value
	fnCreateTexture            js.Value
	fnCreateVertexArray        js.Value
	fnDeleteBuffer             js.Value
	fnDeleteFramebuffer        js.Value
	fnDeleteProgram            js.Value
	fnDeleteRenderbuffer       js.Value
	fnDeleteShader             js.Value
	fnDeleteTexture            js.Value
	fnDeleteVertexArray        js.Value
	fnDisable                  js.Value
	fnDisableVertexAttribArray js.Value
	fnDrawElements             js.Value
	fnEnable                   js.Value
	fnEnableVertexAttribArray  js.Value
	fnFramebufferRenderbuffer  js.Value
	fnFramebufferTexture2D     js.Value
	fnFlush                    js.Value
	fnGetError                 js.Value
	fnGetParameter             js.Value
	fnGetProgramInfoLog        js.Value
	fnGetProgramParameter      js.Value
	fnGetShaderInfoLog         js.Value
	fnGetShaderParameter       js.Value
	fnGetUniformLocation       js.Value
	fnIsProgram                js.Value
	fnLinkProgram              js.Value
	fnPixelStorei              js.Value
	fnReadPixels               js.Value
	fnRenderbufferStorage      js.Value
	fnScissor                  js.Value
	fnShaderSource             js.Value
	fnStencilFunc              js.Value
	fnStencilMask              js.Value
	fnStencilOpSeparate        js.Value
	fnTexImage2D               js.Value
	fnTexSubImage2D            js.Value
	fnTexParameteri            js.Value
	fnUniform1fv               js.Value
	fnUniform1i                js.Value
	fnUniform1iv               js.Value
	fnUniform2fv               js.Value
	fnUniform2iv               js.Value
	fnUniform3fv               js.Value
	fnUniform3iv               js.Value
	fnUniform4fv               js.Value
	fnUniform4iv               js.Value
	fnUniformMatrix2fv         js.Value
	fnUniformMatrix3fv         js.Value
	fnUniformMatrix4fv         js.Value
	fnUseProgram               js.Value
	fnVertexAttribPointer      js.Value
	fnViewport                 js.Value

	buffers          values
	framebuffers     values
	programs         values
	renderbuffers    values
	shaders          values
	textures         values
	vertexArrays     values
	uniformLocations map[uint32]*values
}

type values struct {
	idToValue map[uint32]js.Value
	lastID    uint32
}

func (v *values) create(value js.Value) uint32 {
	v.lastID++
	id := v.lastID
	if v.idToValue == nil {
		v.idToValue = map[uint32]js.Value{}
	}
	v.idToValue[id] = value
	return id
}

func (v *values) get(id uint32) js.Value {
	return v.idToValue[id]
}

func (v *values) getID(value js.Value) (uint32, bool) {
	for id, v := range v.idToValue {
		if v.Equal(value) {
			return id, true
		}
	}
	return 0, false
}

func (v *values) getOrCreate(value js.Value) uint32 {
	id, ok := v.getID(value)
	if ok {
		return id
	}
	return v.create(value)
}

func (v *values) delete(id uint32) {
	delete(v.idToValue, id)
}

func NewDefaultContext(v js.Value) (Context, error) {
	// Passing a Go string to the JS world is expensive. This causes conversion to UTF-16 (#1438).
	// In order to reduce the cost when calling functions, create the function objects by bind and use them.
	g := &defaultContext{
		fnActiveTexture:            v.Get("activeTexture").Call("bind", v),
		fnAttachShader:             v.Get("attachShader").Call("bind", v),
		fnBindAttribLocation:       v.Get("bindAttribLocation").Call("bind", v),
		fnBindBuffer:               v.Get("bindBuffer").Call("bind", v),
		fnBindFramebuffer:          v.Get("bindFramebuffer").Call("bind", v),
		fnBindRenderbuffer:         v.Get("bindRenderbuffer").Call("bind", v),
		fnBindTexture:              v.Get("bindTexture").Call("bind", v),
		fnBindVertexArray:          v.Get("bindVertexArray").Call("bind", v),
		fnBlendEquationSeparate:    v.Get("blendEquationSeparate").Call("bind", v),
		fnBlendFuncSeparate:        v.Get("blendFuncSeparate").Call("bind", v),
		fnBufferData:               v.Get("bufferData").Call("bind", v),
		fnBufferSubData:            v.Get("bufferSubData").Call("bind", v),
		fnCheckFramebufferStatus:   v.Get("checkFramebufferStatus").Call("bind", v),
		fnClear:                    v.Get("clear").Call("bind", v),
		fnColorMask:                v.Get("colorMask").Call("bind", v),
		fnCompileShader:            v.Get("compileShader").Call("bind", v),
		fnCreateBuffer:             v.Get("createBuffer").Call("bind", v),
		fnCreateFramebuffer:        v.Get("createFramebuffer").Call("bind", v),
		fnCreateProgram:            v.Get("createProgram").Call("bind", v),
		fnCreateRenderbuffer:       v.Get("createRenderbuffer").Call("bind", v),
		fnCreateShader:             v.Get("createShader").Call("bind", v),
		fnCreateTexture:            v.Get("createTexture").Call("bind", v),
		fnCreateVertexArray:        v.Get("createVertexArray").Call("bind", v),
		fnDeleteBuffer:             v.Get("deleteBuffer").Call("bind", v),
		fnDeleteFramebuffer:        v.Get("deleteFramebuffer").Call("bind", v),
		fnDeleteProgram:            v.Get("deleteProgram").Call("bind", v),
		fnDeleteRenderbuffer:       v.Get("deleteRenderbuffer").Call("bind", v),
		fnDeleteShader:             v.Get("deleteShader").Call("bind", v),
		fnDeleteTexture:            v.Get("deleteTexture").Call("bind", v),
		fnDeleteVertexArray:        v.Get("deleteVertexArray").Call("bind", v),
		fnDisable:                  v.Get("disable").Call("bind", v),
		fnDisableVertexAttribArray: v.Get("disableVertexAttribArray").Call("bind", v),
		fnDrawElements:             v.Get("drawElements").Call("bind", v),
		fnEnable:                   v.Get("enable").Call("bind", v),
		fnEnableVertexAttribArray:  v.Get("enableVertexAttribArray").Call("bind", v),
		fnFramebufferRenderbuffer:  v.Get("framebufferRenderbuffer").Call("bind", v),
		fnFramebufferTexture2D:     v.Get("framebufferTexture2D").Call("bind", v),
		fnFlush:                    v.Get("flush").Call("bind", v),
		fnGetError:                 v.Get("getError").Call("bind", v),
		fnGetParameter:             v.Get("getParameter").Call("bind", v),
		fnGetProgramInfoLog:        v.Get("getProgramInfoLog").Call("bind", v),
		fnGetProgramParameter:      v.Get("getProgramParameter").Call("bind", v),
		fnGetShaderInfoLog:         v.Get("getShaderInfoLog").Call("bind", v),
		fnGetShaderParameter:       v.Get("getShaderParameter").Call("bind", v),
		fnGetUniformLocation:       v.Get("getUniformLocation").Call("bind", v),
		fnIsProgram:                v.Get("isProgram").Call("bind", v),
		fnLinkProgram:              v.Get("linkProgram").Call("bind", v),
		fnPixelStorei:              v.Get("pixelStorei").Call("bind", v),
		fnReadPixels:               v.Get("readPixels").Call("bind", v),
		fnRenderbufferStorage:      v.Get("renderbufferStorage").Call("bind", v),
		fnScissor:                  v.Get("scissor").Call("bind", v),
		fnShaderSource:             v.Get("shaderSource").Call("bind", v),
		fnStencilFunc:              v.Get("stencilFunc").Call("bind", v),
		fnStencilMask:              v.Get("stencilMask").Call("bind", v),
		fnStencilOpSeparate:        v.Get("stencilOpSeparate").Call("bind", v),
		fnTexImage2D:               v.Get("texImage2D").Call("bind", v),
		fnTexSubImage2D:            v.Get("texSubImage2D").Call("bind", v),
		fnTexParameteri:            v.Get("texParameteri").Call("bind", v),
		fnUniform1fv:               v.Get("uniform1fv").Call("bind", v),
		fnUniform1i:                v.Get("uniform1i").Call("bind", v),
		fnUniform1iv:               v.Get("uniform1iv").Call("bind", v),
		fnUniform2fv:               v.Get("uniform2fv").Call("bind", v),
		fnUniform2iv:               v.Get("uniform2iv").Call("bind", v),
		fnUniform3fv:               v.Get("uniform3fv").Call("bind", v),
		fnUniform3iv:               v.Get("uniform3iv").Call("bind", v),
		fnUniform4fv:               v.Get("uniform4fv").Call("bind", v),
		fnUniform4iv:               v.Get("uniform4iv").Call("bind", v),
		fnUniformMatrix2fv:         v.Get("uniformMatrix2fv").Call("bind", v),
		fnUniformMatrix3fv:         v.Get("uniformMatrix3fv").Call("bind", v),
		fnUniformMatrix4fv:         v.Get("uniformMatrix4fv").Call("bind", v),
		fnUseProgram:               v.Get("useProgram").Call("bind", v),
		fnVertexAttribPointer:      v.Get("vertexAttribPointer").Call("bind", v),
		fnViewport:                 v.Get("viewport").Call("bind", v),
	}

	return g, nil
}

func (c *defaultContext) getUniformLocation(location int32) js.Value {
	program := uint32(location) >> 5
	return c.uniformLocations[program].get(uint32(location) & ((1 << 5) - 1))
}

func (c *defaultContext) LoadFunctions() error {
	return nil
}

func (c *defaultContext) IsES() bool {
	// WebGL is compatible with GLES.
	return true
}

func (c *defaultContext) ActiveTexture(texture uint32) {
	c.fnActiveTexture.Invoke(texture)
}

func (c *defaultContext) AttachShader(program uint32, shader uint32) {
	c.fnAttachShader.Invoke(c.programs.get(program), c.shaders.get(shader))
}

func (c *defaultContext) BindAttribLocation(program uint32, index uint32, name string) {
	c.fnBindAttribLocation.Invoke(c.programs.get(program), index, name)
}

func (c *defaultContext) BindBuffer(target uint32, buffer uint32) {
	c.fnBindBuffer.Invoke(target, c.buffers.get(buffer))
}

func (c *defaultContext) BindFramebuffer(target uint32, framebuffer uint32) {
	c.fnBindFramebuffer.Invoke(target, c.framebuffers.get(framebuffer))
}

func (c *defaultContext) BindRenderbuffer(target uint32, renderbuffer uint32) {
	c.fnBindRenderbuffer.Invoke(target, c.renderbuffers.get(renderbuffer))
}

func (c *defaultContext) BindTexture(target uint32, texture uint32) {
	c.fnBindTexture.Invoke(target, c.textures.get(texture))
}

func (c *defaultContext) BindVertexArray(array uint32) {
	c.fnBindVertexArray.Invoke(c.vertexArrays.get(array))
}

func (c *defaultContext) BlendEquationSeparate(modeRGB uint32, modeAlpha uint32) {
	c.fnBlendEquationSeparate.Invoke(modeRGB, modeAlpha)
}

func (c *defaultContext) BlendFuncSeparate(srcRGB uint32, dstRGB uint32, srcAlpha uint32, dstAlpha uint32) {
	c.fnBlendFuncSeparate.Invoke(srcRGB, dstRGB, srcAlpha, dstAlpha)
}

func (c *defaultContext) BufferInit(target uint32, size int, usage uint32) {
	c.fnBufferData.Invoke(target, size, usage)
}

func (c *defaultContext) BufferSubData(target uint32, offset int, data []byte) {
	l := len(data)
	arr := tmpUint8ArrayFromUint8Slice(l, data)
	c.fnBufferSubData.Invoke(target, offset, arr, 0, l)
}

func (c *defaultContext) CheckFramebufferStatus(target uint32) uint32 {
	return uint32(c.fnCheckFramebufferStatus.Invoke(target).Int())
}

func (c *defaultContext) Clear(mask uint32) {
	c.fnClear.Invoke(mask)
}

func (c *defaultContext) ColorMask(red, green, blue, alpha bool) {
	c.fnColorMask.Invoke(red, green, blue, alpha)
}

func (c *defaultContext) CompileShader(shader uint32) {
	c.fnCompileShader.Invoke(c.shaders.get(shader))
}

func (c *defaultContext) CreateBuffer() uint32 {
	return c.buffers.create(c.fnCreateBuffer.Invoke())
}

func (c *defaultContext) CreateFramebuffer() uint32 {
	return c.framebuffers.create(c.fnCreateFramebuffer.Invoke())
}

func (c *defaultContext) CreateProgram() uint32 {
	return c.programs.create(c.fnCreateProgram.Invoke())
}

func (c *defaultContext) CreateRenderbuffer() uint32 {
	return c.renderbuffers.create(c.fnCreateRenderbuffer.Invoke())
}

func (c *defaultContext) CreateShader(xtype uint32) uint32 {
	return c.shaders.create(c.fnCreateShader.Invoke(xtype))
}

func (c *defaultContext) CreateTexture() uint32 {
	return c.textures.create(c.fnCreateTexture.Invoke())
}

func (c *defaultContext) CreateVertexArray() uint32 {
	return c.vertexArrays.create(c.fnCreateVertexArray.Invoke())
}

func (c *defaultContext) DeleteBuffer(buffer uint32) {
	c.fnDeleteBuffer.Invoke(c.buffers.get(buffer))
	c.buffers.delete(buffer)
}

func (c *defaultContext) DeleteFramebuffer(framebuffer uint32) {
	c.fnDeleteFramebuffer.Invoke(c.framebuffers.get(framebuffer))
	c.framebuffers.delete(framebuffer)
}

func (c *defaultContext) DeleteProgram(program uint32) {
	c.fnDeleteProgram.Invoke(c.programs.get(program))
	c.programs.delete(program)
	delete(c.uniformLocations, program)
}

func (c *defaultContext) DeleteRenderbuffer(renderbuffer uint32) {
	c.fnDeleteRenderbuffer.Invoke(c.renderbuffers.get(renderbuffer))
	c.renderbuffers.delete(renderbuffer)
}

func (c *defaultContext) DeleteShader(shader uint32) {
	c.fnDeleteShader.Invoke(c.shaders.get(shader))
	c.shaders.delete(shader)
}

func (c *defaultContext) DeleteTexture(texture uint32) {
	c.fnDeleteTexture.Invoke(c.textures.get(texture))
	c.textures.delete(texture)
}

func (c *defaultContext) DeleteVertexArray(array uint32) {
	c.fnDeleteVertexArray.Invoke(c.vertexArrays.get(array))
	c.textures.delete(array)
}

func (c *defaultContext) Disable(cap uint32) {
	c.fnDisable.Invoke(cap)
}

func (c *defaultContext) DisableVertexAttribArray(index uint32) {
	c.fnDisableVertexAttribArray.Invoke(index)
}

func (c *defaultContext) DrawElements(mode uint32, count int32, xtype uint32, offset int) {
	c.fnDrawElements.Invoke(mode, count, xtype, offset)
}

func (c *defaultContext) Enable(cap uint32) {
	c.fnEnable.Invoke(cap)
}

func (c *defaultContext) EnableVertexAttribArray(index uint32) {
	c.fnEnableVertexAttribArray.Invoke(index)
}

func (c *defaultContext) Flush() {
	c.fnFlush.Invoke()
}

func (c *defaultContext) FramebufferRenderbuffer(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	c.fnFramebufferRenderbuffer.Invoke(target, attachment, renderbuffertarget, c.renderbuffers.get(renderbuffer))
}

func (c *defaultContext) FramebufferTexture2D(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	c.fnFramebufferTexture2D.Invoke(target, attachment, textarget, c.textures.get(texture), level)
}

func (c *defaultContext) GetError() uint32 {
	return uint32(c.fnGetError.Invoke().Int())
}

func (c *defaultContext) GetInteger(pname uint32) int {
	ret := c.fnGetParameter.Invoke(pname)
	switch pname {
	case FRAMEBUFFER_BINDING:
		id, ok := c.framebuffers.getID(ret)
		if !ok {
			return 0
		}
		return int(id)
	case MAX_TEXTURE_SIZE:
		return ret.Int()
	default:
		panic(fmt.Sprintf("gl: unexpected pname at GetInteger: %d", pname))
	}
}

func (c *defaultContext) GetProgramInfoLog(program uint32) string {
	return c.fnGetProgramInfoLog.Invoke(c.programs.get(program)).String()
}

func (c *defaultContext) GetProgrami(program uint32, pname uint32) int {
	v := c.fnGetProgramParameter.Invoke(c.programs.get(program), pname)
	switch v.Type() {
	case js.TypeNumber:
		return v.Int()
	case js.TypeBoolean:
		if v.Bool() {
			return TRUE
		}
		return FALSE
	default:
		panic(fmt.Sprintf("gl: unexpected return type at GetProgrami: %v", v))
	}
}

func (c *defaultContext) GetShaderInfoLog(shader uint32) string {
	return c.fnGetShaderInfoLog.Invoke(c.shaders.get(shader)).String()
}

func (c *defaultContext) GetShaderi(shader uint32, pname uint32) int {
	v := c.fnGetShaderParameter.Invoke(c.shaders.get(shader), pname)
	switch v.Type() {
	case js.TypeNumber:
		return v.Int()
	case js.TypeBoolean:
		if v.Bool() {
			return TRUE
		}
		return FALSE
	default:
		panic(fmt.Sprintf("gl: unexpected return type at GetShaderi: %v", v))
	}

}

func (c *defaultContext) GetUniformLocation(program uint32, name string) int32 {
	location := c.fnGetUniformLocation.Invoke(c.programs.get(program), name)
	if c.uniformLocations == nil {
		c.uniformLocations = map[uint32]*values{}
	}
	vs, ok := c.uniformLocations[program]
	if !ok {
		vs = &values{}
		c.uniformLocations[program] = vs
	}
	idx := vs.getOrCreate(location)
	return int32((program << 5) | idx)
}

func (c *defaultContext) IsProgram(program uint32) bool {
	return c.fnIsProgram.Invoke(c.programs.get(program)).Bool()
}

func (c *defaultContext) LinkProgram(program uint32) {
	c.fnLinkProgram.Invoke(c.programs.get(program))
}

func (c *defaultContext) PixelStorei(pname uint32, param int32) {
	c.fnPixelStorei.Invoke(pname, param)
}

func (c *defaultContext) ReadPixels(dst []byte, x int32, y int32, width int32, height int32, format uint32, xtype uint32) {
	if dst == nil {
		c.fnReadPixels.Invoke(x, y, width, height, format, xtype, 0)
		return
	}
	p := tmpUint8ArrayFromUint8Slice(len(dst), nil)
	c.fnReadPixels.Invoke(x, y, width, height, format, xtype, p)
	js.CopyBytesToGo(dst, p)
}

func (c *defaultContext) RenderbufferStorage(target uint32, internalFormat uint32, width int32, height int32) {
	c.fnRenderbufferStorage.Invoke(target, internalFormat, width, height)
}

func (c *defaultContext) Scissor(x, y, width, height int32) {
	c.fnScissor.Invoke(x, y, width, height)
}

func (c *defaultContext) ShaderSource(shader uint32, xstring string) {
	c.fnShaderSource.Invoke(c.shaders.get(shader), xstring)
}

func (c *defaultContext) StencilFunc(func_ uint32, ref int32, mask uint32) {
	c.fnStencilFunc.Invoke(func_, ref, mask)
}

func (c *defaultContext) StencilOpSeparate(face, sfail, dpfail, dppass uint32) {
	c.fnStencilOpSeparate.Invoke(face, sfail, dpfail, dppass)
}

func (c *defaultContext) TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	if pixels != nil {
		panic("gl: TexImage2D with non-nil pixels is not implemented")
	}
	c.fnTexImage2D.Invoke(target, level, internalformat, width, height, 0, format, xtype, nil)
}

func (c *defaultContext) TexParameteri(target uint32, pname uint32, param int32) {
	c.fnTexParameteri.Invoke(target, pname, param)
}

func (c *defaultContext) TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels []byte) {
	arr := tmpUint8ArrayFromUint8Slice(len(pixels), pixels)
	// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
	//                    GLsizei width, GLsizei height,
	//                    GLenum format, GLenum type, ArrayBufferView pixels, srcOffset);
	c.fnTexSubImage2D.Invoke(target, level, xoffset, yoffset, width, height, format, xtype, arr, 0)
}

func (c *defaultContext) Uniform1fv(location int32, value []float32) {
	l := c.getUniformLocation(location)
	arr := tmpFloat32ArrayFromFloat32Slice(len(value), value)
	c.fnUniform1fv.Invoke(l, arr, 0, len(value))
}

func (c *defaultContext) Uniform1i(location int32, v0 int32) {
	l := c.getUniformLocation(location)
	c.fnUniform1i.Invoke(l, v0)
}

func (c *defaultContext) Uniform1iv(location int32, value []int32) {
	l := c.getUniformLocation(location)
	arr := tmpInt32ArrayFromInt32Slice(len(value), value)
	c.fnUniform1iv.Invoke(l, arr, 0, len(value))
}

func (c *defaultContext) Uniform2fv(location int32, value []float32) {
	l := c.getUniformLocation(location)
	arr := tmpFloat32ArrayFromFloat32Slice(len(value), value)
	c.fnUniform2fv.Invoke(l, arr, 0, len(value))
}

func (c *defaultContext) Uniform2iv(location int32, value []int32) {
	l := c.getUniformLocation(location)
	arr := tmpInt32ArrayFromInt32Slice(len(value), value)
	c.fnUniform2iv.Invoke(l, arr, 0, len(value))
}

func (c *defaultContext) Uniform3fv(location int32, value []float32) {
	l := c.getUniformLocation(location)
	arr := tmpFloat32ArrayFromFloat32Slice(len(value), value)
	c.fnUniform3fv.Invoke(l, arr, 0, len(value))
}

func (c *defaultContext) Uniform3iv(location int32, value []int32) {
	l := c.getUniformLocation(location)
	arr := tmpInt32ArrayFromInt32Slice(len(value), value)
	c.fnUniform3iv.Invoke(l, arr, 0, len(value))
}

func (c *defaultContext) Uniform4fv(location int32, value []float32) {
	l := c.getUniformLocation(location)
	arr := tmpFloat32ArrayFromFloat32Slice(len(value), value)
	c.fnUniform4fv.Invoke(l, arr, 0, len(value))
}

func (c *defaultContext) Uniform4iv(location int32, value []int32) {
	l := c.getUniformLocation(location)
	arr := tmpInt32ArrayFromInt32Slice(len(value), value)
	c.fnUniform4iv.Invoke(l, arr, 0, len(value))
}

func (c *defaultContext) UniformMatrix2fv(location int32, value []float32) {
	l := c.getUniformLocation(location)
	arr := tmpFloat32ArrayFromFloat32Slice(len(value), value)
	c.fnUniformMatrix2fv.Invoke(l, false, arr, 0, len(value))
}

func (c *defaultContext) UniformMatrix3fv(location int32, value []float32) {
	l := c.getUniformLocation(location)
	arr := tmpFloat32ArrayFromFloat32Slice(len(value), value)
	c.fnUniformMatrix3fv.Invoke(l, false, arr, 0, len(value))
}

func (c *defaultContext) UniformMatrix4fv(location int32, value []float32) {
	l := c.getUniformLocation(location)
	arr := tmpFloat32ArrayFromFloat32Slice(len(value), value)
	c.fnUniformMatrix4fv.Invoke(l, false, arr, 0, len(value))
}

func (c *defaultContext) UseProgram(program uint32) {
	c.fnUseProgram.Invoke(c.programs.get(program))
}

func (c *defaultContext) VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset int) {
	c.fnVertexAttribPointer.Invoke(index, size, xtype, normalized, stride, offset)
}

func (c *defaultContext) Viewport(x int32, y int32, width int32, height int32) {
	c.fnViewport.Invoke(x, y, width, height)
}
