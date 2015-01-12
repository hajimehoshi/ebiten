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

func IsKeyPressed(key Key) bool {
	return currentInput.keyPressed[key]
}

func CursorPosition() (x, y int) {
	return currentInput.cursorX, currentInput.cursorY
}

func IsMouseButtonPressed(button MouseButton) bool {
	return currentInput.mouseButtonPressed[button]
}

func GamepadAxisNum(id int) int {
	if len(currentInput.gamepads) <= id {
		return 0
	}
	return currentInput.gamepads[id].axisNum
}

func GamepadAxis(id int, axis int) float64 {
	if len(currentInput.gamepads) <= id {
		return 0
	}
	return currentInput.gamepads[id].axes[axis]
}

func GamepadButtonNum(id int) int {
	if len(currentInput.gamepads) <= id {
		return 0
	}
	return currentInput.gamepads[id].buttonNum
}

func IsGamepadButtonPressed(id int, button GamepadButton) bool {
	if len(currentInput.gamepads) <= id {
		return false
	}
	return currentInput.gamepads[id].buttonPressed[button]
}

var currentInput input

type input struct {
	keyPressed         [256]bool
	mouseButtonPressed [256]bool
	cursorX            int
	cursorY            int
	gamepads           [16]gamePad
}

type gamePad struct {
	axisNum       int
	axes          [16]float64
	buttonNum     int
	buttonPressed [256]bool
}
