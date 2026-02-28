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
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

type nativeGamepadsGC struct{}

func newNativeGamepadsGC() nativeGamepads {
	return &nativeGamepadsGC{}
}

func (*nativeGamepadsGC) init(gamepads *gamepads) error {
	initializeGCGamepads()
	return nil
}

func (*nativeGamepadsGC) update(gamepads *gamepads) error {
	return nil
}

type nativeGamepadGC struct {
	controller           uintptr
	buttonMask           uint16
	hasDualshockTouchpad bool
	hasXboxPaddles       bool
	hasXboxShareButton   bool
	leftMotor            uintptr
	rightMotor           uintptr
	vib                  bool
	vibEnd               time.Time

	axes    []float64
	buttons []bool
	hats    []int
}

func (g *nativeGamepadGC) update(gamepad *gamepads) error {
	g.updateGCGamepad()
	if g.vib && time.Now().Sub(g.vibEnd) >= 0 {
		vibrateGCGamepad(g.rightMotor, g.leftMotor, 0, 0)
		g.vib = false
	}
	return nil
}

func (*nativeGamepadGC) hasOwnStandardLayoutMapping() bool {
	return false
}

func (*nativeGamepadGC) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	return nil
}

func (*nativeGamepadGC) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	return nil
}

func (g *nativeGamepadGC) axisCount() int {
	return len(g.axes)
}

func (g *nativeGamepadGC) buttonCount() int {
	return len(g.buttons)
}

func (g *nativeGamepadGC) hatCount() int {
	return len(g.hats)
}

func (g *nativeGamepadGC) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
}

func (g *nativeGamepadGC) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axes) {
		return 0
	}
	return g.axes[axis]
}

func (g *nativeGamepadGC) isButtonPressed(button int) bool {
	if button < 0 || button >= len(g.buttons) {
		return false
	}
	return g.buttons[button]
}

func (g *nativeGamepadGC) buttonValue(button int) float64 {
	if g.isButtonPressed(button) {
		return 1
	}
	return 0
}

func (g *nativeGamepadGC) hatState(hat int) int {
	if hat < 0 || hat >= len(g.hats) {
		return 0
	}
	return g.hats[hat]
}

func (g *nativeGamepadGC) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	if strongMagnitude <= 0 && weakMagnitude <= 0 {
		g.vib = false
		vibrateGCGamepad(g.leftMotor, g.rightMotor, 0, 0)
		return
	}
	g.vib = true
	g.vibEnd = time.Now().Add(duration)
	vibrateGCGamepad(g.leftMotor, g.rightMotor, strongMagnitude, weakMagnitude)
}
