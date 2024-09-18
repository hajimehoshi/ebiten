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
	"errors"
	"math"
	"sync"
	"syscall/js"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/file"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
)

type graphicsDriverCreatorImpl struct {
	canvas     js.Value
	colorSpace graphicsdriver.ColorSpace
}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	graphics, err := g.newOpenGL()
	return graphics, GraphicsLibraryOpenGL, err
}

func (g *graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics(g.canvas, g.colorSpace)
}

func (*graphicsDriverCreatorImpl) newDirectX() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: DirectX is not supported in this environment")
}

func (*graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: Metal is not supported in this environment")
}

func (*graphicsDriverCreatorImpl) newPlayStation5() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: PlayStation 5 is not supported in this environment")
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
	case CursorShapeNESWResize:
		return "nesw-resize"
	case CursorShapeNWSEResize:
		return "nwse-resize"
	case CursorShapeMove:
		return "move"
	case CursorShapeNotAllowed:
		return "not-allowed"
	}
	return "auto"
}

type userInterfaceImpl struct {
	graphicsDriver graphicsdriver.Graphics

	runnableOnUnfocused bool
	fpsMode             FPSModeType
	renderingScheduled  bool
	cursorMode          CursorMode
	cursorPrevMode      CursorMode
	captureCursorLater  bool
	cursorShape         CursorShape
	onceUpdateCalled    bool
	lastCaptureExitTime time.Time
	hiDPIEnabled        bool

	context                   *context
	inputState                InputState
	keyDurationsByKeyProperty map[Key]int
	cursorXInClient           float64
	cursorYInClient           float64
	origCursorXInClient       float64
	origCursorYInClient       float64
	touchesInClient           []touchInClient

	savedCursorX              float64
	savedCursorY              float64
	savedOutsideWidth         float64
	savedOutsideHeight        float64
	outsideSizeUnchangedCount int

	keyboardLayoutMap js.Value

	m         sync.Mutex
	dropFileM sync.Mutex
}

var (
	window                = js.Global().Get("window")
	document              = js.Global().Get("document")
	screen                = js.Global().Get("screen")
	canvas                js.Value
	requestAnimationFrame = js.Global().Get("requestAnimationFrame")
	setTimeout            = js.Global().Get("setTimeout")
)

var (
	documentHasFocus = document.Get("hasFocus").Call("bind", document)
	documentHidden   = js.Global().Get("Object").Call("getOwnPropertyDescriptor", js.Global().Get("Document").Get("prototype"), "hidden").Get("get").Call("bind", document)
)

