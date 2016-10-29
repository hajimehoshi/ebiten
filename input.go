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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/internal/ui"
)

// IsKeyPressed returns a boolean indicating whether key is pressed.
//
// This function is concurrent-safe.
func IsKeyPressed(key Key) bool {
	return ui.CurrentInput().IsKeyPressed(ui.Key(key))
}

// CursorPosition returns a position of a mouse cursor.
//
// This function is concurrent-safe.
func CursorPosition() (x, y int) {
	return ui.CurrentInput().CursorPosition()
}

// IsMouseButtonPressed returns a boolean indicating whether mouseButton is pressed.
//
// This function is concurrent-safe.
//
// Note that touch events not longer affect this function's result as of 1.4.0-alpha.
// Use Touches instead.
func IsMouseButtonPressed(mouseButton MouseButton) bool {
	return ui.CurrentInput().IsMouseButtonPressed(ui.MouseButton(mouseButton))
}

// GamepadAxisNum returns the number of axes of the gamepad.
//
// This function is concurrent-safe.
//
// NOTE: Gamepad API is available only on desktops, Chrome and Firefox.
// To use this API, browsers might require rebooting the browser.
func GamepadAxisNum(id int) int {
	return ui.CurrentInput().GamepadAxisNum(id)
}

// GamepadAxis returns the float value [-1.0 - 1.0] of the axis.
//
// This function is concurrent-safe.
//
// NOTE: Gamepad API is available only on desktops, Chrome and Firefox.
// To use this API, browsers might require rebooting the browser.
func GamepadAxis(id int, axis int) float64 {
	return ui.CurrentInput().GamepadAxis(id, axis)
}

// GamepadButtonNum returns the number of the buttons of the gamepad.
//
// This function is concurrent-safe.
//
// NOTE: Gamepad API is available only on desktops, Chrome and Firefox.
// To use this API, browsers might require rebooting the browser.
func GamepadButtonNum(id int) int {
	return ui.CurrentInput().GamepadButtonNum(id)
}

// IsGamepadButtonPressed returns the boolean indicating the buttons is pressed or not.
//
// This function is concurrent-safe.
//
// The key states vary depending on environments.
// There can be differences even between Chrome and Firefox.
// Don't assume that states of a keys are always same when same buttons are pressed.
//
// NOTE: Gamepad API is available only on desktops, Chrome and Firefox.
// To use this API, browsers might require rebooting the browser.
func IsGamepadButtonPressed(id int, button GamepadButton) bool {
	return ui.CurrentInput().IsGamepadButtonPressed(id, ui.GamepadButton(button))
}

// Touch represents a pointer state.
type Touch interface {
	ID() int
	Position() (x, y int)
}

// Touches returns the current touch states.
func Touches() []Touch {
	t := ui.CurrentInput().Touches()
	tt := make([]Touch, len(t))
	for i := 0; i < len(tt); i++ {
		tt[i] = t[i]
	}
	return tt
}
