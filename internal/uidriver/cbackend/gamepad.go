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

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

func (i *Input) AppendGamepadIDs(gamepadIDs []driver.GamepadID) []driver.GamepadID {
	return gamepad.AppendGamepadIDs(gamepadIDs)
}

func (i *Input) GamepadSDLID(id driver.GamepadID) string {
	g := gamepad.Get(id)
	if g == nil {
		return ""
	}
	return g.SDLID()
}

func (i *Input) GamepadName(id driver.GamepadID) string {
	g := gamepad.Get(id)
	if g == nil {
		return ""
	}
	return g.Name()
}

func (i *Input) GamepadAxisValue(id driver.GamepadID, axis int) float64 {
	g := gamepad.Get(id)
	if g == nil {
		return 0
	}
	return g.Axis(axis)
}

func (i *Input) GamepadAxisNum(id driver.GamepadID) int {
	g := gamepad.Get(id)
	if g == nil {
		return 0
	}
	return g.AxisCount()
}

func (i *Input) GamepadButtonNum(id driver.GamepadID) int {
	g := gamepad.Get(id)
	if g == nil {
		return 0
	}
	return g.ButtonCount()
}

func (i *Input) IsGamepadButtonPressed(id driver.GamepadID, button driver.GamepadButton) bool {
	g := gamepad.Get(id)
	if g == nil {
		return false
	}
	return g.Button(int(button))
}

func (i *Input) IsStandardGamepadButtonPressed(id driver.GamepadID, button driver.StandardGamepadButton) bool {
	g := gamepad.Get(id)
	if g == nil {
		return false
	}
	return g.IsStandardButtonPressed(button)
}

func (i *Input) IsStandardGamepadLayoutAvailable(id driver.GamepadID) bool {
	g := gamepad.Get(id)
	if g == nil {
		return false
	}
	return g.IsStandardLayoutAvailable()
}

func (i *Input) StandardGamepadAxisValue(id driver.GamepadID, axis driver.StandardGamepadAxis) float64 {
	g := gamepad.Get(id)
	if g == nil {
		return 0
	}
	return g.StandardAxisValue(axis)
}

func (i *Input) StandardGamepadButtonValue(id driver.GamepadID, button driver.StandardGamepadButton) float64 {
	g := gamepad.Get(id)
	if g == nil {
		return 0
	}
	return g.StandardButtonValue(button)
}

func (i *Input) VibrateGamepad(id driver.GamepadID, duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	g := gamepad.Get(id)
	if g == nil {
		return
	}
	g.Vibrate(duration, strongMagnitude, weakMagnitude)
}
