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
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
)

type pos struct {
	x int
	y int
}

type inputState struct {
	keyDurations     []int
	prevKeyDurations []int

	mouseButtonDurations     map[ebiten.MouseButton]int
	prevMouseButtonDurations map[ebiten.MouseButton]int

	gamepadIDs     map[ebiten.GamepadID]struct{}
	prevGamepadIDs map[ebiten.GamepadID]struct{}

	gamepadButtonDurations     map[ebiten.GamepadID][]int
	prevGamepadButtonDurations map[ebiten.GamepadID][]int

	standardGamepadButtonDurations     map[ebiten.GamepadID][]int
	prevStandardGamepadButtonDurations map[ebiten.GamepadID][]int

	touchIDs           map[ebiten.TouchID]struct{}
	touchDurations     map[ebiten.TouchID]int
	touchPositions     map[ebiten.TouchID]pos
	prevTouchDurations map[ebiten.TouchID]int
	prevTouchPositions map[ebiten.TouchID]pos

	gamepadIDsBuf []ebiten.GamepadID
	touchIDsBuf   []ebiten.TouchID

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

	standardGamepadButtonDurations:     map[ebiten.GamepadID][]int{},
	prevStandardGamepadButtonDurations: map[ebiten.GamepadID][]int{},

	touchIDs:           map[ebiten.TouchID]struct{}{},
	touchDurations:     map[ebiten.TouchID]int{},
	touchPositions:     map[ebiten.TouchID]pos{},
	prevTouchDurations: map[ebiten.TouchID]int{},
	prevTouchPositions: map[ebiten.TouchID]pos{},
}

func init() {
	hook.AppendHookOnBeforeUpdate(func() error {
		theInputState.update()
		return nil
	})
}

func (i *inputState) update() {
	i.m.Lock()
	defer i.m.Unlock()

	// Keyboard
	copy(i.prevKeyDurations, i.keyDurations)
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if ebiten.IsKeyPressed(k) {
			i.keyDurations[k]++
		} else {
			i.keyDurations[k] = 0
		}
	}

	// Mouse
	for b := ebiten.MouseButton(0); b <= ebiten.MouseButtonMax; b++ {
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

	for id := range i.prevStandardGamepadButtonDurations {
		delete(i.prevStandardGamepadButtonDurations, id)
	}
	for id, ds := range i.standardGamepadButtonDurations {
		i.prevStandardGamepadButtonDurations[id] = append([]int{}, ds...)
	}

	for id := range i.gamepadIDs {
		delete(i.gamepadIDs, id)
	}
	i.gamepadIDsBuf = ebiten.AppendGamepadIDs(i.gamepadIDsBuf[:0])
	for _, id := range i.gamepadIDsBuf {
		i.gamepadIDs[id] = struct{}{}

		if _, ok := i.gamepadButtonDurations[id]; !ok {
			i.gamepadButtonDurations[id] = make([]int, ebiten.GamepadButtonMax+1)
		}
		for b := ebiten.GamepadButton(0); b <= ebiten.GamepadButtonMax; b++ {
			if ebiten.IsGamepadButtonPressed(id, b) {
				i.gamepadButtonDurations[id][b]++
			} else {
				i.gamepadButtonDurations[id][b] = 0
			}
		}

		if _, ok := i.standardGamepadButtonDurations[id]; !ok {
			i.standardGamepadButtonDurations[id] = make([]int, ebiten.StandardGamepadButtonMax+1)
		}
		for b := ebiten.StandardGamepadButton(0); b <= ebiten.StandardGamepadButtonMax; b++ {
			if ebiten.IsStandardGamepadButtonPressed(id, b) {
				i.standardGamepadButtonDurations[id][b]++
			} else {
				i.standardGamepadButtonDurations[id][b] = 0
			}
		}
	}
	for id := range i.gamepadButtonDurations {
		if _, ok := i.gamepadIDs[id]; !ok {
			delete(i.gamepadButtonDurations, id)
		}
	}
	for id := range i.standardGamepadButtonDurations {
		if _, ok := i.gamepadIDs[id]; !ok {
			delete(i.standardGamepadButtonDurations, id)
		}
	}

	// Touches

	// Copy the touch durations and positions.
	for id := range i.prevTouchDurations {
		delete(i.prevTouchDurations, id)
	}
	for id := range i.touchDurations {
		i.prevTouchDurations[id] = i.touchDurations[id]
	}
	for id := range i.prevTouchPositions {
		delete(i.prevTouchPositions, id)
	}
	for id := range i.touchPositions {
		i.prevTouchPositions[id] = i.touchPositions[id]
	}

	for id := range i.touchIDs {
		delete(i.touchIDs, id)
	}
	i.touchIDsBuf = ebiten.AppendTouchIDs(i.touchIDsBuf[:0])
	for _, id := range i.touchIDsBuf {
		i.touchIDs[id] = struct{}{}
		i.touchDurations[id]++
		x, y := ebiten.TouchPosition(id)
		i.touchPositions[id] = pos{x: x, y: y}
	}
	for id := range i.touchDurations {
		if _, ok := i.touchIDs[id]; !ok {
			delete(i.touchDurations, id)
			delete(i.touchPositions, id)
		}
	}
}

