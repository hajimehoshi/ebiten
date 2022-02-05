// Copyright 2022 The Ebiten Authors
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

//go:build ebitencbackend
// +build ebitencbackend

package cbackend

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/cbackend"
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
)

func (i *Input) updateGamepads() {
	i.gamepads = i.gamepads[:0]
	i.gamepads = cbackend.AppendGamepads(i.gamepads)
}

func (i *Input) AppendGamepadIDs(gamepadIDs []driver.GamepadID) []driver.GamepadID {
	i.m.Lock()
	defer i.m.Unlock()

	for _, g := range i.gamepads {
		gamepadIDs = append(gamepadIDs, g.ID)
	}
	return gamepadIDs
}

func (i *Input) GamepadSDLID(id driver.GamepadID) string {
	return ""
}

func (i *Input) GamepadName(id driver.GamepadID) string {
	return ""
}

func (i *Input) GamepadAxisValue(id driver.GamepadID, axis int) float64 {
	i.m.Lock()
	defer i.m.Unlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		if axis < 0 {
			return 0
		}
		if g.AxisNum <= axis {
			return 0
		}
		if len(g.AxisValues) <= axis {
			return 0
		}
		return g.AxisValues[axis]
	}
	return 0
}

func (i *Input) GamepadAxisNum(id driver.GamepadID) int {
	i.m.Lock()
	defer i.m.Unlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.AxisNum
	}
	return 0
}

func (i *Input) GamepadButtonNum(id driver.GamepadID) int {
	i.m.Lock()
	defer i.m.Unlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.ButtonNum
	}
	return 0
}

func (i *Input) IsGamepadButtonPressed(id driver.GamepadID, button driver.GamepadButton) bool {
	i.m.Lock()
	defer i.m.Unlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		if button < 0 {
			return false
		}
		if g.ButtonNum <= int(button) {
			return false
		}
		if len(g.ButtonPressed) <= int(button) {
			return false
		}
		return g.ButtonPressed[button]
	}
	return false
}

func (i *Input) IsStandardGamepadButtonPressed(id driver.GamepadID, button driver.StandardGamepadButton) bool {
	i.m.Lock()
	defer i.m.Unlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		if !g.Standard {
			return false
		}
		if button < 0 {
			return false
		}
		if g.ButtonNum <= int(button) {
			return false
		}
		if len(g.ButtonPressed) <= int(button) {
			return false
		}
		return g.ButtonPressed[button]
	}
	return false
}

func (i *Input) IsStandardGamepadLayoutAvailable(id driver.GamepadID) bool {
	i.m.Lock()
	defer i.m.Unlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		return g.Standard
	}
	return false
}

func (i *Input) StandardGamepadAxisValue(id driver.GamepadID, axis driver.StandardGamepadAxis) float64 {
	i.m.Lock()
	defer i.m.Unlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		if !g.Standard {
			return 0
		}
		if axis < 0 {
			return 0
		}
		if g.AxisNum <= int(axis) {
			return 0
		}
		if len(g.AxisValues) <= int(axis) {
			return 0
		}
		return g.AxisValues[axis]
	}
	return 0
}

func (i *Input) StandardGamepadButtonValue(id driver.GamepadID, button driver.StandardGamepadButton) float64 {
	i.m.Lock()
	defer i.m.Unlock()

	for _, g := range i.gamepads {
		if g.ID != id {
			continue
		}
		if !g.Standard {
			return 0
		}
		if button < 0 {
			return 0
		}
		if g.ButtonNum <= int(button) {
			return 0
		}
		if len(g.ButtonValues) <= int(button) {
			return 0
		}
		return g.ButtonValues[button]
	}
	return 0
}

func (i *Input) VibrateGamepad(id driver.GamepadID, duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	cbackend.VibrateGamepad(id, duration, strongMagnitude, weakMagnitude)
}
