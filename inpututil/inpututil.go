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
	"maps"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
	"github.com/hajimehoshi/ebiten/v2/internal/inputstate"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

type gamepadState struct {
	buttonDurations         [ebiten.GamepadButtonMax + 1]int
	standardButtonDurations [ebiten.StandardGamepadButtonMax + 1]int
}

type touchState struct {
	duration int
	x        float64
	y        float64
}

type inputState struct {
	gamepadStates     map[ebiten.GamepadID]gamepadState
	prevGamepadStates map[ebiten.GamepadID]gamepadState

	touchStates     map[ebiten.TouchID]touchState
	prevTouchStates map[ebiten.TouchID]touchState

	gamepadIDsBuf []ebiten.GamepadID
	touchIDsBuf   []ebiten.TouchID

	m sync.RWMutex
}

var theInputState = &inputState{
	gamepadStates:     map[ebiten.GamepadID]gamepadState{},
	prevGamepadStates: map[ebiten.GamepadID]gamepadState{},
	touchStates:       map[ebiten.TouchID]touchState{},
	prevTouchStates:   map[ebiten.TouchID]touchState{},
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

	// Gamepads

	// Copy the gamepad states.
	clear(i.prevGamepadStates)
	maps.Copy(i.prevGamepadStates, i.gamepadStates)

	i.gamepadIDsBuf = ebiten.AppendGamepadIDs(i.gamepadIDsBuf[:0])
	for _, id := range i.gamepadIDsBuf {
		state := i.gamepadStates[id]

		for b := range i.gamepadStates[id].buttonDurations {
			if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton(b)) {
				state.buttonDurations[b]++
			} else {
				state.buttonDurations[b] = 0
			}
		}

		for b := range i.gamepadStates[id].standardButtonDurations {
			if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButton(b)) {
				state.standardButtonDurations[b]++
			} else {
				state.standardButtonDurations[b] = 0
			}
		}

		i.gamepadStates[id] = state
	}

	// Remove disconnected gamepads.
	for id := range i.gamepadStates {
		if !slices.Contains(i.gamepadIDsBuf, id) {
			delete(i.gamepadStates, id)
		}
	}

	// Touches

	// Copy the touch durations and positions.
	clear(i.prevTouchStates)
	maps.Copy(i.prevTouchStates, i.touchStates)

	i.touchIDsBuf = ebiten.AppendTouchIDs(i.touchIDsBuf[:0])
	for _, id := range i.touchIDsBuf {
		state := i.touchStates[id]
		state.duration++
		state.x, state.y = ebiten.TouchPositionF(id)
		i.touchStates[id] = state
	}

	// Remove released touches.
	for id := range i.touchStates {
		if !slices.Contains(i.touchIDsBuf, id) {
			delete(i.touchStates, id)
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
	return inputstate.AppendPressedKeys(keys)
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
	return inputstate.AppendJustPressedKeys(keys)
}

// AppendJustReleasedKeys append just released keyboard keys to keys and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustReleasedKeys must be called in a game's Update, not Draw.
//
// AppendJustReleasedKeys is concurrent safe.
func AppendJustReleasedKeys(keys []ebiten.Key) []ebiten.Key {
	return inputstate.AppendJustReleasedKeys(keys)
}

// IsKeyJustPressed returns a boolean value indicating
// whether the given key is pressed just in the current tick.
//
// IsKeyJustPressed always returns false for a key that represents multiple keys like [ebiten.KeyShift].
// Use its left and right variants like [ebiten.KeyShiftLeft] and [ebiten.KeyShiftRight] instead.
//
// IsKeyJustPressed must be called in a game's Update, not Draw.
//
// IsKeyJustPressed is concurrent safe.
func IsKeyJustPressed(key ebiten.Key) bool {
	return inputstate.Get().IsKeyJustPressed(ui.Key(key))
}

// IsKeyJustReleased returns a boolean value indicating
// whether the given key is released just in the current tick.
//
// IsKeyJustReleased always returns false for a key that represents multiple keys like [ebiten.KeyShift].
// Use its left and right variants like [ebiten.KeyShiftLeft] and [ebiten.KeyShiftRight] instead.
//
// IsKeyJustReleased must be called in a game's Update, not Draw.
//
// IsKeyJustReleased is concurrent safe.
func IsKeyJustReleased(key ebiten.Key) bool {
	return inputstate.Get().IsKeyJustReleased(ui.Key(key))
}

// KeyPressDuration returns how long the key is pressed in ticks (Update).
//
// KeyPressDuration follows [ebiten.IsKeyPressed], and thus returns a positive value for a modifier key
// released in the current tick.
//
// KeyPressDuration always returns 0 for a key that represents multiple keys like [ebiten.KeyShift].
// Use its left and right variants like [ebiten.KeyShiftLeft] and [ebiten.KeyShiftRight] instead.
//
// KeyPressDuration must be called in a game's Update, not Draw.
//
// KeyPressDuration is concurrent safe.
func KeyPressDuration(key ebiten.Key) int {
	return int(inputstate.Get().KeyPressDuration(ui.Key(key)))
}

// IsMouseButtonJustPressed returns a boolean value indicating
// whether the given mouse button is pressed just in the current tick.
//
// IsMouseButtonJustPressed must be called in a game's Update, not Draw.
//
// IsMouseButtonJustPressed is concurrent safe.
func IsMouseButtonJustPressed(button ebiten.MouseButton) bool {
	return inputstate.Get().IsMouseButtonJustPressed(ui.MouseButton(button))
}

// IsMouseButtonJustReleased returns a boolean value indicating
// whether the given mouse button is released just in the current tick.
//
// IsMouseButtonJustReleased must be called in a game's Update, not Draw.
//
// IsMouseButtonJustReleased is concurrent safe.
func IsMouseButtonJustReleased(button ebiten.MouseButton) bool {
	return inputstate.Get().IsMouseButtonJustReleased(ui.MouseButton(button))
}

// MouseButtonPressDuration returns how long the mouse button is pressed in ticks (Update).
//
// MouseButtonPressDuration must be called in a game's Update, not Draw.
//
// MouseButtonPressDuration is concurrent safe.
func MouseButtonPressDuration(button ebiten.MouseButton) int {
	return int(inputstate.Get().MouseButtonPressDuration(ui.MouseButton(button)))
}

// AppendJustConnectedGamepadIDs appends gamepad IDs that are connected just in the current tick to gamepadIDs,
// and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustConnectedGamepadIDs must be called in a game's Update, not Draw.
//
// AppendJustConnectedGamepadIDs is concurrent safe.
func AppendJustConnectedGamepadIDs(gamepadIDs []ebiten.GamepadID) []ebiten.GamepadID {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	origLen := len(gamepadIDs)
	for id := range theInputState.gamepadStates {
		if _, ok := theInputState.prevGamepadStates[id]; !ok {
			gamepadIDs = append(gamepadIDs, id)
		}
	}

	slices.Sort(gamepadIDs[origLen:])
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
	defer theInputState.m.RUnlock()

	_, current := theInputState.gamepadStates[id]
	_, prev := theInputState.prevGamepadStates[id]
	return !current && prev
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

	state, ok := theInputState.gamepadStates[id]
	if !ok {
		return buttons
	}

	for b, d := range state.buttonDurations {
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

	state, ok := theInputState.gamepadStates[id]
	if !ok {
		return buttons
	}

	for b, d := range state.buttonDurations {
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

	state, ok := theInputState.gamepadStates[id]
	if !ok {
		return buttons
	}
	prevState, ok := theInputState.prevGamepadStates[id]
	if !ok {
		return buttons
	}

	for b := range state.buttonDurations {
		if state.buttonDurations[b] != 0 {
			continue
		}
		if prevState.buttonDurations[b] == 0 {
			continue
		}
		buttons = append(buttons, ebiten.GamepadButton(b))
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
	if button < 0 || ebiten.GamepadButtonMax < button {
		return false
	}

	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	state, ok := theInputState.gamepadStates[id]
	if !ok {
		return false
	}
	prevState, ok := theInputState.prevGamepadStates[id]
	if !ok {
		return false
	}

	return state.buttonDurations[button] == 0 && prevState.buttonDurations[button] > 0
}

// GamepadButtonPressDuration returns how long the gamepad button of the gamepad id is pressed in ticks (Update).
//
// GamepadButtonPressDuration must be called in a game's Update, not Draw.
//
// GamepadButtonPressDuration is concurrent safe.
func GamepadButtonPressDuration(id ebiten.GamepadID, button ebiten.GamepadButton) int {
	if button < 0 || ebiten.GamepadButtonMax < button {
		return 0
	}

	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	state, ok := theInputState.gamepadStates[id]
	if !ok {
		return 0
	}

	return state.buttonDurations[button]
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

	state, ok := theInputState.gamepadStates[id]
	if !ok {
		return buttons
	}

	for i, d := range state.standardButtonDurations {
		if d == 0 {
			continue
		}
		buttons = append(buttons, ebiten.StandardGamepadButton(i))
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

	state, ok := theInputState.gamepadStates[id]
	if !ok {
		return buttons
	}

	for b, d := range state.standardButtonDurations {
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

	state, ok := theInputState.gamepadStates[id]
	if !ok {
		return buttons
	}
	prevState, ok := theInputState.prevGamepadStates[id]
	if !ok {
		return buttons
	}

	for b := range state.standardButtonDurations {
		if state.standardButtonDurations[b] != 0 {
			continue
		}
		if prevState.standardButtonDurations[b] == 0 {
			continue
		}
		buttons = append(buttons, ebiten.StandardGamepadButton(b))
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
	if button < 0 || ebiten.StandardGamepadButtonMax < button {
		return false
	}

	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	state, ok := theInputState.gamepadStates[id]
	if !ok {
		return false
	}
	prevState, ok := theInputState.prevGamepadStates[id]
	if !ok {
		return false
	}

	return state.standardButtonDurations[button] == 0 && prevState.standardButtonDurations[button] > 0
}

// StandardGamepadButtonPressDuration returns how long the standard gamepad button of the gamepad id is pressed in ticks (Update).
//
// StandardGamepadButtonPressDuration must be called in a game's Update, not Draw.
//
// StandardGamepadButtonPressDuration is concurrent safe.
func StandardGamepadButtonPressDuration(id ebiten.GamepadID, button ebiten.StandardGamepadButton) int {
	if button < 0 || ebiten.StandardGamepadButtonMax < button {
		return 0
	}

	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	state, ok := theInputState.gamepadStates[id]
	if !ok {
		return 0
	}

	return state.standardButtonDurations[button]
}

// AppendJustPressedTouchIDs append touch IDs that are created just in the current tick to touchIDs,
// and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// The touches are sampled once per tick, so a touch which was created and released between two
// ticks is not reported at all.
// A new touch which reuses an ID that is still being tracked is not reported either, as its ID is
// not new.
//
// AppendJustPressedTouchIDs must be called in a game's Update, not Draw.
//
// AppendJustPressedTouchIDs is concurrent safe.
func AppendJustPressedTouchIDs(touchIDs []ebiten.TouchID) []ebiten.TouchID {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	origLen := len(touchIDs)
	for id, state := range theInputState.touchStates {
		if state.duration != 1 {
			continue
		}
		touchIDs = append(touchIDs, id)
	}

	slices.Sort(touchIDs[origLen:])
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
// A touch which was created and released between two ticks is not reported as released, and a new
// touch which reused the ID of a released one is reported as a continuation of the previous touch.
//
// AppendJustReleasedTouchIDs must be called in a game's Update, not Draw.
//
// AppendJustReleasedTouchIDs is concurrent safe.
func AppendJustReleasedTouchIDs(touchIDs []ebiten.TouchID) []ebiten.TouchID {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	origLen := len(touchIDs)
	// Iterate prevTouchStates instead of touchStates since touchStates doesn't have released touches.
	for id, state := range theInputState.prevTouchStates {
		if state.duration == 0 {
			continue
		}
		if theInputState.touchStates[id].duration != 0 {
			continue
		}
		touchIDs = append(touchIDs, id)
	}

	slices.Sort(touchIDs[origLen:])
	return touchIDs
}

// IsTouchJustReleased returns a boolean value indicating
// whether the given touch is released just in the current tick.
//
// A touch which was created and released between two ticks is not reported as released, and a new
// touch which reused the ID of a released one is reported as a continuation of the previous touch.
//
// IsTouchJustReleased must be called in a game's Update, not Draw.
//
// IsTouchJustReleased is concurrent safe.
func IsTouchJustReleased(id ebiten.TouchID) bool {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	current := theInputState.touchStates[id]
	prev := theInputState.prevTouchStates[id]
	return current.duration == 0 && prev.duration > 0
}

// TouchPressDuration returns how long the touch remains in ticks (Update).
//
// The duration is counted per touch ID, not per touch: a new touch which reused the ID of a touch
// released between two ticks continues the duration of the previous touch.
//
// TouchPressDuration must be called in a game's Update, not Draw.
//
// TouchPressDuration is concurrent safe.
func TouchPressDuration(id ebiten.TouchID) int {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()
	return theInputState.touchStates[id].duration
}

// TouchPositionInPreviousTick returns the position in the previous tick.
// If the touch is a just-released touch, TouchPositionInPreviousTick returns the last position of the touch.
//
// The position is tracked per touch ID, not per touch: a new touch which reused the ID of a touch
// released between two ticks continues the position of the previous touch.
//
// TouchPositionInPreviousTick must be called in a game's Update, not Draw.
//
// TouchPositionInPreviousTick is concurrent safe.
func TouchPositionInPreviousTick(id ebiten.TouchID) (int, int) {
	x, y := TouchPositionFInPreviousTick(id)
	return int(x), int(y)
}

// TouchPositionFInPreviousTick returns the high-precision position in the previous tick.
// If the touch is a just-released touch, TouchPositionFInPreviousTick returns the last position of the touch.
//
// The position is tracked per touch ID, not per touch: a new touch which reused the ID of a touch
// released between two ticks continues the position of the previous touch.
//
// TouchPositionFInPreviousTick must be called in a game's Update, not Draw.
//
// TouchPositionFInPreviousTick is concurrent safe.
func TouchPositionFInPreviousTick(id ebiten.TouchID) (float64, float64) {
	theInputState.m.RLock()
	defer theInputState.m.RUnlock()

	state := theInputState.prevTouchStates[id]
	return state.x, state.y
}