func (u *UserInterface) SetFullscreen(fullscreen bool) {
	if !canvas.Truthy() {
		return
	}
	if !document.Truthy() {
		return
	}
	if fullscreen == u.IsFullscreen() {
		return
	}

	if u.cursorMode == CursorModeCaptured {
		u.saveCursorPosition()
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

func (u *UserInterface) IsFullscreen() bool {
	if !document.Truthy() {
		return false
	}
	if !document.Get("fullscreenElement").Truthy() && !document.Get("webkitFullscreenElement").Truthy() {
		return false
	}
	return true
}

func (u *UserInterface) IsFocused() bool {
	return u.isFocused()
}

func (u *UserInterface) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.runnableOnUnfocused = runnableOnUnfocused
}

func (u *UserInterface) IsRunnableOnUnfocused() bool {
	return u.runnableOnUnfocused
}

func (u *UserInterface) FPSMode() FPSModeType {
	return u.fpsMode
}

func (u *UserInterface) SetFPSMode(mode FPSModeType) {
	u.fpsMode = mode
}

func (u *UserInterface) ScheduleFrame() {
	u.renderingScheduled = true
}

func (u *UserInterface) CursorMode() CursorMode {
	if !canvas.Truthy() {
		return CursorModeHidden
	}
	return u.cursorMode
}

func (u *UserInterface) SetCursorMode(mode CursorMode) {
	if mode == CursorModeCaptured && !u.canCaptureCursor() {
		u.captureCursorLater = true
		return
	}
	u.setCursorMode(mode)
}

func (u *UserInterface) setCursorMode(mode CursorMode) {
	u.captureCursorLater = false

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
		u.lastCaptureExitTime = time.Now()
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

func (u *UserInterface) recoverCursorMode() {
	if u.cursorPrevMode == CursorModeCaptured {
		panic("ui: cursorPrevMode must not be CursorModeCaptured at recoverCursorMode")
	}
	u.SetCursorMode(u.cursorPrevMode)
}

func (u *UserInterface) CursorShape() CursorShape {
	if !canvas.Truthy() {
		return CursorShapeDefault
	}
	return u.cursorShape
}

func (u *UserInterface) SetCursorShape(shape CursorShape) {
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

func (u *UserInterface) outsideSize() (float64, float64) {
	if document.Truthy() {
		body := document.Get("body")
		bw := body.Get("clientWidth").Float()
		bh := body.Get("clientHeight").Float()
		return bw, bh
	}

	// Node.js
	return 640, 480
}

func (u *UserInterface) suspended() bool {
	if u.runnableOnUnfocused {
		return false
	}
	return !u.isFocused()
}

func (u *UserInterface) isFocused() bool {
	if !documentHasFocus.Invoke().Bool() {
		return false
	}
	if documentHidden.Invoke().Bool() {
		return false
	}
	return true
}

// canCaptureCursor reports whether a cursor can be captured or not now.
// Just after escaping from a capture, a browser might not be able to capture a cursor (#2693).
// If it is too early to capture a cursor, Ebitengine tries to delay it.
//
// See also https://w3c.github.io/pointerlock/#extensions-to-the-element-interface
//
// > Pointer lock is a transient activation-gated API, therefore a requestPointerLock() call
// > MUST fail if the relevant global object of this does not have transient activation.
// > This prevents locking upon initial navigation or re-acquiring lock without user's attention.
func (u *UserInterface) canCaptureCursor() bool {
	// 1.5 [sec] seems enough in the real world.
	return time.Now().Sub(u.lastCaptureExitTime) >= 1500*time.Millisecond
}

func (u *UserInterface) update() error {
	if u.captureCursorLater && u.canCaptureCursor() {
		u.setCursorMode(CursorModeCaptured)
	}

	if u.suspended() {
		return hook.SuspendAudio()
	}
	if err := hook.ResumeAudio(); err != nil {
		return err
	}
	return u.updateImpl(false)
}

func (u *UserInterface) updateImpl(force bool) error {
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

	// TODO: If DeviceScaleFactor changes, call updateScreenSize.
	// Now there is not a good way to detect the change.
	// See also https://crbug.com/123694.

	w, h := u.outsideSize()
	if force {
		if err := u.context.forceUpdateFrame(u.graphicsDriver, w, h, theMonitor.DeviceScaleFactor(), u); err != nil {
			return err
		}
	} else {
		if err := u.context.updateFrame(u.graphicsDriver, w, h, theMonitor.DeviceScaleFactor(), u); err != nil {
			return err
		}
	}
	return nil
}

func (u *UserInterface) needsUpdate() bool {
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

func (u *UserInterface) loopGame() error {
	// Initialize the screen size first (#3034).
	// If ebiten.SetRunnableOnUnfocused(false) and the canvas is not focused,
	// suspended() returns true and the update routine cannot start.
	u.updateScreenSize()

	errCh := make(chan error, 1)
	reqStopAudioCh := make(chan struct{})
	resStopAudioCh := make(chan struct{})

	var cf js.Func
	f := func() {
		if err := u.error(); err != nil {
			errCh <- err
			return
		}
		if u.needsUpdate() {
			defer func() {
				u.onceUpdateCalled = true
			}()
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
	cf = js.FuncOf(func(this js.Value, args []js.Value) any {
		// f can be blocked but callbacks must not be blocked. Create a goroutine (#1161).
		go f()
		return nil
	})

	// Call f asyncly since ch is used in f.
	go f()

	// Run another loop to watch suspended() as the above update function is never called when the tab is hidden.
	// To check the document's visibility, visibilitychange event should usually be used. However, this event is
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
					if err := hook.SuspendAudio(); err != nil {
						errCh <- err
						return
					}
				} else {
					if err := hook.ResumeAudio(); err != nil {
						errCh <- err
						return
					}
				}
			case <-reqStopAudioCh:
				return
			}
		}
	}()

	return <-errCh
}

func (u *UserInterface) init() error {
	u.userInterfaceImpl = userInterfaceImpl{
		runnableOnUnfocused: true,
		savedCursorX:        math.NaN(),
		savedCursorY:        math.NaN(),
		hiDPIEnabled:        true,
	}

	// document is undefined on node.js
	if !document.Truthy() {
		return nil
	}

	if !document.Get("body").Truthy() {
		ch := make(chan struct{})
		window.Call("addEventListener", "load", js.FuncOf(func(this js.Value, args []js.Value) any {
			close(ch)
			return nil
		}))
		<-ch
	}

	u.setWindowEventHandlers(window)

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
	canvasStyle.Set("display", "block")

	// Make the canvas focusable.
	canvas.Call("setAttribute", "tabindex", 1)
	canvas.Get("style").Set("outline", "none")

	u.setCanvasEventHandlers(canvas)

	// Pointer Lock
	document.Call("addEventListener", "pointerlockchange", js.FuncOf(func(this js.Value, args []js.Value) any {
		if document.Get("pointerLockElement").Truthy() {
			return nil
		}
		// Recover the state correctly when the pointer lock exits.

		// A user can exit the pointer lock by pressing ESC. In this case, sync the cursor mode state.
		if u.cursorMode == CursorModeCaptured {
			u.recoverCursorMode()
		}
		u.recoverCursorPosition()
		return nil
	}))
	document.Call("addEventListener", "pointerlockerror", js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Get("console").Call("error", "pointerlockerror event is fired. 'sandbox=\"allow-pointer-lock\"' might be required at an iframe. This function on browsers must be called as a result of a gestural interaction or orientation change.")
		if u.cursorMode == CursorModeCaptured {
			u.recoverCursorMode()
		}
		u.recoverCursorPosition()
		return nil
	}))
	document.Call("addEventListener", "fullscreenerror", js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Get("console").Call("error", "fullscreenerror event is fired. 'allow=\"fullscreen\"' or 'allowfullscreen' might be required at an iframe. This function on browsers must be called as a result of a gestural interaction or orientation change.")
		return nil
	}))
	document.Call("addEventListener", "webkitfullscreenerror", js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Get("console").Call("error", "webkitfullscreenerror event is fired. 'allow=\"fullscreen\"' or 'allowfullscreen' might be required at an iframe. This function on browsers must be called as a result of a gestural interaction or orientation change.")
		return nil
	}))

	return nil
}

