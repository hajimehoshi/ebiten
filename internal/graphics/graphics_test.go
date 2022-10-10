// Copyright 2014 Hajime Hoshi
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

package graphics_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
)

func TestInternalImageSize(t *testing.T) {
	testCases := []struct {
		expected int
		arg      int
	}{
		{256, 255},
		{256, 256},
		{512, 257},
	}

	for _, testCase := range testCases {
		got := graphics.InternalImageSize(testCase.arg)
		wanted := testCase.expected
		if wanted != got {
			t.Errorf("Clp(%d) = %d, wanted %d", testCase.arg, got, wanted)
		}

	}
}

func TestAdjustPixel(t *testing.T) {
	tests := []struct {
		X     float32
		Y     float32
		Delta float32
	}{
		{
			X:     -0.1,
			Y:     0.9,
			Delta: 1,
		},
		{
			X:     -1,
			Y:     0,
			Delta: 1,
		},
		{
			X:     -1.9,
			Y:     1.1,
			Delta: 3,
		},
		{
			X:     -2,
			Y:     1,
			Delta: 3,
		},
	}
	for _, tc := range tests {
		if rx, ry := graphics.AdjustDestinationPixelForTesting(tc.X)+tc.Delta, graphics.AdjustDestinationPixelForTesting(tc.Y); rx != ry {
			t.Errorf("adjustDestinationPixel(%f) + 1 must equal to adjustDestinationPixel(%f) but not (%f vs %f)", tc.X, tc.Y, rx, ry)
		}
	}
}

func BenchmarkAdjustPixel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		graphics.AdjustDestinationPixelForTesting(float32(i) / 17)
	}
}
