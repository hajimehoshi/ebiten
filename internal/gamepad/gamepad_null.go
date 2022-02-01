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

//go:build (!darwin || ios) && !js && !windows
// +build !darwin ios
// +build !js
// +build !windows

package gamepad

import (
	"time"
)

type nativeGamepads struct{}

func (*nativeGamepads) init(gamepads *gamepads) error {
	return nil
}

func (*nativeGamepads) update(gamepads *gamepads) error {
	return nil
}

type nativeGamepad struct{}

func (*nativeGamepad) update(gamepad *gamepads) error {
	return nil
}

func (*nativeGamepad) hasOwnStandardLayoutMapping() bool {
	return false
}

func (*nativeGamepad) axisCount() int {
	return 0
}

func (*nativeGamepad) buttonCount() int {
	return 0
}

func (*nativeGamepad) hatCount() int {
	return 0
}

func (*nativeGamepad) axisValue(axis int) float64 {
	return 0
}

func (*nativeGamepad) isButtonPressed(button int) bool {
	return false
}

func (*nativeGamepad) buttonValue(button int) float64 {
	panic("gamepad: buttonValue is not implemented")
}

func (*nativeGamepad) hatState(hat int) int {
	return hatCentered
}

func (g *nativeGamepad) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// TODO: Implement this (#1452)
}
