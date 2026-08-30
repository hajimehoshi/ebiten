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
	"math"
	"syscall/js"
	"unicode"
)

var (
	stringAlt     = js.ValueOf("Alt")
	stringControl = js.ValueOf("Control")
	stringMeta    = js.ValueOf("Meta")
	stringShift   = js.ValueOf("Shift")

	stringKeydown    = js.ValueOf("keydown")
	stringKeyup      = js.ValueOf("keyup")
	stringMousedown  = js.ValueOf("mousedown")
	stringMouseup    = js.ValueOf("mouseup")
	stringMousemove  = js.ValueOf("mousemove")
	stringWheel      = js.ValueOf("wheel")
	stringTouchstart = js.ValueOf("touchstart")
	stringTouchend   = js.ValueOf("touchend")
	stringTouchmove  = js.ValueOf("touchmove")

	stringCapsLock = js.ValueOf("CapsLock")
	stringNumLock  = js.ValueOf("NumLock")
)

type touchInClient struct {
	id TouchID
	x  float64
	y  float64
}

func jsCodeToID(code js.Value) Key {
	// js.Value cannot be used as a map key.
	// As the number of keys is around 100, just a dumb loop should work.
	for uiKey, jsCode := range uiKeyToJSCode {
		if jsCode.Equal(code) {
			return uiKey
		}
	}
	return -1
}

var codeToMouseButton = map[int]MouseButton{
	0: MouseButton0, // Left
	1: MouseButton1, // Middle
	2: MouseButton2, // Right
	3: MouseButton3,
	4: MouseButton4,
}

func eventToKeys(e js.Value) (key0, key1 Key) {
	id := jsCodeToID(e.Get("code"))
	if id >= 0 {
		return id, -1
	}

	// With a virtual keyboard on mobile devices, e.code is empty. Use a 'key' property instead (#2898).
	key := e.Get("key")

	// The key property doesn't distinghlish between left and right modifier keys.
	// Let's assume both keys are pressed.
	switch {
	case key.Equal(stringAlt):
		return KeyAltLeft, KeyAltRight
	case key.Equal(stringControl):
		return KeyControlLeft, KeyControlRight
	case key.Equal(stringMeta):
		return KeyMetaLeft, KeyMetaRight
	case key.Equal(stringShift):
		return KeyShiftLeft, KeyShiftRight
	}

	for uiKey, jsKey := range uiKeyToJSKey {
		if key.Equal(jsKey) {
			return uiKey, -1
		}
	}

	return -1, -1
}

// An inputButtonEvent is a key or mouse-button state change reported by a browser event handler,
// queued without a timestamp.
//
// A browser fires its event handlers on its own event loop, so an event can arrive at any moment —
// even after the game loop has already read the input snapshot for the current tick. Stamping such
// an event with the live tick would attribute its edge to a tick that was already sampled, so a key
// pressed and released between two samples would go unnoticed (#3317). The handlers queue the event
// here and (*UserInterface).applyInputEvents stamps it with InputTime when the game loop drains the
// queue, so the edge is always attributed to the tick that observes it.
type inputButtonEvent struct {
	key     Key
	button  MouseButton
	pressed bool

	// isKey distinguishes a key event (key) from a mouse-button event (button).
	isKey bool
}

func (u *UserInterface) enqueueKeys(key0, key1 Key, pressed bool) {
	if key0 < 0 && key1 < 0 {
		return
	}
	u.inputMu.Lock()
	defer u.inputMu.Unlock()
	if key0 >= 0 {
		u.inputEvents = append(u.inputEvents, inputButtonEvent{key: key0, pressed: pressed, isKey: true})
	}
	if key1 >= 0 {
		u.inputEvents = append(u.inputEvents, inputButtonEvent{key: key1, pressed: pressed, isKey: true})
	}
}

func (u *UserInterface) enqueueMouseButton(button MouseButton, pressed bool) {
	if button < 0 || MouseButtonMax < button {
		return
	}
	u.inputMu.Lock()
	defer u.inputMu.Unlock()
	u.inputEvents = append(u.inputEvents, inputButtonEvent{button: button, pressed: pressed})
}

func (u *UserInterface) keyDown(event js.Value) {
	// Ignore key repeats for now.
	if event.Get("repeat").Bool() {
		return
	}
	key0, key1 := eventToKeys(event)
	u.enqueueKeys(key0, key1, true)
}

func (u *UserInterface) keyUp(event js.Value) {
	key0, key1 := eventToKeys(event)
	u.enqueueKeys(key0, key1, false)
}

func (u *UserInterface) mouseDown(code int) {
	u.enqueueMouseButton(codeToMouseButton[code], true)
}

func (u *UserInterface) mouseUp(code int) {
	u.enqueueMouseButton(codeToMouseButton[code], false)
}

