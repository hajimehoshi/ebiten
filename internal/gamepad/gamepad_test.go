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
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
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
