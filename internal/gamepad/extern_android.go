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

package gamepad

import (
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

func AddAndroidGamepad(androidDeviceID int, name, sdlID string, axisCount, hatCount int) {
	theGamepads.addAndroidGamepad(androidDeviceID, name, sdlID, axisCount, hatCount)
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

func UpdateAndroidGamepadHat(androidDeviceID int, hat int, xValue, yValue int) {
	theGamepads.updateAndroidGamepadHat(androidDeviceID, hat, xValue, yValue)
}

func (g *gamepads) addAndroidGamepad(androidDeviceID int, name, sdlID string, axisCount, hatCount int) {
	g.m.Lock()
	defer g.m.Unlock()

	gp := g.add(name, sdlID)
	gp.native = &nativeGamepadImpl{
		androidDeviceID: androidDeviceID,
		axesReady:       make([]bool, axisCount),
		axes:            make([]float64, axisCount),
		buttons:         make([]bool, gamepaddb.SDLControllerButtonMax+1),
		hats:            make([]int, hatCount),
	}
}

func (g *gamepads) removeAndroidGamepad(androidDeviceID int) {
	g.m.Lock()
	defer g.m.Unlock()

	g.remove(func(gamepad *Gamepad) bool {
		return gamepad.native.(*nativeGamepadImpl).androidDeviceID == androidDeviceID
	})
}

func (g *gamepads) updateAndroidGamepadAxis(androidDeviceID int, axis int, value float64) {
	g.m.Lock()
	defer g.m.Unlock()

	gp := g.find(func(gamepad *Gamepad) bool {
		return gamepad.native.(*nativeGamepadImpl).androidDeviceID == androidDeviceID
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
		return gamepad.native.(*nativeGamepadImpl).androidDeviceID == androidDeviceID
	})
	if gp == nil {
		return
	}
	gp.updateAndroidGamepadButton(button, pressed)
}

func (g *gamepads) updateAndroidGamepadHat(androidDeviceID int, hat int, xValue, yValue int) {
	g.m.Lock()
	defer g.m.Unlock()

	gp := g.find(func(gamepad *Gamepad) bool {
		return gamepad.native.(*nativeGamepadImpl).androidDeviceID == androidDeviceID
	})
	if gp == nil {
		return
	}
	gp.updateAndroidGamepadHat(hat, xValue, yValue)
}

func (g *Gamepad) updateAndroidGamepadAxis(axis int, value float64) {
	g.m.Lock()
	defer g.m.Unlock()

	n := g.native.(*nativeGamepadImpl)
	if axis < 0 || axis >= len(n.axes) {
		return
	}
	n.axes[axis] = value

	// MotionEvent with 0 value can be sent when a gamepad is connected even though an axis is not touched (#2598).
	// This is problematic when an axis is a trigger button where -1 should be the default value.
	// When MotionEvent with non-0 value is sent, it seems fine to assume that the axis is actually touched and ready.
	if value != 0 {
		n.axesReady[axis] = true
	}
}

func (g *Gamepad) updateAndroidGamepadButton(button Button, pressed bool) {
	g.m.Lock()
	defer g.m.Unlock()

	n := g.native.(*nativeGamepadImpl)
	if button < 0 || int(button) >= len(n.buttons) {
		return
	}
	n.buttons[button] = pressed
}

func (g *Gamepad) updateAndroidGamepadHat(hat int, xValue, yValue int) {
	g.m.Lock()
	defer g.m.Unlock()

	n := g.native.(*nativeGamepadImpl)
	if hat < 0 || hat >= len(n.hats) {
		return
	}
	var v int
	switch {
	case xValue < 0:
		v |= hatLeft
	case xValue > 0:
		v |= hatRight
	}
	switch {
	case yValue < 0:
		v |= hatUp
	case yValue > 0:
		v |= hatDown
	}
	n.hats[hat] = v

	// Update the gamepad buttons in addition to hats.
	// See https://github.com/libsdl-org/SDL/blob/47f2373dc13b66c48bf4024fcdab53cd0bdd59bb/src/joystick/android/SDL_sysjoystick.c#L290-L301
	n.buttons[gamepaddb.SDLControllerButtonDpadLeft] = v&hatLeft != 0
	n.buttons[gamepaddb.SDLControllerButtonDpadRight] = v&hatRight != 0
	n.buttons[gamepaddb.SDLControllerButtonDpadUp] = v&hatUp != 0
	n.buttons[gamepaddb.SDLControllerButtonDpadDown] = v&hatDown != 0
}
