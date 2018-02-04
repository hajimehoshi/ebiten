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

// +build example

package blocks

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/inpututil"
)

// Input manages the input state including gamepads and keyboards.
type Input struct {
	virtualGamepadButtonStates map[virtualGamepadButton]int
	gamepadConfig              gamepadConfig
}

// IsAnyGamepadButtonPressed returns a boolean value indicating
// whether any gamepad button is pressed.
func (i *Input) IsAnyGamepadButtonPressed() bool {
	const gamepadID = 0
	for b := ebiten.GamepadButton(0); b <= ebiten.GamepadButtonMax; b++ {
		if ebiten.IsGamepadButtonPressed(gamepadID, b) {
			return true
		}
	}
	return false
}

func (i *Input) stateForVirtualGamepadButton(b virtualGamepadButton) int {
	if i.virtualGamepadButtonStates == nil {
		return 0
	}
	return i.virtualGamepadButtonStates[b]
}

func (i *Input) Update() {
	if i.virtualGamepadButtonStates == nil {
		i.virtualGamepadButtonStates = map[virtualGamepadButton]int{}
	}
	for _, b := range virtualGamepadButtons {
		if !i.gamepadConfig.IsButtonPressed(b) {
			i.virtualGamepadButtonStates[b] = 0
			continue
		}
		i.virtualGamepadButtonStates[b]++
	}
}

func (i *Input) IsRotateRightJustPressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyX) {
		return true
	}
	return i.stateForVirtualGamepadButton(virtualGamepadButtonButtonB) == 1
}

func (i *Input) IsRotateLeftJustPressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		return true
	}
	return i.stateForVirtualGamepadButton(virtualGamepadButtonButtonA) == 1
}

func (i *Input) StateForLeft() int {
	if v := inpututil.KeyPressDuration(ebiten.KeyLeft); 0 < v {
		return v
	}
	return i.stateForVirtualGamepadButton(virtualGamepadButtonLeft)
}

func (i *Input) StateForRight() int {
	if v := inpututil.KeyPressDuration(ebiten.KeyRight); 0 < v {
		return v
	}
	return i.stateForVirtualGamepadButton(virtualGamepadButtonRight)
}

func (i *Input) StateForDown() int {
	if v := inpututil.KeyPressDuration(ebiten.KeyDown); 0 < v {
		return v
	}
	return i.stateForVirtualGamepadButton(virtualGamepadButtonDown)
}
