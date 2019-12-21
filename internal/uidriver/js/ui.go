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

package js

import (
	"image"
	"log"
	"runtime"
	"syscall/js"
	"time"

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/jsutil"
)

type UserInterface struct {
	runnableInBackground bool
	vsync                bool
	running              bool

	sizeChanged bool
	contextLost bool

	lastDeviceScaleFactor float64

	context driver.UIContext
	input   Input
}

var theUI = &UserInterface{
	sizeChanged: true,
	vsync:       true,
}

func init() {
	theUI.input.ui = theUI
}

func Get() *UserInterface {
	return theUI
}

var (
	window                = js.Global().Get("window")
	document              = js.Global().Get("document")
	canvas                js.Value
	requestAnimationFrame = window.Get("requestAnimationFrame")
	setTimeout            = window.Get("setTimeout")
)

func (u *UserInterface) ScreenSizeInFullscreen() (int, int) {
	return window.Get("innerWidth").Int(), window.Get("innerHeight").Int()
}

func (u *UserInterface) SetFullscreen(fullscreen bool) {
	// Do nothing
}

func (u *UserInterface) IsFullscreen() bool {
	return false
}

func (u *UserInterface) SetRunnableInBackground(runnableInBackground bool) {
	u.runnableInBackground = runnableInBackground
}

func (u *UserInterface) IsRunnableInBackground() bool {
	return u.runnableInBackground
}

func (u *UserInterface) SetVsyncEnabled(enabled bool) {
	u.vsync = enabled
}

func (u *UserInterface) IsVsyncEnabled() bool {
	return u.vsync
}

func (u *UserInterface) CursorMode() driver.CursorMode {
	if canvas.Get("style").Get("cursor").String() != "none" {
		return driver.CursorModeVisible
	}
	return driver.CursorModeHidden
}

func (u *UserInterface) SetCursorMode(mode driver.CursorMode) {
	var visible bool
	switch mode {
	case driver.CursorModeVisible:
		visible = true
	case driver.CursorModeHidden:
		visible = false
	default:
		return
	}

	if visible {
		canvas.Get("style").Set("cursor", "auto")
	} else {
		canvas.Get("style").Set("cursor", "none")
	}
}

func (u *UserInterface) SetWindowTitle(title string) {
	// TODO: As the page should be in an iframe, this might be useless.
	document.Set("title", title)
}

func (u *UserInterface) SetWindowIcon(iconImages []image.Image) {
	// Do nothing
}

func (u *UserInterface) IsWindowDecorated() bool {
	return false
}

func (u *UserInterface) SetWindowDecorated(decorated bool) {
	// Do nothing
}

func (u *UserInterface) IsWindowResizable() bool {
	return false
}

func (u *UserInterface) SetWindowResizable(decorated bool) {
	// Do nothing
}

func (u *UserInterface) SetWindowSize(width, height int) {
	// TODO: This is too tricky: Even though browsers don't have windows, SetWindowSize is called whenever the
	// screen size is changed. Fix this hack.
	u.sizeChanged = true
}

func (u *UserInterface) DeviceScaleFactor() float64 {
	return devicescale.GetAt(0, 0)
}

func (u *UserInterface) updateSize() {
	a := u.DeviceScaleFactor()
	if u.lastDeviceScaleFactor != a {
		u.updateScreenSize()
	}
	u.lastDeviceScaleFactor = a

	if u.sizeChanged {
		u.sizeChanged = false
		body := document.Get("body")
		bw := body.Get("clientWidth").Float()
		bh := body.Get("clientHeight").Float()
		u.context.Layout(bw, bh)
	}
}

func (u *UserInterface) suspended() bool {
	if u.runnableInBackground {
		return false
	}

	if !document.Call("hasFocus").Bool() {
		return true
	}
	if document.Get("hidden").Bool() {
		return true
	}
	return false
}

func (u *UserInterface) update() error {
	if u.suspended() {
		hooks.SuspendAudio()
		return nil
	}
	hooks.ResumeAudio()

	u.input.UpdateGamepads()
	u.updateSize()
	if err := u.context.Update(func() {
		u.updateSize()
	}); err != nil {
		return err
	}
	return nil
}

func (u *UserInterface) loop(context driver.UIContext) <-chan error {
	u.context = context

	ch := make(chan error)
	var cf js.Func
	f := func(this js.Value, args []js.Value) interface{} {
		if u.contextLost {
			requestAnimationFrame.Invoke(cf)
			return nil
		}

		if err := u.update(); err != nil {
			ch <- err
			close(ch)
			return nil
		}
		if u.vsync {
			requestAnimationFrame.Invoke(cf)
		} else {
			setTimeout.Invoke(cf, 0)
		}
		return nil
	}
	// TODO: Should cf be released after the game ends?
	cf = js.FuncOf(f)
	// Call f asyncly to be async since ch is used in f.
	go func() {
		f(js.Value{}, nil)
	}()

	// Run another loop to watch suspended() as the above update function is never called when the tab is hidden.
	// To check the document's visiblity, visibilitychange event should usually be used. However, this event is
	// not reliable and sometimes it is not fired (#961). Then, watch the state regularly instead.
	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for range t.C {
			if u.suspended() {
				hooks.SuspendAudio()
			} else {
				hooks.ResumeAudio()
			}
		}
	}()

	return ch
}

