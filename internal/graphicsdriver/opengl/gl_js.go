// Copyright 2021 The Ebiten Authors
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

package opengl

import (
	"syscall/js"
)

type gl struct {
	activeTexture            js.Value
	attachShader             js.Value
	bindAttribLocation       js.Value
	bindBuffer               js.Value
	bindFramebuffer          js.Value
	bindRenderbuffer         js.Value
	bindTexture              js.Value
	blendFunc                js.Value
	bufferData               js.Value
	bufferSubData            js.Value
	checkFramebufferStatus   js.Value
	clear                    js.Value
	colorMask                js.Value
	compileShader            js.Value
	createBuffer             js.Value
	createFramebuffer        js.Value
	createProgram            js.Value
	createRenderbuffer       js.Value
	createShader             js.Value
	createTexture            js.Value
	deleteBuffer             js.Value
	deleteFramebuffer        js.Value
	deleteProgram            js.Value
	deleteRenderbuffer       js.Value
	deleteShader             js.Value
	deleteTexture            js.Value
	disable                  js.Value
	disableVertexAttribArray js.Value
	drawElements             js.Value
	enable                   js.Value
	enableVertexAttribArray  js.Value
	framebufferRenderbuffer  js.Value
	framebufferTexture2D     js.Value
	flush                    js.Value
	getBufferSubData         js.Value
	getExtension             js.Value
	getParameter             js.Value
	getProgramInfoLog        js.Value
	getProgramParameter      js.Value
	getShaderInfoLog         js.Value
	getShaderParameter       js.Value
	getShaderPrecisionFormat js.Value
	getUniformLocation       js.Value
	isContextLost            js.Value
	isFramebuffer            js.Value
	isProgram                js.Value
	isRenderbuffer           js.Value
	isTexture                js.Value
	linkProgram              js.Value
	pixelStorei              js.Value
	readPixels               js.Value
	renderbufferStorage      js.Value
	scissor                  js.Value
	shaderSource             js.Value
	stencilFunc              js.Value
	stencilMask              js.Value
	stencilOp                js.Value
	texImage2D               js.Value
	texSubImage2D            js.Value
	texParameteri            js.Value
	uniform1f                js.Value
	uniform1fv               js.Value
	uniform2fv               js.Value
	uniform3fv               js.Value
	uniform4fv               js.Value
	uniform1i                js.Value
	uniformMatrix2fv         js.Value
	uniformMatrix3fv         js.Value
	uniformMatrix4fv         js.Value
	useProgram               js.Value
	vertexAttribPointer      js.Value
	viewport                 js.Value
}

