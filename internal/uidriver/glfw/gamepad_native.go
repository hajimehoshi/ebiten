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

//go:build darwin && !ios
// +build darwin,!ios

package glfw

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	gamepadpkg "github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

type nativeGamepads struct{}

// updateGamepads must be called on the main thread.
func (i *Input) updateGamepads() {
	gamepadpkg.Update()
}

func (i *Input) AppendGamepadIDs(gamepadIDs []driver.GamepadID) []driver.GamepadID {
	var gs []driver.GamepadID
	i.ui.t.Call(func() {
		gs = gamepadpkg.AppendGamepadIDs(gamepadIDs)
	})
	return gs
}

func (i *Input) gamepad(id driver.GamepadID) *gamepadpkg.Gamepad {
	var g *gamepadpkg.Gamepad
	i.ui.t.Call(func() {
		g = gamepadpkg.Get(id)
	})
	return g
}

func (i *Input) GamepadSDLID(id driver.GamepadID) string {
	g := i.gamepad(id)
	if g == nil {
		return ""
	}
	return g.SDLID()
}

func (i *Input) GamepadName(id driver.GamepadID) string {
	g := i.gamepad(id)
	if g == nil {
		return ""
	}
	return g.Name()
}

func (i *Input) GamepadAxisNum(id driver.GamepadID) int {
	g := i.gamepad(id)
	if g == nil {
		return 0
	}
	return g.AxisNum()
}

func (i *Input) GamepadAxisValue(id driver.GamepadID, axis int) float64 {
	g := i.gamepad(id)
	if g == nil {
		return 0
	}
	return g.Axis(axis)
}

func (i *Input) GamepadButtonNum(id driver.GamepadID) int {
	g := i.gamepad(id)
	if g == nil {
		return 0
	}

	// For backward compatibility, hats are treated as buttons in GLFW.
	return g.ButtonNum() + g.HatNum()*4
}

func (i *Input) IsGamepadButtonPressed(id driver.GamepadID, button driver.GamepadButton) bool {
	g := i.gamepad(id)
	if g == nil {
		return false
	}

	nbuttons := g.ButtonNum()
	if int(button) < nbuttons {
		return g.Button(int(button))
	}

	// For backward compatibility, hats are treated as buttons in GLFW.
	if hat := (int(button) - nbuttons) / 4; hat < g.HatNum() {
		dir := (int(button) - nbuttons) % 4
		return g.Hat(hat)&(1<<dir) != 0
	}

	return false
}

func (i *Input) IsStandardGamepadLayoutAvailable(id driver.GamepadID) bool {
	g := i.gamepad(id)
	if g == nil {
		return false
	}
	return g.IsStandardLayoutAvailable()
}

func (i *Input) StandardGamepadAxisValue(id driver.GamepadID, axis driver.StandardGamepadAxis) float64 {
	g := i.gamepad(id)
	if g == nil {
		return 0
	}
	return g.StandardAxisValue(axis)
}

func (i *Input) StandardGamepadButtonValue(id driver.GamepadID, button driver.StandardGamepadButton) float64 {
	g := i.gamepad(id)
	if g == nil {
		return 0
	}
	return g.StandardButtonValue(button)
}

func (i *Input) IsStandardGamepadButtonPressed(id driver.GamepadID, button driver.StandardGamepadButton) bool {
	g := i.gamepad(id)
	if g == nil {
		return false
	}
	return g.IsStandardButtonPressed(button)
}

func (i *Input) VibrateGamepad(id driver.GamepadID, duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	g := i.gamepad(id)
	if g == nil {
		return
	}
	g.Vibrate(duration, strongMagnitude, weakMagnitude)
}
