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
	"fmt"
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

func geoMToString(g GeoM) string {
	a := g.Element(0, 0)
	b := g.Element(0, 1)
	c := g.Element(1, 0)
	d := g.Element(1, 1)
	tx := g.Element(0, 2)
	ty := g.Element(1, 2)
	return fmt.Sprintf("{a: %f, b: %f, c: %f, d: %f, tx: %f, ty: %f}", a, b, c, d, tx, ty)
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
		GeoM GeoM
		InX  float64
		InY  float64
		OutX float64
		OutY float64
	}{
		{
			GeoM: GeoM{},
			InX:  3.14159,
			InY:  2.81828,
			OutX: 3.14159,
			OutY: 2.81828,
		},
		{
			GeoM: trans,
			InX:  3.14159,
			InY:  2.81828,
			OutX: 4.14159,
			OutY: 4.81828,
		},
		{
			GeoM: scale,
			InX:  3.14159,
			InY:  2.81828,
			OutX: 4.71239,
			OutY: 7.04570,
		},
		{
			GeoM: cpx,
			InX:  3.14159,
			InY:  2.81828,
			OutX: -6.71239,
			OutY: -10.04570,
		},
	}

	const delta = 0.00001

	for _, c := range cases {
		rx, ry := c.GeoM.Apply(c.InX, c.InY)
		if math.Abs(rx-c.OutX) > delta || math.Abs(ry-c.OutY) > delta {
			t.Errorf("%s.Apply(%f, %f) = (%f, %f), want (%f, %f)", geoMToString(c.GeoM), c.InX, c.InY, rx, ry, c.OutX, c.OutY)
		}
	}
}

func TestGeoMIsInvert(t *testing.T) {
	zero := GeoM{}
	zero.Scale(0, 0)

	trans := GeoM{}
	trans.Translate(1, 2)

	scale := GeoM{}
	scale.Scale(1.5, 2.5)

	cpx := GeoM{}
	cpx.Rotate(math.Pi)
	cpx.Scale(1.5, 2.5)
	cpx.Translate(-2, -3)

	cpx2 := GeoM{}
	cpx2.Scale(2, 3)
	cpx2.Rotate(0.234)
	cpx2.Translate(100, 100)

	cases := []struct {
		GeoM       GeoM
		Invertible bool
	}{
		{
			GeoM:       zero,
			Invertible: false,
		},
		{
			GeoM:       GeoM{},
			Invertible: true,
		},
		{
			GeoM:       trans,
			Invertible: true,
		},
		{
			GeoM:       scale,
			Invertible: true,
		},
		{
			GeoM:       cpx,
			Invertible: true,
		},
		{
			GeoM:       cpx2,
			Invertible: true,
		},
	}

	pts := []struct {
		X float64
		Y float64
	}{
		{
			X: 0,
			Y: 0,
		},
		{
			X: 1,
			Y: 1,
		},
		{
			X: 3.14159,
			Y: 2.81828,
		},
		{
			X: -1000,
			Y: 1000,
		},
	}

	const delta = 0.00001

	for _, c := range cases {
		if c.GeoM.IsInvertible() != c.Invertible {
			t.Errorf("%s.IsInvertible(): got: %t, want: %t", geoMToString(c.GeoM), c.GeoM.IsInvertible(), c.Invertible)
		}
		if !c.GeoM.IsInvertible() {
			continue
		}
		invGeoM := c.GeoM
		invGeoM.Invert()
		for _, p := range pts {
			x, y := p.X, p.Y
			gotX, gotY := invGeoM.Apply(c.GeoM.Apply(x, y))
			if math.Abs(gotX-x) > delta || math.Abs(gotY-y) > delta {
				t.Errorf("%s.Apply(%s.Apply(%f, %f)): got: (%f, %f), want: (%f, %f)", geoMToString(invGeoM), geoMToString(c.GeoM), x, y, gotX, gotY, x, y)
			}
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
