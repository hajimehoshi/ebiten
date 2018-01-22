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
)

type Input struct {
	keyStates                  map[ebiten.Key]int
	anyGamepadButtonPressed    bool
	virtualGamepadButtonStates map[virtualGamepadButton]int
	gamepadConfig              gamepadConfig
}

func (i *Input) StateForKey(key ebiten.Key) int {
	if i.keyStates == nil {
		return 0
	}
	return i.keyStates[key]
}

func (i *Input) IsAnyGamepadButtonPressed() bool {
	return i.anyGamepadButtonPressed
}

func (i *Input) stateForVirtualGamepadButton(b virtualGamepadButton) int {
	if i.virtualGamepadButtonStates == nil {
		return 0
	}
	return i.virtualGamepadButtonStates[b]
}

func (i *Input) Update() {
	if i.keyStates == nil {
		i.keyStates = map[ebiten.Key]int{}
	}
	for key := ebiten.Key(0); key <= ebiten.KeyMax; key++ {
		if !ebiten.IsKeyPressed(ebiten.Key(key)) {
			i.keyStates[key] = 0
			continue
		}
		i.keyStates[key]++
	}

	const gamepadID = 0
	i.anyGamepadButtonPressed = false
	for b := ebiten.GamepadButton(0); b <= ebiten.GamepadButtonMax; b++ {
		if ebiten.IsGamepadButtonPressed(gamepadID, b) {
			i.anyGamepadButtonPressed = true
			break
		}
	}

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
	if i.StateForKey(ebiten.KeySpace) == 1 || i.StateForKey(ebiten.KeyX) == 1 {
		return true
	}
	return i.stateForVirtualGamepadButton(virtualGamepadButtonButtonB) == 1
}

func (i *Input) IsRotateLeftJustPressed() bool {
	if i.StateForKey(ebiten.KeyZ) == 1 {
		return true
	}
	return i.stateForVirtualGamepadButton(virtualGamepadButtonButtonA) == 1
}

func (i *Input) StateForLeft() int {
	v := i.StateForKey(ebiten.KeyLeft)
	if 0 < v {
		return v
	}
	return i.stateForVirtualGamepadButton(virtualGamepadButtonLeft)
}

func (i *Input) StateForRight() int {
	v := i.StateForKey(ebiten.KeyRight)
	if 0 < v {
		return v
	}
	return i.stateForVirtualGamepadButton(virtualGamepadButtonRight)
}

func (i *Input) StateForDown() int {
	v := i.StateForKey(ebiten.KeyDown)
	if 0 < v {
		return v
	}
	return i.stateForVirtualGamepadButton(virtualGamepadButtonDown)
}