// applyInputEvents folds the queued key and mouse-button events into the input state, stamping each
// with the current input time so the edge belongs to the tick that reads it (#3317). The caller must
// hold inputMu.
func (u *UserInterface) applyInputEvents() {
	for _, e := range u.inputEvents {
		t := u.InputTime()
		if e.isKey {
			if e.pressed {
				u.inputState.setKeyPressed(e.key, t)
			} else {
				u.inputState.setKeyReleased(e.key, t)
			}
			continue
		}
		if e.pressed {
			u.inputState.setMouseButtonPressed(e.button, t)
		} else {
			u.inputState.setMouseButtonReleased(e.button, t)
		}
	}
	u.inputEvents = u.inputEvents[:0]

	if u.releaseAllPending {
		u.inputState.releaseAllButtons(u.InputTime())
		u.releaseAllPending = false
	}
}

func (u *UserInterface) updateInputFromEvent(e js.Value) error {
	// Avoid using js.Value.String() as String creates a Uint8Array via a TextEncoder and causes a heavy
	// overhead (#1437).
	t := e.Get("type")
	switch {
	case t.Equal(stringKeydown):
		if str := e.Get("key").String(); isKeyString(str) {
			u.inputMu.Lock()
			for _, r := range str {
				u.inputState.appendRune(r)
			}
			u.inputMu.Unlock()
		}
		u.keyDown(e)
	case t.Equal(stringKeyup):
		u.keyUp(e)
	case t.Equal(stringMousedown):
		u.mouseDown(e.Get("button").Int())
		u.setMouseCursorFromEvent(e)
	case t.Equal(stringMouseup):
		u.mouseUp(e.Get("button").Int())
		u.setMouseCursorFromEvent(e)
	case t.Equal(stringMousemove):
		u.setMouseCursorFromEvent(e)
	case t.Equal(stringWheel):
		// TODO: What if e.deltaMode is not DOM_DELTA_PIXEL?
		dx := -e.Get("deltaX").Float()
		dy := -e.Get("deltaY").Float()
		u.inputMu.Lock()
		u.inputState.WheelX += dx
		u.inputState.WheelY += dy
		u.inputMu.Unlock()
	case t.Equal(stringTouchstart) || t.Equal(stringTouchend) || t.Equal(stringTouchmove):
		u.updateTouchesFromEvent(e)
	}

	// A browser reports the lock key states only as part of an input event. KeyboardEvent and
	// MouseEvent carry them; TouchEvent does not.
	switch {
	case t.Equal(stringKeydown), t.Equal(stringKeyup), t.Equal(stringMousedown), t.Equal(stringMouseup),
		t.Equal(stringMousemove), t.Equal(stringWheel):
		capsLock := NewLockKeyStateFromBool(e.Call("getModifierState", stringCapsLock).Bool())
		numLock := NewLockKeyStateFromBool(e.Call("getModifierState", stringNumLock).Bool())
		u.inputMu.Lock()
		u.inputState.CapsLock = capsLock
		u.inputState.NumLock = numLock
		u.inputMu.Unlock()
	}

	u.forceUpdateOnMinimumFPSMode()
	return nil
}

func (u *UserInterface) setMouseCursorFromEvent(e js.Value) {
	if u.context == nil {
		return
	}

	u.inputMu.Lock()
	defer u.inputMu.Unlock()

	u.origCursorXInClient = e.Get("clientX").Float()
	u.origCursorYInClient = e.Get("clientY").Float()

	if u.cursorMode == CursorModeCaptured {
		u.cursorXInClient += e.Get("movementX").Float()
		u.cursorYInClient += e.Get("movementY").Float()
		return
	}

	u.cursorXInClient = u.origCursorXInClient
	u.cursorYInClient = u.origCursorYInClient
}

func (u *UserInterface) recoverCursorPosition() {
	u.inputMu.Lock()
	defer u.inputMu.Unlock()
	u.cursorXInClient = u.origCursorXInClient
	u.cursorYInClient = u.origCursorYInClient
}

func (u *UserInterface) updateTouchesFromEvent(e js.Value) {
	u.inputMu.Lock()
	defer u.inputMu.Unlock()

	u.touchesInClient = u.touchesInClient[:0]

	touches := e.Get("targetTouches")
	for i := 0; i < touches.Length(); i++ {
		t := touches.Call("item", i)
		u.touchesInClient = append(u.touchesInClient, touchInClient{
			id: TouchID(t.Get("identifier").Int()),
			x:  t.Get("clientX").Float(),
			y:  t.Get("clientY").Float(),
		})
	}
}

