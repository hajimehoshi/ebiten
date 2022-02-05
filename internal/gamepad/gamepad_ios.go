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

//go:build ios && !ebitencbackend
// +build ios,!ebitencbackend

package gamepad

import (
	"time"
)

type nativeGamepads struct{}

func (*nativeGamepads) init(gamepads *gamepads) error {
	initializeIOSGamepads()
	return nil
}

func (*nativeGamepads) update(gamepads *gamepads) error {
	return nil
}

type nativeGamepad struct {
	controller           uintptr
	buttonMask           uint16
	hasDualshockTouchpad bool
	hasXboxPaddles       bool
	hasXboxShareButton   bool

	axes    []float64
	buttons []bool
	hats    []int
}

func (g *nativeGamepad) update(gamepad *gamepads) error {
	g.updateIOSGamepad()
	return nil
}

func (*nativeGamepad) hasOwnStandardLayoutMapping() bool {
	return false
}

func (g *nativeGamepad) axisCount() int {
	return len(g.axes)
}

func (g *nativeGamepad) buttonCount() int {
	return len(g.buttons)
}

func (g *nativeGamepad) hatCount() int {
	return len(g.hats)
}

func (g *nativeGamepad) axisValue(axis int) float64 {
	if axis < 0 || axis >= len(g.axes) {
		return 0
	}
	return g.axes[axis]
}

func (g *nativeGamepad) isButtonPressed(button int) bool {
	if button < 0 || button >= len(g.buttons) {
		return false
	}
	return g.buttons[button]
}

func (*nativeGamepad) buttonValue(button int) float64 {
	panic("gamepad: buttonValue is not implemented")
}

func (g *nativeGamepad) hatState(hat int) int {
	if hat < 0 || hat >= len(g.hats) {
		return 0
	}
	return g.hats[hat]
}

func (g *nativeGamepad) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// TODO: Implement this (#1452)
}
