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

package ebiten_test

import (
	"math"
	"testing"

	. "github.com/hajimehoshi/ebiten"
)

func TestGeoMInit(t *testing.T) {
	var m GeoM
	for i := 0; i < GeoMDim-1; i++ {
		for j := 0; j < GeoMDim; j++ {
			got := m.Element(i, j)
			want := 0.0
			if i == j {
				want = 1
			}
			if want != got {
				t.Errorf("m.Element(%d, %d) = %f, want %f", i, j, got, want)
			}
		}
	}
}

func TestGeoMAssign(t *testing.T) {
	m := GeoM{}
	m.SetElement(0, 0, 1)
	m2 := m
	m.SetElement(0, 0, 0)
	got := m2.Element(0, 0)
	want := 1.0
	if want != got {
		t.Errorf("m2.Element(%d, %d) = %f, want %f", 0, 0, got, want)
	}
}

func TestGeoMConcat(t *testing.T) {
	matrix1 := GeoM{}
	matrix1.Scale(2, 2)
	matrix2 := GeoM{}
	matrix2.Translate(1, 1)

	matrix3 := matrix1
	matrix3.Concat(matrix2)
	expected := [][]float64{
		{2, 0, 1},
		{0, 2, 1},
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			got := matrix3.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix3.Element(%d, %d) = %f,"+
					" want %f",
					i, j, got, want)
			}
		}
	}

	matrix4 := matrix2
	matrix4.Concat(matrix1)
	expected = [][]float64{
		{2, 0, 2},
		{0, 2, 2},
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			got := matrix4.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix4.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}
}

func TestGeoMConcatSelf(t *testing.T) {
	m := GeoM{}
	m.SetElement(0, 0, 1)
	m.SetElement(0, 1, 2)
	m.SetElement(0, 2, 3)
	m.SetElement(1, 0, 4)
	m.SetElement(1, 1, 5)
	m.SetElement(1, 2, 6)
	m.Concat(m)
	expected := [][]float64{
		{9, 12, 18},
		{24, 33, 48},
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			got := m.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("m.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}
}

func TestGeoMApply(t *testing.T) {
	trans := GeoM{}
	trans.Translate(1, 2)

	scale := GeoM{}
	scale.Scale(1.5, 2.5)

	cpx := GeoM{}
	cpx.Rotate(math.Pi)
	cpx.Scale(1.5, 2.5)
	cpx.Translate(-2, -3)

	cases := []struct {
		GeoM  GeoM
		InX   float64
		InY   float64
		OutX  float64
		OutY  float64
		Delta float64
	}{
		{
			GeoM:  GeoM{},
			InX:   3.14159,
			InY:   2.81828,
			OutX:  3.14159,
			OutY:  2.81828,
			Delta: 0.00001,
		},
		{
			GeoM:  trans,
			InX:   3.14159,
			InY:   2.81828,
			OutX:  4.14159,
			OutY:  4.81828,
			Delta: 0.00001,
		},
		{
			GeoM:  scale,
			InX:   3.14159,
			InY:   2.81828,
			OutX:  4.71239,
			OutY:  7.04570,
			Delta: 0.00001,
		},
		{
			GeoM:  cpx,
			InX:   3.14159,
			InY:   2.81828,
			OutX:  -6.71239,
			OutY:  -10.04570,
			Delta: 0.00001,
		},
	}
	for _, c := range cases {
		rx, ry := c.GeoM.Apply(c.InX, c.InY)
		if math.Abs(rx-c.OutX) > c.Delta || math.Abs(ry-c.OutY) > c.Delta {
			t.Errorf("%v.Apply(%v, %v) = (%v, %v), want (%v, %v)", c.GeoM, c.InX, c.InY, rx, ry, c.OutX, c.OutY)
		}
	}
}

func BenchmarkGeoM(b *testing.B) {
	var m GeoM
	for i := 0; i < b.N; i++ {
		m = GeoM{}
		m.Translate(10, 20)
		m.Scale(2, 3)
		m.Rotate(math.Pi / 2)
	}
}
