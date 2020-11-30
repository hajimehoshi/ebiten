// Copyright 2019 The Ebiten Authors
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

package driver

type GamepadID int

type TouchID int

type Input interface {
	CursorPosition() (x, y int)
	GamepadSDLID(id GamepadID) string
	GamepadName(id GamepadID) string
	GamepadAxis(id GamepadID, axis int) float64
	GamepadAxisNum(id GamepadID) int
	GamepadButtonNum(id GamepadID) int
	GamepadIDs() []GamepadID
	IsGamepadButtonPressed(id GamepadID, button GamepadButton) bool
	IsKeyPressed(key Key) bool
	IsMouseButtonPressed(button MouseButton) bool
	RuneBuffer() []rune
	TouchIDs() []TouchID
	TouchPosition(id TouchID) (x, y int)
	Wheel() (xoff, yoff float64)
}
