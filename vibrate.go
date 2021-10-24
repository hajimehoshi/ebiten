// Copyright 2021 The Ebiten Authors
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

package ebiten

import (
	"time"
)

// Vibrate vibrates the device.
//
// Vibrate works on mobiles and browsers.
// On browsers, StrongManitude and WeakMagnitude might be ignored.
//
// Vibrate is concurrent-safe.
func Vibrate(duration time.Duration) {
	uiDriver().Vibrate(duration)
}

// VibrateGamepadOptions represents the options to vibrate a gamepad.
type VibrateGamepadOptions struct {
	// Duration is the time duration of the effect.
	Duration time.Duration

	// StrongMagnitude is the rumble intensity of a low-frequency rumble motor.
	// The value is in between 0 and 1.
	StrongMagnitude float64

	// StrongMagnitude is the rumble intensity of a high-frequency rumble motor.
	// The value is in between 0 and 1.
	WeakMagnitude float64
}

// VibrateGamepad vibrates the specified gamepad with the specified options.
//
// VibrateGamepad is concurrent-safe.
func VibrateGamepad(gamepadID GamepadID, options *VibrateGamepadOptions) {
	uiDriver().Input().VibrateGamepad(gamepadID, options.Duration, options.StrongMagnitude, options.WeakMagnitude)
}
