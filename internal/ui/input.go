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
)

var currentInput = &Input{}

type Input struct {
	keyPressed         [256]bool
	mouseButtonPressed [256]bool
	cursorX            int
	cursorY            int
	gamepads           [16]gamePad
	m                  sync.RWMutex
}

type gamePad struct {
	axisNum       int
	axes          [16]float64
	buttonNum     int
	buttonPressed [256]bool
}

func CurrentInput() *Input {
	return currentInput
}

func (i *Input) IsKeyPressed(key Key) bool {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.keyPressed[key]
}

func (i *Input) CursorPosition() (x, y int) {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.cursorX, currentInput.cursorY
}

func (i *Input) IsMouseButtonPressed(button MouseButton) bool {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.mouseButtonPressed[button]
}

func (i *Input) GamepadAxisNum(id int) int {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return 0
	}
	return i.gamepads[id].axisNum
}

func (i *Input) GamepadAxis(id int, axis int) float64 {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return 0
	}
	return i.gamepads[id].axes[axis]
}

func (i *Input) GamepadButtonNum(id int) int {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return 0
	}
	return i.gamepads[id].buttonNum
}

func (i *Input) IsGamepadButtonPressed(id int, button GamepadButton) bool {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return false
	}
	return i.gamepads[id].buttonPressed[button]
}
