// Copyright 2023 The Ebitengine Authors
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

package restorable_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
)

func areEqualRectangles(a, b []image.Rectangle) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestRemoveDuplicatedRegions(t *testing.T) {
	cases := []struct {
		In  []image.Rectangle
		Out []image.Rectangle
	}{
		{
			In:  nil,
			Out: nil,
		},
		{
			In: []image.Rectangle{
				image.Rect(0, 0, 2, 2),
			},
			Out: []image.Rectangle{
				image.Rect(0, 0, 2, 2),
			},
		},
		{
			In: []image.Rectangle{
				image.Rect(0, 0, 2, 2),
				image.Rect(0, 0, 1, 1),
			},
			Out: []image.Rectangle{
				image.Rect(0, 0, 2, 2),
			},
		},
		{
			In: []image.Rectangle{
				image.Rect(0, 0, 1, 1),
				image.Rect(0, 0, 2, 2),
			},
			Out: []image.Rectangle{
				image.Rect(0, 0, 2, 2),
			},
		},
		{
			In: []image.Rectangle{
				image.Rect(0, 0, 1, 3),
				image.Rect(0, 0, 2, 2),
				image.Rect(0, 0, 3, 1),
			},
			Out: []image.Rectangle{
				image.Rect(0, 0, 1, 3),
				image.Rect(0, 0, 2, 2),
				image.Rect(0, 0, 3, 1),
			},
		},
		{
			In: []image.Rectangle{
				image.Rect(0, 0, 1, 3),
				image.Rect(0, 0, 2, 2),
				image.Rect(0, 0, 3, 1),
				image.Rect(0, 0, 4, 4),
			},
			Out: []image.Rectangle{
				image.Rect(0, 0, 4, 4),
			},
		},
		{
			In: []image.Rectangle{
				image.Rect(0, 0, 1, 3),
				image.Rect(0, 0, 2, 2),
				image.Rect(0, 0, 3, 1),
				image.Rect(0, 0, 4, 4),
				image.Rect(1, 1, 2, 2),
			},
			Out: []image.Rectangle{
				image.Rect(0, 0, 4, 4),
			},
		},
		{
			In: []image.Rectangle{
				image.Rect(0, 0, 1, 3),
				image.Rect(0, 0, 2, 2),
				image.Rect(0, 0, 3, 1),
				image.Rect(0, 0, 4, 4),
				image.Rect(0, 0, 5, 5),
			},
			Out: []image.Rectangle{
				image.Rect(0, 0, 5, 5),
			},
		},
	}

	for _, c := range cases {
		n := restorable.RemoveDuplicatedRegions(c.In)
		got := c.In[:n]
		want := c.Out
		if !areEqualRectangles(got, want) {
			t.Errorf("restorable.RemoveDuplicatedRegions(%#v): got: %#v, want: %#v", c.In, got, want)
		}
	}
}
