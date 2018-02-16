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
	keyStates     map[ebiten.Key]int
	prevKeyStates map[ebiten.Key]int

	mouseButtonStates     map[ebiten.MouseButton]int
	prevMouseButtonStates map[ebiten.MouseButton]int

	gamepadButtonStates     map[int]map[ebiten.GamepadButton]int
	prevGamepadButtonStates map[int]map[ebiten.GamepadButton]int

	touchStates map[int]int

	m sync.RWMutex
}

var theInputState = &inputState{
	keyStates:     map[ebiten.Key]int{},
	prevKeyStates: map[ebiten.Key]int{},

	mouseButtonStates:     map[ebiten.MouseButton]int{},
	prevMouseButtonStates: map[ebiten.MouseButton]int{},

	gamepadButtonStates:     map[int]map[ebiten.GamepadButton]int{},
	prevGamepadButtonStates: map[int]map[ebiten.GamepadButton]int{},

	touchStates: map[int]int{},
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

	// Keyboard
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		i.prevKeyStates[k] = i.keyStates[k]
		if ebiten.IsKeyPressed(k) {
			i.keyStates[k]++
		} else {
			i.keyStates[k] = 0
		}
	}

	// Mouse
	for _, b := range []ebiten.MouseButton{
		ebiten.MouseButtonLeft,
		ebiten.MouseButtonRight,
		ebiten.MouseButtonMiddle,
	} {
		i.prevMouseButtonStates[b] = i.mouseButtonStates[b]
		if ebiten.IsMouseButtonPressed(b) {
			i.mouseButtonStates[b]++
		} else {
			i.mouseButtonStates[b] = 0
		}
	}

	// Gamepads

	// Reset the previous states first since some gamepad IDs might be already gone.
	for id := range i.prevGamepadButtonStates {
		for b := range i.prevGamepadButtonStates[id] {
			i.prevGamepadButtonStates[id][b] = 0
		}
	}
	ids := map[int]struct{}{}
	for _, id := range ebiten.GamepadIDs() {
		ids[id] = struct{}{}

		if _, ok := i.prevGamepadButtonStates[id]; !ok {
			i.prevGamepadButtonStates[id] = map[ebiten.GamepadButton]int{}
		}
		if _, ok := i.gamepadButtonStates[id]; !ok {
			i.gamepadButtonStates[id] = map[ebiten.GamepadButton]int{}
		}

		n := ebiten.GamepadButtonNum(id)
		for b := ebiten.GamepadButton(0); b < ebiten.GamepadButton(n); b++ {
			i.prevGamepadButtonStates[id][b] = i.gamepadButtonStates[id][b]
			if ebiten.IsGamepadButtonPressed(id, b) {
				i.gamepadButtonStates[id][b]++
			} else {
				i.gamepadButtonStates[id][b] = 0
			}
		}
	}
	idsToDelete := []int{}
	for id := range i.gamepadButtonStates {
		if _, ok := ids[id]; !ok {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(i.gamepadButtonStates, id)
	}

	// Touches
	ids = map[int]struct{}{}
	for _, t := range ebiten.Touches() {
		ids[t.ID()] = struct{}{}
		i.touchStates[t.ID()]++
	}
	idsToDelete = []int{}
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
//
// IsKeyJustPressed is concurrent safe.
func IsKeyJustPressed(key ebiten.Key) bool {
	return KeyPressDuration(key) == 1
}

// IsKeyJustReleased returns a boolean value indicating
// whether the given key is released just in the current frame.
//
// IsKeyJustReleased is concurrent safe.
func IsKeyJustReleased(key ebiten.Key) bool {
	theInputState.m.RLock()
	r := theInputState.keyStates[key] == 0 && theInputState.prevKeyStates[key] > 0
	theInputState.m.RUnlock()
	return r
}

// KeyPressDuration returns how long the key is pressed in frames.
//
// KeyPressDuration is concurrent safe.
func KeyPressDuration(key ebiten.Key) int {
	theInputState.m.RLock()
	s := theInputState.keyStates[key]
	theInputState.m.RUnlock()
	return s
}

// IsMouseButtonJustPressed returns a boolean value indicating
// whether the given mouse button is pressed just in the current frame.
//
// IsMouseButtonJustPressed is concurrent safe.
func IsMouseButtonJustPressed(button ebiten.MouseButton) bool {
	return MouseButtonPressDuration(button) == 1
}

// IsMouseButtonJustReleased returns a boolean value indicating
// whether the given mouse button is released just in the current frame.
//
// IsMouseButtonJustReleased is concurrent safe.
func IsMouseButtonJustReleased(button ebiten.MouseButton) bool {
	theInputState.m.RLock()
	r := theInputState.mouseButtonStates[button] == 0 &&
		theInputState.prevMouseButtonStates[button] > 0
	theInputState.m.RUnlock()
	return r
}

// MouseButtonPressDuration returns how long the mouse button is pressed in frames.
//
// MouseButtonPressDuration is concurrent safe.
func MouseButtonPressDuration(button ebiten.MouseButton) int {
	theInputState.m.RLock()
	s := theInputState.mouseButtonStates[button]
	theInputState.m.RUnlock()
	return s
}

// IsGamepadButtonJustPressed returns a boolean value indicating
// whether the given gamepad button of the gamepad id is pressed just in the current frame.
//
// IsGamepadButtonJustPressed is concurrent safe.
func IsGamepadButtonJustPressed(id int, button ebiten.GamepadButton) bool {
	return GamepadButtonPressDuration(id, button) == 1
}

// IsGamepadButtonJustReleased returns a boolean value indicating
// whether the given gamepad button of the gamepad id is released just in the current frame.
//
// IsGamepadButtonJustReleased is concurrent safe.
func IsGamepadButtonJustReleased(id int, button ebiten.GamepadButton) bool {
	theInputState.m.RLock()
	prev := 0
	if _, ok := theInputState.prevGamepadButtonStates[id]; ok {
		prev = theInputState.prevGamepadButtonStates[id][button]
	}
	current := 0
	if _, ok := theInputState.gamepadButtonStates[id]; ok {
		current = theInputState.gamepadButtonStates[id][button]
	}
	theInputState.m.RUnlock()
	return current == 0 && prev > 0
}

// GamepadButtonPressDuration returns how long the gamepad button of the gamepad id is pressed in frames.
//
// GamepadButtonPressDuration is concurrent safe.
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
//
// IsJustTouched is concurrent safe.
func IsJustTouched(id int) bool {
	return TouchDuration(id) == 1
}

// TouchDuration returns how long the touch remains in frames.
//
// TouchDuration is concurrent safe.
func TouchDuration(id int) int {
	theInputState.m.RLock()
	s := theInputState.touchStates[id]
	theInputState.m.RUnlock()
	return s
}
