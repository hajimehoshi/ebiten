// Copyright 2018 The Ebiten Authors
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

// Package inpututil provides utility functions of input like keyboard or mouse.
//
// Note: This package is experimental and API might be changed.
package inpututil

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/sync"
)

type inputState struct {
	keyStates           map[ebiten.Key]int
	mouseButtonStates   map[ebiten.MouseButton]int
	gamepadButtonStates map[int]map[ebiten.GamepadButton]int
	touchStates         map[int]int

	m sync.RWMutex
}

var theInputState = &inputState{
	keyStates:           map[ebiten.Key]int{},
	mouseButtonStates:   map[ebiten.MouseButton]int{},
	gamepadButtonStates: map[int]map[ebiten.GamepadButton]int{},
	touchStates:         map[int]int{},
}

func init() {
	hooks.AppendHookOnUpdate(func() error {
		theInputState.update()
		return nil
	})
}

func (i *inputState) update() {
	i.m.Lock()
	defer i.m.Unlock()

	for k := ebiten.Key(0); k < ebiten.KeyMax; k++ {
		if ebiten.IsKeyPressed(k) {
			i.keyStates[k]++
		} else {
			i.keyStates[k] = 0
		}
	}

	for _, b := range []ebiten.MouseButton{
		ebiten.MouseButtonLeft,
		ebiten.MouseButtonRight,
		ebiten.MouseButtonMiddle,
	} {
		if ebiten.IsMouseButtonPressed(b) {
			i.mouseButtonStates[b]++
		} else {
			i.mouseButtonStates[b] = 0
		}
	}

	for _, id := range ebiten.GamepadIDs() {
		if _, ok := i.gamepadButtonStates[id]; !ok {
			i.gamepadButtonStates[id] = map[ebiten.GamepadButton]int{}
		}
		n := ebiten.GamepadButtonNum(id)
		for b := ebiten.GamepadButton(0); b < ebiten.GamepadButton(n); b++ {
			if ebiten.IsGamepadButtonPressed(id, b) {
				i.gamepadButtonStates[id][b]++
			} else {
				i.gamepadButtonStates[id][b] = 0
			}
		}
	}

	ids := map[int]struct{}{}
	for _, t := range ebiten.Touches() {
		ids[t.ID()] = struct{}{}
		i.touchStates[t.ID()]++
	}
	idsToDelete := []int{}
	for id := range i.touchStates {
		if _, ok := ids[id]; !ok {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(i.touchStates, id)
	}
}

// IsKeyJustPressed returns a boolean value indicating
// whether the given key is pressed just in the current frame.
func IsKeyJustPressed(key ebiten.Key) bool {
	return KeyPressDuration(key) == 1
}

// KeyPressDuration returns how long the key is pressed in frames.
func KeyPressDuration(key ebiten.Key) int {
	theInputState.m.RLock()
	s := theInputState.keyStates[key]
	theInputState.m.RUnlock()
	return s
}

// IsMouseButtonJustPressed returns a boolean value indicating
// whether the given mouse button is pressed just in the current frame.
func IsMouseButtonJustPressed(button ebiten.MouseButton) bool {
	return MouseButtonPressDuration(button) == 1
}

// MouseButtonPressDuration returns how long the mouse button is pressed in frames.
func MouseButtonPressDuration(button ebiten.MouseButton) int {
	theInputState.m.RLock()
	s := theInputState.mouseButtonStates[button]
	theInputState.m.RUnlock()
	return s
}

// IsGamepadButtonJustPressed returns a boolean value indicating
// whether the given gamepad button of the gamepad id is pressed just in the current frame.
func IsGamepadButtonJustPressed(id int, button ebiten.GamepadButton) bool {
	return GamepadButtonPressDuration(id, button) == 1
}

// GamepadButtonPressDuration returns how long the gamepad button of the gamepad id is pressed in frames.
func GamepadButtonPressDuration(id int, button ebiten.GamepadButton) int {
	theInputState.m.RLock()
	s := 0
	if _, ok := theInputState.gamepadButtonStates[id]; ok {
		s = theInputState.gamepadButtonStates[id][button]
	}
	theInputState.m.RUnlock()
	return s
}

// IsJustTouched returns a boolean value indicating
// whether the given touch is pressed just in the current frame.
func IsJustTouched(id int) bool {
	return TouchDuration(id) == 1
}

// TouchDuration returns how long the touch remains in frames.
func TouchDuration(id int) int {
	theInputState.m.RLock()
	s := theInputState.touchStates[id]
	theInputState.m.RUnlock()
	return s
}
