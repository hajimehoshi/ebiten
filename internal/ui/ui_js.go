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

package ui

import (
	"sync"
	"syscall/js"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/devicescale"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

type graphicsDriverCreatorImpl struct{}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	graphics, err := g.newOpenGL()
	return graphics, GraphicsLibraryOpenGL, err
}

func (*graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics()
}

func (*graphicsDriverCreatorImpl) newDirectX() (graphicsdriver.Graphics, error) {
	return nil, nil
}

func (*graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return nil, nil
}

var (
	stringNone        = js.ValueOf("none")
	stringTransparent = js.ValueOf("transparent")
)

func driverCursorShapeToCSSCursor(cursor CursorShape) string {
	switch cursor {
	case CursorShapeDefault:
		return "default"
	case CursorShapeText:
		return "text"
	case CursorShapeCrosshair:
		return "crosshair"
	case CursorShapePointer:
		return "pointer"
	case CursorShapeEWResize:
		return "ew-resize"
	case CursorShapeNSResize:
		return "ns-resize"
	}
	return "auto"
}

type userInterfaceImpl struct {
	graphicsDriver graphicsdriver.Graphics

	runnableOnUnfocused bool
	fpsMode             FPSModeType
	renderingScheduled  bool
	running             bool
	initFocused         bool
	cursorMode          CursorMode
	cursorPrevMode      CursorMode
	cursorShape         CursorShape
	onceUpdateCalled    bool

	lastDeviceScaleFactor float64

	context *context
	input   Input

	m sync.Mutex
}

func init() {
	theUI.userInterfaceImpl = userInterfaceImpl{
		runnableOnUnfocused: true,
		initFocused:         true,
	}
	theUI.input.ui = &theUI.userInterfaceImpl
}

var (
	window                = js.Global().Get("window")
	document              = js.Global().Get("document")
	canvas                js.Value
	requestAnimationFrame = js.Global().Get("requestAnimationFrame")
	setTimeout            = js.Global().Get("setTimeout")
)

var (
	documentHasFocus js.Value
	documentHidden   js.Value
)

func init() {
	documentHasFocus = document.Get("hasFocus").Call("bind", document)
	documentHidden = js.Global().Get("Object").Call("getOwnPropertyDescriptor", js.Global().Get("Document").Get("prototype"), "hidden").Get("get").Call("bind", document)
}

func (u *userInterfaceImpl) ScreenSizeInFullscreen() (int, int) {
	return window.Get("innerWidth").Int(), window.Get("innerHeight").Int()
}

func (u *userInterfaceImpl) SetFullscreen(fullscreen bool) {
	if !canvas.Truthy() {
		return
	}
	if !document.Truthy() {
		return
	}
	if fullscreen == u.IsFullscreen() {
		return
	}
	if fullscreen {
		f := canvas.Get("requestFullscreen")
		if !f.Truthy() {
			f = canvas.Get("webkitRequestFullscreen")
		}
		f.Call("bind", canvas).Invoke()
		return
	}
	f := document.Get("exitFullscreen")
	if !f.Truthy() {
		f = document.Get("webkitExitFullscreen")
	}
	f.Call("bind", document).Invoke()
}

func (u *userInterfaceImpl) IsFullscreen() bool {
	if !document.Truthy() {
		return false
	}
	if !document.Get("fullscreenElement").Truthy() && !document.Get("webkitFullscreenElement").Truthy() {
		return false
	}
	return true
}

func (u *userInterfaceImpl) IsFocused() bool {
	return u.isFocused()
}

func (u *userInterfaceImpl) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.runnableOnUnfocused = runnableOnUnfocused
}

func (u *userInterfaceImpl) IsRunnableOnUnfocused() bool {
	return u.runnableOnUnfocused
}

func (u *userInterfaceImpl) SetFPSMode(mode FPSModeType) {
	u.fpsMode = mode
}

func (u *userInterfaceImpl) ScheduleFrame() {
	u.renderingScheduled = true
}

func (u *userInterfaceImpl) CursorMode() CursorMode {
	if !canvas.Truthy() {
		return CursorModeHidden
	}
	return u.cursorMode
}

