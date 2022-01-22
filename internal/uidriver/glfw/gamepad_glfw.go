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

//go:build !android && !js && !darwin && !windows
// +build !android,!js,!darwin,!windows

package glfw

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

type nativeGamepads struct {
}

func (i *Input) updateGamepads() error {
	for id := glfw.Joystick(0); id < glfw.Joystick(len(i.gamepads)); id++ {
		i.gamepads[id].valid = false
		if !id.Present() {
			continue
		}

		buttons := id.GetButtons()

		// A gamepad can be detected even though there are not. Apparently, some special devices are
		// recognized as gamepads by GLFW. In this case, the number of the 'buttons' can exceeds the
		// maximum. Skip such devices as a tentative solution (#1173).
		if len(buttons) > driver.GamepadButtonNum {
			continue
		}

		i.gamepads[id].valid = true

		i.gamepads[id].buttonNum = len(buttons)
		for b := 0; b < len(i.gamepads[id].buttonPressed); b++ {
			if len(buttons) <= b {
				i.gamepads[id].buttonPressed[b] = false
				continue
			}
			i.gamepads[id].buttonPressed[b] = glfw.Action(buttons[b]) == glfw.Press
		}

		axes32 := id.GetAxes()
		i.gamepads[id].axisNum = len(axes32)
		for a := 0; a < len(i.gamepads[id].axes); a++ {
			if len(axes32) <= a {
				i.gamepads[id].axes[a] = 0
				continue
			}
			i.gamepads[id].axes[a] = float64(axes32[a])
		}

		hats := id.GetHats()
		i.gamepads[id].hatsNum = len(hats)
		for h := 0; h < len(i.gamepads[id].hats); h++ {
			if len(hats) <= h {
				i.gamepads[id].hats[h] = 0
				continue
			}
			i.gamepads[id].hats[h] = int(hats[h])
		}

		// Note that GLFW's gamepad GUID follows SDL's GUID.
		i.gamepads[id].guid = id.GetGUID()
		i.gamepads[id].name = id.GetName()
	}

	return nil
}

func (i *Input) AppendGamepadIDs(gamepadIDs []driver.GamepadID) []driver.GamepadID {
	if !i.ui.isRunning() {
		return nil
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	for id, g := range i.gamepads {
		if g.valid {
			gamepadIDs = append(gamepadIDs, driver.GamepadID(id))
		}
	}
	return gamepadIDs
}

func (i *Input) GamepadSDLID(id driver.GamepadID) string {
	if !i.ui.isRunning() {
		return ""
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return ""
	}
	return i.gamepads[id].guid
}

func (i *Input) GamepadName(id driver.GamepadID) string {
	if !i.ui.isRunning() {
		return ""
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return ""
	}
	if name := gamepaddb.Name(i.gamepads[id].guid); name != "" {
		return name
	}
	return i.gamepads[id].name
}

func (i *Input) GamepadAxisNum(id driver.GamepadID) int {
	if !i.ui.isRunning() {
		return 0
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return 0
	}
	return i.gamepads[id].axisNum
}

func (i *Input) GamepadAxisValue(id driver.GamepadID, axis int) float64 {
	if !i.ui.isRunning() {
		return 0
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return 0
	}
	return i.gamepads[id].axes[axis]
}

func (i *Input) GamepadButtonNum(id driver.GamepadID) int {
	if !i.ui.isRunning() {
		return 0
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return 0
	}
	return i.gamepads[id].buttonNum
}

func (i *Input) IsGamepadButtonPressed(id driver.GamepadID, button driver.GamepadButton) bool {
	if !i.ui.isRunning() {
		return false
	}

	i.ui.m.RLock()
	defer i.ui.m.RUnlock()
	if len(i.gamepads) <= int(id) {
		return false
	}
	return i.gamepads[id].buttonPressed[button]
}

func (i *Input) IsStandardGamepadLayoutAvailable(id driver.GamepadID) bool {
	i.ui.m.Lock()
	defer i.ui.m.Unlock()

	if len(i.gamepads) <= int(id) {
		return false
	}
	g := i.gamepads[int(id)]
	return gamepaddb.HasStandardLayoutMapping(g.guid)
}

func (i *Input) StandardGamepadAxisValue(id driver.GamepadID, axis driver.StandardGamepadAxis) float64 {
	i.ui.m.Lock()
	defer i.ui.m.Unlock()

	if len(i.gamepads) <= int(id) {
		return 0
	}
	g := i.gamepads[int(id)]
	return gamepaddb.AxisValue(g.guid, axis, gamepadState{&g})
}

func (i *Input) StandardGamepadButtonValue(id driver.GamepadID, button driver.StandardGamepadButton) float64 {
	i.ui.m.Lock()
	defer i.ui.m.Unlock()

	if len(i.gamepads) <= int(id) {
		return 0
	}
	g := i.gamepads[int(id)]
	return gamepaddb.ButtonValue(g.guid, button, gamepadState{&g})
}

func (i *Input) IsStandardGamepadButtonPressed(id driver.GamepadID, button driver.StandardGamepadButton) bool {
	i.ui.m.Lock()
	defer i.ui.m.Unlock()

	if len(i.gamepads) <= int(id) {
		return false
	}
	g := i.gamepads[int(id)]
	return gamepaddb.IsButtonPressed(g.guid, button, gamepadState{&g})
}

func (i *Input) VibrateGamepad(id driver.GamepadID, duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// TODO: Implement this (#1452)
}

func init() {
	// Confirm that all the hat state values are the same.
	if gamepaddb.HatUp != glfw.HatUp {
		panic("glfw: gamepaddb.HatUp must equal to glfw.HatUp but not")
	}
	if gamepaddb.HatRight != glfw.HatRight {
		panic("glfw: gamepaddb.HatRight must equal to glfw.HatRight but not")
	}
	if gamepaddb.HatDown != glfw.HatDown {
		panic("glfw: gamepaddb.HatDown must equal to glfw.HatDown but not")
	}
	if gamepaddb.HatLeft != glfw.HatLeft {
		panic("glfw: gamepaddb.HatLeft must equal to glfw.HatLeft but not")
	}
}

type gamepadState struct {
	g *gamepad
}

func (s gamepadState) Axis(index int) float64 {
	return s.g.axes[index]
}

func (s gamepadState) Button(index int) bool {
	return s.g.buttonPressed[index]
}

func (s gamepadState) Hat(index int) int {
	return s.g.hats[index]
}
