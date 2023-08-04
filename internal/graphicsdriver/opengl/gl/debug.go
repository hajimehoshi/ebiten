// Copyright 2023 The Ebitengine Authors
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
	"os"
)

type DebugContext struct {
	Context Context
}

var _ Context = (*DebugContext)(nil)

func (d *DebugContext) ActiveTexture(arg0 uint32) {
	d.Context.ActiveTexture(arg0)
	fmt.Fprintln(os.Stderr, "ActiveTexture")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at ActiveTexture", e))
	}
}

func (d *DebugContext) AttachShader(arg0 uint32, arg1 uint32) {
	d.Context.AttachShader(arg0, arg1)
	fmt.Fprintln(os.Stderr, "AttachShader")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at AttachShader", e))
	}
}

func (d *DebugContext) BindAttribLocation(arg0 uint32, arg1 uint32, arg2 string) {
	d.Context.BindAttribLocation(arg0, arg1, arg2)
	fmt.Fprintln(os.Stderr, "BindAttribLocation")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at BindAttribLocation", e))
	}
}

func (d *DebugContext) BindBuffer(arg0 uint32, arg1 uint32) {
	d.Context.BindBuffer(arg0, arg1)
	fmt.Fprintln(os.Stderr, "BindBuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at BindBuffer", e))
	}
}

func (d *DebugContext) BindFramebuffer(arg0 uint32, arg1 uint32) {
	d.Context.BindFramebuffer(arg0, arg1)
	fmt.Fprintln(os.Stderr, "BindFramebuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at BindFramebuffer", e))
	}
}

func (d *DebugContext) BindRenderbuffer(arg0 uint32, arg1 uint32) {
	d.Context.BindRenderbuffer(arg0, arg1)
	fmt.Fprintln(os.Stderr, "BindRenderbuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at BindRenderbuffer", e))
	}
}

func (d *DebugContext) BindTexture(arg0 uint32, arg1 uint32) {
	d.Context.BindTexture(arg0, arg1)
	fmt.Fprintln(os.Stderr, "BindTexture")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at BindTexture", e))
	}
}

func (d *DebugContext) BindVertexArray(arg0 uint32) {
	d.Context.BindVertexArray(arg0)
	fmt.Fprintln(os.Stderr, "BindVertexArray")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at BindVertexArray", e))
	}
}

func (d *DebugContext) BlendEquationSeparate(arg0 uint32, arg1 uint32) {
	d.Context.BlendEquationSeparate(arg0, arg1)
	fmt.Fprintln(os.Stderr, "BlendEquationSeparate")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at BlendEquationSeparate", e))
	}
}

func (d *DebugContext) BlendFuncSeparate(arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32) {
	d.Context.BlendFuncSeparate(arg0, arg1, arg2, arg3)
	fmt.Fprintln(os.Stderr, "BlendFuncSeparate")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at BlendFuncSeparate", e))
	}
}

func (d *DebugContext) BufferInit(arg0 uint32, arg1 int, arg2 uint32) {
	d.Context.BufferInit(arg0, arg1, arg2)
	fmt.Fprintln(os.Stderr, "BufferInit")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at BufferInit", e))
	}
}

func (d *DebugContext) BufferSubData(arg0 uint32, arg1 int, arg2 []uint8) {
	d.Context.BufferSubData(arg0, arg1, arg2)
	fmt.Fprintln(os.Stderr, "BufferSubData")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at BufferSubData", e))
	}
}

func (d *DebugContext) CheckFramebufferStatus(arg0 uint32) uint32 {
	out0 := d.Context.CheckFramebufferStatus(arg0)
	fmt.Fprintln(os.Stderr, "CheckFramebufferStatus")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at CheckFramebufferStatus", e))
	}
	return out0
}

func (d *DebugContext) Clear(arg0 uint32) {
	d.Context.Clear(arg0)
	fmt.Fprintln(os.Stderr, "Clear")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Clear", e))
	}
}

