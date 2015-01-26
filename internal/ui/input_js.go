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
	"github.com/gopherjs/gopherjs/js"
)

func CurrentInput() *input {
	return &currentInput
}

func (i *input) KeyDown(key int) {
	k, ok := keyCodeToKey[key]
	if !ok {
		return
	}
	i.keyPressed[k] = true
}

func (i *input) KeyUp(key int) {
	k, ok := keyCodeToKey[key]
	if !ok {
		return
	}
	i.keyPressed[k] = false
}

func (i *input) MouseDown(button int) {
	p := &i.mouseButtonPressed
	switch button {
	case 0:
		p[MouseButtonLeft] = true
	case 1:
		p[MouseButtonMiddle] = true
	case 2:
		p[MouseButtonRight] = true
	}
}

func (i *input) MouseUp(button int) {
	p := &i.mouseButtonPressed
	switch button {
	case 0:
		p[MouseButtonLeft] = false
	case 1:
		p[MouseButtonMiddle] = false
	case 2:
		p[MouseButtonRight] = false
	}
}

func (i *input) SetMouseCursor(x, y int) {
	i.cursorX, i.cursorY = x, y
}

func (i *input) UpdateGamepads() {
	nav := js.Global.Get("navigator")
	if nav.Get("getGamepads") == js.Undefined {
		return
	}
	gamepads := nav.Call("getGamepads")
	l := gamepads.Get("length").Int()
	for id := 0; id < l; id++ {
		gamepad := gamepads.Index(id)
		if gamepad == js.Undefined || gamepad == nil {
			continue
		}

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
