// Copyright 2020 The Ebiten Authors
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

package monogame

import (
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/monogame"
)

type Input struct {
	game *monogame.Game
}

func (i *Input) CursorPosition() (x, y int) {
	return 0, 0
}

func (i *Input) GamepadSDLID(id int) string {
	return ""
}

func (i *Input) GamepadName(id int) string {
	return ""
}

func (i *Input) GamepadAxis(id int, axis int) float64 {
	return 0
}

func (i *Input) GamepadAxisNum(id int) int {
	return 0
}

func (i *Input) GamepadButtonNum(id int) int {
	return 0
}

func (i *Input) GamepadIDs() []int {
	return nil
}

func (i *Input) IsGamepadButtonPressed(id int, button driver.GamepadButton) bool {
	return false
}

func (i *Input) IsKeyPressed(key driver.Key) bool {
	return i.game.IsKeyPressed(key)
}

func (i *Input) IsMouseButtonPressed(button driver.MouseButton) bool {
	return false
}

func (i *Input) RuneBuffer() []rune {
	return nil
}

func (i *Input) TouchIDs() []int {
	return nil
}

func (i *Input) TouchPosition(id int) (x, y int) {
	return 0, 0
}

func (i *Input) Wheel() (xoff, yoff float64) {
	return 0, 0
}
