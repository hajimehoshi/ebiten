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

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/vibrate"
)

// VibrateOptions represents the options for device vibration.
type VibrateOptions struct {
	// Duration is the time duration of the effect.
	Duration time.Duration

	// Magnitude is the strength of the device vibration.
	// The value is in between 0 and 1.
	Magnitude float64
}

// Vibrate vibrates the device with the specified options.
//
// Vibrate works on mobiles and browsers.
//
// On browsers, Magnitude in the options is ignored.
//
// On Android, this line is required in the manifest setting to use Vibrate:
//
//	<uses-permission android:name="android.permission.VIBRATE"/>
//
// On Android, Magnitude in the options is recognized only when the API Level is 26 or newer.
// Otherwise, Magnitude is ignored.
//
// On iOS, CoreHaptics.framework is required to use Vibrate.
//
// On iOS, Vibrate works only when iOS version is 13.0 or newer.
// Otherwise, Vibrate does nothing.
//
// Vibrate is concurrent-safe.
func Vibrate(options *VibrateOptions) {
	vibrate.Vibrate(options.Duration, options.Magnitude)
}

// VibrateGamepadOptions represents the options for gamepad vibration.
type VibrateGamepadOptions struct {
	// Duration is the time duration of the effect.
	Duration time.Duration

	// StrongMagnitude is the rumble intensity of a low-frequency rumble motor.
	// The value is in between 0 and 1.
	StrongMagnitude float64

	// WeakMagnitude is the rumble intensity of a high-frequency rumble motor.
	// The value is in between 0 and 1.
	WeakMagnitude float64
}

// VibrateGamepad vibrates the specified gamepad with the specified options.
//
// VibrateGamepad works only on browsers and Nintendo Switch so far.
//
// VibrateGamepad is concurrent-safe.
func VibrateGamepad(gamepadID GamepadID, options *VibrateGamepadOptions) {
	g := gamepad.Get(gamepadID)
	if g == nil {
		return
	}
	g.Vibrate(options.Duration, options.StrongMagnitude, options.WeakMagnitude)
}