func (d *DebugContext) ColorMask(arg0 bool, arg1 bool, arg2 bool, arg3 bool) {
	d.Context.ColorMask(arg0, arg1, arg2, arg3)
	fmt.Fprintln(os.Stderr, "ColorMask")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at ColorMask", e))
	}
}

func (d *DebugContext) CompileShader(arg0 uint32) {
	d.Context.CompileShader(arg0)
	fmt.Fprintln(os.Stderr, "CompileShader")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at CompileShader", e))
	}
}

func (d *DebugContext) CreateBuffer() uint32 {
	out0 := d.Context.CreateBuffer()
	fmt.Fprintln(os.Stderr, "CreateBuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at CreateBuffer", e))
	}
	return out0
}

func (d *DebugContext) CreateFramebuffer() uint32 {
	out0 := d.Context.CreateFramebuffer()
	fmt.Fprintln(os.Stderr, "CreateFramebuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at CreateFramebuffer", e))
	}
	return out0
}

func (d *DebugContext) CreateProgram() uint32 {
	out0 := d.Context.CreateProgram()
	fmt.Fprintln(os.Stderr, "CreateProgram")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at CreateProgram", e))
	}
	return out0
}

func (d *DebugContext) CreateRenderbuffer() uint32 {
	out0 := d.Context.CreateRenderbuffer()
	fmt.Fprintln(os.Stderr, "CreateRenderbuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at CreateRenderbuffer", e))
	}
	return out0
}

func (d *DebugContext) CreateShader(arg0 uint32) uint32 {
	out0 := d.Context.CreateShader(arg0)
	fmt.Fprintln(os.Stderr, "CreateShader")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at CreateShader", e))
	}
	return out0
}

func (d *DebugContext) CreateTexture() uint32 {
	out0 := d.Context.CreateTexture()
	fmt.Fprintln(os.Stderr, "CreateTexture")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at CreateTexture", e))
	}
	return out0
}

func (d *DebugContext) CreateVertexArray() uint32 {
	out0 := d.Context.CreateVertexArray()
	fmt.Fprintln(os.Stderr, "CreateVertexArray")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at CreateVertexArray", e))
	}
	return out0
}

func (d *DebugContext) DeleteBuffer(arg0 uint32) {
	d.Context.DeleteBuffer(arg0)
	fmt.Fprintln(os.Stderr, "DeleteBuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at DeleteBuffer", e))
	}
}

func (d *DebugContext) DeleteFramebuffer(arg0 uint32) {
	d.Context.DeleteFramebuffer(arg0)
	fmt.Fprintln(os.Stderr, "DeleteFramebuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at DeleteFramebuffer", e))
	}
}

func (d *DebugContext) DeleteProgram(arg0 uint32) {
	d.Context.DeleteProgram(arg0)
	fmt.Fprintln(os.Stderr, "DeleteProgram")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at DeleteProgram", e))
	}
}

func (d *DebugContext) DeleteRenderbuffer(arg0 uint32) {
	d.Context.DeleteRenderbuffer(arg0)
	fmt.Fprintln(os.Stderr, "DeleteRenderbuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at DeleteRenderbuffer", e))
	}
}

func (d *DebugContext) DeleteShader(arg0 uint32) {
	d.Context.DeleteShader(arg0)
	fmt.Fprintln(os.Stderr, "DeleteShader")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at DeleteShader", e))
	}
}

func (d *DebugContext) DeleteTexture(arg0 uint32) {
	d.Context.DeleteTexture(arg0)
	fmt.Fprintln(os.Stderr, "DeleteTexture")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at DeleteTexture", e))
	}
}

func (d *DebugContext) DeleteVertexArray(arg0 uint32) {
	d.Context.DeleteVertexArray(arg0)
	fmt.Fprintln(os.Stderr, "DeleteVertexArray")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at DeleteVertexArray", e))
	}
}