func init() {
	if jsutil.Equal(document.Get("body"), js.Null()) {
		ch := make(chan struct{})
		window.Call("addEventListener", "load", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			close(ch)
			return nil
		}))
		<-ch
	}

	window.Call("addEventListener", "resize", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		theUI.updateScreenSize()
		return nil
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
	bodyStyle.Set("height", "100%")
	bodyStyle.Set("margin", "0")
	bodyStyle.Set("padding", "0")

	// TODO: This is OK as long as the game is in an independent iframe.
	// What if the canvas is embedded in a HTML directly?
	document.Get("body").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		canvas.Call("focus")
		return nil
	}))

	canvasStyle := canvas.Get("style")
	canvasStyle.Set("width", "100%")
	canvasStyle.Set("height", "100%")
	canvasStyle.Set("margin", "0")
	canvasStyle.Set("padding", "0")

	// Make the canvas focusable.
	canvas.Call("setAttribute", "tabindex", 1)
	canvas.Get("style").Set("outline", "none")

	// Keyboard
	canvas.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Focus the canvas explicitly to activate tha game (#961).
		canvas.Call("focus")

		e := args[0]
		// Don't 'preventDefault' on keydown events or keypress events wouldn't work (#715).
		theUI.input.Update(e)
		return nil
	}))
	canvas.Call("addEventListener", "keypress", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.Update(e)
		return nil
	}))
	canvas.Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.Update(e)
		return nil
	}))

	// Mouse
	canvas.Call("addEventListener", "mousedown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Focus the canvas explicitly to activate tha game (#961).
		canvas.Call("focus")

		e := args[0]
		e.Call("preventDefault")
		theUI.input.Update(e)
		return nil
	}))
	canvas.Call("addEventListener", "mouseup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.Update(e)
		return nil
	}))
	canvas.Call("addEventListener", "mousemove", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.Update(e)
		return nil
	}))
	canvas.Call("addEventListener", "wheel", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.Update(e)
		return nil
	}))

	// Touch
	canvas.Call("addEventListener", "touchstart", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Focus the canvas explicitly to activate tha game (#961).
		canvas.Call("focus")

		e := args[0]
		e.Call("preventDefault")
		theUI.input.Update(e)
		return nil
	}))
	canvas.Call("addEventListener", "touchend", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.Update(e)
		return nil
	}))
	canvas.Call("addEventListener", "touchmove", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.Update(e)
		return nil
	}))

	// Gamepad
	window.Call("addEventListener", "gamepadconnected", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Do nothing.
		return nil
	}))

	canvas.Call("addEventListener", "contextmenu", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		return nil
	}))

	// Context
	canvas.Call("addEventListener", "webglcontextlost", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.contextLost = true
		return nil
	}))
	canvas.Call("addEventListener", "webglcontextrestored", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		theUI.contextLost = false
		return nil
	}))
}

func (u *UserInterface) Run(context driver.UIContext, graphics driver.Graphics) error {
	canvas.Call("focus")
	u.running = true
	ch := u.loop(context)
	if runtime.GOARCH == "wasm" {
		return <-ch
	}

	// On GopherJS, the main goroutine cannot be blocked due to the bug (gopherjs/gopherjs#826).
	// Return immediately.
	go func() {
		defer func() {
			u.running = false
		}()
		if err := <-ch; err != nil {
			log.Fatal(err)
		}
	}()
	return nil
}

func (u *UserInterface) RunWithoutMainLoop(width, height int, scale float64, title string, context driver.UIContext, graphics driver.Graphics) <-chan error {
	panic("js: RunWithoutMainLoop is not implemented")
}

func (u *UserInterface) updateScreenSize() {
	body := document.Get("body")
	bw := int(body.Get("clientWidth").Float() * u.DeviceScaleFactor())
	bh := int(body.Get("clientHeight").Float() * u.DeviceScaleFactor())
	canvas.Set("width", bw)
	canvas.Set("height", bh)
	u.sizeChanged = true
}

func (u *UserInterface) SetWindowPosition(x, y int) {
	// Do nothing
}

func (u *UserInterface) WindowPosition() (int, int) {
	if !u.running {
		panic("js: WindowPosition can't be called before the main loop starts")
	}
	return 0, 0
}

func (u *UserInterface) SetScreenTransparent(transparent bool) {
	if u.running {
		panic("js: SetScreenTransparent can't be called after the main loop starts")
	}

	bodyStyle := document.Get("body").Get("style")
	if transparent {
		bodyStyle.Set("backgroundColor", "transparent")
	} else {
		bodyStyle.Set("backgroundColor", "#000")
	}
}

func (u *UserInterface) IsScreenTransparent() bool {
	bodyStyle := document.Get("body").Get("style")
	return bodyStyle.Get("backgroundColor").String() == "transparent"
}

func (u *UserInterface) MonitorPosition() (int, int) {
	return 0, 0
}

func (u *UserInterface) CanHaveWindow() bool {
	return false
}

func (u *UserInterface) Input() driver.Input {
	return &u.input
}
