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

type Input interface {
	CursorPosition() (x, y int)
	GamepadSDLID(id int) string
	GamepadName(id int) string
	GamepadAxis(id int, axis int) float64
	GamepadAxisNum(id int) int
	GamepadButtonNum(id int) int
	GamepadIDs() []int
	IsGamepadButtonPressed(id int, button GamepadButton) bool
	IsKeyPressed(key Key) bool
	IsMouseButtonPressed(button MouseButton) bool
	RuneBuffer() []rune
	TouchIDs() []int
	TouchPosition(id int) (x, y int)
	Wheel() (xoff, yoff float64)
}