func (d *DebugContext) Disable(arg0 uint32) {
	d.Context.Disable(arg0)
	fmt.Fprintln(os.Stderr, "Disable")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Disable", e))
	}
}

func (d *DebugContext) DisableVertexAttribArray(arg0 uint32) {
	d.Context.DisableVertexAttribArray(arg0)
	fmt.Fprintln(os.Stderr, "DisableVertexAttribArray")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at DisableVertexAttribArray", e))
	}
}

func (d *DebugContext) DrawElements(arg0 uint32, arg1 int32, arg2 uint32, arg3 int) {
	d.Context.DrawElements(arg0, arg1, arg2, arg3)
	fmt.Fprintln(os.Stderr, "DrawElements")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at DrawElements", e))
	}
}

func (d *DebugContext) Enable(arg0 uint32) {
	d.Context.Enable(arg0)
	fmt.Fprintln(os.Stderr, "Enable")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Enable", e))
	}
}

func (d *DebugContext) EnableVertexAttribArray(arg0 uint32) {
	d.Context.EnableVertexAttribArray(arg0)
	fmt.Fprintln(os.Stderr, "EnableVertexAttribArray")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at EnableVertexAttribArray", e))
	}
}

func (d *DebugContext) Flush() {
	d.Context.Flush()
	fmt.Fprintln(os.Stderr, "Flush")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Flush", e))
	}
}

func (d *DebugContext) FramebufferRenderbuffer(arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32) {
	d.Context.FramebufferRenderbuffer(arg0, arg1, arg2, arg3)
	fmt.Fprintln(os.Stderr, "FramebufferRenderbuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at FramebufferRenderbuffer", e))
	}
}

func (d *DebugContext) FramebufferTexture2D(arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 int32) {
	d.Context.FramebufferTexture2D(arg0, arg1, arg2, arg3, arg4)
	fmt.Fprintln(os.Stderr, "FramebufferTexture2D")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at FramebufferTexture2D", e))
	}
}

func (d *DebugContext) GetError() uint32 {
	out0 := d.Context.GetError()
	fmt.Fprintln(os.Stderr, "GetError")
	return out0
}

func (d *DebugContext) GetInteger(arg0 uint32) int {
	out0 := d.Context.GetInteger(arg0)
	fmt.Fprintln(os.Stderr, "GetInteger")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at GetInteger", e))
	}
	return out0
}

func (d *DebugContext) GetProgramInfoLog(arg0 uint32) string {
	out0 := d.Context.GetProgramInfoLog(arg0)
	fmt.Fprintln(os.Stderr, "GetProgramInfoLog")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at GetProgramInfoLog", e))
	}
	return out0
}

func (d *DebugContext) GetProgrami(arg0 uint32, arg1 uint32) int {
	out0 := d.Context.GetProgrami(arg0, arg1)
	fmt.Fprintln(os.Stderr, "GetProgrami")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at GetProgrami", e))
	}
	return out0
}

func (d *DebugContext) GetShaderInfoLog(arg0 uint32) string {
	out0 := d.Context.GetShaderInfoLog(arg0)
	fmt.Fprintln(os.Stderr, "GetShaderInfoLog")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at GetShaderInfoLog", e))
	}
	return out0
}

func (d *DebugContext) GetShaderi(arg0 uint32, arg1 uint32) int {
	out0 := d.Context.GetShaderi(arg0, arg1)
	fmt.Fprintln(os.Stderr, "GetShaderi")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at GetShaderi", e))
	}
	return out0
}

func (d *DebugContext) GetUniformLocation(arg0 uint32, arg1 string) int32 {
	out0 := d.Context.GetUniformLocation(arg0, arg1)
	fmt.Fprintln(os.Stderr, "GetUniformLocation")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at GetUniformLocation", e))
	}
	return out0
}

func (d *DebugContext) IsES() bool {
	out0 := d.Context.IsES()
	return out0
}

