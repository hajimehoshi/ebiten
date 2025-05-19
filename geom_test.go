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

	"github.com/hajimehoshi/ebiten/v2"
)

func TestGeoMInit(t *testing.T) {
	var m ebiten.GeoM
	for i := range ebiten.GeoMDim - 1 {
		for j := range ebiten.GeoMDim {
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
	m := ebiten.GeoM{}
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
	matrix1 := ebiten.GeoM{}
	matrix1.Scale(2, 2)
	matrix2 := ebiten.GeoM{}
	matrix2.Translate(1, 1)

	matrix3 := matrix1
	matrix3.Concat(matrix2)
	expected := [][]float64{
		{2, 0, 1},
		{0, 2, 1},
	}
	for i := range 2 {
		for j := range 3 {
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
	for i := range 2 {
		for j := range 3 {
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
	m := ebiten.GeoM{}
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
	for i := range 2 {
		for j := range 3 {
			got := m.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("m.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}
}

func geoMToString(g ebiten.GeoM) string {
	a := g.Element(0, 0)
	b := g.Element(0, 1)
	c := g.Element(1, 0)
	d := g.Element(1, 1)
	tx := g.Element(0, 2)
	ty := g.Element(1, 2)
	return fmt.Sprintf("{a: %f, b: %f, c: %f, d: %f, tx: %f, ty: %f}", a, b, c, d, tx, ty)
}

func TestGeoMApply(t *testing.T) {
	trans := ebiten.GeoM{}
	trans.Translate(1, 2)

	scale := ebiten.GeoM{}
	scale.Scale(1.5, 2.5)

	cpx := ebiten.GeoM{}
	cpx.Rotate(math.Pi)
	cpx.Scale(1.5, 2.5)
	cpx.Translate(-2, -3)

	cases := []struct {
		GeoM ebiten.GeoM
		InX  float64
		InY  float64
		OutX float64
		OutY float64
	}{
		{
			GeoM: ebiten.GeoM{},
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
	zero := ebiten.GeoM{}
	zero.Scale(0, 0)

	trans := ebiten.GeoM{}
	trans.Translate(1, 2)

	scale := ebiten.GeoM{}
	scale.Scale(1.5, 2.5)

	cpx := ebiten.GeoM{}
	cpx.Rotate(math.Pi)
	cpx.Scale(1.5, 2.5)
	cpx.Translate(-2, -3)

	cpx2 := ebiten.GeoM{}
	cpx2.Scale(2, 3)
	cpx2.Rotate(0.234)
	cpx2.Translate(100, 100)

	skew := ebiten.GeoM{}
	skew.Skew(1, 1)

	cases := []struct {
		GeoM       ebiten.GeoM
		Invertible bool
	}{
		{
			GeoM:       zero,
			Invertible: false,
		},
		{
			GeoM:       ebiten.GeoM{},
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
		{
			GeoM:       skew,
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

	const delta = 0.001

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

func newGeoM(a, b, c, d, tx, ty float64) ebiten.GeoM {
	outp := ebiten.GeoM{}
	outp.SetElement(0, 0, a)
	outp.SetElement(0, 1, b)
	outp.SetElement(0, 2, tx)
	outp.SetElement(1, 0, c)
	outp.SetElement(1, 1, d)
	outp.SetElement(1, 2, ty)
	return outp
}

func TestGeomSkew(t *testing.T) {
	testSkew := func(skewX, skewY float64, input, expected ebiten.GeoM) {
		input.Skew(skewX, skewY)
		for i := range 2 {
			for j := range 3 {
				got := input.Element(i, j)
				want := expected.Element(i, j)
				if want != got {
					t.Errorf("Skew(%f, %f): got %s, want: %s", skewX, skewY, input.String(), expected.String())
					return
				}
			}
		}
	}
	// skewX = 0.25
	expectedX := newGeoM(1, math.Tan(0.25), math.Tan(0), 1, 0, 0)
	testSkew(0.25, 0, ebiten.GeoM{}, expectedX)

	// skewY = 0.25
	expectedY := newGeoM(1, math.Tan(0), math.Tan(0.5), 1, 0, 0)
	testSkew(0, 0.5, ebiten.GeoM{}, expectedY)

	// skewX, skewY = 0.3, 0.8
	expectedXY := newGeoM(1, math.Tan(0.3), math.Tan(0.8), 1, 0, 0)
	testSkew(0.3, 0.8, ebiten.GeoM{}, expectedXY)

	// skewX, skewY = 0.4, -1.8 ; b, c = 2, 3
	expectedOffDiag := newGeoM(1+3*math.Tan(0.4), 2+math.Tan(0.4), 3+math.Tan(-1.8), 1+2*math.Tan(-1.8), 0, 0)
	inputOffDiag := newGeoM(1, 2, 3, 1, 0, 0)
	testSkew(0.4, -1.8, inputOffDiag, expectedOffDiag)

	// skewX, skewY = -1.5, 1.5 ; tx, ty = 5, 6
	expectedTrn := newGeoM(1, math.Tan(-1.5), math.Tan(1.5), 1, 5+math.Tan(-1.5)*6, 6+5*math.Tan(1.5))
	inputTrn := newGeoM(1, 0, 0, 1, 5, 6)
	testSkew(-1.5, 1.5, inputTrn, expectedTrn)
}

func TestGeoMEquals(t *testing.T) {
	tests := []struct {
		a    ebiten.GeoM
		b    ebiten.GeoM
		want bool
	}{
		{
			a:    ebiten.GeoM{},
			b:    ebiten.GeoM{},
			want: true,
		},
		{
			a:    newGeoM(3, 1, 4, 1, 5, 9),
			b:    newGeoM(3, 1, 4, 1, 5, 9),
			want: true,
		},
		{
			a:    newGeoM(3, 1, 4, 1, 5, 9),
			b:    newGeoM(3, 1, 4, 1, 5, 10),
			want: false,
		},
	}
	for _, test := range tests {
		got := (test.a == test.b)
		want := test.want
		if got != want {
			t.Errorf("%#v == %#v: got %t, want: %t", test.a, test.b, got, want)
		}
	}
}

func BenchmarkGeoM(b *testing.B) {
	var m ebiten.GeoM
	for range b.N {
		m = ebiten.GeoM{}
		m.Translate(10, 20)
		m.Scale(2, 3)
		m.Rotate(math.Pi / 2)
	}
}