func (c *context) newGL(v js.Value) *gl {
	// Passing a Go string to the JS world is expensive. This causes conversion to UTF-16 (#1438).
	// In order to reduce the cost when calling functions, create the function objects by bind and use them.
	g := &gl{
		activeTexture:            v.Get("activeTexture").Call("bind", v),
		attachShader:             v.Get("attachShader").Call("bind", v),
		bindAttribLocation:       v.Get("bindAttribLocation").Call("bind", v),
		bindBuffer:               v.Get("bindBuffer").Call("bind", v),
		bindFramebuffer:          v.Get("bindFramebuffer").Call("bind", v),
		bindRenderbuffer:         v.Get("bindRenderbuffer").Call("bind", v),
		bindTexture:              v.Get("bindTexture").Call("bind", v),
		blendFunc:                v.Get("blendFunc").Call("bind", v),
		bufferData:               v.Get("bufferData").Call("bind", v),
		bufferSubData:            v.Get("bufferSubData").Call("bind", v),
		checkFramebufferStatus:   v.Get("checkFramebufferStatus").Call("bind", v),
		clear:                    v.Get("clear").Call("bind", v),
		colorMask:                v.Get("colorMask").Call("bind", v),
		compileShader:            v.Get("compileShader").Call("bind", v),
		createBuffer:             v.Get("createBuffer").Call("bind", v),
		createFramebuffer:        v.Get("createFramebuffer").Call("bind", v),
		createProgram:            v.Get("createProgram").Call("bind", v),
		createRenderbuffer:       v.Get("createRenderbuffer").Call("bind", v),
		createShader:             v.Get("createShader").Call("bind", v),
		createTexture:            v.Get("createTexture").Call("bind", v),
		deleteBuffer:             v.Get("deleteBuffer").Call("bind", v),
		deleteFramebuffer:        v.Get("deleteFramebuffer").Call("bind", v),
		deleteProgram:            v.Get("deleteProgram").Call("bind", v),
		deleteRenderbuffer:       v.Get("deleteRenderbuffer").Call("bind", v),
		deleteShader:             v.Get("deleteShader").Call("bind", v),
		deleteTexture:            v.Get("deleteTexture").Call("bind", v),
		disable:                  v.Get("disable").Call("bind", v),
		disableVertexAttribArray: v.Get("disableVertexAttribArray").Call("bind", v),
		drawElements:             v.Get("drawElements").Call("bind", v),
		enable:                   v.Get("enable").Call("bind", v),
		enableVertexAttribArray:  v.Get("enableVertexAttribArray").Call("bind", v),
		framebufferRenderbuffer:  v.Get("framebufferRenderbuffer").Call("bind", v),
		framebufferTexture2D:     v.Get("framebufferTexture2D").Call("bind", v),
		flush:                    v.Get("flush").Call("bind", v),
		getParameter:             v.Get("getParameter").Call("bind", v),
		getProgramInfoLog:        v.Get("getProgramInfoLog").Call("bind", v),
		getProgramParameter:      v.Get("getProgramParameter").Call("bind", v),
		getShaderInfoLog:         v.Get("getShaderInfoLog").Call("bind", v),
		getShaderParameter:       v.Get("getShaderParameter").Call("bind", v),
		getShaderPrecisionFormat: v.Get("getShaderPrecisionFormat").Call("bind", v),
		getUniformLocation:       v.Get("getUniformLocation").Call("bind", v),
		isContextLost:            v.Get("isContextLost").Call("bind", v),
		isFramebuffer:            v.Get("isFramebuffer").Call("bind", v),
		isProgram:                v.Get("isProgram").Call("bind", v),
		isRenderbuffer:           v.Get("isRenderbuffer").Call("bind", v),
		isTexture:                v.Get("isTexture").Call("bind", v),
		linkProgram:              v.Get("linkProgram").Call("bind", v),
		pixelStorei:              v.Get("pixelStorei").Call("bind", v),
		readPixels:               v.Get("readPixels").Call("bind", v),
		renderbufferStorage:      v.Get("renderbufferStorage").Call("bind", v),
		scissor:                  v.Get("scissor").Call("bind", v),
		shaderSource:             v.Get("shaderSource").Call("bind", v),
		stencilFunc:              v.Get("stencilFunc").Call("bind", v),
		stencilMask:              v.Get("stencilMask").Call("bind", v),
		stencilOp:                v.Get("stencilOp").Call("bind", v),
		texImage2D:               v.Get("texImage2D").Call("bind", v),
		texSubImage2D:            v.Get("texSubImage2D").Call("bind", v),
		texParameteri:            v.Get("texParameteri").Call("bind", v),
		uniform1f:                v.Get("uniform1f").Call("bind", v),
		uniform1fv:               v.Get("uniform1fv").Call("bind", v),
		uniform2fv:               v.Get("uniform2fv").Call("bind", v),
		uniform3fv:               v.Get("uniform3fv").Call("bind", v),
		uniform4fv:               v.Get("uniform4fv").Call("bind", v),
		uniform1i:                v.Get("uniform1i").Call("bind", v),
		uniformMatrix2fv:         v.Get("uniformMatrix2fv").Call("bind", v),
		uniformMatrix3fv:         v.Get("uniformMatrix3fv").Call("bind", v),
		uniformMatrix4fv:         v.Get("uniformMatrix4fv").Call("bind", v),
		useProgram:               v.Get("useProgram").Call("bind", v),
		vertexAttribPointer:      v.Get("vertexAttribPointer").Call("bind", v),
		viewport:                 v.Get("viewport").Call("bind", v),
	}
	if c.usesWebGL2() {
		g.getExtension = v.Get("getBufferSubData").Call("bind", v)
	} else {
		g.getExtension = v.Get("getExtension").Call("bind", v)
	}
	return g
}
