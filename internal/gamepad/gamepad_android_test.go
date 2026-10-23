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

package gamepad_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

func TestAndroidGamepadGenericButtons(t *testing.T) {
	const deviceID = 1234
	const sdlID = "android-generic-buttons"
	gamepad.AddAndroidGamepad(deviceID, "Generic Pad", sdlID, 0, 0)
	defer gamepad.RemoveAndroidGamepad(deviceID)

	var g *gamepad.Gamepad
	for _, id := range gamepad.AppendGamepadIDs(nil) {
		if candidate := gamepad.Get(id); candidate.SDLID() == sdlID {
			g = candidate
			break
		}
	}
	if g == nil {
		t.Fatal("Android gamepad was not added")
	}
	if got, want := g.ButtonCount(), gamepad.ButtonCount; got != want {
		t.Errorf("ButtonCount() = %d; want %d", got, want)
	}
	for button := gamepad.Button20; button < gamepad.ButtonCount; button++ {
		gamepad.UpdateAndroidGamepadButton(deviceID, button, true)
		if !g.Button(int(button)) {
			t.Errorf("Button(%d) = false after pressing it", button)
		}
		gamepad.UpdateAndroidGamepadButton(deviceID, button, false)
		if g.Button(int(button)) {
			t.Errorf("Button(%d) = true after releasing it", button)
		}
	}
	gamepad.UpdateAndroidGamepadButton(deviceID, gamepad.ButtonCount, true)
	if g.Button(gamepad.ButtonCount) {
		t.Errorf("Button(%d) = true; want false", gamepad.ButtonCount)
	}
}