func (u *UserInterface) setWindowEventHandlers(v js.Value) {
	v.Call("addEventListener", "resize", js.FuncOf(func(this js.Value, args []js.Value) any {
		u.updateScreenSize()

		// updateImpl can block. Use goroutine.
		// See https://pkg.go.dev/syscall/js#FuncOf.
		go func() {
			if err := u.updateImpl(true); err != nil {
				u.setError(err)
				return
			}
		}()
		return nil
	}))
}

func (u *UserInterface) setCanvasEventHandlers(v js.Value) {
	// Keyboard
	v.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) any {
		// Focus the canvas explicitly to activate tha game (#961).
		v.Call("focus")

		e := args[0]
		e.Call("preventDefault")
		if err := u.updateInputFromEvent(e); err != nil {
			u.setError(err)
			return nil
		}
		return nil
	}))
	v.Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		if err := u.updateInputFromEvent(e); err != nil {
			u.setError(err)
			return nil
		}
		return nil
	}))

	// Mouse
	v.Call("addEventListener", "mousedown", js.FuncOf(func(this js.Value, args []js.Value) any {
		// Focus the canvas explicitly to activate tha game (#961).
		v.Call("focus")

		e := args[0]
		e.Call("preventDefault")
		if err := u.updateInputFromEvent(e); err != nil {
			u.setError(err)
			return nil
		}
		return nil
	}))
	v.Call("addEventListener", "mouseup", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		if err := u.updateInputFromEvent(e); err != nil {
			u.setError(err)
			return nil
		}
		return nil
	}))
	v.Call("addEventListener", "mousemove", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		if err := u.updateInputFromEvent(e); err != nil {
			u.setError(err)
			return nil
		}
		return nil
	}))
	v.Call("addEventListener", "wheel", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		if err := u.updateInputFromEvent(e); err != nil {
			u.setError(err)
			return nil
		}
		return nil
	}))

	// Touch
	v.Call("addEventListener", "touchstart", js.FuncOf(func(this js.Value, args []js.Value) any {
		// Focus the canvas explicitly to activate tha game (#961).
		v.Call("focus")

		e := args[0]
		e.Call("preventDefault")
		if err := u.updateInputFromEvent(e); err != nil {
			u.setError(err)
			return nil
		}
		return nil
	}))
	v.Call("addEventListener", "touchend", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		if err := u.updateInputFromEvent(e); err != nil {
			u.setError(err)
			return nil
		}
		return nil
	}))
	v.Call("addEventListener", "touchmove", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		if err := u.updateInputFromEvent(e); err != nil {
			u.setError(err)
			return nil
		}
		return nil
	}))

	// Context menu
	v.Call("addEventListener", "contextmenu", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		return nil
	}))

	// Context
	v.Call("addEventListener", "webglcontextlost", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		window.Get("location").Call("reload")
		return nil
	}))

	// Drop
	v.Call("addEventListener", "dragover", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		return nil
	}))
	v.Call("addEventListener", "drop", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		data := e.Get("dataTransfer")
		if !data.Truthy() {
			return nil
		}

		go u.appendDroppedFiles(data)
		return nil
	}))

	// Blur
	v.Call("addEventListener", "blur", js.FuncOf(func(this js.Value, args []js.Value) any {
		u.inputState.resetForBlur()
		return nil
	}))
}

