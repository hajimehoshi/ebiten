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
package inpututil

import (
	"sort"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

type inputState struct {
	keyDurations     []int
	prevKeyDurations []int

	mouseButtonDurations     map[ebiten.MouseButton]int
	prevMouseButtonDurations map[ebiten.MouseButton]int

	gamepadIDs     map[ebiten.GamepadID]struct{}
	prevGamepadIDs map[ebiten.GamepadID]struct{}

	gamepadButtonDurations     map[ebiten.GamepadID][]int
	prevGamepadButtonDurations map[ebiten.GamepadID][]int

	touchIDs           map[ebiten.TouchID]struct{}
	touchDurations     map[ebiten.TouchID]int
	prevTouchDurations map[ebiten.TouchID]int

	m sync.RWMutex
}

var theInputState = &inputState{
	keyDurations:     make([]int, ebiten.KeyMax+1),
	prevKeyDurations: make([]int, ebiten.KeyMax+1),

	mouseButtonDurations:     map[ebiten.MouseButton]int{},
	prevMouseButtonDurations: map[ebiten.MouseButton]int{},

	gamepadIDs:     map[ebiten.GamepadID]struct{}{},
	prevGamepadIDs: map[ebiten.GamepadID]struct{}{},

	gamepadButtonDurations:     map[ebiten.GamepadID][]int{},
	prevGamepadButtonDurations: map[ebiten.GamepadID][]int{},

	touchIDs:           map[ebiten.TouchID]struct{}{},
	touchDurations:     map[ebiten.TouchID]int{},
	prevTouchDurations: map[ebiten.TouchID]int{},
}

func init() {
	hooks.AppendHookOnBeforeUpdate(func() error {
		theInputState.update()
		return nil
	})
}

func (i *inputState) update() {
	i.m.Lock()
	defer i.m.Unlock()

	// Keyboard
	copy(i.prevKeyDurations[:], i.keyDurations[:])
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if ebiten.IsKeyPressed(k) {
			i.keyDurations[k]++
		} else {
			i.keyDurations[k] = 0
		}
	}

	// Mouse
	for _, b := range []ebiten.MouseButton{
		ebiten.MouseButtonLeft,
		ebiten.MouseButtonRight,
		ebiten.MouseButtonMiddle,
	} {
		i.prevMouseButtonDurations[b] = i.mouseButtonDurations[b]
		if ebiten.IsMouseButtonPressed(b) {
			i.mouseButtonDurations[b]++
		} else {
			i.mouseButtonDurations[b] = 0
		}
	}

	// Gamepads

	// Copy the gamepad IDs.
	for id := range i.prevGamepadIDs {
		delete(i.prevGamepadIDs, id)
	}
	for id := range i.gamepadIDs {
		i.prevGamepadIDs[id] = struct{}{}
	}

	// Copy the gamepad button durations.
	for id := range i.prevGamepadButtonDurations {
		delete(i.prevGamepadButtonDurations, id)
	}
	for id, ds := range i.gamepadButtonDurations {
		i.prevGamepadButtonDurations[id] = append([]int{}, ds...)
	}

	for id := range i.gamepadIDs {
		delete(i.gamepadIDs, id)
	}
	for _, id := range ebiten.GamepadIDs() {
		i.gamepadIDs[id] = struct{}{}
		if _, ok := i.gamepadButtonDurations[id]; !ok {
			i.gamepadButtonDurations[id] = make([]int, ebiten.GamepadButtonMax+1)
		}
		n := ebiten.GamepadButtonNum(id)
		for b := ebiten.GamepadButton(0); b < ebiten.GamepadButton(n); b++ {
			if ebiten.IsGamepadButtonPressed(id, b) {
				i.gamepadButtonDurations[id][b]++
			} else {
				i.gamepadButtonDurations[id][b] = 0
			}
		}
	}
	for id := range i.gamepadButtonDurations {
		if _, ok := i.gamepadIDs[id]; !ok {
			delete(i.gamepadButtonDurations, id)
		}
	}

	// Touches

	// Copy the touch durations.
	for id := range i.prevTouchDurations {
		delete(i.prevTouchDurations, id)
	}
	for id := range i.touchDurations {
		i.prevTouchDurations[id] = i.touchDurations[id]
	}

	for id := range i.touchIDs {
		delete(i.touchIDs, id)
	}
	for _, id := range ebiten.TouchIDs() {
		i.touchIDs[id] = struct{}{}
		i.touchDurations[id]++
	}
	for id := range i.touchDurations {
		if _, ok := i.touchIDs[id]; !ok {
			delete(i.touchDurations, id)
		}
	}
}

// PressedKeys returns a set of currently pressed keyboard keys.
//
// PressedKeys is concurrent safe.
func PressedKeys() []ebiten.Key {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	keys := make([]ebiten.Key, 0, len(theInputState.keyDurations))
	for i, d := range theInputState.keyDurations {
		if d == 0 {
			continue
		}
		keys = append(keys, ebiten.Key(i))
	}
	return keys
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
	r := theInputState.keyDurations[key] == 0 && theInputState.prevKeyDurations[key] > 0
	theInputState.m.RUnlock()
	return r
}

