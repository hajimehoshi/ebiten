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

type nativeGamepadsImpl struct{}

func newNativeGamepadsImpl() nativeGamepads {
	return &nativeGamepadsImpl{}
}

func (*nativeGamepadsImpl) init(gamepads *gamepads) error {
	initializeIOSGamepads()
	return nil
}

func (*nativeGamepadsImpl) update(gamepads *gamepads) error {
	return nil
}

type nativeGamepadImpl struct {
	controller           uintptr
	buttonMask           uint16
	hasDualshockTouchpad bool
	hasXboxPaddles       bool
	hasXboxShareButton   bool

	axes    []float64
	buttons []bool
	hats    []int
}

func (g *nativeGamepadImpl) update(gamepad *gamepads) error {
	g.updateIOSGamepad()
	return nil
}

func (*nativeGamepadImpl) hasOwnStandardLayoutMapping() bool {
	return false
}

func (*nativeGamepadImpl) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	return nil
}

func (*nativeGamepadImpl) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	return nil
}

func (g *nativeGamepadImpl) axisCount() int {
	return len(g.axes)
}

func (g *nativeGamepadImpl) buttonCount() int {
	return len(g.buttons)
}

func (g *nativeGamepadImpl) hatCount() int {
	return len(g.hats)
}

func (g *nativeGamepadImpl) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
}

func (g *nativeGamepadImpl) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axes) {
		return 0
	}
	return g.axes[axis]
}

func (g *nativeGamepadImpl) isButtonPressed(button int) bool {
	if button < 0 || button >= len(g.buttons) {
		return false
	}
	return g.buttons[button]
}

func (g *nativeGamepadImpl) buttonValue(button int) float64 {
	if g.isButtonPressed(button) {
		return 1
	}
	return 0
}

func (g *nativeGamepadImpl) hatState(hat int) int {
	if hat < 0 || hat >= len(g.hats) {
		return 0
	}
	return g.hats[hat]
}

func (g *nativeGamepadImpl) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// TODO: Implement this (#1452)
}
