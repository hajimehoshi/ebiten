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
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

func TestGameControllerButtonsWithHats(t *testing.T) {
	tests := []struct {
		name            string
		buttons         []bool
		hats            []int
		wantButtonCount int
		wantPressed     []int
	}{
		{
			name:            "no hats",
			buttons:         []bool{true, false},
			wantButtonCount: 2,
			wantPressed:     []int{0},
		},
		{
			name:            "one hat",
			buttons:         []bool{true, false, true, true, false, false},
			hats:            []int{gamepaddb.HatUp | gamepaddb.HatRight},
			wantButtonCount: 6,
			wantPressed:     []int{0, 2, 3},
		},
		{
			name:            "two hats",
			buttons:         []bool{false, true, true, false, false, true, false, false, true, false},
			hats:            []int{gamepaddb.HatUp | gamepaddb.HatLeft, gamepaddb.HatDown},
			wantButtonCount: 10,
			wantPressed:     []int{1, 2, 5, 8},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := gamepad.NewGCGamepadForTest(test.buttons, test.hats)
			if got := g.ButtonCountWithHats(); got != test.wantButtonCount {
				t.Errorf("ButtonCountWithHats() = %d; want %d", got, test.wantButtonCount)
			}
			for button := -1; button <= len(test.buttons)+4*len(test.hats); button++ {
				if got, want := g.IsButtonPressedWithHats(button), slices.Contains(test.wantPressed, button); got != want {
					t.Errorf("IsButtonPressedWithHats(%d) = %t; want %t", button, got, want)
				}
			}
		})
	}
}
