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
	"math"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

func TestMotorMagnitude(t *testing.T) {
	tests := []struct {
		in   float64
		want uint16
	}{
		{
			in:   math.NaN(),
			want: 0,
		},
		{
			in:   math.Inf(-1),
			want: 0,
		},
		{
			in:   -1,
			want: 0,
		},
		{
			in:   0,
			want: 0,
		},
		{
			in:   0.5,
			want: 0x7fff,
		},
		{
			in:   1,
			want: 0xffff,
		},
		{
			in:   2,
			want: 0xffff,
		},
		{
			in:   math.Inf(1),
			want: 0xffff,
		},
	}
	for _, test := range tests {
		if got := gamepad.MotorMagnitude(test.in); got != test.want {
			t.Errorf("gamepad.MotorMagnitude(%v): got: %#x, want: %#x", test.in, got, test.want)
		}
	}
}

func TestStandardLayoutQueryReadsOneReport(t *testing.T) {
	const sdlID = "00000000000000000000000000009301"
	if err := gamepaddb.Update([]byte(sdlID + ",Report Pad,leftx:+a0,lefttrigger:a1,\n")); err != nil {
		t.Fatal(err)
	}

	g := gamepad.NewGamepadForTest(sdlID)

	fullReport := []float64{1, 1}
	var emptyReport []float64
	g.SetReportForTest(fullReport, nil, nil)

	deadline := time.Now().Add(200 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Go(func() {
		for time.Now().Before(deadline) {
			g.SetReportForTest(fullReport, nil, nil)
			g.SetReportForTest(emptyReport, nil, nil)
		}
	})
	wg.Go(func() {
		for time.Now().Before(deadline) {
			if got := g.StandardAxisValue(gamepaddb.StandardAxisLeftStickHorizontal); got != 1 && got != 0 {
				t.Errorf("StandardAxisValue(LeftStickHorizontal) = %v; want 1 or 0 (the value mixes two reports)", got)
				return
			}
			if got := g.StandardButtonValue(gamepaddb.StandardButtonFrontBottomLeft); got != 1 && got != 0 {
				t.Errorf("StandardButtonValue(FrontBottomLeft) = %v; want 1 or 0 (the value mixes two reports)", got)
				return
			}
		}
	})
	wg.Wait()
}

func TestAxisButtonPressedMatchesValue(t *testing.T) {
	const dbID = "00000000000000000000000000009303"
	const ownID = "00000000000000000000000000009304"
	if err := gamepaddb.Update([]byte(dbID + ",Trigger Pad,lefttrigger:a2,\n")); err != nil {
		t.Fatal(err)
	}

	gamepads := []struct {
		name string
		g    *gamepad.Gamepad
	}{
		{
			name: "gamepaddb",
			g:    gamepad.NewGamepadForTest(dbID),
		},
		{
			name: "own layout",
			g: gamepad.NewGamepadWithAxisButtonsForTest(ownID, map[gamepaddb.StandardButton]int{
				gamepaddb.StandardButtonFrontBottomLeft: 2,
			}),
		},
	}
	tests := []struct {
		axis        float64
		wantPressed bool
		wantValue   float64
	}{
		{
			axis:        -1,
			wantPressed: false,
			wantValue:   0,
		},
		{
			axis:        -0.8,
			wantPressed: false,
			wantValue:   0.1,
		},
		{
			axis:        -0.7,
			wantPressed: true,
			wantValue:   0.15,
		},
		{
			axis:        -0.5,
			wantPressed: true,
			wantValue:   0.25,
		},
		{
			axis:        0,
			wantPressed: true,
			wantValue:   0.5,
		},
		{
			axis:        1,
			wantPressed: true,
			wantValue:   1,
		},
	}
	for _, gp := range gamepads {
		for _, test := range tests {
			gp.g.SetReportForTest([]float64{0, 0, test.axis}, nil, nil)
			if got := gp.g.IsStandardButtonPressed(gamepaddb.StandardButtonFrontBottomLeft); got != test.wantPressed {
				t.Errorf("%s: IsStandardButtonPressed(FrontBottomLeft) with axis %v = %t; want %t", gp.name, test.axis, got, test.wantPressed)
			}
			if got := gp.g.StandardButtonValue(gamepaddb.StandardButtonFrontBottomLeft); math.Abs(got-test.wantValue) > 1e-9 {
				t.Errorf("%s: StandardButtonValue(FrontBottomLeft) with axis %v = %v; want %v", gp.name, test.axis, got, test.wantValue)
			}
		}
	}
}

func TestButtonsWithHats(t *testing.T) {
	g := gamepad.NewGamepadForTest("00000000000000000000000000009302")
	g.SetReportForTest(nil, []bool{true, false}, []int{gamepaddb.HatUp | gamepaddb.HatRight, 0})

	if got, want := g.ButtonCountWithHats(), 2+2*4; got != want {
		t.Errorf("ButtonCountWithHats() = %d; want %d", got, want)
	}

	tests := []struct {
		button int
		want   bool
	}{
		{button: -1, want: false},
		{button: 0, want: true},
		{button: 1, want: false},
		{button: 2, want: true},
		{button: 3, want: true},
		{button: 4, want: false},
		{button: 5, want: false},
		{button: 6, want: false},
		{button: 7, want: false},
		{button: 8, want: false},
		{button: 9, want: false},
		{button: 10, want: false},
	}
	for _, test := range tests {
		if got := g.IsButtonPressedWithHats(test.button); got != test.want {
			t.Errorf("IsButtonPressedWithHats(%d) = %t; want %t", test.button, got, test.want)
		}
	}
}
