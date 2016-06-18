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
	"strconv"

	"github.com/gopherjs/gopherjs/js"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

var canvas *js.Object

type userInterface struct {
	scale       float64
	deviceScale float64
	sizeChanged bool
}

var currentUI = &userInterface{
	sizeChanged: true,
}

func CurrentUI() UserInterface {
	return currentUI
}

// NOTE: This returns true even when the browser is not active.
func shown() bool {
	return !js.Global.Get("document").Get("hidden").Bool()
}

func vsync() {
	ch := make(chan struct{})
	n := 1
	var l func()
	l = func() {
		if 0 < n {
			n--
			// TODO: In iOS8, this is called at every 1/30[sec] frame.
			// Can we use DOMHighResTimeStamp?
			js.Global.Get("window").Call("requestAnimationFrame", l)
			return
		}
		close(ch)
	}
	l()
	<-ch
}

func (u *userInterface) SetScreenSize(width, height int) bool {
	return u.setScreenSize(width, height, u.scale)
}

func (u *userInterface) SetScreenScale(scale float64) bool {
	width, height := u.size()
	return u.setScreenSize(width, height, scale)
}

func (u *userInterface) ScreenScale() float64 {
	return u.scale
}

func (u *userInterface) ActualScreenScale() float64 {
	return u.scale * u.deviceScale
}

func (u *userInterface) Update() (interface{}, error) {
	currentInput.updateGamepads()
	if u.sizeChanged {
		u.sizeChanged = false
		w, h := u.size()
		e := ScreenSizeEvent{
			Width:       w,
			Height:      h,
			ActualScale: u.ActualScreenScale(),
		}
		return e, nil
	}
	// Dummy channel
	ch := make(chan struct{}, 1)
	return RenderEvent{ch}, nil
}

func (u *userInterface) Terminate() error {
	// Do nothing.
	return nil
}

func (u *userInterface) SwapBuffers() error {
	vsync()
	for !shown() {
		vsync()
	}
	return nil
}

func (u *userInterface) FinishRendering() error {
	return nil
}

func touchEventToTouches(e *js.Object) []touch {
	scale := currentUI.scale
	j := e.Get("targetTouches")
	rect := canvas.Call("getBoundingClientRect")
	left, top := rect.Get("left").Int(), rect.Get("top").Int()
	t := make([]touch, j.Get("length").Int())
	for i := 0; i < len(t); i++ {
		jj := j.Call("item", i)
		t[i].id = jj.Get("identifier").Int()
		t[i].x = int(float64(jj.Get("clientX").Int()-left) / scale)
		t[i].y = int(float64(jj.Get("clientY").Int()-top) / scale)
	}
	return t
}

func initialize() (*opengl.Context, error) {
	// Do nothing in node.js.
	if js.Global.Get("require") != js.Undefined {
		c, err := opengl.NewContext()
		if err != nil {
			return nil, err
		}
		return c, nil
	}

	doc := js.Global.Get("document")
	window := js.Global.Get("window")
	if doc.Get("body") == nil {
		ch := make(chan struct{})
		window.Call("addEventListener", "load", func() {
			close(ch)
		})
		<-ch
	}

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
	// TODO: This is OK as long as the game is in an independent iframe.
	// What if the canvas is embedded in a HTML directly?
	doc.Get("body").Call("addEventListener", "click", func() {
		canvas.Call("focus")
	})

	canvasStyle := canvas.Get("style")
	canvasStyle.Set("position", "absolute")

	// Make the canvas focusable.
	canvas.Call("setAttribute", "tabindex", 1)
	canvas.Get("style").Set("outline", "none")

	// Keyboard
	canvas.Call("addEventListener", "keydown", func(e *js.Object) {
		e.Call("preventDefault")
		code := e.Get("keyCode").Int()
		currentInput.keyDown(code)
	})
	canvas.Call("addEventListener", "keyup", func(e *js.Object) {
		e.Call("preventDefault")
		code := e.Get("keyCode").Int()
		currentInput.keyUp(code)
	})

	// Mouse
	canvas.Call("addEventListener", "mousedown", func(e *js.Object) {
		e.Call("preventDefault")
		button := e.Get("button").Int()
		currentInput.mouseDown(button)
		setMouseCursorFromEvent(e)
	})
	canvas.Call("addEventListener", "mouseup", func(e *js.Object) {
		e.Call("preventDefault")
		button := e.Get("button").Int()
		currentInput.mouseUp(button)
		setMouseCursorFromEvent(e)
	})
	canvas.Call("addEventListener", "mousemove", func(e *js.Object) {
		e.Call("preventDefault")
		setMouseCursorFromEvent(e)
	})
	canvas.Call("addEventListener", "contextmenu", func(e *js.Object) {
		e.Call("preventDefault")
	})

	// Touch
	canvas.Call("addEventListener", "touchstart", func(e *js.Object) {
		e.Call("preventDefault")
		currentInput.updateTouches(touchEventToTouches(e))
	})
	canvas.Call("addEventListener", "touchend", func(e *js.Object) {
		e.Call("preventDefault")
		currentInput.updateTouches(touchEventToTouches(e))
	})
	canvas.Call("addEventListener", "touchmove", func(e *js.Object) {
		e.Call("preventDefault")
		currentInput.updateTouches(touchEventToTouches(e))
	})

	// Gamepad
	window.Call("addEventListener", "gamepadconnected", func(e *js.Object) {
		// Do nothing.
	})

	c, err := opengl.NewContext()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func setMouseCursorFromEvent(e *js.Object) {
	scale := currentUI.scale
	rect := canvas.Call("getBoundingClientRect")
	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
	x -= rect.Get("left").Int()
	y -= rect.Get("top").Int()
	currentInput.setMouseCursor(int(float64(x)/scale), int(float64(y)/scale))
}

func devicePixelRatio() float64 {
	ratio := js.Global.Get("window").Get("devicePixelRatio").Float()
	if ratio == 0 {
		ratio = 1
	}
	return ratio
}

func Main() error {
	// Do nothing
	return nil
}

func (u *userInterface) Start(width, height int, scale float64, title string) error {
	doc := js.Global.Get("document")
	doc.Set("title", title)
	u.setScreenSize(width, height, scale)
	canvas.Call("focus")
	return nil
}

func (u *userInterface) size() (width, height int) {
	a := int(u.ActualScreenScale())
	if a == 0 {
		// a == 0 only on the initial state.
		return
	}
	width = canvas.Get("width").Int() / a
	height = canvas.Get("height").Int() / a
	return
}

func (u *userInterface) setScreenSize(width, height int, scale float64) bool {
	w, h := u.size()
	s := u.scale
	if w == width && h == height && s == scale {
		return false
	}
	u.scale = scale
	u.deviceScale = devicePixelRatio()
	canvas.Set("width", int(float64(width)*u.ActualScreenScale()))
	canvas.Set("height", int(float64(height)*u.ActualScreenScale()))
	canvasStyle := canvas.Get("style")

	cssWidth := int(float64(width) * scale)
	cssHeight := int(float64(height) * scale)
	canvasStyle.Set("width", strconv.Itoa(cssWidth)+"px")
	canvasStyle.Set("height", strconv.Itoa(cssHeight)+"px")
	// CSS calc requires space chars.
	canvasStyle.Set("left", "calc((100% - "+strconv.Itoa(cssWidth)+"px) / 2)")
	canvasStyle.Set("top", "calc((100% - "+strconv.Itoa(cssHeight)+"px) / 2)")
	u.sizeChanged = true
	return true
}
