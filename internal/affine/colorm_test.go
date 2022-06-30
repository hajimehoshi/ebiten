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

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
)

func TestColorMScale(t *testing.T) {
	cases := []struct {
		In  affine.ColorM
		Out affine.ColorM
	}{
		{
			affine.ColorMIdentity{},
			affine.ColorMIdentity{}.Scale(0.25, 0.5, 0.75, 1),
		},
		{
			affine.ColorMIdentity{}.Scale(0.5, 0.5, 0.5, 0.8),
			affine.ColorMIdentity{}.Scale(0.125, 0.25, 0.375, 0.8),
		},
		{
			affine.ColorMIdentity{}.Translate(0, 0, 0, 0),
			affine.ColorMIdentity{}.Scale(0.25, 0.5, 0.75, 1),
		},
	}
	for _, c := range cases {
		got := c.In.Scale(0.25, 0.5, 0.75, 1)
		want := c.Out
		if !got.Equals(want) {
			t.Errorf("%v.Scale(): got: %v, want: %v", c.In, got, want)
		}
	}
}

func TestColorMScaleOnly(t *testing.T) {
	cases := []struct {
		In  affine.ColorM
		Out bool
	}{
		{
			affine.ColorMIdentity{},
			true,
		},
		{
			affine.ColorMIdentity{}.Translate(0, 0, 0, 0),
			true,
		},
		{
			affine.ColorMIdentity{}.Translate(1, 0, 0, 0),
			false,
		},
		{
			affine.ColorMIdentity{}.Translate(0, 0, 0, -1),
			false,
		},
		{
			affine.ColorMIdentity{}.Scale(1, 1, 1, 1),
			true,
		},
		{
			affine.ColorMIdentity{}.Scale(0, 0, 0, 0),
			true,
		},
		{
			affine.ColorMIdentity{}.Scale(0.1, 0.2, 0.3, 0.4),
			true,
		},
		{
			affine.ColorMIdentity{}.Scale(0.1, 0.2, 0.3, 0.4).Translate(1, 0, 0, 0),
			false,
		},
		{
			affine.ChangeHSV(affine.ColorMIdentity{}, math.Pi/2, 0.5, 0.5),
			false,
		},
		{
			affine.ColorMSetElement(affine.ColorMIdentity{}, 0, 0, 2),
			true,
		},
		{
			affine.ColorMSetElement(affine.ColorMIdentity{}, 0, 1, 2),
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
	var m affine.ColorM = affine.ColorMIdentity{}
	m = affine.ColorMSetElement(m, 1, 0, .5)
	m = affine.ColorMSetElement(m, 1, 1, .5)
	m = affine.ColorMSetElement(m, 1, 2, .5)
	m = affine.ColorMSetElement(m, 1, 3, .5)
	m = affine.ColorMSetElement(m, 1, 4, .5)

	var cidentity affine.ColorM = affine.ColorMIdentity{}
	var cinvalid affine.ColorM = affine.ColorMIdentity{}
	cinvalid = affine.ColorMSetElement(cinvalid, 0, 0, 0)
	cinvalid = affine.ColorMSetElement(cinvalid, 1, 1, 0)
	cinvalid = affine.ColorMSetElement(cinvalid, 2, 2, 0)
	cinvalid = affine.ColorMSetElement(cinvalid, 3, 3, 0)

	cases := []struct {
		In  affine.ColorM
		Out bool
	}{
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
			t.Errorf("%s.IsInvertible(): got: %t, want: %t", c.In, got, want)
		}
	}
}

func arrayToColorM(es [4][5]float32) affine.ColorM {
	var a affine.ColorM = affine.ColorMIdentity{}
	for j := 0; j < 5; j++ {
		for i := 0; i < 4; i++ {
			a = affine.ColorMSetElement(a, i, j, es[i][j])
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

func equalWithDelta(a, b affine.ColorM, delta float32) bool {
	for j := 0; j < 5; j++ {
		for i := 0; i < 4; i++ {
			ea := a.At(i, j)
			eb := b.At(i, j)
			if abs(ea-eb) > delta {
				return false
			}
		}
	}
	return true
}

func TestColorMInvert(t *testing.T) {
	cases := []struct {
		In  affine.ColorM
		Out affine.ColorM
	}{
		{
			In:  affine.ColorMIdentity{},
			Out: affine.ColorMIdentity{},
		},
		{
			In: arrayToColorM([4][5]float32{
				{1, 0, 0, 0, 0},
				{0, 1, 0, 0, 0},
				{0, 0, 1, 0, 0},
				{0, 0, 0, 1, 0},
			}),
			Out: arrayToColorM([4][5]float32{
				{1, 0, 0, 0, 0},
				{0, 1, 0, 0, 0},
				{0, 0, 1, 0, 0},
				{0, 0, 0, 1, 0},
			}),
		},
		{
			In: affine.ColorMIdentity{}.Scale(1, 2, 4, 8),
			Out: arrayToColorM([4][5]float32{
				{1, 0, 0, 0, 0},
				{0, 0.5, 0, 0, 0},
				{0, 0, 0.25, 0, 0},
				{0, 0, 0, 0.125, 0},
			}),
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
	var m affine.ColorM = affine.ColorMIdentity{}
	for m.IsIdentity() || !m.IsInvertible() {
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

func TestColorMConcat(t *testing.T) {
	cases := []struct {
		In0 affine.ColorM
		In1 affine.ColorM
		Out affine.ColorM
	}{
		{
			affine.ColorMIdentity{},
			affine.ColorMIdentity{},
			affine.ColorMIdentity{},
		},
		{
			affine.ColorMIdentity{}.Scale(1, 2, 3, 4),
			affine.ColorMIdentity{}.Scale(5, 6, 7, 8),
			affine.ColorMIdentity{}.Scale(5, 12, 21, 32),
		},
		{
			affine.ColorMIdentity{}.Scale(5, 6, 7, 8),
			affine.ColorMIdentity{}.Scale(1, 2, 3, 4),
			affine.ColorMIdentity{}.Scale(5, 12, 21, 32),
		},
		{
			arrayToColorM([4][5]float32{
				{1, 2, 3, 4, 5},
				{5, 1, 2, 3, 4},
				{4, 5, 1, 2, 3},
				{3, 4, 5, 1, 2},
			}),
			affine.ColorMIdentity{}.Scale(1, 2, 3, 4),
			arrayToColorM([4][5]float32{
				{1, 2, 3, 4, 5},
				{10, 2, 4, 6, 8},
				{12, 15, 3, 6, 9},
				{12, 16, 20, 4, 8},
			}),
		},
		{
			affine.ColorMIdentity{}.Scale(1, 2, 3, 4),
			arrayToColorM([4][5]float32{
				{1, 2, 3, 4, 5},
				{5, 1, 2, 3, 4},
				{4, 5, 1, 2, 3},
				{3, 4, 5, 1, 2},
			}),
			arrayToColorM([4][5]float32{
				{1, 4, 9, 16, 5},
				{5, 2, 6, 12, 4},
				{4, 10, 3, 8, 3},
				{3, 8, 15, 4, 2},
			}),
		},
	}
	for _, c := range cases {
		got := c.In0.Concat(c.In1)
		want := c.Out
		if !equalWithDelta(got, want, 1e-6) {
			t.Errorf("%s.Concat(%s): got: %v, want: %v", c.In0, c.In1, got, want)
		}
	}
}

// Issue #2170
func TestColorMChangeHSVScale(t *testing.T) {
	var c affine.ColorM = affine.ColorMIdentity{}
	c = affine.ChangeHSV(c, 0, 1, 0.12345)
	if got, want := c.ScaleOnly(), true; got != want {
		t.Errorf("got: %t, want: %t", got, want)
	}
}
