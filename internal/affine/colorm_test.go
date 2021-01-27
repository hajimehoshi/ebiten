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
	"math/rand"
	"testing"

	. "github.com/hajimehoshi/ebiten/v2/internal/affine"
)

func TestColorMScale(t *testing.T) {
	cases := []struct {
		In  *ColorM
		Out *ColorM
	}{
		{
			nil,
			(*ColorM)(nil).Scale(0.25, 0.5, 0.75, 1),
		},
		{
			(*ColorM)(nil).Scale(0.5, 0.5, 0.5, 0.8),
			(*ColorM)(nil).Scale(0.125, 0.25, 0.375, 0.8),
		},
		{
			(*ColorM)(nil).Translate(0, 0, 0, 0),
			(*ColorM)(nil).Scale(0.25, 0.5, 0.75, 1),
		},
	}
	for _, c := range cases {
		got := c.In.Scale(0.25, 0.5, 0.75, 1)
		want := c.Out
		if got != want {
			t.Errorf("%v.Scale(): got: %v, want: %v", c.In, got, want)
		}
	}
}

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

func TestColorMIsInvertible(t *testing.T) {
	m := &ColorM{}
	m = m.SetElement(1, 0, .5)
	m = m.SetElement(1, 1, .5)
	m = m.SetElement(1, 2, .5)
	m = m.SetElement(1, 3, .5)
	m = m.SetElement(1, 4, .5)

	cidentity := &ColorM{}
	cinvalid := &ColorM{}
	cinvalid = cinvalid.SetElement(0, 0, 0)
	cinvalid = cinvalid.SetElement(1, 1, 0)
	cinvalid = cinvalid.SetElement(2, 2, 0)
	cinvalid = cinvalid.SetElement(3, 3, 0)

	cases := []struct {
		In  *ColorM
		Out bool
	}{
		{
			nil,
			true,
		},
		{
			cidentity,
			true,
		},
		{
			m,
			true,
		},
		{
			cinvalid,
			false,
		},
	}
	for _, c := range cases {
		got := c.In.IsInvertible()
		want := c.Out
		if got != want {
			t.Errorf("%v.IsInvertible(): got: %t, want: %t", c.In, got, want)
		}
	}
}

func arrayToColorM(es [4][5]float32) *ColorM {
	var a = &ColorM{}
	for j := 0; j < 5; j++ {
		for i := 0; i < 4; i++ {
			a = a.SetElement(i, j, es[i][j])
		}
	}
	return a
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func equalWithDelta(a, b *ColorM, delta float32) bool {
	for j := 0; j < 5; j++ {
		for i := 0; i < 4; i++ {
			ea := a.Element(i, j)
			eb := b.Element(i, j)
			if abs(ea-eb) > delta {
				return false
			}
		}
	}
	return true
}

func TestColorMInvert(t *testing.T) {
	cases := []struct {
		In  *ColorM
		Out *ColorM
	}{
		{
			In:  nil,
			Out: nil,
		},
		{
			In: arrayToColorM([4][5]float32{
				{1, 0, 0, 0, 0},
				{8, 1, 0, 0, 0},
				{-9, 0, 1, 0, 0},
				{7, 4, 2, 1, 0},
			}),
			Out: arrayToColorM([4][5]float32{
				{1, 0, 0, 0, 0},
				{-8, 1, 0, 0, 0},
				{9, 0, 1, 0, 0},
				{7, -4, -2, 1, 0},
			}),
		},
		{
			In: arrayToColorM([4][5]float32{
				{1, 2, 3, 4, 5},
				{5, 1, 2, 3, 4},
				{4, 5, 1, 2, 3},
				{3, 4, 5, 1, 2},
			}),
			Out: arrayToColorM([4][5]float32{
				{-6 / 35.0, 3 / 14.0, 1 / 70.0, 1 / 70.0, -1 / 14.0},
				{1 / 35.0, -13 / 70.0, 3 / 14.0, 1 / 70.0, -1 / 14.0},
				{1 / 35.0, 1 / 70.0, -13 / 70.0, 3 / 14.0, -1 / 14.0},
				{9 / 35.0, 1 / 35.0, 1 / 35.0, -6 / 35.0, -8 / 7.0},
			}),
		},
	}

	for _, c := range cases {
		if got, want := c.In.Invert(), c.Out; !equalWithDelta(got, want, 1e-6) {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}
}

func BenchmarkColorMInvert(b *testing.B) {
	r := rand.Float32

	b.StopTimer()
	var m *ColorM
	for m == nil || !m.IsInvertible() {
		m = arrayToColorM([4][5]float32{
			{r(), r(), r(), r(), r() * 10},
			{r(), r(), r(), r(), r() * 10},
			{r(), r(), r(), r(), r() * 10},
			{r(), r(), r(), r(), r() * 10},
		})
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		m = m.Invert()
	}
}
