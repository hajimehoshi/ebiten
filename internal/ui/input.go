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

var currentInput = &input{}

type Input interface {
	IsKeyPressed(key Key) bool
	IsMouseButtonPressed(button MouseButton) bool
	CursorPosition() (x, y int)
	GamepadAxis(id int, axis int) float64
	GamepadAxisNum(id int) int
	GamepadButtonNum(id int) int
	IsGamepadButtonPressed(id int, button GamepadButton) bool
	Touches() []Touch
}

type Touch interface {
	ID() int
	Position() (x, y int)
}

func CurrentInput() Input {
	return currentInput
}

func (i *input) CursorPosition() (x, y int) {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.cursorX, i.cursorY
}

func (i *input) GamepadAxisNum(id int) int {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return 0
	}
	return i.gamepads[id].axisNum
}

func (i *input) GamepadAxis(id int, axis int) float64 {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return 0
	}
	return i.gamepads[id].axes[axis]
}

func (i *input) GamepadButtonNum(id int) int {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return 0
	}
	return i.gamepads[id].buttonNum
}

func (i *input) IsGamepadButtonPressed(id int, button GamepadButton) bool {
	i.m.RLock()
	defer i.m.RUnlock()
	if len(i.gamepads) <= id {
		return false
	}
	return i.gamepads[id].buttonPressed[button]
}

func (in *input) Touches() []Touch {
	in.m.RLock()
	defer in.m.RUnlock()
	t := make([]Touch, len(in.touches))
	for i := 0; i < len(t); i++ {
		t[i] = &in.touches[i]
	}
	return t
}

type gamePad struct {
	axisNum       int
	axes          [16]float64
	buttonNum     int
	buttonPressed [256]bool
}

type touch struct {
	id int
	x  int
	y  int
}

func (t *touch) ID() int {
	return t.id
}

func (t *touch) Position() (x, y int) {
	return t.x, t.y
}