func (u *userInterfaceImpl) SetCursorMode(mode CursorMode) {
	if !canvas.Truthy() {
		return
	}
	if u.cursorMode == mode {
		return
	}
	// Remember the previous cursor mode in the case when the pointer lock exits by pressing ESC.
	u.cursorPrevMode = u.cursorMode
	if u.cursorMode == CursorModeCaptured {
		document.Call("exitPointerLock")
	}
	u.cursorMode = mode
	switch mode {
	case CursorModeVisible:
		canvas.Get("style").Set("cursor", driverCursorShapeToCSSCursor(u.cursorShape))
	case CursorModeHidden:
		canvas.Get("style").Set("cursor", stringNone)
	case CursorModeCaptured:
		canvas.Call("requestPointerLock")
	}
}

func (u *userInterfaceImpl) recoverCursorMode() {
	if theUI.cursorPrevMode == CursorModeCaptured {
		panic("ui: cursorPrevMode must not be CursorModeCaptured at recoverCursorMode")
	}
	u.SetCursorMode(u.cursorPrevMode)
}

func (u *userInterfaceImpl) CursorShape() CursorShape {
	if !canvas.Truthy() {
		return CursorShapeDefault
	}
	return u.cursorShape
}

func (u *userInterfaceImpl) SetCursorShape(shape CursorShape) {
	if !canvas.Truthy() {
		return
	}
	if u.cursorShape == shape {
		return
	}

	u.cursorShape = shape
	if u.cursorMode == CursorModeVisible {
		canvas.Get("style").Set("cursor", driverCursorShapeToCSSCursor(u.cursorShape))
	}
}

func (u *userInterfaceImpl) DeviceScaleFactor() float64 {
	return devicescale.GetAt(0, 0)
}

func (u *userInterfaceImpl) outsideSize() (float64, float64) {
	if document.Truthy() {
		body := document.Get("body")
		bw := body.Get("clientWidth").Float()
		bh := body.Get("clientHeight").Float()
		return bw, bh
	}

	// Node.js
	return 640, 480
}

func (u *userInterfaceImpl) suspended() bool {
	if u.runnableOnUnfocused {
		return false
	}
	return !u.isFocused()
}

func (u *userInterfaceImpl) isFocused() bool {
	if !documentHasFocus.Invoke().Bool() {
		return false
	}
	if documentHidden.Invoke().Bool() {
		return false
	}
	return true
}

func (u *userInterfaceImpl) update() error {
	if u.suspended() {
		return hooks.SuspendAudio()
	}
	if err := hooks.ResumeAudio(); err != nil {
		return err
	}
	return u.updateImpl(false)
}

func (u *userInterfaceImpl) updateImpl(force bool) error {
	// Guard updateImpl as this function cannot be invoked until this finishes (#2339).
	u.m.Lock()
	defer u.m.Unlock()

	// context can be nil when an event is fired but the loop doesn't start yet (#1928).
	if u.context == nil {
		return nil
	}

	if err := gamepad.Update(); err != nil {
		return err
	}

	a := u.DeviceScaleFactor()
	if u.lastDeviceScaleFactor != a {
		u.updateScreenSize()
	}
	u.lastDeviceScaleFactor = a

	w, h := u.outsideSize()
	if force {
		if err := u.context.forceUpdateFrame(u.graphicsDriver, w, h, u.DeviceScaleFactor()); err != nil {
			return err
		}
	} else {
		if err := u.context.updateFrame(u.graphicsDriver, w, h, u.DeviceScaleFactor()); err != nil {
			return err
		}
	}
	return nil
}

func (u *userInterfaceImpl) needsUpdate() bool {
	if u.fpsMode != FPSModeVsyncOffMinimum {
		return true
	}
	if !u.onceUpdateCalled {
		return true
	}
	if u.renderingScheduled {
		return true
	}
	// TODO: Watch the gamepad state?
	return false
}

