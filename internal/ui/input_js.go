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
	"strings"
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

	// On mobile browsers, treat enter key as if this is from a `key` property.
	if IsVirtualKeyboard() && id == KeyEnter {
		return KeyEnter, -1
	}
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

func (u *UserInterface) keyDown(event js.Value) {
	// Ignore key repeats for now.
	if event.Get("repeat").Bool() {
		return
	}
	now := u.InputTime()
	key0, key1 := eventToKeys(event)
	if key0 >= 0 {
		u.inputState.setKeyPressed(key0, now)
	}
	if key1 >= 0 {
		u.inputState.setKeyPressed(key1, now)
	}
}

func (u *UserInterface) keyUp(event js.Value) {
	now := u.InputTime()
	key0, key1 := eventToKeys(event)
	if key0 >= 0 {
		u.inputState.setKeyReleased(key0, now)
	}
	if key1 >= 0 {
		u.inputState.setKeyReleased(key1, now)
	}
}

func (u *UserInterface) mouseDown(code int) {
	u.inputState.setMouseButtonPressed(codeToMouseButton[code], u.InputTime())
}

func (u *UserInterface) mouseUp(code int) {
	u.inputState.setMouseButtonReleased(codeToMouseButton[code], u.InputTime())
}

func (u *UserInterface) updateInputFromEvent(e js.Value) error {
	// Avoid using js.Value.String() as String creates a Uint8Array via a TextEncoder and causes a heavy
	// overhead (#1437).
	switch t := e.Get("type"); {
	case t.Equal(stringKeydown):
		if str := e.Get("key").String(); isKeyString(str) {
			for _, r := range str {
				u.inputState.appendRune(r)
			}
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
		u.inputState.WheelX = -e.Get("deltaX").Float()
		u.inputState.WheelY = -e.Get("deltaY").Float()
	case t.Equal(stringTouchstart) || t.Equal(stringTouchend) || t.Equal(stringTouchmove):
		u.updateTouchesFromEvent(e)
	}

	u.forceUpdateOnMinimumFPSMode()
	return nil
}

func (u *UserInterface) setMouseCursorFromEvent(e js.Value) {
	if u.context == nil {
		return
	}

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
	u.cursorXInClient = u.origCursorXInClient
	u.cursorYInClient = u.origCursorYInClient
}

func (u *UserInterface) updateTouchesFromEvent(e js.Value) {
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
	u.savedCursorX = u.inputState.CursorX
	u.savedCursorY = u.inputState.CursorY
	w, h := u.outsideSize()
	u.savedOutsideWidth = w
	u.savedOutsideHeight = h
}

func (u *UserInterface) updateInputStateForFrame() error {
	s := theMonitor.DeviceScaleFactor()

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
			X:  int(x),
			Y:  int(y),
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

func IsVirtualKeyboard() bool {
	// Detect a virtual keyboard by the user agent.
	// Note that this is not a correct way to detect a virtual keyboard.
	// In the future, we should use the `navigator.virtualKeyboard` API.
	// https://developer.mozilla.org/en-US/docs/Web/API/Navigator/virtualKeyboard
	ua := js.Global().Get("navigator").Get("userAgent").String()
	if strings.Contains(ua, "Android") || strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad") || strings.Contains(ua, "iPod") {
		return true
	}
	return false
}
