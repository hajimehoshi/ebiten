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