// AppendPressedKeys append currently pressed keyboard keys to keys and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendPressedKeys must be called in a game's Update, not Draw.
//
// AppendPressedKeys is concurrent safe.
func AppendPressedKeys(keys []ebiten.Key) []ebiten.Key {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	for i, d := range theInputState.keyDurations {
		if d == 0 {
			continue
		}
		keys = append(keys, ebiten.Key(i))
	}
	return keys
}

// PressedKeys returns a set of currently pressed keyboard keys.
//
// PressedKeys must be called in a game's Update, not Draw.
//
// Deprecated: as of v2.2. Use AppendPressedKeys instead.
func PressedKeys() []ebiten.Key {
	return AppendPressedKeys(nil)
}

// AppendJustPressedKeys append just pressed keyboard keys to keys and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustPressedKeys must be called in a game's Update, not Draw.
//
// AppendJustPressedKeys is concurrent safe.
func AppendJustPressedKeys(keys []ebiten.Key) []ebiten.Key {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	for i, d := range theInputState.keyDurations {
		if d != 1 {
			continue
		}
		keys = append(keys, ebiten.Key(i))
	}
	return keys
}

// AppendJustReleasedKeys append just released keyboard keys to keys and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustReleasedKeys must be called in a game's Update, not Draw.
//
// AppendJustReleasedKeys is concurrent safe.
func AppendJustReleasedKeys(keys []ebiten.Key) []ebiten.Key {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if theInputState.keyDurations[k] != 0 {
			continue
		}
		if theInputState.prevKeyDurations[k] == 0 {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}

// IsKeyJustPressed returns a boolean value indicating
// whether the given key is pressed just in the current tick.
//
// IsKeyJustPressed must be called in a game's Update, not Draw.
//
// IsKeyJustPressed is concurrent safe.
func IsKeyJustPressed(key ebiten.Key) bool {
	return KeyPressDuration(key) == 1
}

// IsKeyJustReleased returns a boolean value indicating
// whether the given key is released just in the current tick.
//
// IsKeyJustReleased must be called in a game's Update, not Draw.
//
// IsKeyJustReleased is concurrent safe.
func IsKeyJustReleased(key ebiten.Key) bool {
	theInputState.m.RLock()
	r := theInputState.keyDurations[key] == 0 && theInputState.prevKeyDurations[key] > 0
	theInputState.m.RUnlock()
	return r
}

// KeyPressDuration returns how long the key is pressed in ticks (Update).
//
// KeyPressDuration must be called in a game's Update, not Draw.
//
// KeyPressDuration is concurrent safe.
func KeyPressDuration(key ebiten.Key) int {
	theInputState.m.RLock()
	s := theInputState.keyDurations[key]
	theInputState.m.RUnlock()
	return s
}

// IsMouseButtonJustPressed returns a boolean value indicating
// whether the given mouse button is pressed just in the current tick.
//
// IsMouseButtonJustPressed must be called in a game's Update, not Draw.
//
// IsMouseButtonJustPressed is concurrent safe.
func IsMouseButtonJustPressed(button ebiten.MouseButton) bool {
	return MouseButtonPressDuration(button) == 1
}

// IsMouseButtonJustReleased returns a boolean value indicating
// whether the given mouse button is released just in the current tick.
//
// IsMouseButtonJustReleased must be called in a game's Update, not Draw.
//
// IsMouseButtonJustReleased is concurrent safe.
func IsMouseButtonJustReleased(button ebiten.MouseButton) bool {
	theInputState.m.RLock()
	r := theInputState.mouseButtonDurations[button] == 0 &&
		theInputState.prevMouseButtonDurations[button] > 0
	theInputState.m.RUnlock()
	return r
}

// MouseButtonPressDuration returns how long the mouse button is pressed in ticks (Update).
//
// MouseButtonPressDuration must be called in a game's Update, not Draw.
//
// MouseButtonPressDuration is concurrent safe.
func MouseButtonPressDuration(button ebiten.MouseButton) int {
	theInputState.m.RLock()
	s := theInputState.mouseButtonDurations[button]
	theInputState.m.RUnlock()
	return s
}

// AppendJustConnectedGamepadIDs appends gamepad IDs that are connected just in the current tick to gamepadIDs,
// and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustConnectedGamepadIDs must be called in a game's Update, not Draw.
//
// AppendJustConnectedGamepadIDs is concurrent safe.
func AppendJustConnectedGamepadIDs(gamepadIDs []ebiten.GamepadID) []ebiten.GamepadID {
	origLen := len(gamepadIDs)
	theInputState.m.RLock()
	for id := range theInputState.gamepadIDs {
		if _, ok := theInputState.prevGamepadIDs[id]; !ok {
			gamepadIDs = append(gamepadIDs, id)
		}
	}
	theInputState.m.RUnlock()
	s := gamepadIDs[origLen:]
	sort.Slice(s, func(a, b int) bool {
		return s[a] < s[b]
	})
	return gamepadIDs
}

// JustConnectedGamepadIDs returns gamepad IDs that are connected just in the current tick.
//
// JustConnectedGamepadIDs must be called in a game's Update, not Draw.
//
// Deprecated: as of v2.2. Use AppendJustConnectedGamepadIDs instead.
func JustConnectedGamepadIDs() []ebiten.GamepadID {
	return AppendJustConnectedGamepadIDs(nil)
}

// IsGamepadJustDisconnected returns a boolean value indicating
// whether the gamepad of the given id is released just in the current tick.
//
// IsGamepadJustDisconnected must be called in a game's Update, not Draw.
//
// IsGamepadJustDisconnected is concurrent safe.
func IsGamepadJustDisconnected(id ebiten.GamepadID) bool {
	theInputState.m.RLock()
	_, prev := theInputState.prevGamepadIDs[id]
	_, current := theInputState.gamepadIDs[id]
	theInputState.m.RUnlock()
	return prev && !current
}

// AppendPressedGamepadButtons append currently pressed gamepad buttons to buttons and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendPressedGamepadButtons must be called in a game's Update, not Draw.
//
// AppendPressedGamepadButtons is concurrent safe.
func AppendPressedGamepadButtons(id ebiten.GamepadID, buttons []ebiten.GamepadButton) []ebiten.GamepadButton {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	if _, ok := theInputState.gamepadButtonDurations[id]; !ok {
		return buttons
	}

	for b, d := range theInputState.gamepadButtonDurations[id] {
		if d == 0 {
			continue
		}
		buttons = append(buttons, ebiten.GamepadButton(b))
	}

	return buttons
}

// AppendJustPressedGamepadButtons append just pressed gamepad buttons to buttons and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustPressedGamepadButtons must be called in a game's Update, not Draw.
//
// AppendJustPressedGamepadButtons is concurrent safe.
func AppendJustPressedGamepadButtons(id ebiten.GamepadID, buttons []ebiten.GamepadButton) []ebiten.GamepadButton {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	if _, ok := theInputState.gamepadButtonDurations[id]; !ok {
		return buttons
	}

	for b, d := range theInputState.gamepadButtonDurations[id] {
		if d != 1 {
			continue
		}
		buttons = append(buttons, ebiten.GamepadButton(b))
	}

	return buttons
}

// AppendJustReleasedGamepadButtons append just released gamepad buttons to buttons and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustReleasedGamepadButtons must be called in a game's Update, not Draw.
//
// AppendJustReleasedGamepadButtons is concurrent safe.
func AppendJustReleasedGamepadButtons(id ebiten.GamepadID, buttons []ebiten.GamepadButton) []ebiten.GamepadButton {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	if _, ok := theInputState.gamepadButtonDurations[id]; !ok {
		return buttons
	}
	if _, ok := theInputState.prevGamepadButtonDurations[id]; !ok {
		return buttons
	}

	for b := ebiten.GamepadButton(0); b <= ebiten.GamepadButtonMax; b++ {
		if theInputState.gamepadButtonDurations[id][b] != 0 {
			continue
		}
		if theInputState.prevGamepadButtonDurations[id][b] == 0 {
			continue
		}
		buttons = append(buttons, b)
	}

	return buttons
}

// IsGamepadButtonJustPressed returns a boolean value indicating
// whether the given gamepad button of the gamepad id is pressed just in the current tick.
//
// IsGamepadButtonJustPressed must be called in a game's Update, not Draw.
//
// IsGamepadButtonJustPressed is concurrent safe.
func IsGamepadButtonJustPressed(id ebiten.GamepadID, button ebiten.GamepadButton) bool {
	return GamepadButtonPressDuration(id, button) == 1
}

// IsGamepadButtonJustReleased returns a boolean value indicating
// whether the given gamepad button of the gamepad id is released just in the current tick.
//
// IsGamepadButtonJustReleased must be called in a game's Update, not Draw.
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

// GamepadButtonPressDuration returns how long the gamepad button of the gamepad id is pressed in ticks (Update).
//
// GamepadButtonPressDuration must be called in a game's Update, not Draw.
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

// AppendPressedStandardGamepadButtons append currently pressed standard gamepad buttons to buttons and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendPressedStandardGamepadButtons must be called in a game's Update, not Draw.
//
// AppendPressedStandardGamepadButtons is concurrent safe.
func AppendPressedStandardGamepadButtons(id ebiten.GamepadID, buttons []ebiten.StandardGamepadButton) []ebiten.StandardGamepadButton {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	if _, ok := theInputState.standardGamepadButtonDurations[id]; !ok {
		return buttons
	}

	for b, d := range theInputState.standardGamepadButtonDurations[id] {
		if d == 0 {
			continue
		}
		buttons = append(buttons, ebiten.StandardGamepadButton(b))
	}

	return buttons
}

// AppendJustPressedStandardGamepadButtons append just pressed standard gamepad buttons to buttons and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustPressedStandardGamepadButtons must be called in a game's Update, not Draw.
//
// AppendJustPressedStandardGamepadButtons is concurrent safe.
func AppendJustPressedStandardGamepadButtons(id ebiten.GamepadID, buttons []ebiten.StandardGamepadButton) []ebiten.StandardGamepadButton {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	if _, ok := theInputState.gamepadButtonDurations[id]; !ok {
		return buttons
	}

	for b, d := range theInputState.standardGamepadButtonDurations[id] {
		if d != 1 {
			continue
		}
		buttons = append(buttons, ebiten.StandardGamepadButton(b))
	}

	return buttons
}

// AppendJustReleasedStandardGamepadButtons append just released standard gamepad buttons to buttons and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustReleasedStandardGamepadButtons must be called in a game's Update, not Draw.
//
// AppendJustReleasedStandardGamepadButtons is concurrent safe.
func AppendJustReleasedStandardGamepadButtons(id ebiten.GamepadID, buttons []ebiten.StandardGamepadButton) []ebiten.StandardGamepadButton {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	if _, ok := theInputState.gamepadButtonDurations[id]; !ok {
		return buttons
	}
	if _, ok := theInputState.prevGamepadButtonDurations[id]; !ok {
		return buttons
	}

	for b := ebiten.StandardGamepadButton(0); b <= ebiten.StandardGamepadButtonMax; b++ {
		if theInputState.standardGamepadButtonDurations[id][b] != 0 {
			continue
		}
		if theInputState.prevStandardGamepadButtonDurations[id][b] == 0 {
			continue
		}
		buttons = append(buttons, b)
	}

	return buttons
}

// IsStandardGamepadButtonJustPressed returns a boolean value indicating
// whether the given standard gamepad button of the gamepad id is pressed just in the current tick.
//
// IsStandardGamepadButtonJustPressed must be called in a game's Update, not Draw.
//
// IsStandardGamepadButtonJustPressed is concurrent safe.
func IsStandardGamepadButtonJustPressed(id ebiten.GamepadID, button ebiten.StandardGamepadButton) bool {
	return StandardGamepadButtonPressDuration(id, button) == 1
}

// IsStandardGamepadButtonJustReleased returns a boolean value indicating
// whether the given standard gamepad button of the gamepad id is released just in the current tick.
//
// IsStandardGamepadButtonJustReleased must be called in a game's Update, not Draw.
//
// IsStandardGamepadButtonJustReleased is concurrent safe.
func IsStandardGamepadButtonJustReleased(id ebiten.GamepadID, button ebiten.StandardGamepadButton) bool {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	var prev int
	if _, ok := theInputState.prevStandardGamepadButtonDurations[id]; ok {
		prev = theInputState.prevStandardGamepadButtonDurations[id][button]
	}
	var current int
	if _, ok := theInputState.standardGamepadButtonDurations[id]; ok {
		current = theInputState.standardGamepadButtonDurations[id][button]
	}
	return current == 0 && prev > 0
}

// StandardGamepadButtonPressDuration returns how long the standard gamepad button of the gamepad id is pressed in ticks (Update).
//
// StandardGamepadButtonPressDuration must be called in a game's Update, not Draw.
//
// StandardGamepadButtonPressDuration is concurrent safe.
func StandardGamepadButtonPressDuration(id ebiten.GamepadID, button ebiten.StandardGamepadButton) int {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	if _, ok := theInputState.standardGamepadButtonDurations[id]; ok {
		return theInputState.standardGamepadButtonDurations[id][button]
	}
	return 0
}

// AppendJustPressedTouchIDs append touch IDs that are created just in the current tick to touchIDs,
// and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustPressedTouchIDs must be called in a game's Update, not Draw.
//
// AppendJustPressedTouchIDs is concurrent safe.
func AppendJustPressedTouchIDs(touchIDs []ebiten.TouchID) []ebiten.TouchID {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	origLen := len(touchIDs)
	for id, s := range theInputState.touchDurations {
		if s == 1 {
			touchIDs = append(touchIDs, id)
		}
	}

	s := touchIDs[origLen:]
	sort.Slice(s, func(a, b int) bool {
		return s[a] < s[b]
	})

	return touchIDs
}

// JustPressedTouchIDs returns touch IDs that are created just in the current tick.
//
// JustPressedTouchIDs must be called in a game's Update, not Draw.
//
// Deprecated: as of v2.2. Use AppendJustPressedTouchIDs instead.
func JustPressedTouchIDs() []ebiten.TouchID {
	return AppendJustPressedTouchIDs(nil)
}

// AppendJustReleasedTouchIDs append touch IDs that are released just in the current tick to touchIDs,
// and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustReleasedTouchIDs must be called in a game's Update, not Draw.
//
// AppendJustReleasedTouchIDs is concurrent safe.
func AppendJustReleasedTouchIDs(touchIDs []ebiten.TouchID) []ebiten.TouchID {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	origLen := len(touchIDs)
	for id := range theInputState.prevTouchDurations {
		if theInputState.touchDurations[id] == 0 && theInputState.prevTouchDurations[id] > 0 {
			touchIDs = append(touchIDs, id)
		}
	}

	s := touchIDs[origLen:]
	sort.Slice(s, func(a, b int) bool {
		return s[a] < s[b]
	})

	return touchIDs
}

// IsTouchJustReleased returns a boolean value indicating
// whether the given touch is released just in the current tick.
//
// IsTouchJustReleased must be called in a game's Update, not Draw.
//
// IsTouchJustReleased is concurrent safe.
func IsTouchJustReleased(id ebiten.TouchID) bool {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	return theInputState.touchDurations[id] == 0 && theInputState.prevTouchDurations[id] > 0
}

// TouchPressDuration returns how long the touch remains in ticks (Update).
//
// TouchPressDuration must be called in a game's Update, not Draw.
//
// TouchPressDuration is concurrent safe.
func TouchPressDuration(id ebiten.TouchID) int {
	theInputState.m.RLock()
	s := theInputState.touchDurations[id]
	theInputState.m.RUnlock()
	return s
}

// TouchPositionInPreviousTick returns the position in the previous tick.
// If the touch is a just-released touch, TouchPositionInPreviousTick returns the last position of the touch.
//
// TouchPositionInPreviousTick must be called in a game's Update, not Draw.
//
// TouchJustReleasedPosition is concurrent safe.
func TouchPositionInPreviousTick(id ebiten.TouchID) (int, int) {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	p := theInputState.prevTouchPositions[id]
	return p.x, p.y
}
