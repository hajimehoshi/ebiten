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

//go:build nintendosdk

package ui

import (
	"github.com/hajimehoshi/ebiten/v2/internal/nintendosdk"
)

func (u *userInterfaceImpl) updateInputState() {
	u.nativeTouches = u.nativeTouches[:0]
	u.nativeTouches = nintendosdk.AppendTouches(u.nativeTouches)

	for i := range u.inputState.Touches {
		u.inputState.Touches[i].Valid = false
	}
	for i, t := range u.nativeTouches {
		x, y := u.context.clientPositionToLogicalPosition(float64(t.X), float64(t.Y), deviceScaleFactor)
		u.inputState.Touches[i] = Touch{
			Valid: true,
			ID:    TouchID(t.ID),
			X:     int(x),
			Y:     int(y),
		}
	}
}

func KeyName(key Key) string {
	return ""
}
