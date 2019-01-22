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

package affine_test

import (
	"math"
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/affine"
)

func TestColorMScaleOnly(t *testing.T) {
	cases := []struct {
		In  *ColorM
		Out bool
	}{
		{
			nil,
			true,
		},
		{
			(*ColorM)(nil).Translate(0, 0, 0, 0),
			true,
		},
		{
			(*ColorM)(nil).Translate(1, 0, 0, 0),
			false,
		},
		{
			(*ColorM)(nil).Translate(0, 0, 0, -1),
			false,
		},
		{
			(*ColorM)(nil).Scale(1, 1, 1, 1),
			true,
		},
		{
			(*ColorM)(nil).Scale(0, 0, 0, 0),
			true,
		},
		{
			(*ColorM)(nil).Scale(0.1, 0.2, 0.3, 0.4),
			true,
		},
		{
			(*ColorM)(nil).Scale(0.1, 0.2, 0.3, 0.4).Translate(1, 0, 0, 0),
			false,
		},
		{
			(*ColorM)(nil).ChangeHSV(math.Pi/2, 0.5, 0.5),
			false,
		},
		{
			(*ColorM)(nil).SetElement(0, 0, 2),
			true,
		},
		{
			(*ColorM)(nil).SetElement(0, 1, 2),
			false,
		},
	}
	for _, c := range cases {
		got := c.In.ScaleOnly()
		want := c.Out
		if got != want {
			t.Errorf("%v.ScaleOnly(): got: %t, want: %t", c.In, got, want)
		}
	}
}
