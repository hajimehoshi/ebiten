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

func jsKeyToID(key js.Value) Key {
	// js.Value cannot be used as a map key.
	// As the number of keys is around 100, just a dumb loop should work.
	for uiKey, jsKey := range uiKeyToJSKey {
		if jsKey.Equal(key) {
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

func (u *userInterfaceImpl) keyDown(code js.Value) {
	id := jsKeyToID(code)
	if id < 0 {
		return
	}
	u.inputState.KeyPressed[id] = true
}

func (u *userInterfaceImpl) keyUp(code js.Value) {
	id := jsKeyToID(code)
	if id < 0 {
		return
	}
	u.inputState.KeyPressed[id] = false
}

func (u *userInterfaceImpl) mouseDown(code int) {
	u.inputState.MouseButtonPressed[codeToMouseButton[code]] = true
}

func (u *userInterfaceImpl) mouseUp(code int) {
	u.inputState.MouseButtonPressed[codeToMouseButton[code]] = false
}

func (u *userInterfaceImpl) updateInputFromEvent(e js.Value) error {
	// Avoid using js.Value.String() as String creates a Uint8Array via a TextEncoder and causes a heavy
	// overhead (#1437).
	switch t := e.Get("type"); {
	case t.Equal(stringKeydown):
		if str := e.Get("key").String(); isKeyString(str) {
			for _, r := range str {
				u.inputState.appendRune(r)
			}
		}
		u.keyDown(e.Get("code"))
	case t.Equal(stringKeyup):
		u.keyUp(e.Get("code"))
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

func (u *userInterfaceImpl) setMouseCursorFromEvent(e js.Value) {
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

func (u *userInterfaceImpl) recoverCursorPosition() {
	u.cursorXInClient = u.origCursorXInClient
	u.cursorYInClient = u.origCursorYInClient
}

func (u *userInterfaceImpl) updateTouchesFromEvent(e js.Value) {
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
	jsKeyboard                     = js.Global().Get("navigator").Get("keyboard")
	jsKeyboardGetLayoutMap         js.Value
	jsKeyboardGetLayoutMapCh       chan js.Value
	jsKeyboardGetLayoutMapCallback js.Func
)

func init() {
	if !jsKeyboard.Truthy() {
		return
	}

	jsKeyboardGetLayoutMap = jsKeyboard.Get("getLayoutMap").Call("bind", jsKeyboard)
	jsKeyboardGetLayoutMapCh = make(chan js.Value, 1)
	jsKeyboardGetLayoutMapCallback = js.FuncOf(func(this js.Value, args []js.Value) any {
		jsKeyboardGetLayoutMapCh <- args[0]
		return nil
	})
}

func KeyName(key Key) string {
	return theUI.keyName(key)
}

func (u *userInterfaceImpl) keyName(key Key) string {
	if !u.running {
		return ""
	}

	// keyboardLayoutMap is reset every tick.
	if u.keyboardLayoutMap.IsUndefined() {
		if !jsKeyboard.Truthy() {
			return ""
		}

		// Invoke getLayoutMap every tick to detect the keyboard change.
		// TODO: Calling this every tick might be inefficient. Is there a way to detect a keyboard change?
		jsKeyboardGetLayoutMap.Invoke().Call("then", jsKeyboardGetLayoutMapCallback)
		u.keyboardLayoutMap = <-jsKeyboardGetLayoutMapCh
	}

	n := u.keyboardLayoutMap.Call("get", uiKeyToJSKey[key])
	if n.IsUndefined() {
		return ""
	}
	return n.String()
}

func UpdateInputFromEvent(e js.Value) {
	theUI.updateInputFromEvent(e)
}

func (u *userInterfaceImpl) saveCursorPosition() {
	u.savedCursorX = u.inputState.CursorX
	u.savedCursorY = u.inputState.CursorY
	w, h := u.outsideSize()
	u.savedOutsideWidth = w
	u.savedOutsideHeight = h
}

func (u *userInterfaceImpl) updateInputState() error {
	s := u.DeviceScaleFactor()

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
