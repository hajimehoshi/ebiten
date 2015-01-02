// Copyright 2015 Hajime Hoshi
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

// +build js

package ui

import (
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/webgl"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"time"
)

var canvas js.Object
var context *opengl.Context

func Use(f func(*opengl.Context)) {
	f(context)
}

func DoEvents() {
	time.Sleep(1)
}

func Terminate() {
	// Do nothing.
}

func IsClosed() bool {
	return false
}

func SwapBuffers() {
	// Do nothing.
}

func init() {
	ch := make(chan struct{})
	js.Global.Get("window").Set("onload", func() {
		close(ch)
	})
	<-ch

	doc := js.Global.Get("document")
	canvas = doc.Call("createElement", "canvas")
	canvas.Set("width", 16)
	canvas.Set("height", 16)
	doc.Get("body").Call("appendChild", canvas)
	webglContext, err := webgl.NewContext(canvas, &webgl.ContextAttributes{
		Alpha:              true,
		PremultipliedAlpha: true,
	})
	if err != nil {
		panic(err)
	}
	context = opengl.NewContext(webglContext)

	// Make the canvas focusable.
	canvas.Call("setAttribute", "tabindex", 1)
	canvas.Get("style").Set("outline", "none")

	canvas.Set("onkeydown", func(e js.Object) {
		defer e.Call("preventDefault")
		code := e.Get("keyCode").Int()
		currentInput.keyDown(code)
	})
	canvas.Set("onkeyup", func(e js.Object) {
		defer e.Call("preventDefault")
		code := e.Get("keyCode").Int()
		currentInput.keyUp(code)
	})
}

func Start(width, height, scale int, title string) (actualScale int, err error) {
	canvas.Set("width", width*scale)
	canvas.Set("height", height*scale)
	return scale, nil
}