func (u *userInterfaceImpl) loop(game Game) <-chan error {
	u.context = newContext(game)

	errCh := make(chan error, 1)
	reqStopAudioCh := make(chan struct{})
	resStopAudioCh := make(chan struct{})

	var cf js.Func
	f := func() {
		if u.needsUpdate() {
			u.onceUpdateCalled = true
			u.renderingScheduled = false
			if err := u.update(); err != nil {
				close(reqStopAudioCh)
				<-resStopAudioCh

				errCh <- err
				return
			}
		}
		switch u.fpsMode {
		case FPSModeVsyncOn:
			requestAnimationFrame.Invoke(cf)
		case FPSModeVsyncOffMaximum:
			setTimeout.Invoke(cf, 0)
		case FPSModeVsyncOffMinimum:
			requestAnimationFrame.Invoke(cf)
		}
	}

	// TODO: Should cf be released after the game ends?
	cf = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// f can be blocked but callbacks must not be blocked. Create a goroutine (#1161).
		go f()
		return nil
	})

	// Call f asyncly since ch is used in f.
	go f()

	// Run another loop to watch suspended() as the above update function is never called when the tab is hidden.
	// To check the document's visiblity, visibilitychange event should usually be used. However, this event is
	// not reliable and sometimes it is not fired (#961). Then, watch the state regularly instead.
	go func() {
		defer close(resStopAudioCh)

		const interval = 100 * time.Millisecond
		t := time.NewTicker(interval)
		defer func() {
			t.Stop()

			// This is a dirty hack. (*time.Ticker).Stop() just marks the timer 'deleted' [1] and
			// something might run even after Stop. On Wasm, this causes an issue to execute Go program
			// even after finishing (#1027). Sleep for the interval time duration to ensure that
			// everything related to the timer is finished.
			//
			// [1] runtime.deltimer
			time.Sleep(interval)
		}()

		for {
			select {
			case <-t.C:
				if u.suspended() {
					if err := hooks.SuspendAudio(); err != nil {
						errCh <- err
						return
					}
				} else {
					if err := hooks.ResumeAudio(); err != nil {
						errCh <- err
						return
					}
				}
			case <-reqStopAudioCh:
				return
			}
		}
	}()

	return errCh
}

func init() {
	// docuemnt is undefined on node.js
	if !document.Truthy() {
		return
	}

	if !document.Get("body").Truthy() {
		ch := make(chan struct{})
		window.Call("addEventListener", "load", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			close(ch)
			return nil
		}))
		<-ch
	}

	setWindowEventHandlers(window)

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

	canvasStyle := canvas.Get("style")
	canvasStyle.Set("width", "100%")
	canvasStyle.Set("height", "100%")
	canvasStyle.Set("margin", "0")
	canvasStyle.Set("padding", "0")

	// Make the canvas focusable.
	canvas.Call("setAttribute", "tabindex", 1)
	canvas.Get("style").Set("outline", "none")

	setCanvasEventHandlers(canvas)

	// Pointer Lock
	document.Call("addEventListener", "pointerlockchange", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if document.Get("pointerLockElement").Truthy() {
			return nil
		}
		// Recover the state correctly when the pointer lock exits.

		// A user can exit the pointer lock by pressing ESC. In this case, sync the cursor mode state.
		if theUI.cursorMode == CursorModeCaptured {
			theUI.recoverCursorMode()
		}
		theUI.input.recoverCursorPosition()
		return nil
	}))
	document.Call("addEventListener", "pointerlockerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		js.Global().Get("console").Call("error", "pointerlockerror event is fired. 'sandbox=\"allow-pointer-lock\"' might be required at an iframe. This function on browsers must be called as a result of a gestural interaction or orientation change.")
		return nil
	}))
	document.Call("addEventListener", "fullscreenerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		js.Global().Get("console").Call("error", "fullscreenerror event is fired. 'allow=\"fullscreen\"' or 'allowfullscreen' might be required at an iframe. This function on browsers must be called as a result of a gestural interaction or orientation change.")
		return nil
	}))
	document.Call("addEventListener", "webkitfullscreenerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		js.Global().Get("console").Call("error", "webkitfullscreenerror event is fired. 'allow=\"fullscreen\"' or 'allowfullscreen' might be required at an iframe. This function on browsers must be called as a result of a gestural interaction or orientation change.")
		return nil
	}))
}

func setWindowEventHandlers(v js.Value) {
	v.Call("addEventListener", "resize", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		theUI.updateScreenSize()

		// updateImpl can block. Use goroutine.
		// See https://pkg.go.dev/syscall/js#FuncOf.
		go func() {
			if err := theUI.updateImpl(true); err != nil {
				panic(err)
			}
		}()
		return nil
	}))
}

