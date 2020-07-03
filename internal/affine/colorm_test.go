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

func TestColorMIsInvertible(t *testing.T) {
	m := &ColorM{}
	m = m.SetElement(1, 0, .5)
	m = m.SetElement(1, 1, .5)
	m = m.SetElement(1, 2, .5)
	m = m.SetElement(1, 3, .5)
	m = m.SetElement(1, 4, .5)
	cidentity := &ColorM{}
	//
	cinvalid := &ColorM{}
	cinvalid = cinvalid.SetElement(0, 0, 0)
	cinvalid = cinvalid.SetElement(1, 1, 0)
	cinvalid = cinvalid.SetElement(2, 2, 0)
	cinvalid = cinvalid.SetElement(3, 3, 0)
	//
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

func TestColorMInvert(t *testing.T) {
	var m [5][4]float32

	m = [5][4]float32{
		{1, 8, -9, 7},
		{0, 1, 0, 4},
		{0, 0, 1, 2},
		{0, 0, 0, 1},
		{0, 0, 0, 0},
	}
	a := &ColorM{}
	for j := 0; j < 5; j++ {
		for i := 0; i < 4; i++ {
			a = a.SetElement(i, j, m[j][i])
		}
	}

	m = [5][4]float32{
		{1, -8, 9, 7},
		{0, 1, 0, -4},
		{0, 0, 1, -2},
		{0, 0, 0, 1},
		{0, 0, 0, 0},
	}
	ia := &ColorM{}
	for j := 0; j < 5; j++ {
		for i := 0; i < 4; i++ {
			ia = ia.SetElement(i, j, m[j][i])
		}
	}

	ia2 := a.Invert()
	if !ia.Equals(ia2) {
		t.Fail()
	}
}

func BenchmarkColorMInvert(b *testing.B) {
	b.StopTimer()
	m := &ColorM{}
	m = m.SetElement(1, 0, rand.Float32())
	m = m.SetElement(2, 0, rand.Float32())
	m = m.SetElement(3, 0, rand.Float32())
	m = m.SetElement(0, 1, rand.Float32())
	m = m.SetElement(2, 1, rand.Float32())
	m = m.SetElement(3, 1, rand.Float32())
	m = m.SetElement(0, 2, rand.Float32())
	m = m.SetElement(1, 2, rand.Float32())
	m = m.SetElement(3, 2, rand.Float32())
	m = m.SetElement(0, 3, rand.Float32())
	m = m.SetElement(1, 3, rand.Float32())
	m = m.SetElement(2, 3, rand.Float32())
	m = m.SetElement(0, 4, rand.Float32()*10)
	m = m.SetElement(1, 4, rand.Float32()*10)
	m = m.SetElement(2, 4, rand.Float32()*10)
	m = m.SetElement(3, 4, rand.Float32()*10)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if m.IsInvertible() {
			m = m.Invert()
		}
	}
}