func isKeyString(str string) bool {
	// From https://www.w3.org/TR/uievents-key/#keys-unicode,
	//
	//     A key string is a string containing a 0 or 1 non-control characters
	//     ("base" characters) followed by 0 or more combining characters. The
	//     string MUST be in Normalized Form C (NFC) as described in
	//     [UnicodeNormalizationForms].
	//
	//     A non-control character is any valid Unicode character except those
	//     that are part of the "Other, Control" ("Cc") General Category.
	//
	//     A combining character is any valid Unicode character in the "Mark,
	//     Spacing Combining" ("Mc") General Category or with a non-zero
	//     Combining Class.
	for i, r := range str {
		if i == 0 {
			if unicode.Is(unicode.Cc, r) {
				return false
			}
			continue
		}
		if !unicode.Is(unicode.Mc, r) {
			return false
		}
	}
	return true
}

var (
	jsKeyboard                          = js.Global().Get("navigator").Get("keyboard")
	jsKeyboardLayoutAvailable           bool
	jsKeyboardGetLayoutMap              js.Value
	jsKeyboardGetLayoutMapCh            chan js.Value
	jsKeyboardGetLayoutMapThenCallback  js.Func
	jsKeyboardGetLayoutMapCatchCallback js.Func
)

func init() {
	if !jsKeyboard.Truthy() {
		return
	}

	jsKeyboardGetLayoutMap = jsKeyboard.Get("getLayoutMap").Call("bind", jsKeyboard)
	jsKeyboardGetLayoutMapCh = make(chan js.Value, 1)
	jsKeyboardGetLayoutMapThenCallback = js.FuncOf(func(this js.Value, args []js.Value) any {
		jsKeyboardGetLayoutMapCh <- args[0]
		return nil
	})
	jsKeyboardGetLayoutMapCatchCallback = js.FuncOf(func(this js.Value, args []js.Value) any {
		err := args[0]
		js.Global().Get("console").Call("error", "ui: navigator.keyboard.getLayoutMap() failed:", err)
		jsKeyboardLayoutAvailable = false
		jsKeyboardGetLayoutMapCh <- js.Undefined()
		return nil
	})
	jsKeyboardLayoutAvailable = true
}

func (u *UserInterface) KeyName(key Key) string {
	if !u.isRunning() {
		return ""
	}

	if !jsKeyboardLayoutAvailable {
		return ""
	}

	// keyboardLayoutMap is reset every tick.
	if u.keyboardLayoutMap.IsUndefined() {
		// Invoke getLayoutMap every tick to detect the keyboard change.
		// TODO: Calling this every tick might be inefficient. Is there a way to detect a keyboard change?
		jsKeyboardGetLayoutMap.Invoke().Call("then", jsKeyboardGetLayoutMapThenCallback).Call("catch", jsKeyboardGetLayoutMapCatchCallback)
		u.keyboardLayoutMap = <-jsKeyboardGetLayoutMapCh
	}
	if u.keyboardLayoutMap.IsUndefined() {
		return ""
	}

	n := u.keyboardLayoutMap.Call("get", uiKeyToJSCode[key])
	if n.IsUndefined() {
		return ""
	}
	return n.String()
}

func (u *UserInterface) UpdateInputFromEvent(e js.Value) {
	u.updateInputFromEvent(e)
}

func (u *UserInterface) saveCursorPosition() {
	w, h := u.outsideSize()

	u.inputMu.Lock()
	defer u.inputMu.Unlock()

	u.savedCursorX = u.inputState.CursorX
	u.savedCursorY = u.inputState.CursorY
	u.savedOutsideWidth = w
	u.savedOutsideHeight = h
}

func (u *UserInterface) updateInputStateForFrame(deviceScaleFactor float64) error {
	s := deviceScaleFactor

	// cursorXInClient and touchesInClient are written by the browser's mouse and touch handlers, so
	// read and reset them under inputMu.
	u.inputMu.Lock()
	defer u.inputMu.Unlock()

	if !math.IsNaN(u.savedCursorX) && !math.IsNaN(u.savedCursorY) {
		// If savedCursorX and savedCursorY are valid values, the cursor is saved just before entering or exiting from fullscreen.
		// Even after entering or exiting from fullscreening, the outside (body) size is not updated for a while.
		// Wait for the outside size updated.
		if w, h := u.outsideSize(); u.savedOutsideWidth != w || u.savedOutsideHeight != h {
			u.inputState.CursorX = u.savedCursorX
			u.inputState.CursorY = u.savedCursorY
			cx, cy := u.context.logicalPositionToClientPosition(u.inputState.CursorX, u.inputState.CursorY, s)
			u.cursorXInClient = cx
			u.cursorYInClient = cy
			u.savedCursorX = math.NaN()
			u.savedCursorY = math.NaN()
			u.savedOutsideWidth = 0
			u.savedOutsideHeight = 0
			u.outsideSizeUnchangedCount = 0
		} else {
			u.outsideSizeUnchangedCount++

			// If the outside size is not changed for a while, probably the screen size is not actually changed.
			// Reset the state.
			if u.outsideSizeUnchangedCount > 60 {
				u.savedCursorX = math.NaN()
				u.savedCursorY = math.NaN()
				u.savedOutsideWidth = 0
				u.savedOutsideHeight = 0
				u.outsideSizeUnchangedCount = 0
			}
		}
	} else {
		cx, cy := u.context.clientPositionToLogicalPosition(u.cursorXInClient, u.cursorYInClient, s)
		u.inputState.CursorX = cx
		u.inputState.CursorY = cy
	}

	u.inputState.Touches = u.inputState.Touches[:0]
	for _, t := range u.touchesInClient {
		x, y := u.context.clientPositionToLogicalPosition(t.x, t.y, s)
		u.inputState.Touches = append(u.inputState.Touches, Touch{
			ID: t.id,
			X:  x,
			Y:  y,
		})
	}

	return nil
}

