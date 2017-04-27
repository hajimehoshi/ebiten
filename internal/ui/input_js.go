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
	"sync"

	"github.com/gopherjs/gopherjs/js"
)

type input struct {
	keyPressed         map[string]bool
	keyPressedSafari   map[int]bool
	mouseButtonPressed map[int]bool
	cursorX            int
	cursorY            int
	gamepads           [16]gamePad
	touches            []touch
	m                  sync.RWMutex
}

func (i *input) IsKeyPressed(key Key) bool {
	i.m.RLock()
	defer i.m.RUnlock()
	if i.keyPressed != nil {
		for _, c := range keyToCodes[key] {
			if i.keyPressed[c] {
				return true
			}
		}
	}
	if i.keyPressedSafari != nil {
		for c, k := range keyCodeToKeySafari {
			if k != key {
				continue
			}
			if i.keyPressedSafari[c] {
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

func (i *input) IsMouseButtonPressed(button MouseButton) bool {
	i.m.RLock()
	defer i.m.RUnlock()
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

func (i *input) keyDown(code string) {
	i.m.Lock()
	defer i.m.Unlock()
	if i.keyPressed == nil {
		i.keyPressed = map[string]bool{}
	}
	i.keyPressed[code] = true
}

func (i *input) keyUp(code string) {
	i.m.Lock()
	defer i.m.Unlock()
	if i.keyPressed == nil {
		i.keyPressed = map[string]bool{}
	}
	i.keyPressed[code] = false
}

func (i *input) keyDownSafari(code int) {
	i.m.Lock()
	defer i.m.Unlock()
	if i.keyPressedSafari == nil {
		i.keyPressedSafari = map[int]bool{}
	}
	i.keyPressedSafari[code] = true
}

func (i *input) keyUpSafari(code int) {
	i.m.Lock()
	defer i.m.Unlock()
	if i.keyPressedSafari == nil {
		i.keyPressedSafari = map[int]bool{}
	}
	i.keyPressedSafari[code] = false
}

func (i *input) mouseDown(code int) {
	i.m.Lock()
	defer i.m.Unlock()
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	i.mouseButtonPressed[code] = true
}

func (i *input) mouseUp(code int) {
	i.m.Lock()
	defer i.m.Unlock()
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	i.mouseButtonPressed[code] = false
}

func (i *input) setMouseCursor(x, y int) {
	i.m.Lock()
	defer i.m.Unlock()
	i.cursorX, i.cursorY = x, y
}

func (i *input) updateGamepads() {
	i.m.Lock()
	defer i.m.Unlock()
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

func (i *input) updateTouches(t []touch) {
	i.m.Lock()
	defer i.m.Unlock()
	i.touches = make([]touch, len(t))
	copy(i.touches, t)
}
