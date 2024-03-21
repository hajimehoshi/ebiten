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

//go:build !darwin && !js && !linux && !windows

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

type nativeGamepadImpl struct{}

func (*nativeGamepadImpl) update(gamepad *gamepads) error {
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

func (*nativeGamepadImpl) axisCount() int {
	return 0
}

func (*nativeGamepadImpl) buttonCount() int {
	return 0
}

func (*nativeGamepadImpl) hatCount() int {
	return 0
}

func (g *nativeGamepadImpl) isAxisReady(axis int) bool {
	return false
}

func (*nativeGamepadImpl) axisValue(axis int) float64 {
	return 0
}

func (*nativeGamepadImpl) isButtonPressed(button int) bool {
	return false
}

func (*nativeGamepadImpl) buttonValue(button int) float64 {
	return 0
}

func (*nativeGamepadImpl) hatState(hat int) int {
	return hatCentered
}

func (g *nativeGamepadImpl) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
}
