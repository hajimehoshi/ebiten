// Copyright 2014 Hajime Hoshi
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

package blocks

import (
	"github.com/hajimehoshi/ebiten"
)

type Input struct {
	keyStates           [256]int
	gamepadButtonStates [4]int
}

func NewInput() *Input {
	return &Input{}
}

func (i *Input) StateForKey(key ebiten.Key) int {
	return i.keyStates[key]
}

func (i *Input) StateForGamepadButton(button ebiten.GamepadButton) int {
	return i.gamepadButtonStates[button]
}

func (i *Input) Update() {
	for key := range i.keyStates {
		if !ebiten.IsKeyPressed(ebiten.Key(key)) {
			i.keyStates[key] = 0
			continue
		}
		i.keyStates[key]++
	}

	const gamepadID = 0
	for b := range i.gamepadButtonStates {
		if !ebiten.IsGamepadButtonPressed(gamepadID, ebiten.GamepadButton(b)) {
			i.gamepadButtonStates[b] = 0
			continue
		}
		i.gamepadButtonStates[b]++
	}
}