func (u *UserInterface) appendDroppedFiles(data js.Value) {
	u.dropFileM.Lock()
	defer u.dropFileM.Unlock()
	items := data.Get("items")

	var entries []js.Value
	for i := 0; i < items.Length(); i++ {
		kind := items.Index(i).Get("kind").String()
		switch kind {
		case "file":
			entries = append(entries, items.Index(i).Call("webkitGetAsEntry").Get("filesystem").Get("root"))
		}
	}
	if len(entries) > 0 {
		fs, err := file.NewFileEntryFS(entries)
		if err != nil {
			u.setError(err)
			return
		}
		u.inputState.DroppedFiles = fs
	}
}

func (u *UserInterface) forceUpdateOnMinimumFPSMode() {
	if u.fpsMode != FPSModeVsyncOffMinimum {
		return
	}

	// updateImpl can block. Use goroutine.
	// See https://pkg.go.dev/syscall/js#FuncOf.
	go func() {
		if err := u.updateImpl(true); err != nil {
			u.setError(err)
		}
	}()
}

func (u *UserInterface) shouldFocusFirst(options *RunOptions) bool {
	if options.InitUnfocused {
		return false
	}
	if !window.Truthy() {
		return false
	}

	// Do not focus the canvas when the current document is in an iframe.
	// Otherwise, the parent page tries to focus the iframe on every loading, which is annoying (#1373).
	parent := window.Get("parent")
	isInIframe := !window.Get("location").Equal(parent.Get("location"))
	if !isInIframe {
		return true
	}

	return false
}

func (u *UserInterface) initOnMainThread(options *RunOptions) error {
	u.setRunning(true)

	u.hiDPIEnabled = !options.DisableHiDPI

	if u.shouldFocusFirst(options) {
		canvas.Call("focus")
	}

	g, lib, err := newGraphicsDriver(&graphicsDriverCreatorImpl{
		canvas:     canvas,
		colorSpace: options.ColorSpace,
	}, options.GraphicsLibrary)
	if err != nil {
		return err
	}
	u.graphicsDriver = g
	u.setGraphicsLibrary(lib)

	if bodyStyle := document.Get("body").Get("style"); options.ScreenTransparent {
		bodyStyle.Set("backgroundColor", "transparent")
	} else {
		bodyStyle.Set("backgroundColor", "#000")
	}

	return nil
}

func (u *UserInterface) updateScreenSize() {
	if document.Truthy() {
		body := document.Get("body")
		f := theMonitor.DeviceScaleFactor()
		bw := int(body.Get("clientWidth").Float() * f)
		bh := int(body.Get("clientHeight").Float() * f)
		canvas.Set("width", bw)
		canvas.Set("height", bh)
	}
}

func (u *UserInterface) readInputState(inputState *InputState) {
	u.inputState.copyAndReset(inputState)
	u.keyboardLayoutMap = js.Value{}
}

func (u *UserInterface) Window() Window {
	return &nullWindow{}
}

type Monitor struct {
	deviceScaleFactor float64
}

var theMonitor = &Monitor{}

func (m *Monitor) Name() string {
	return ""
}

func (m *Monitor) DeviceScaleFactor() float64 {
	if !theUI.hiDPIEnabled {
		return 1
	}

	if m.deviceScaleFactor != 0 {
		return m.deviceScaleFactor
	}

	ratio := window.Get("devicePixelRatio").Float()
	if ratio == 0 {
		ratio = 1
	}
	m.deviceScaleFactor = ratio
	return m.deviceScaleFactor
}

func (m *Monitor) Size() (int, int) {
	return screen.Get("width").Int(), screen.Get("height").Int()
}

func (u *UserInterface) AppendMonitors(mons []*Monitor) []*Monitor {
	return append(mons, theMonitor)
}

func (u *UserInterface) Monitor() *Monitor {
	return theMonitor
}

func (u *UserInterface) updateIconIfNeeded() error {
	return nil
}

func IsScreenTransparentAvailable() bool {
	return true
}

func dipToNativePixels(x float64, scale float64) float64 {
	return x
}
