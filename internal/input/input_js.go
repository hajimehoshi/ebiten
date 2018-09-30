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

package input

import (
	"unicode"

	"github.com/gopherjs/gopherwasm/js"
)

type mockRWLock struct{}

func (m mockRWLock) Lock()    {}
func (m mockRWLock) Unlock()  {}
func (m mockRWLock) RLock()   {}
func (m mockRWLock) RUnlock() {}

type Input struct {
	keyPressed         map[string]bool
	keyPressedEdge     map[int]bool
	mouseButtonPressed map[int]bool
	cursorX            int
	cursorY            int
	wheelX             float64
	wheelY             float64
	gamepads           [16]gamePad
	touches            []*Touch
	runeBuffer         []rune
	m                  mockRWLock
}

func (i *Input) RuneBuffer() []rune {
	return i.runeBuffer
}

func (i *Input) ClearRuneBuffer() {
	i.runeBuffer = nil
}

func (i *Input) ResetWheelValues() {
	i.wheelX = 0
	i.wheelY = 0
}

func (i *Input) IsKeyPressed(key Key) bool {
	if i.keyPressed != nil {
		for _, c := range keyToCodes[key] {
			if i.keyPressed[c] {
				return true
			}
		}
	}
	if i.keyPressedEdge != nil {
		for c, k := range keyCodeToKeyEdge {
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

func (i *Input) keyDown(code string) {
	if i.keyPressed == nil {
		i.keyPressed = map[string]bool{}
	}
	i.keyPressed[code] = true
}

func (i *Input) keyUp(code string) {
	if i.keyPressed == nil {
		i.keyPressed = map[string]bool{}
	}
	i.keyPressed[code] = false
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

func (i *Input) setMouseCursor(x, y int) {
	i.cursorX, i.cursorY = x, y
}

func (i *Input) UpdateGamepads() {
	nav := js.Global().Get("navigator")
	if nav.Get("getGamepads") == js.Undefined() {
		return
	}
	gamepads := nav.Call("getGamepads")
	l := gamepads.Get("length").Int()
	for id := 0; id < l; id++ {
		i.gamepads[id].valid = false
		gamepad := gamepads.Index(id)
		if gamepad == js.Undefined() || gamepad == js.Null() {
			continue
		}
		i.gamepads[id].valid = true

		axes := gamepad.Get("axes")
		axesNum := axes.Get("length").Int()
		i.gamepads[id].axisNum = axesNum
		for a := 0; a < len(i.gamepads[id].axes); a++ {
			if axesNum <= a {
				i.gamepads[id].axes[a] = 0
				continue
			}
			i.gamepads[id].axes[a] = axes.Index(a).Float()
		}

		buttons := gamepad.Get("buttons")
		buttonsNum := buttons.Get("length").Int()
		i.gamepads[id].buttonNum = buttonsNum
		for b := 0; b < len(i.gamepads[id].buttonPressed); b++ {
			if buttonsNum <= b {
				i.gamepads[id].buttonPressed[b] = false
				continue
			}
			i.gamepads[id].buttonPressed[b] = buttons.Index(b).Get("pressed").Bool()
		}
	}
}

func OnKeyDown(e js.Value) {
	c := e.Get("code")
	if c == js.Undefined() {
		code := e.Get("keyCode").Int()
		if keyCodeToKeyEdge[code] == KeyUp ||
			keyCodeToKeyEdge[code] == KeyDown ||
			keyCodeToKeyEdge[code] == KeyLeft ||
			keyCodeToKeyEdge[code] == KeyRight ||
			keyCodeToKeyEdge[code] == KeyBackspace ||
			keyCodeToKeyEdge[code] == KeyTab {
			e.Call("preventDefault")
		}
		theInput.keyDownEdge(code)
		return
	}
	cs := c.String()
	if cs == keyToCodes[KeyUp][0] ||
		cs == keyToCodes[KeyDown][0] ||
		cs == keyToCodes[KeyLeft][0] ||
		cs == keyToCodes[KeyRight][0] ||
		cs == keyToCodes[KeyBackspace][0] ||
		cs == keyToCodes[KeyTab][0] {
		e.Call("preventDefault")
	}
	theInput.keyDown(cs)
}

func OnKeyPress(e js.Value) {
	if r := rune(e.Get("charCode").Int()); unicode.IsPrint(r) {
		theInput.runeBuffer = append(theInput.runeBuffer, r)
	}
}

func OnKeyUp(e js.Value) {
	if e.Get("code") == js.Undefined() {
		// Assume that UA is Edge.
		code := e.Get("keyCode").Int()
		theInput.keyUpEdge(code)
		return
	}
	code := e.Get("code").String()
	theInput.keyUp(code)
}

func OnMouseDown(e js.Value) {
	button := e.Get("button").Int()
	theInput.mouseDown(button)
	setMouseCursorFromEvent(e)
}

func OnMouseUp(e js.Value) {
	button := e.Get("button").Int()
	theInput.mouseUp(button)
	setMouseCursorFromEvent(e)
}

func OnMouseMove(e js.Value) {
	setMouseCursorFromEvent(e)
}

func OnWheel(e js.Value) {
	// TODO: What if e.deltaMode is not DOM_DELTA_PIXEL?
	theInput.wheelX = -e.Get("deltaX").Float()
	theInput.wheelY = -e.Get("deltaY").Float()
}

func OnTouchStart(e js.Value) {
	theInput.updateTouches(e)
}

func OnTouchEnd(e js.Value) {
	theInput.updateTouches(e)
}

func OnTouchMove(e js.Value) {
	theInput.updateTouches(e)
}

func setMouseCursorFromEvent(e js.Value) {
	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
	theInput.setMouseCursor(x, y)
}

func (i *Input) updateTouches(e js.Value) {
	j := e.Get("targetTouches")
	ts := make([]*Touch, j.Get("length").Int())
	for i := 0; i < len(ts); i++ {
		jj := j.Call("item", i)
		id := jj.Get("identifier").Int()
		ts[i] = &Touch{
			id: id,
			x:  jj.Get("clientX").Int(),
			y:  jj.Get("clientY").Int(),
		}
	}
	i.touches = ts
}