func (d *DebugContext) IsFramebuffer(arg0 uint32) bool {
	out0 := d.Context.IsFramebuffer(arg0)
	fmt.Fprintln(os.Stderr, "IsFramebuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at IsFramebuffer", e))
	}
	return out0
}

func (d *DebugContext) IsProgram(arg0 uint32) bool {
	out0 := d.Context.IsProgram(arg0)
	fmt.Fprintln(os.Stderr, "IsProgram")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at IsProgram", e))
	}
	return out0
}

func (d *DebugContext) IsRenderbuffer(arg0 uint32) bool {
	out0 := d.Context.IsRenderbuffer(arg0)
	fmt.Fprintln(os.Stderr, "IsRenderbuffer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at IsRenderbuffer", e))
	}
	return out0
}

func (d *DebugContext) IsTexture(arg0 uint32) bool {
	out0 := d.Context.IsTexture(arg0)
	fmt.Fprintln(os.Stderr, "IsTexture")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at IsTexture", e))
	}
	return out0
}

func (d *DebugContext) LinkProgram(arg0 uint32) {
	d.Context.LinkProgram(arg0)
	fmt.Fprintln(os.Stderr, "LinkProgram")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at LinkProgram", e))
	}
}

func (d *DebugContext) LoadFunctions() error {
	out0 := d.Context.LoadFunctions()
	return out0
}

func (d *DebugContext) PixelStorei(arg0 uint32, arg1 int32) {
	d.Context.PixelStorei(arg0, arg1)
	fmt.Fprintln(os.Stderr, "PixelStorei")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at PixelStorei", e))
	}
}

func (d *DebugContext) ReadPixels(arg0 []uint8, arg1 int32, arg2 int32, arg3 int32, arg4 int32, arg5 uint32, arg6 uint32) {
	d.Context.ReadPixels(arg0, arg1, arg2, arg3, arg4, arg5, arg6)
	fmt.Fprintln(os.Stderr, "ReadPixels")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at ReadPixels", e))
	}
}

func (d *DebugContext) RenderbufferStorage(arg0 uint32, arg1 uint32, arg2 int32, arg3 int32) {
	d.Context.RenderbufferStorage(arg0, arg1, arg2, arg3)
	fmt.Fprintln(os.Stderr, "RenderbufferStorage")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at RenderbufferStorage", e))
	}
}

func (d *DebugContext) Scissor(arg0 int32, arg1 int32, arg2 int32, arg3 int32) {
	d.Context.Scissor(arg0, arg1, arg2, arg3)
	fmt.Fprintln(os.Stderr, "Scissor")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Scissor", e))
	}
}

func (d *DebugContext) ShaderSource(arg0 uint32, arg1 string) {
	d.Context.ShaderSource(arg0, arg1)
	fmt.Fprintln(os.Stderr, "ShaderSource")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at ShaderSource", e))
	}
}

func (d *DebugContext) StencilFunc(arg0 uint32, arg1 int32, arg2 uint32) {
	d.Context.StencilFunc(arg0, arg1, arg2)
	fmt.Fprintln(os.Stderr, "StencilFunc")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at StencilFunc", e))
	}
}

func (d *DebugContext) StencilOp(arg0 uint32, arg1 uint32, arg2 uint32) {
	d.Context.StencilOp(arg0, arg1, arg2)
	fmt.Fprintln(os.Stderr, "StencilOp")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at StencilOp", e))
	}
}

func (d *DebugContext) TexImage2D(arg0 uint32, arg1 int32, arg2 int32, arg3 int32, arg4 int32, arg5 uint32, arg6 uint32, arg7 []uint8) {
	d.Context.TexImage2D(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
	fmt.Fprintln(os.Stderr, "TexImage2D")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at TexImage2D", e))
	}
}

