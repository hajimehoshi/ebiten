// Copyright 2018 The Ebiten Authors
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

package graphicsutil_test

import (
	"math"
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/graphicsutil"
)

func TestMipmapLevel(t *testing.T) {
	inf := float32(math.Inf(1))
	cases := []struct {
		In  float32
		Out int
	}{
		{0, -1},
		{1, 0},
		{-1, 0},
		{2, 0},
		{-2, 0},
		{100, 0},
		{-100, 0},
		{1.0 / 2.0, 0},
		{-1.0 / 2.0, 0},
		{1.0 / 4.0, 0},
		{-1.0 / 4.0, 0},
		{math.Nextafter32(1.0/4.0, 0), 1},
		{math.Nextafter32(-1.0/4.0, 0), 1},
		{1.0 / 8.0, 1},
		{-1.0 / 8.0, 1},
		{1.0 / 16.0, 1},
		{-1.0 / 16.0, 1},
		{math.Nextafter32(1.0/16.0, 0), 2},
		{math.Nextafter32(-1.0/16.0, 0), 2},
		{math.Nextafter32(1.0/256.0, 0), 4},
		{math.Nextafter32(-1.0/256.0, 0), 4},
		{math.SmallestNonzeroFloat32, 74},
		{-math.SmallestNonzeroFloat32, 74},
		{inf, 0},
		{-inf, 0},
	}

	for _, c := range cases {
		got := MipmapLevel(c.In)
		want := c.Out
		if got != want {
			t.Errorf("MipmapLevel(%v): got %v, want %v", c.In, got, want)
		}
	}
}
