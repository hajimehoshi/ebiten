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
	"github.com/hajimehoshi/ebiten/exp/gamepad"
)

var gamepadStdButtons = []gamepad.StdButton{
	gamepad.StdButtonLL,
	gamepad.StdButtonLR,
	gamepad.StdButtonLD,
	gamepad.StdButtonRD,
	gamepad.StdButtonRR,
}

type Input struct {
	keyStates              [256]int
	gamepadButtonStates    [256]int
	gamepadStdButtonStates [16]int
	gamepadConfig          gamepad.Configuration
}

func (i *Input) StateForKey(key ebiten.Key) int {
	return i.keyStates[key]
}

func (i *Input) StateForGamepadButton(b ebiten.GamepadButton) int {
	return i.gamepadButtonStates[b]
}

func (i *Input) stateForGamepadStdButton(b gamepad.StdButton) int {
	return i.gamepadStdButtonStates[b]
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

	for _, b := range gamepadStdButtons {
		if !i.gamepadConfig.IsButtonPressed(gamepadID, b) {
			i.gamepadStdButtonStates[b] = 0
			continue
		}
		i.gamepadStdButtonStates[b]++
	}
}

func (i *Input) IsRotateRightTrigger() bool {
	if i.StateForKey(ebiten.KeySpace) == 1 || i.StateForKey(ebiten.KeyX) == 1 {
		return true
	}
	return i.stateForGamepadStdButton(gamepad.StdButtonRR) == 1
}

func (i *Input) IsRotateLeftTrigger() bool {
	if i.StateForKey(ebiten.KeyZ) == 1 {
		return true
	}
	return i.stateForGamepadStdButton(gamepad.StdButtonRD) == 1
}

func (i *Input) StateForLeft() int {
	v := i.StateForKey(ebiten.KeyLeft)
	if 0 < v {
		return v
	}
	return i.stateForGamepadStdButton(gamepad.StdButtonLL)
}

func (i *Input) StateForRight() int {
	v := i.StateForKey(ebiten.KeyRight)
	if 0 < v {
		return v
	}
	return i.stateForGamepadStdButton(gamepad.StdButtonLR)
}

func (i *Input) StateForDown() int {
	v := i.StateForKey(ebiten.KeyDown)
	if 0 < v {
		return v
	}
	return i.stateForGamepadStdButton(gamepad.StdButtonLD)
}
