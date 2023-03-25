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

//go:build (android || ios) && !nintendosdk

package ui

type TouchForInput struct {
	ID TouchID

	// X is in device-independent pixels.
	X float64

	// Y is in device-independent pixels.
	Y float64
}

func (u *userInterfaceImpl) updateInputState(keys map[Key]struct{}, runes []rune, touches []TouchForInput) {
	u.m.Lock()
	defer u.m.Unlock()

	for k := range u.inputState.KeyPressed {
		_, ok := keys[Key(k)]
		u.inputState.KeyPressed[k] = ok
	}

	u.inputState.Runes = append(u.inputState.Runes, runes...)

	u.inputState.Touches = u.inputState.Touches[:0]
	for _, t := range touches {
		x, y := u.context.clientPositionToLogicalPosition(t.X, t.Y, u.DeviceScaleFactor())
		u.inputState.Touches = append(u.inputState.Touches, Touch{
			ID: t.ID,
			X:  int(x),
			Y:  int(y),
		})
	}
}

func KeyName(key Key) string {
	// TODO: Implement this.
	return ""
}
