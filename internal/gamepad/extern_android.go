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

//go:build !ebitencbackend
// +build !ebitencbackend

package gamepad

import (
	"fmt"
)

type AndroidHatDirection int

const (
	AndroidHatDirectionX AndroidHatDirection = iota
	AndroidHatDirectionY
)

func AddAndroidGamepad(androidDeviceID int, name, sdlID string, axisCount, buttonCount, hatCount int) {
	theGamepads.addAndroidGamepad(androidDeviceID, name, sdlID, axisCount, buttonCount, hatCount)
}

func RemoveAndroidGamepad(androidDeviceID int) {
	theGamepads.removeAndroidGamepad(androidDeviceID)
}

func UpdateAndroidGamepadAxis(androidDeviceID int, axis int, value float64) {
	theGamepads.updateAndroidGamepadAxis(androidDeviceID, axis, value)
}

func UpdateAndroidGamepadButton(androidDeviceID int, button Button, pressed bool) {
	theGamepads.updateAndroidGamepadButton(androidDeviceID, button, pressed)
}

func UpdateAndroidGamepadHat(androidDeviceID int, hat int, dir AndroidHatDirection, value int) {
	theGamepads.updateAndroidGamepadHat(androidDeviceID, hat, dir, value)
}

func (g *gamepads) addAndroidGamepad(androidDeviceID int, name, sdlID string, axisCount, buttonCount, hatCount int) {
	g.m.Lock()
	defer g.m.Unlock()

	gp := g.add(name, sdlID)
	gp.androidDeviceID = androidDeviceID
	gp.axes = make([]float64, axisCount)
	gp.buttons = make([]bool, buttonCount)
	gp.hats = make([]int, hatCount)
}

func (g *gamepads) removeAndroidGamepad(androidDeviceID int) {
	g.m.Lock()
	defer g.m.Unlock()

	g.remove(func(gamepad *Gamepad) bool {
		return gamepad.androidDeviceID == androidDeviceID
	})
}

func (g *gamepads) updateAndroidGamepadAxis(androidDeviceID int, axis int, value float64) {
	g.m.Lock()
	defer g.m.Unlock()

	gp := g.find(func(gamepad *Gamepad) bool {
		return gamepad.androidDeviceID == androidDeviceID
	})
	if gp == nil {
		return
	}
	gp.updateAndroidGamepadAxis(axis, value)
}

func (g *gamepads) updateAndroidGamepadButton(androidDeviceID int, button Button, pressed bool) {
	g.m.Lock()
	defer g.m.Unlock()

	gp := g.find(func(gamepad *Gamepad) bool {
		return gamepad.androidDeviceID == androidDeviceID
	})
	if gp == nil {
		return
	}
	gp.updateAndroidGamepadButton(button, pressed)
}

func (g *gamepads) updateAndroidGamepadHat(androidDeviceID int, hat int, dir AndroidHatDirection, value int) {
	g.m.Lock()
	defer g.m.Unlock()

	gp := g.find(func(gamepad *Gamepad) bool {
		return gamepad.androidDeviceID == androidDeviceID
	})
	if gp == nil {
		return
	}
	gp.updateAndroidGamepadHat(hat, dir, value)
}

func (g *Gamepad) updateAndroidGamepadAxis(axis int, value float64) {
	g.m.Lock()
	defer g.m.Unlock()

	if axis < 0 || axis >= len(g.axes) {
		return
	}
	g.axes[axis] = value
}

func (g *Gamepad) updateAndroidGamepadButton(button Button, pressed bool) {
	g.m.Lock()
	defer g.m.Unlock()

	if button < 0 || int(button) >= len(g.buttons) {
		return
	}
	g.buttons[button] = pressed
}

func (g *Gamepad) updateAndroidGamepadHat(hat int, dir AndroidHatDirection, value int) {
	g.m.Lock()
	defer g.m.Unlock()

	if hat < 0 || hat >= len(g.hats) {
		return
	}
	v := g.hats[hat]
	switch dir {
	case AndroidHatDirectionX:
		switch {
		case value < 0:
			v |= hatLeft
			v &^= hatRight
		case value > 0:
			v &^= hatLeft
			v |= hatRight
		default:
			v &^= (hatLeft | hatRight)
		}
	case AndroidHatDirectionY:
		switch {
		case value < 0:
			v |= hatUp
			v &^= hatDown
		case value > 0:
			v &^= hatUp
			v |= hatDown
		default:
			v &^= (hatUp | hatDown)
		}
	default:
		panic(fmt.Sprintf("gamepad: invalid direction: %d", dir))
	}
	g.hats[hat] = v
}
