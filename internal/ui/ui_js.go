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
	"log"
	"runtime"
	"strconv"

	"github.com/gopherjs/gopherwasm/js"

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/input"
)

var canvas js.Value

type userInterface struct {
	width                int
	height               int
	scale                float64
	fullscreen           bool
	runnableInBackground bool
	vsync                bool

	sizeChanged bool
	windowFocus bool
	pageVisible bool
	contextLost bool

	lastActualScale float64
}

var currentUI = &userInterface{
	sizeChanged: true,
	windowFocus: true,
	pageVisible: true,
	vsync:       true,
}

var (
	window                = js.Global().Get("window")
	document              = js.Global().Get("document")
	requestAnimationFrame = window.Get("requestAnimationFrame")
	setTimeout            = window.Get("setTimeout")
)

func ScreenSizeInFullscreen() (int, int) {
	return window.Get("innerWidth").Int(), window.Get("innerHeight").Int()
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

func SetVsyncEnabled(enabled bool) {
	currentUI.vsync = enabled
}

func IsVsyncEnabled() bool {
	return currentUI.vsync
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
	document.Set("title", title)
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

func IsWindowResizable() bool {
	return false
}

func SetWindowResizable(decorated bool) {
	// Do nothing
}

func DeviceScaleFactor() float64 {
	return devicescale.GetAt(0, 0)
}

func (u *userInterface) getScale() float64 {
	if !u.fullscreen {
		return u.scale
	}
	body := document.Get("body")
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
	return u.getScale() * devicescale.GetAt(0, 0)
}

func (u *userInterface) updateGraphicsContext(g GraphicsContext) {
	a := u.actualScreenScale()
	if u.lastActualScale != a {
		u.updateScreenSize()
	}
	u.lastActualScale = a

	if u.sizeChanged {
		u.sizeChanged = false
		g.SetSize(u.width, u.height, a)
	}
}

func (u *userInterface) suspended() bool {
	return !u.runnableInBackground && (!u.windowFocus || !u.pageVisible)
}

func (u *userInterface) update(g GraphicsContext) error {
	if u.suspended() {
		hooks.SuspendAudio()
		return nil
	}
	hooks.ResumeAudio()

	input.Get().UpdateGamepads()
	u.updateGraphicsContext(g)
	if err := g.Update(func() {
		input.Get().ClearRuneBuffer()
		input.Get().ResetWheelValues()
		u.updateGraphicsContext(g)
	}); err != nil {
		return err
	}
	return nil
}

func (u *userInterface) loop(g GraphicsContext) <-chan error {
	ch := make(chan error)
	var cf js.Callback
	f := func([]js.Value) {
		if u.contextLost {
			requestAnimationFrame.Invoke(cf)
			return
		}

		if err := u.update(g); err != nil {
			ch <- err
			close(ch)
			return
		}
		if u.vsync {
			requestAnimationFrame.Invoke(cf)
		} else {
			setTimeout.Invoke(cf, 0)
		}
	}
	cf = js.NewCallback(f)
	// Call f asyncly to be async since ch is used in f.
	go func() {
		f(nil)
	}()
	return ch
}

func init() {
	if document.Get("body") == js.Null() {
		ch := make(chan struct{})
		window.Call("addEventListener", "load", js.NewCallback(func([]js.Value) {
			close(ch)
		}))
		<-ch
	}

	window.Call("addEventListener", "focus", js.NewCallback(func([]js.Value) {
		currentUI.windowFocus = true
		if currentUI.suspended() {
			hooks.SuspendAudio()
		} else {
			hooks.ResumeAudio()
		}
	}))
	window.Call("addEventListener", "blur", js.NewCallback(func([]js.Value) {
		currentUI.windowFocus = false
		if currentUI.suspended() {
			hooks.SuspendAudio()
		} else {
			hooks.ResumeAudio()
		}
	}))
	document.Call("addEventListener", "visibilitychange", js.NewCallback(func([]js.Value) {
		currentUI.pageVisible = !document.Get("hidden").Bool()
		if currentUI.suspended() {
			hooks.SuspendAudio()
		} else {
			hooks.ResumeAudio()
		}
	}))
	window.Call("addEventListener", "resize", js.NewCallback(func([]js.Value) {
		currentUI.updateScreenSize()
	}))

	// Adjust the initial scale to 1.
	// https://developer.mozilla.org/en/docs/Mozilla/Mobile/Viewport_meta_tag
	meta := document.Call("createElement", "meta")
	meta.Set("name", "viewport")
	meta.Set("content", "width=device-width, initial-scale=1")
	document.Get("head").Call("appendChild", meta)

	canvas = document.Call("createElement", "canvas")
	canvas.Set("width", 16)
	canvas.Set("height", 16)
	document.Get("body").Call("appendChild", canvas)

	htmlStyle := document.Get("documentElement").Get("style")
	htmlStyle.Set("height", "100%")
	htmlStyle.Set("margin", "0")
	htmlStyle.Set("padding", "0")

	bodyStyle := document.Get("body").Get("style")
	bodyStyle.Set("backgroundColor", "#000")
	bodyStyle.Set("position", "relative")
	bodyStyle.Set("height", "100%")
	bodyStyle.Set("margin", "0")
	bodyStyle.Set("padding", "0")
	// TODO: This is OK as long as the game is in an independent iframe.
	// What if the canvas is embedded in a HTML directly?
	document.Get("body").Call("addEventListener", "click", js.NewCallback(func([]js.Value) {
		canvas.Call("focus")
	}))

	canvasStyle := canvas.Get("style")
	canvasStyle.Set("position", "absolute")

	// Make the canvas focusable.
	canvas.Call("setAttribute", "tabindex", 1)
	canvas.Get("style").Set("outline", "none")

	// Keyboard
	// Don't 'preventDefault' on keydown events or keypress events wouldn't work (#715).
	canvas.Call("addEventListener", "keydown", js.NewEventCallback(0, input.OnKeyDown))
	canvas.Call("addEventListener", "keypress", js.NewEventCallback(js.PreventDefault, input.OnKeyPress))
	canvas.Call("addEventListener", "keyup", js.NewEventCallback(js.PreventDefault, input.OnKeyUp))

	// Mouse
	canvas.Call("addEventListener", "mousedown", js.NewEventCallback(js.PreventDefault, input.OnMouseDown))
	canvas.Call("addEventListener", "mouseup", js.NewEventCallback(js.PreventDefault, input.OnMouseUp))
	canvas.Call("addEventListener", "mousemove", js.NewEventCallback(js.PreventDefault, input.OnMouseMove))
	canvas.Call("addEventListener", "wheel", js.NewEventCallback(js.PreventDefault, input.OnWheel))

	// Touch
	canvas.Call("addEventListener", "touchstart", js.NewEventCallback(js.PreventDefault, input.OnTouchStart))
	canvas.Call("addEventListener", "touchend", js.NewEventCallback(js.PreventDefault, input.OnTouchEnd))
	canvas.Call("addEventListener", "touchmove", js.NewEventCallback(js.PreventDefault, input.OnTouchMove))

	// Gamepad
	window.Call("addEventListener", "gamepadconnected", js.NewCallback(func(e []js.Value) {
		// Do nothing.
	}))

	canvas.Call("addEventListener", "contextmenu", js.NewEventCallback(js.PreventDefault, func(js.Value) {
		// Do nothing.
	}))

	// Context
	canvas.Call("addEventListener", "webglcontextlost", js.NewEventCallback(js.PreventDefault, func(js.Value) {
		currentUI.contextLost = true
	}))
	canvas.Call("addEventListener", "webglcontextrestored", js.NewCallback(func(e []js.Value) {
		currentUI.contextLost = false
	}))
}

func Loop(ch <-chan error) error {
	return <-ch
}

func Run(width, height int, scale float64, title string, g GraphicsContext, mainloop bool) error {
	u := currentUI
	document.Set("title", title)
	u.setScreenSize(width, height, scale, u.fullscreen)
	canvas.Call("focus")
	ch := u.loop(g)
	if runtime.GOARCH == "wasm" {
		return <-ch
	}

	// On GopherJS, the main goroutine cannot be blocked due to the bug (gopherjs/gopherjs#826).
	// Return immediately.
	go func() {
		if err := <-ch; err != nil {
			log.Fatal(err)
		}
	}()
	return nil
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
