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

package inpututil

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestGamepadButtonOutOfRange(t *testing.T) {
	for _, button := range []ebiten.GamepadButton{-1, ebiten.GamepadButtonMax + 1} {
		if got := GamepadButtonPressDuration(0, button); got != 0 {
			t.Errorf("GamepadButtonPressDuration(%d): got: %d, want: 0", button, got)
		}
		if IsGamepadButtonJustPressed(0, button) {
			t.Errorf("IsGamepadButtonJustPressed(%d): got true, want: false", button)
		}
		if IsGamepadButtonJustReleased(0, button) {
			t.Errorf("IsGamepadButtonJustReleased(%d): got true, want: false", button)
		}
	}
}

func TestStandardGamepadButtonOutOfRange(t *testing.T) {
	for _, button := range []ebiten.StandardGamepadButton{-1, ebiten.StandardGamepadButtonMax + 1} {
		if got := StandardGamepadButtonPressDuration(0, button); got != 0 {
			t.Errorf("StandardGamepadButtonPressDuration(%d): got: %d, want: 0", button, got)
		}
		if IsStandardGamepadButtonJustPressed(0, button) {
			t.Errorf("IsStandardGamepadButtonJustPressed(%d): got true, want: false", button)
		}
		if IsStandardGamepadButtonJustReleased(0, button) {
			t.Errorf("IsStandardGamepadButtonJustReleased(%d): got true, want: false", button)
		}
	}
}
