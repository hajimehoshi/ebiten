// Copyright 2026 The Ebitengine Authors
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
)

// rumbler drives the vibration motors of one gamepad. Rumble on Windows goes
// through a different API per device family, so every nativeGamepadDesktop is
// assigned the rumbler for its family when the device is detected:
//
//   - XInput devices: xinputRumbler
//   - PlayStation controllers: sonyRumbler
//   - other DirectInput devices: noRumbler
type rumbler interface {
	// vibrate starts vibrating the motors for the duration, or stops them
	// when both magnitudes are 0 or less. The magnitudes are in the range
	// 0 to 1.
	vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64)

	// update stops the motors once the requested duration has passed. It is
	// called once per frame, as the device keeps rumbling until told
	// otherwise.
	update()

	// close releases the rumbler's resources when the gamepad is
	// disconnected.
	close()
}

// noRumbler is for devices whose rumble is not supported.
//
// TODO: Support rumble for DirectInput devices with force feedback (#2014).
type noRumbler struct{}

func (noRumbler) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
}

func (noRumbler) update() {
}

func (noRumbler) close() {
}

// xinputRumbler drives an XInput device's motors.
type xinputRumbler struct {
	index  int
	vib    bool
	vibEnd time.Time
}

func (x *xinputRumbler) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	if strongMagnitude <= 0 && weakMagnitude <= 0 {
		x.stop()
		return
	}

	x.vib = true
	x.vibEnd = time.Now().Add(duration)
	_ = _XInputSetState(uint32(x.index), &_XINPUT_VIBRATION{
		wLeftMotorSpeed:  motorMagnitude(strongMagnitude),
		wRightMotorSpeed: motorMagnitude(weakMagnitude),
	})
}

func (x *xinputRumbler) update() {
	if x.vib && time.Since(x.vibEnd) >= 0 {
		x.stop()
	}
}

func (x *xinputRumbler) stop() {
	x.vib = false
	_ = _XInputSetState(uint32(x.index), &_XINPUT_VIBRATION{})
}

func (x *xinputRumbler) close() {
}
