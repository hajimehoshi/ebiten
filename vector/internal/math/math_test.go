// Copyright 2019 The Ebiten Authors
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

package math_test

import (
	"testing"

	. "github.com/hajimehoshi/ebiten/vector/internal/math"
)

func TestIntersectionAsLine(t *testing.T) {
	cases := []struct {
		S0   Segment
		S1   Segment
		Want Point
	}{
		{
			S0:   Segment{Point{0.5, 0}, Point{0.5, 0.5}},
			S1:   Segment{Point{1, 1}, Point{2, 1}},
			Want: Point{0.5, 1},
		},
		{
			S0:   Segment{Point{0.5, 0}, Point{0.5, 1.5}},
			S1:   Segment{Point{1, 1}, Point{2, 1}},
			Want: Point{0.5, 1},
		},
	}
	for _, c := range cases {
		got := c.S0.IntersectionAsLines(c.S1)
		want := c.Want
		if got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}
}
