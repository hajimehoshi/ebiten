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
	"strconv"
)

var canvas js.Object
var context *opengl.Context

func shown() bool {
	return !js.Global.Get("document").Get("hidden").Bool()
}

func Use(f func(*opengl.Context)) {
	f(context)
}

func vsync() {
	ch := make(chan struct{})
	js.Global.Get("window").Call("requestAnimationFrame", func() {
		close(ch)
	})
	<-ch
}

func DoEvents() {
	vsync()
	for !shown() {
		vsync()
	}
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
	// TODO: Implement this with node-webgl mainly for testing.

	doc := js.Global.Get("document")
	if doc.Get("body") == nil {
		ch := make(chan struct{})
		js.Global.Get("window").Call("addEventListener", "load", func() {
			close(ch)
		})
		<-ch
	}
	doc.Set("onkeydown", func(e js.Object) bool {
		code := e.Get("keyCode").Int()
		// Backspace
		if code == 8 {
			return false
		}
		// Functions
		if 112 <= code && code <= 123 {
			return false
		}
		// Alt and arrows
		if code == 37 && code == 39 {
			// Don't need to check Alt.
			return false
		}
		return true
	})

	canvas = doc.Call("createElement", "canvas")
	canvas.Set("width", 16)
	canvas.Set("height", 16)
	doc.Get("body").Call("appendChild", canvas)

	htmlStyle := doc.Get("documentElement").Get("style")
	htmlStyle.Set("height", "100%")
	htmlStyle.Set("margin", "0")
	htmlStyle.Set("padding", "0")

	bodyStyle := doc.Get("body").Get("style")
	bodyStyle.Set("backgroundColor", "#000")
	bodyStyle.Set("position", "relative")
	bodyStyle.Set("height", "100%")
	bodyStyle.Set("margin", "0")
	bodyStyle.Set("padding", "0")
	doc.Get("body").Set("onclick", func() {
		canvas.Call("focus")
	})

	canvasStyle := canvas.Get("style")
	canvasStyle.Set("position", "absolute")

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

	canvas.Set("onkeydown", func(e js.Object) bool {
		code := e.Get("keyCode").Int()
		currentInput.keyDown(code)
		return false
	})
	canvas.Set("onkeyup", func(e js.Object) bool {
		code := e.Get("keyCode").Int()
		currentInput.keyUp(code)
		return false
	})
	canvas.Set("onmousedown", func(e js.Object) bool {
		button := e.Get("button").Int()
		currentInput.mouseDown(button)
		return false
	})
	canvas.Set("onmouseup", func(e js.Object) bool {
		button := e.Get("button").Int()
		currentInput.mouseUp(button)
		return false
	})
	canvas.Set("oncontextmenu", func(e js.Object) bool {
		return false
	})
}

func devicePixelRatio() int {
	// TODO: What if ratio is not an integer but a float?
	ratio := js.Global.Get("window").Get("devicePixelRatio").Int()
	if ratio == 0 {
		ratio = 1
	}
	return ratio
}

func Start(width, height, scale int, title string) (actualScale int, err error) {
	doc := js.Global.Get("document")
	doc.Set("title", title)
	// for retina
	actualScale = scale * devicePixelRatio()
	canvas.Set("width", width*actualScale)
	canvas.Set("height", height*actualScale)
	canvasStyle := canvas.Get("style")

	cssWidth := width * scale
	cssHeight := height * scale
	canvasStyle.Set("width", strconv.Itoa(cssWidth)+"px")
	canvasStyle.Set("height", strconv.Itoa(cssHeight)+"px")
	canvasStyle.Set("left", "calc(50% - "+strconv.Itoa(cssWidth/2)+"px)")
	canvasStyle.Set("top", "calc(50% - "+strconv.Itoa(cssHeight/2)+"px)")

	canvas.Set("onmousemove", func(e js.Object) {
		rect := canvas.Call("getBoundingClientRect")
		x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
		x -= rect.Get("left").Int()
		y -= rect.Get("top").Int()
		currentInput.mouseMove(x/scale, y/scale)
	})
	canvas.Call("focus")

	return actualScale, nil
}
