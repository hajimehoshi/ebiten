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
	"image"
	"strconv"

	"github.com/gopherjs/gopherjs/js"

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/input"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/web"
)

var canvas *js.Object

type userInterface struct {
	width                int
	height               int
	scale                float64
	fullscreen           bool
	runnableInBackground bool

	sizeChanged bool
	windowFocus bool
}

var currentUI = &userInterface{
	sizeChanged: true,
	windowFocus: true,
}

func MonitorSize() (int, int) {
	w := js.Global.Get("window")
	return w.Get("innerWidth").Int(), w.Get("innerHeight").Int()
}

func SetScreenSize(width, height int) bool {
	return currentUI.setScreenSize(width, height, currentUI.scale, currentUI.fullscreen)
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

func ScreenPadding() (x0, y0, x1, y1 float64) {
	return 0, 0, 0, 0
}

func adjustPosition(x, y int) (int, int) {
	rect := canvas.Call("getBoundingClientRect")
	x -= rect.Get("left").Int()
	y -= rect.Get("top").Int()
	scale := currentUI.getScale()
	return int(float64(x) / scale), int(float64(y) / scale)
}

func AdjustedCursorPosition() (x, y int) {
	return adjustPosition(input.Get().CursorPosition())
}

func AdjustedTouches() []*input.Touch {
	ts := input.Get().Touches()
	adjusted := make([]*input.Touch, len(ts))
	for i, t := range ts {
		x, y := adjustPosition(t.Position())
		adjusted[i] = input.NewTouch(t.ID(), x, y)
	}
	return adjusted
}

func IsCursorVisible() bool {
	// The initial value is an empty string, so don't compare with "auto" here.
	return canvas.Get("style").Get("cursor").String() != "none"
}

func SetCursorVisible(visible bool) {
	if visible {
		canvas.Get("style").Set("cursor", "auto")
	} else {
		canvas.Get("style").Set("cursor", "none")
	}
}

func SetWindowTitle(title string) {
	doc := js.Global.Get("document")
	doc.Set("title", title)
}

func SetWindowIcon(iconImages []image.Image) {
	// Do nothing
}

func IsWindowDecorated() bool {
	return false
}

func SetWindowDecorated(decorated bool) {
	// Do nothing
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
	// CSS imageRendering property seems useful to enlarge the screen,
	// but doesn't work in some cases (#306):
	// * Chrome just after restoring the lost context
	// * Safari
	// Let's use the devicePixelRatio as it is here.
	return u.getScale() * devicescale.DeviceScale()
}

func (u *userInterface) updateGraphicsContext(g GraphicsContext) {
	if u.sizeChanged {
		u.sizeChanged = false
		g.SetSize(u.width, u.height, u.actualScreenScale())
	}
}

func (u *userInterface) update(g GraphicsContext) error {
	if !u.runnableInBackground && !u.windowFocus {
		return nil
	}
	if opengl.GetContext().IsContextLost() {
		opengl.GetContext().RestoreContext()
		g.Invalidate()

		// Need to return once to wait restored (#526)
		// TODO: Is it necessary to handle webglcontextrestored event?
		return nil
	}

	input.Get().UpdateGamepads()
	u.updateGraphicsContext(g)
	if err := g.Update(func() {
		input.Get().ClearRuneBuffer()
		// The offscreens must be updated every frame (#490).
		u.updateGraphicsContext(g)
	}); err != nil {
		return err
	}
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

func init() {
	if err := initialize(); err != nil {
		panic(err)
	}
}

func initialize() error {
	// Do nothing in node.js.
	if web.IsNodeJS() {
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
	canvas.Call("addEventListener", "keydown", input.OnKeyDown)
	canvas.Call("addEventListener", "keypress", input.OnKeyPress)
	canvas.Call("addEventListener", "keyup", input.OnKeyUp)

	// Mouse
	canvas.Call("addEventListener", "mousedown", input.OnMouseDown)
	canvas.Call("addEventListener", "mouseup", input.OnMouseUp)
	canvas.Call("addEventListener", "mousemove", input.OnMouseMove)
	canvas.Call("addEventListener", "contextmenu", func(e *js.Object) {
		e.Call("preventDefault")
	})

	// Touch
	canvas.Call("addEventListener", "touchstart", input.OnTouchStart)
	canvas.Call("addEventListener", "touchend", input.OnTouchEnd)
	canvas.Call("addEventListener", "touchmove", input.OnTouchMove)

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

func RunMainThreadLoop(ch <-chan error) error {
	return <-ch
}

func Run(width, height int, scale float64, title string, g GraphicsContext, mainloop bool) error {
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
