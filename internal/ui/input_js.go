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
	stringKeypress   = js.ValueOf("keypress")
	stringKeyup      = js.ValueOf("keyup")
	stringMousedown  = js.ValueOf("mousedown")
	stringMouseup    = js.ValueOf("mouseup")
	stringMousemove  = js.ValueOf("mousemove")
	stringWheel      = js.ValueOf("wheel")
	stringTouchstart = js.ValueOf("touchstart")
	stringTouchend   = js.ValueOf("touchend")
	stringTouchmove  = js.ValueOf("touchmove")
)

var jsKeys []js.Value

func init() {
	for _, k := range uiKeyToJSKey {
		jsKeys = append(jsKeys, k)
	}
}

func jsKeyToID(key js.Value) int {
	// js.Value cannot be used as a map key.
	// As the number of keys is around 100, just a dumb loop should work.
	for i, k := range jsKeys {
		if k.Equal(key) {
			return i
		}
	}
	return -1
}

type pos struct {
	X int
	Y int
}

type Input struct {
	keyPressed         map[int]bool
	keyPressedEdge     map[int]bool
	mouseButtonPressed map[int]bool
	cursorX            int
	cursorY            int
	origCursorX        int
	origCursorY        int
	wheelX             float64
	wheelY             float64
	touches            map[TouchID]pos
	runeBuffer         []rune
	ui                 *userInterfaceImpl
}

func (i *Input) CursorPosition() (x, y int) {
	if i.ui.context == nil {
		return 0, 0
	}
	xf, yf := i.ui.context.adjustPosition(float64(i.cursorX), float64(i.cursorY), i.ui.DeviceScaleFactor())
	return int(xf), int(yf)
}

func (i *Input) AppendTouchIDs(touchIDs []TouchID) []TouchID {
	for id := range i.touches {
		touchIDs = append(touchIDs, id)
	}
	return touchIDs
}

func (i *Input) TouchPosition(id TouchID) (x, y int) {
	d := i.ui.DeviceScaleFactor()
	for tid, pos := range i.touches {
		if id == tid {
			x, y := i.ui.context.adjustPosition(float64(pos.X), float64(pos.Y), d)
			return int(x), int(y)
		}
	}
	return 0, 0
}

func (i *Input) AppendInputChars(runes []rune) []rune {
	return append(runes, i.runeBuffer...)
}

func (i *Input) resetForTick() {
	i.runeBuffer = nil
	i.wheelX = 0
	i.wheelY = 0
}

func (i *Input) IsKeyPressed(key Key) bool {
	if i.keyPressed != nil {
		if i.keyPressed[jsKeyToID(uiKeyToJSKey[key])] {
			return true
		}
	}
	if i.keyPressedEdge != nil {
		for c, k := range edgeKeyCodeToUIKey {
			if k != key {
				continue
			}
			if i.keyPressedEdge[c] {
				return true
			}
		}
	}
	return false
}

var codeToMouseButton = map[int]MouseButton{
	0: MouseButtonLeft,
	1: MouseButtonMiddle,
	2: MouseButtonRight,
}

func (i *Input) IsMouseButtonPressed(button MouseButton) bool {
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	for c, b := range codeToMouseButton {
		if b != button {
			continue
		}
		if i.mouseButtonPressed[c] {
			return true
		}
	}
	return false
}

func (i *Input) Wheel() (xoff, yoff float64) {
	return i.wheelX, i.wheelY
}

func (i *Input) keyDown(code js.Value) {
	if i.keyPressed == nil {
		i.keyPressed = map[int]bool{}
	}
	i.keyPressed[jsKeyToID(code)] = true
}

func (i *Input) keyUp(code js.Value) {
	if i.keyPressed == nil {
		i.keyPressed = map[int]bool{}
	}
	i.keyPressed[jsKeyToID(code)] = false
}

func (i *Input) keyDownEdge(code int) {
	if i.keyPressedEdge == nil {
		i.keyPressedEdge = map[int]bool{}
	}
	i.keyPressedEdge[code] = true
}

func (i *Input) keyUpEdge(code int) {
	if i.keyPressedEdge == nil {
		i.keyPressedEdge = map[int]bool{}
	}
	i.keyPressedEdge[code] = false
}

func (i *Input) mouseDown(code int) {
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	i.mouseButtonPressed[code] = true
}

func (i *Input) mouseUp(code int) {
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	i.mouseButtonPressed[code] = false
}

