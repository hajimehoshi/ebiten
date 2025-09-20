// Copyright 2016 Hajime Hoshi
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

//go:build android || ios

package ui

type TouchForInput struct {
	ID TouchID

	// X is in device-independent pixels.
	X float64

	// Y is in device-independent pixels.
	Y float64
}

func (u *UserInterface) updateInputStateFromOutside(keyPressedTimes, keyReleasedTimes [KeyMax + 1]InputTime, runes []rune, touches []TouchForInput) {
	u.m.Lock()
	defer u.m.Unlock()

	u.inputState.KeyPressedTimes = keyPressedTimes
	u.inputState.KeyReleasedTimes = keyReleasedTimes
	u.inputState.Runes = append(u.inputState.Runes, runes...)
	u.touches = u.touches[:0]
	for _, t := range touches {
		u.touches = append(u.touches, t)
	}
}

func (u *UserInterface) updateInputStateForFrame() error {
	u.m.Lock()
	defer u.m.Unlock()

	s := theMonitor.DeviceScaleFactor()

	u.inputState.Touches = u.inputState.Touches[:0]
	for _, t := range u.touches {
		x, y := u.context.clientPositionToLogicalPosition(t.X, t.Y, s)
		u.inputState.Touches = append(u.inputState.Touches, Touch{
			ID: t.ID,
			X:  int(x),
			Y:  int(y),
		})
	}
	return nil
}

func (u *UserInterface) KeyName(key Key) string {
	// TODO: Implement this.
	return ""
}
