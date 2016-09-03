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

var gamepadAbstractButtons = []abstractButton{
	abstractButtonLeft,
	abstractButtonRight,
	abstractButtonDown,
	abstractButtonButtonA,
	abstractButtonButtonB,
}

type Input struct {
	keyStates                   [256]int
	gamepadButtonStates         map[ebiten.GamepadButton]int
	gamepadAbstractButtonStates map[abstractButton]int
	gamepadConfig               gamepadConfig
}

func (i *Input) StateForKey(key ebiten.Key) int {
	return i.keyStates[key]
}

func (i *Input) StateForGamepadButton(b ebiten.GamepadButton) int {
	if i.gamepadButtonStates == nil {
		return 0
	}
	return i.gamepadButtonStates[b]
}

func (i *Input) stateForGamepadAbstractButton(b abstractButton) int {
	if i.gamepadAbstractButtonStates == nil {
		return 0
	}
	return i.gamepadAbstractButtonStates[b]
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
	if i.gamepadButtonStates == nil {
		i.gamepadButtonStates = map[ebiten.GamepadButton]int{}
	}
	for b := ebiten.GamepadButton(0); b <= ebiten.GamepadButtonMax; b++ {
		if !ebiten.IsGamepadButtonPressed(gamepadID, b) {
			i.gamepadButtonStates[b] = 0
			continue
		}
		i.gamepadButtonStates[b]++
	}

	if i.gamepadAbstractButtonStates == nil {
		i.gamepadAbstractButtonStates = map[abstractButton]int{}
	}
	for _, b := range gamepadAbstractButtons {
		if !i.gamepadConfig.IsButtonPressed(gamepadID, b) {
			i.gamepadAbstractButtonStates[b] = 0
			continue
		}
		i.gamepadAbstractButtonStates[b]++
	}
}

func (i *Input) IsRotateRightTrigger() bool {
	if i.StateForKey(ebiten.KeySpace) == 1 || i.StateForKey(ebiten.KeyX) == 1 {
		return true
	}
	return i.stateForGamepadAbstractButton(abstractButtonButtonB) == 1
}

func (i *Input) IsRotateLeftTrigger() bool {
	if i.StateForKey(ebiten.KeyZ) == 1 {
		return true
	}
	return i.stateForGamepadAbstractButton(abstractButtonButtonA) == 1
}

func (i *Input) StateForLeft() int {
	v := i.StateForKey(ebiten.KeyLeft)
	if 0 < v {
		return v
	}
	return i.stateForGamepadAbstractButton(abstractButtonLeft)
}

func (i *Input) StateForRight() int {
	v := i.StateForKey(ebiten.KeyRight)
	if 0 < v {
		return v
	}
	return i.stateForGamepadAbstractButton(abstractButtonRight)
}

func (i *Input) StateForDown() int {
	v := i.StateForKey(ebiten.KeyDown)
	if 0 < v {
		return v
	}
	return i.stateForGamepadAbstractButton(abstractButtonDown)
}
