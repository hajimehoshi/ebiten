// Copyright 2016 Hajime Hoshi
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

//go:build android || ios
// +build android ios

package mobile

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

type Input struct {
	keys     map[driver.Key]struct{}
	runes    []rune
	touches  []Touch
	gamepads []Gamepad
	ui       *UserInterface
}

func (i *Input) CursorPosition() (x, y int) {
	return 0, 0
}

func (i *Input) AppendGamepadIDs(gamepadIDs []driver.GamepadID) []driver.GamepadID {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		gamepadIDs = append(gamepadIDs, g.ID)
	}
	return gamepadIDs
}

func (i *Input) GamepadSDLID(id driver.GamepadID) string {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.SDLID
	}
	return ""
}

func (i *Input) GamepadName(id driver.GamepadID) string {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.Name
	}
	return ""
}

func (i *Input) GamepadAxisNum(id driver.GamepadID) int {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.AxisNum
	}
	return 0
}

func (i *Input) GamepadAxisValue(id driver.GamepadID, axis int) float64 {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		if g.AxisNum <= int(axis) {
			return 0
		}
		return float64(g.Axes[axis])
	}
	return 0
}

func (i *Input) GamepadButtonNum(id driver.GamepadID) int {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.ButtonNum
	}
	return 0
}

func (i *Input) IsGamepadButtonPressed(id driver.GamepadID, button driver.GamepadButton) bool {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		if g.ButtonNum <= int(button) {
			return false
		}
		return g.Buttons[button]
	}
	return false
}

func (i *Input) IsStandardGamepadLayoutAvailable(id driver.GamepadID) bool {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return gamepaddb.HasStandardLayoutMapping(g.SDLID)
	}
	return false
}

func (i *Input) IsStandardGamepadButtonPressed(id driver.GamepadID, button driver.StandardGamepadButton) bool {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return gamepaddb.IsButtonPressed(g.SDLID, button, gamepadState{&g})
	}
	return false
}

func (i *Input) StandardGamepadButtonValue(id driver.GamepadID, button driver.StandardGamepadButton) float64 {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return gamepaddb.ButtonValue(g.SDLID, button, gamepadState{&g})
	}
	return 0
}

func (i *Input) StandardGamepadAxisValue(id driver.GamepadID, axis driver.StandardGamepadAxis) float64 {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return gamepaddb.AxisValue(g.SDLID, axis, gamepadState{&g})
	}
	return 0
}

func (i *Input) AppendTouchIDs(touchIDs []driver.TouchID) []driver.TouchID {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, t := range i.touches {
		touchIDs = append(touchIDs, t.ID)
	}
	return touchIDs
}

func (i *Input) TouchPosition(id driver.TouchID) (x, y int) {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	for _, t := range i.touches {
		if t.ID == id {
			return i.ui.adjustPosition(t.X, t.Y)
		}
	}
	return 0, 0
}

func (i *Input) AppendInputChars(runes []rune) []rune {
	i.ui.m.Lock()
	defer i.ui.m.Unlock()
	return append(runes, i.runes...)
}

func (i *Input) IsKeyPressed(key driver.Key) bool {
	i.ui.m.RLock()
	defer i.ui.m.RUnlock()

	_, ok := i.keys[key]
	return ok
}

func (i *Input) Wheel() (xoff, yoff float64) {
	return 0, 0
}

func (i *Input) IsMouseButtonPressed(key driver.MouseButton) bool {
	return false
}

func (i *Input) VibrateGamepad(id driver.GamepadID, duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// TODO: Implement this (#1452)
}

func (i *Input) update(keys map[driver.Key]struct{}, runes []rune, touches []Touch) {
	i.ui.m.Lock()
	defer i.ui.m.Unlock()

	if i.keys == nil {
		i.keys = map[driver.Key]struct{}{}
	}
	for k := range i.keys {
		delete(i.keys, k)
	}
	for k := range keys {
		i.keys[k] = struct{}{}
	}

	i.runes = i.runes[:0]
	i.runes = append(i.runes, runes...)

	i.touches = i.touches[:0]
	i.touches = append(i.touches, touches...)
}

func (i *Input) resetForFrame() {
	i.runes = nil
}

type gamepadState struct {
	g *Gamepad
}

func (s gamepadState) Axis(index int) float64 {
	return float64(s.g.Axes[index])
}

func (s gamepadState) Button(index int) bool {
	return s.g.Buttons[index]
}

func (s gamepadState) Hat(index int) int {
	return s.g.Hats[index]
}
