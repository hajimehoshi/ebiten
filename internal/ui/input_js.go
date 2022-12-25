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

	if u.cursorMode == CursorModeCaptured {
		x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
		u.origCursorX, u.origCursorY = x, y
		s := u.DeviceScaleFactor()
		dx, dy := e.Get("movementX").Float()/s, e.Get("movementY").Float()/s
		// TODO: Keep float64 values.
		u.inputState.CursorX += int(dx)
		u.inputState.CursorY += int(dy)
		return
	}

	x, y := u.context.clientPositionToLogicalPosition(e.Get("clientX").Float(), e.Get("clientY").Float(), u.DeviceScaleFactor())
	u.inputState.CursorX, u.inputState.CursorY = int(x), int(y)
	u.origCursorX, u.origCursorY = int(x), int(y)
}

func (u *userInterfaceImpl) recoverCursorPosition() {
	u.inputState.CursorX, u.inputState.CursorY = u.origCursorX, u.origCursorY
}

func (u *userInterfaceImpl) updateTouchesFromEvent(e js.Value) {
	u.inputState.Touches = u.inputState.Touches[:0]

	touches := e.Get("targetTouches")
	for i := 0; i < touches.Length(); i++ {
		t := touches.Call("item", i)
		x, y := u.context.clientPositionToLogicalPosition(t.Get("clientX").Float(), t.Get("clientY").Float(), u.DeviceScaleFactor())
		u.inputState.Touches = append(u.inputState.Touches, Touch{
			ID: TouchID(t.Get("identifier").Int()),
			X:  int(x),
			Y:  int(y),
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
