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
	canvas = js.Global.Get("Canvas").New()
	js.Global.Get("document").Get("body").Call("appendChild", canvas)
	webglContext, err := webgl.NewContext(canvas, &webgl.ContextAttributes{
		Alpha:              true,
		PremultipliedAlpha: true,
	})
	if err != nil {
		panic(err)
	}
	context = opengl.NewContext(webglContext)
}

func Start(width, height, scale int, title string) (actualScale int, err error) {
	return scale, nil
}
