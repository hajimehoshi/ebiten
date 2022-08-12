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

//go:build !nintendosdk
// +build !nintendosdk

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
	return nil
}

func (*nativeGamepadsImpl) update(gamepads *gamepads) error {
	return nil
}

type nativeGamepadImpl struct {
	androidDeviceID int

	axes    []float64
	buttons []bool
	hats    []int
}

func (*nativeGamepadImpl) update(gamepad *gamepads) error {
	// Do nothing. The state of gamepads are given via APIs in extern_android.go.
	return nil
}

func (*nativeGamepadImpl) hasOwnStandardLayoutMapping() bool {
	return false
}

func (*nativeGamepadImpl) isStandardAxisAvailableInOwnMapping(axis gamepaddb.StandardAxis) bool {
	return false
}

func (*nativeGamepadImpl) isStandardButtonAvailableInOwnMapping(button gamepaddb.StandardButton) bool {
	return false
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

func (*nativeGamepadImpl) buttonValue(button int) float64 {
	panic("gamepad: buttonValue is not implemented")
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