// KeyPressDuration returns how long the key is pressed in frames.
//
// KeyPressDuration is concurrent safe.
func KeyPressDuration(key ebiten.Key) int {
	theInputState.m.RLock()
	s := theInputState.keyDurations[key]
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
	r := theInputState.mouseButtonDurations[button] == 0 &&
		theInputState.prevMouseButtonDurations[button] > 0
	theInputState.m.RUnlock()
	return r
}

// MouseButtonPressDuration returns how long the mouse button is pressed in frames.
//
// MouseButtonPressDuration is concurrent safe.
func MouseButtonPressDuration(button ebiten.MouseButton) int {
	theInputState.m.RLock()
	s := theInputState.mouseButtonDurations[button]
	theInputState.m.RUnlock()
	return s
}

// JustConnectedGamepadIDs returns gamepad IDs that are connected just in the current frame.
//
// JustConnectedGamepadIDs might return nil when there is no connected gamepad.
//
// JustConnectedGamepadIDs is concurrent safe.
func JustConnectedGamepadIDs() []ebiten.GamepadID {
	var ids []ebiten.GamepadID
	theInputState.m.RLock()
	for id := range theInputState.gamepadIDs {
		if _, ok := theInputState.prevGamepadIDs[id]; !ok {
			ids = append(ids, id)
		}
	}
	theInputState.m.RUnlock()
	sort.Slice(ids, func(a, b int) bool {
		return ids[a] < ids[b]
	})
	return ids
}

// IsGamepadJustDisconnected returns a boolean value indicating
// whether the gamepad of the given id is released just in the current frame.
//
// IsGamepadJustDisconnected is concurrent safe.
func IsGamepadJustDisconnected(id ebiten.GamepadID) bool {
	theInputState.m.RLock()
	_, prev := theInputState.prevGamepadIDs[id]
	_, current := theInputState.gamepadIDs[id]
	theInputState.m.RUnlock()
	return prev && !current
}

// IsGamepadButtonJustPressed returns a boolean value indicating
// whether the given gamepad button of the gamepad id is pressed just in the current frame.
//
// IsGamepadButtonJustPressed is concurrent safe.
func IsGamepadButtonJustPressed(id ebiten.GamepadID, button ebiten.GamepadButton) bool {
	return GamepadButtonPressDuration(id, button) == 1
}

// IsGamepadButtonJustReleased returns a boolean value indicating
// whether the given gamepad button of the gamepad id is released just in the current frame.
//
// IsGamepadButtonJustReleased is concurrent safe.
func IsGamepadButtonJustReleased(id ebiten.GamepadID, button ebiten.GamepadButton) bool {
	theInputState.m.RLock()
	prev := 0
	if _, ok := theInputState.prevGamepadButtonDurations[id]; ok {
		prev = theInputState.prevGamepadButtonDurations[id][button]
	}
	current := 0
	if _, ok := theInputState.gamepadButtonDurations[id]; ok {
		current = theInputState.gamepadButtonDurations[id][button]
	}
	theInputState.m.RUnlock()
	return current == 0 && prev > 0
}

// GamepadButtonPressDuration returns how long the gamepad button of the gamepad id is pressed in frames.
//
// GamepadButtonPressDuration is concurrent safe.
func GamepadButtonPressDuration(id ebiten.GamepadID, button ebiten.GamepadButton) int {
	theInputState.m.RLock()
	s := 0
	if _, ok := theInputState.gamepadButtonDurations[id]; ok {
		s = theInputState.gamepadButtonDurations[id][button]
	}
	theInputState.m.RUnlock()
	return s
}

// JustPressedTouchIDs returns touch IDs that are created just in the current frame.
//
// JustPressedTouchIDs might return nil when there is not touch.
//
// JustPressedTouchIDs is concurrent safe.
func JustPressedTouchIDs() []ebiten.TouchID {
	var ids []ebiten.TouchID
	theInputState.m.RLock()
	for id, s := range theInputState.touchDurations {
		if s == 1 {
			ids = append(ids, id)
		}
	}
	theInputState.m.RUnlock()
	sort.Slice(ids, func(a, b int) bool {
		return ids[a] < ids[b]
	})
	return ids
}

// IsTouchJustReleased returns a boolean value indicating
// whether the given touch is released just in the current frame.
//
// IsTouchJustReleased is concurrent safe.
func IsTouchJustReleased(id ebiten.TouchID) bool {
	theInputState.m.RLock()
	r := theInputState.touchDurations[id] == 0 && theInputState.prevTouchDurations[id] > 0
	theInputState.m.RUnlock()
	return r
}

// TouchPressDuration returns how long the touch remains in frames.
//
// TouchPressDuration is concurrent safe.
func TouchPressDuration(id ebiten.TouchID) int {
	theInputState.m.RLock()
	s := theInputState.touchDurations[id]
	theInputState.m.RUnlock()
	return s
}