func setCanvasEventHandlers(v js.Value) {
	// Keyboard
	v.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Focus the canvas explicitly to activate tha game (#961).
		v.Call("focus")

		e := args[0]
		e.Call("preventDefault")
		theUI.input.updateFromEvent(e)
		return nil
	}))
	v.Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.updateFromEvent(e)
		return nil
	}))

	// Mouse
	v.Call("addEventListener", "mousedown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Focus the canvas explicitly to activate tha game (#961).
		v.Call("focus")

		e := args[0]
		e.Call("preventDefault")
		theUI.input.updateFromEvent(e)
		return nil
	}))
	v.Call("addEventListener", "mouseup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.updateFromEvent(e)
		return nil
	}))
	v.Call("addEventListener", "mousemove", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.updateFromEvent(e)
		return nil
	}))
	v.Call("addEventListener", "wheel", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.updateFromEvent(e)
		return nil
	}))

	// Touch
	v.Call("addEventListener", "touchstart", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Focus the canvas explicitly to activate tha game (#961).
		v.Call("focus")

		e := args[0]
		e.Call("preventDefault")
		theUI.input.updateFromEvent(e)
		return nil
	}))
	v.Call("addEventListener", "touchend", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.updateFromEvent(e)
		return nil
	}))
	v.Call("addEventListener", "touchmove", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		theUI.input.updateFromEvent(e)
		return nil
	}))

	// Context menu
	v.Call("addEventListener", "contextmenu", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		return nil
	}))

	// Context
	v.Call("addEventListener", "webglcontextlost", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		window.Get("location").Call("reload")
		return nil
	}))
}

func (u *userInterfaceImpl) forceUpdateOnMinimumFPSMode() {
	if u.fpsMode != FPSModeVsyncOffMinimum {
		return
	}

	// updateImpl can block. Use goroutine.
	// See https://pkg.go.dev/syscall/js#FuncOf.
	go func() {
		if err := u.updateImpl(true); err != nil {
			panic(err)
		}
	}()
}

func (u *userInterfaceImpl) Run(game Game) error {
	if u.initFocused && window.Truthy() {
		// Do not focus the canvas when the current document is in an iframe.
		// Otherwise, the parent page tries to focus the iframe on every loading, which is annoying (#1373).
		isInIframe := !window.Get("location").Equal(window.Get("parent").Get("location"))
		if !isInIframe {
			canvas.Call("focus")
		}
	}
	u.running = true
	g, err := newGraphicsDriver(&graphicsDriverCreatorImpl{})
	if err != nil {
		return err
	}
	u.graphicsDriver = g
	if g, ok := u.graphicsDriver.(interface{ SetCanvas(js.Value) }); ok {
		g.SetCanvas(canvas)
	}
	return <-u.loop(game)
}

func (u *userInterfaceImpl) updateScreenSize() {
	if document.Truthy() {
		body := document.Get("body")
		bw := int(body.Get("clientWidth").Float() * u.DeviceScaleFactor())
		bh := int(body.Get("clientHeight").Float() * u.DeviceScaleFactor())
		canvas.Set("width", bw)
		canvas.Set("height", bh)
	}
}

func (u *userInterfaceImpl) SetScreenTransparent(transparent bool) {
	if u.running {
		panic("ui: SetScreenTransparent can't be called after the main loop starts")
	}

	bodyStyle := document.Get("body").Get("style")
	if transparent {
		bodyStyle.Set("backgroundColor", "transparent")
	} else {
		bodyStyle.Set("backgroundColor", "#000")
	}
}

func (u *userInterfaceImpl) IsScreenTransparent() bool {
	bodyStyle := document.Get("body").Get("style")
	return bodyStyle.Get("backgroundColor").Equal(stringTransparent)
}

func (u *userInterfaceImpl) resetForTick() {
	u.input.resetForTick()
}

func (u *userInterfaceImpl) SetInitFocused(focused bool) {
	if u.running {
		panic("ui: SetInitFocused must be called before the main loop")
	}
	u.initFocused = focused
}

func (u *userInterfaceImpl) Input() *Input {
	return &u.input
}

func (u *userInterfaceImpl) Window() Window {
	return &nullWindow{}
}