func (d *DebugContext) TexParameteri(arg0 uint32, arg1 uint32, arg2 int32) {
	d.Context.TexParameteri(arg0, arg1, arg2)
	fmt.Fprintln(os.Stderr, "TexParameteri")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at TexParameteri", e))
	}
}

func (d *DebugContext) TexSubImage2D(arg0 uint32, arg1 int32, arg2 int32, arg3 int32, arg4 int32, arg5 int32, arg6 uint32, arg7 uint32, arg8 []uint8) {
	d.Context.TexSubImage2D(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8)
	fmt.Fprintln(os.Stderr, "TexSubImage2D")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at TexSubImage2D", e))
	}
}

func (d *DebugContext) Uniform1fv(arg0 int32, arg1 []float32) {
	d.Context.Uniform1fv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "Uniform1fv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Uniform1fv", e))
	}
}

func (d *DebugContext) Uniform1i(arg0 int32, arg1 int32) {
	d.Context.Uniform1i(arg0, arg1)
	fmt.Fprintln(os.Stderr, "Uniform1i")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Uniform1i", e))
	}
}

func (d *DebugContext) Uniform1iv(arg0 int32, arg1 []int32) {
	d.Context.Uniform1iv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "Uniform1iv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Uniform1iv", e))
	}
}

func (d *DebugContext) Uniform2fv(arg0 int32, arg1 []float32) {
	d.Context.Uniform2fv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "Uniform2fv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Uniform2fv", e))
	}
}

func (d *DebugContext) Uniform2iv(arg0 int32, arg1 []int32) {
	d.Context.Uniform2iv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "Uniform2iv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Uniform2iv", e))
	}
}

func (d *DebugContext) Uniform3fv(arg0 int32, arg1 []float32) {
	d.Context.Uniform3fv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "Uniform3fv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Uniform3fv", e))
	}
}

func (d *DebugContext) Uniform3iv(arg0 int32, arg1 []int32) {
	d.Context.Uniform3iv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "Uniform3iv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Uniform3iv", e))
	}
}

func (d *DebugContext) Uniform4fv(arg0 int32, arg1 []float32) {
	d.Context.Uniform4fv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "Uniform4fv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Uniform4fv", e))
	}
}

func (d *DebugContext) Uniform4iv(arg0 int32, arg1 []int32) {
	d.Context.Uniform4iv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "Uniform4iv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Uniform4iv", e))
	}
}

func (d *DebugContext) UniformMatrix2fv(arg0 int32, arg1 []float32) {
	d.Context.UniformMatrix2fv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "UniformMatrix2fv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at UniformMatrix2fv", e))
	}
}

func (d *DebugContext) UniformMatrix3fv(arg0 int32, arg1 []float32) {
	d.Context.UniformMatrix3fv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "UniformMatrix3fv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at UniformMatrix3fv", e))
	}
}

func (d *DebugContext) UniformMatrix4fv(arg0 int32, arg1 []float32) {
	d.Context.UniformMatrix4fv(arg0, arg1)
	fmt.Fprintln(os.Stderr, "UniformMatrix4fv")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at UniformMatrix4fv", e))
	}
}

func (d *DebugContext) UseProgram(arg0 uint32) {
	d.Context.UseProgram(arg0)
	fmt.Fprintln(os.Stderr, "UseProgram")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at UseProgram", e))
	}
}

func (d *DebugContext) VertexAttribPointer(arg0 uint32, arg1 int32, arg2 uint32, arg3 bool, arg4 int32, arg5 int) {
	d.Context.VertexAttribPointer(arg0, arg1, arg2, arg3, arg4, arg5)
	fmt.Fprintln(os.Stderr, "VertexAttribPointer")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at VertexAttribPointer", e))
	}
}

func (d *DebugContext) Viewport(arg0 int32, arg1 int32, arg2 int32, arg3 int32) {
	d.Context.Viewport(arg0, arg1, arg2, arg3)
	fmt.Fprintln(os.Stderr, "Viewport")
	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %d at Viewport", e))
	}
}