// uiKeyToJSKey is a map from Key values to KeyboardEvent's key values.
// Note that js.Value cannot be a map key.
//
// Reference: https://developer.mozilla.org/en-US/docs/Web/API/UI_Events/Keyboard_event_key_values
var uiKeyToJSKey = map[Key]js.Value{
	KeyCapsLock:       js.ValueOf("CapsLock"),
	KeyNumLock:        js.ValueOf("NumLock"),
	KeyScrollLock:     js.ValueOf("ScrollLock"),
	KeyEnter:          js.ValueOf("Enter"),
	KeyTab:            js.ValueOf("Tab"),
	KeySpace:          js.ValueOf(" "),
	KeyArrowDown:      js.ValueOf("ArrowDown"),
	KeyArrowLeft:      js.ValueOf("ArrowLeft"),
	KeyArrowRight:     js.ValueOf("ArrowRight"),
	KeyArrowUp:        js.ValueOf("ArrowUp"),
	KeyEnd:            js.ValueOf("End"),
	KeyHome:           js.ValueOf("Home"),
	KeyPageDown:       js.ValueOf("PageDown"),
	KeyPageUp:         js.ValueOf("PageUp"),
	KeyBackspace:      js.ValueOf("Backspace"),
	KeyDelete:         js.ValueOf("Delete"),
	KeyInsert:         js.ValueOf("Insert"),
	KeyContextMenu:    js.ValueOf("ContextMenu"),
	KeyEscape:         js.ValueOf("Escape"),
	KeyPause:          js.ValueOf("Pause"),
	KeyPrintScreen:    js.ValueOf("PrintScreen"),
	KeyF1:             js.ValueOf("F1"),
	KeyF2:             js.ValueOf("F2"),
	KeyF3:             js.ValueOf("F3"),
	KeyF4:             js.ValueOf("F4"),
	KeyF5:             js.ValueOf("F5"),
	KeyF6:             js.ValueOf("F6"),
	KeyF7:             js.ValueOf("F7"),
	KeyF8:             js.ValueOf("F8"),
	KeyF9:             js.ValueOf("F9"),
	KeyF10:            js.ValueOf("F10"),
	KeyF11:            js.ValueOf("F11"),
	KeyF12:            js.ValueOf("F12"),
	KeyF13:            js.ValueOf("F13"),
	KeyF14:            js.ValueOf("F14"),
	KeyF15:            js.ValueOf("F15"),
	KeyF16:            js.ValueOf("F16"),
	KeyF17:            js.ValueOf("F17"),
	KeyF18:            js.ValueOf("F18"),
	KeyF19:            js.ValueOf("F19"),
	KeyF20:            js.ValueOf("F20"),
	KeyNumpadDecimal:  js.ValueOf("Decimal"),
	KeyNumpadMultiply: js.ValueOf("Multiply"),
	KeyNumpadAdd:      js.ValueOf("Add"),
	KeyNumpadDivide:   js.ValueOf("Divide"),
	KeyNumpadSubtract: js.ValueOf("Subtract"),
	KeyNumpad0:        js.ValueOf("0"),
	KeyNumpad1:        js.ValueOf("1"),
	KeyNumpad2:        js.ValueOf("2"),
	KeyNumpad3:        js.ValueOf("3"),
	KeyNumpad4:        js.ValueOf("4"),
	KeyNumpad5:        js.ValueOf("5"),
	KeyNumpad6:        js.ValueOf("6"),
	KeyNumpad7:        js.ValueOf("7"),
	KeyNumpad8:        js.ValueOf("8"),
	KeyNumpad9:        js.ValueOf("9"),
}
