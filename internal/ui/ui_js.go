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
	"strings"
	"unicode"

	"github.com/gopherjs/gopherjs/js"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

var canvas *js.Object

type userInterface struct {
	width                int
	height               int
	scale                float64
	fullscreen           bool
	runnableInBackground bool

	deviceScale float64
	sizeChanged bool
	windowFocus bool
}

var currentUI = &userInterface{
	sizeChanged: true,
	windowFocus: true,
}

// NOTE: This returns true even when the browser is not active.
func shown() bool {
	return !js.Global.Get("document").Get("hidden").Bool()
}

func SetScreenSize(width, height int) bool {
	return currentUI.setScreenSize(currentUI.width, currentUI.height, currentUI.scale, currentUI.fullscreen)
}

func SetScreenScale(scale float64) bool {
	return currentUI.setScreenSize(currentUI.width, currentUI.height, scale, currentUI.fullscreen)
}

func ScreenScale() float64 {
	return currentUI.scale
}

func SetFullscreen(fullscreen bool) {
	currentUI.setScreenSize(currentUI.width, currentUI.height, currentUI.scale, fullscreen)
}

func IsFullscreen() bool {
	return currentUI.fullscreen
}

func SetRunnableInBackground(runnableInBackground bool) {
	currentUI.runnableInBackground = runnableInBackground
}

func IsRunnableInBackground() bool {
	return currentUI.runnableInBackground
}

func ScreenOffset() (float64, float64) {
	return 0, 0
}

func adjustCursorPosition(x, y int) (int, int) {
	return x, y
}

func SetCursorVisibility(visibility bool) {
	if visibility {
		canvas.Get("style").Set("cursor", "auto")
	} else {
		canvas.Get("style").Set("cursor", "none")
	}
}

func (u *userInterface) getScale() float64 {
	if !u.fullscreen {
		return u.scale
	}
	doc := js.Global.Get("document")
	body := doc.Get("body")
	bw := body.Get("clientWidth").Float()
	bh := body.Get("clientHeight").Float()
	sw := bw / float64(u.width)
	sh := bh / float64(u.height)
	if sw > sh {
		return sh
	}
	return sw
}

func (u *userInterface) actualScreenScale() float64 {
	return u.getScale() * u.deviceScale
}

func (u *userInterface) update(g GraphicsContext) error {
	if !u.runnableInBackground && !u.windowFocus {
		return nil
	}
	if opengl.GetContext().IsContextLost() {
		opengl.GetContext().RestoreContext()
		g.Invalidate()
	}
	currentInput.updateGamepads()
	if u.sizeChanged {
		u.sizeChanged = false
		g.SetSize(u.width, u.height, u.actualScreenScale())
		return nil
	}
	if err := g.Update(); err != nil {
		return err
	}
	currentInput.runebuffer = currentInput.runebuffer[:0]
	return nil
}

func (u *userInterface) loop(g GraphicsContext) error {
	ch := make(chan error)
	var f func()
	f = func() {
		go func() {
			if err := u.update(g); err != nil {
				ch <- err
				close(ch)
				return
			}
			js.Global.Get("window").Call("requestAnimationFrame", f)
		}()
	}
	f()
	return <-ch
}

func touchEventToTouches(e *js.Object) []touch {
	scale := currentUI.getScale()
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

func init() {
	if err := initialize(); err != nil {
		panic(err)
	}
}

func initialize() error {
	// Do nothing in node.js.
	if js.Global.Get("require") != js.Undefined {
		return nil
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
	window.Call("addEventListener", "focus", func() {
		currentUI.windowFocus = true
	})
	window.Call("addEventListener", "blur", func() {
		currentUI.windowFocus = false
	})
	window.Call("addEventListener", "resize", func() {
		currentUI.updateScreenSize()
	})

	// Adjust the initial scale to 1.
	// https://developer.mozilla.org/en/docs/Mozilla/Mobile/Viewport_meta_tag
	meta := doc.Call("createElement", "meta")
	meta.Set("name", "viewport")
	meta.Set("content", "width=device-width, initial-scale=1")
	doc.Get("head").Call("appendChild", meta)

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
		if e.Get("code") == js.Undefined {
			// Assume that UA is Safari.
			code := e.Get("keyCode").Int()
			currentInput.keyDownSafari(code)
			return
		}
		code := e.Get("code").String()
		currentInput.keyDown(code)
	})
	canvas.Call("addEventListener", "keypress", func(e *js.Object) {
		e.Call("preventDefault")
		if currentInput.runebuffer == nil {
			currentInput.runebuffer = make([]rune, 0, 1024)
		}
		if r := rune(e.Get("charCode").Int()); unicode.IsPrint(r) {
			currentInput.runebuffer = append(currentInput.runebuffer, r)
		}
	})
	canvas.Call("addEventListener", "keyup", func(e *js.Object) {
		e.Call("preventDefault")
		if e.Get("code") == js.Undefined {
			// Assume that UA is Safari.
			code := e.Get("keyCode").Int()
			currentInput.keyUpSafari(code)
		}
		code := e.Get("code").String()
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

	canvas.Call("addEventListener", "webglcontextlost", func(e *js.Object) {
		e.Call("preventDefault")
	})
	canvas.Call("addEventListener", "webglcontextrestored", func(e *js.Object) {
		// Do nothing.
	})

	return nil
}

func setMouseCursorFromEvent(e *js.Object) {
	scale := currentUI.getScale()
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

func RunMainThreadLoop(ch <-chan error) error {
	return <-ch
}

func Run(width, height int, scale float64, title string, g GraphicsContext) error {
	u := currentUI
	doc := js.Global.Get("document")
	doc.Set("title", title)
	u.setScreenSize(width, height, scale, u.fullscreen)
	canvas.Call("focus")
	if err := opengl.Init(); err != nil {
		return err
	}
	return u.loop(g)
}

func isSafari() bool {
	ua := js.Global.Get("navigator").Get("userAgent").String()
	if !strings.Contains(ua, "Safari") {
		return false
	}
	if strings.Contains(ua, "Chrome") {
		return false
	}
	return true
}

func (u *userInterface) setScreenSize(width, height int, scale float64, fullscreen bool) bool {
	if u.width == width && u.height == height &&
		u.scale == scale && fullscreen == u.fullscreen {
		return false
	}
	u.width = width
	u.height = height
	u.scale = scale
	u.fullscreen = fullscreen
	u.updateScreenSize()
	return true
}

func (u *userInterface) updateScreenSize() {
	// CSS imageRendering seems useful to enlarge the screen,
	// but doesn't work in some cases (#306):
	// * Chrome just after restoring the lost context
	// * Safari
	// Let's use the pixel ratio as it is here.
	u.deviceScale = devicePixelRatio()

	canvas.Set("width", int(float64(u.width)*u.actualScreenScale()))
	canvas.Set("height", int(float64(u.height)*u.actualScreenScale()))
	canvasStyle := canvas.Get("style")

	s := u.getScale()
	cssWidth := int(float64(u.width) * s)
	cssHeight := int(float64(u.height) * s)
	canvasStyle.Set("width", strconv.Itoa(cssWidth)+"px")
	canvasStyle.Set("height", strconv.Itoa(cssHeight)+"px")
	// CSS calc requires space chars.
	canvasStyle.Set("left", "calc((100% - "+strconv.Itoa(cssWidth)+"px) / 2)")
	canvasStyle.Set("top", "calc((100% - "+strconv.Itoa(cssHeight)+"px) / 2)")

	u.sizeChanged = true
}