func (i *Input) updateFromEvent(e js.Value) {
	// Avoid using js.Value.String() as String creates a Uint8Array via a TextEncoder and causes a heavy
	// overhead (#1437).
	switch t := e.Get("type"); {
	case t.Equal(stringKeydown):
		c := e.Get("code")
		if c.Type() != js.TypeString {
			code := e.Get("keyCode").Int()
			if edgeKeyCodeToUIKey[code] == KeyArrowUp ||
				edgeKeyCodeToUIKey[code] == KeyArrowDown ||
				edgeKeyCodeToUIKey[code] == KeyArrowLeft ||
				edgeKeyCodeToUIKey[code] == KeyArrowRight ||
				edgeKeyCodeToUIKey[code] == KeyBackspace ||
				edgeKeyCodeToUIKey[code] == KeyTab {
				e.Call("preventDefault")
			}
			i.keyDownEdge(code)
			return
		}
		if c.Equal(uiKeyToJSKey[KeyArrowUp]) ||
			c.Equal(uiKeyToJSKey[KeyArrowDown]) ||
			c.Equal(uiKeyToJSKey[KeyArrowLeft]) ||
			c.Equal(uiKeyToJSKey[KeyArrowRight]) ||
			c.Equal(uiKeyToJSKey[KeyBackspace]) ||
			c.Equal(uiKeyToJSKey[KeyTab]) {
			e.Call("preventDefault")
		}
		i.keyDown(c)
	case t.Equal(stringKeypress):
		if r := rune(e.Get("charCode").Int()); unicode.IsPrint(r) {
			i.runeBuffer = append(i.runeBuffer, r)
		}
	case t.Equal(stringKeyup):
		if e.Get("code").Type() != js.TypeString {
			// Assume that UA is Edge.
			code := e.Get("keyCode").Int()
			i.keyUpEdge(code)
			return
		}
		i.keyUp(e.Get("code"))
	case t.Equal(stringMousedown):
		button := e.Get("button").Int()
		i.mouseDown(button)
		i.setMouseCursorFromEvent(e)
	case t.Equal(stringMouseup):
		button := e.Get("button").Int()
		i.mouseUp(button)
		i.setMouseCursorFromEvent(e)
	case t.Equal(stringMousemove):
		i.setMouseCursorFromEvent(e)
	case t.Equal(stringWheel):
		// TODO: What if e.deltaMode is not DOM_DELTA_PIXEL?
		i.wheelX = -e.Get("deltaX").Float()
		i.wheelY = -e.Get("deltaY").Float()
	case t.Equal(stringTouchstart) || t.Equal(stringTouchend) || t.Equal(stringTouchmove):
		i.updateTouchesFromEvent(e)
	}

	i.ui.forceUpdateOnMinimumFPSMode()
}

func (i *Input) setMouseCursorFromEvent(e js.Value) {
	if i.ui.cursorMode == CursorModeCaptured {
		x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
		i.origCursorX, i.origCursorY = x, y
		dx, dy := e.Get("movementX").Int(), e.Get("movementY").Int()
		i.cursorX += dx
		i.cursorY += dy
		return
	}

	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
	i.cursorX, i.cursorY = x, y
	i.origCursorX, i.origCursorY = x, y
}

func (i *Input) recoverCursorPosition() {
	i.cursorX, i.cursorY = i.origCursorX, i.origCursorY
}

func (in *Input) updateTouchesFromEvent(e js.Value) {
	j := e.Get("targetTouches")
	for k := range in.touches {
		delete(in.touches, k)
	}
	for i := 0; i < j.Length(); i++ {
		jj := j.Call("item", i)
		id := TouchID(jj.Get("identifier").Int())
		if in.touches == nil {
			in.touches = map[TouchID]pos{}
		}
		in.touches[id] = pos{
			X: jj.Get("clientX").Int(),
			Y: jj.Get("clientY").Int(),
		}
	}
}

func (i *Input) updateForGo2Cpp() {
	if !go2cpp.Truthy() {
		return
	}

	for k := range i.touches {
		delete(i.touches, k)
	}
	touchCount := go2cpp.Get("touchCount").Int()
	for idx := 0; idx < touchCount; idx++ {
		id := go2cpp.Call("getTouchId", idx)
		x := go2cpp.Call("getTouchX", idx)
		y := go2cpp.Call("getTouchY", idx)
		if i.touches == nil {
			i.touches = map[TouchID]pos{}
		}
		i.touches[TouchID(id.Int())] = pos{
			X: x.Int(),
			Y: y.Int(),
		}
	}
}
