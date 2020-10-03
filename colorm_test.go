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
	"image/color"
	"math"
	"testing"

	. "github.com/hajimehoshi/ebiten/v2"
)

func TestColorMInit(t *testing.T) {
	var m ColorM
	for i := 0; i < ColorMDim-1; i++ {
		for j := 0; j < ColorMDim; j++ {
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

	m.SetElement(0, 0, 1)
	for i := 0; i < ColorMDim-1; i++ {
		for j := 0; j < ColorMDim; j++ {
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

func TestColorMAssign(t *testing.T) {
	m := ColorM{}
	m.SetElement(0, 0, 1)
	m2 := m
	m.SetElement(0, 0, 0)
	got := m2.Element(0, 0)
	want := 1.0
	if want != got {
		t.Errorf("m2.Element(%d, %d) = %f, want %f", 0, 0, got, want)
	}
}

func TestColorMTranslate(t *testing.T) {
	expected := [4][5]float64{
		{1, 0, 0, 0, 0.5},
		{0, 1, 0, 0, 1.5},
		{0, 0, 1, 0, 2.5},
		{0, 0, 0, 1, 3.5},
	}
	m := ColorM{}
	m.Translate(0.5, 1.5, 2.5, 3.5)
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			got := m.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("m.Element(%d, %d) = %f, want %f", i, j, got, want)
			}
		}
	}
}

func TestColorMScale(t *testing.T) {
	expected := [4][5]float64{
		{0.5, 0, 0, 0, 0},
		{0, 1.5, 0, 0, 0},
		{0, 0, 2.5, 0, 0},
		{0, 0, 0, 3.5, 0},
	}
	m := ColorM{}
	m.Scale(0.5, 1.5, 2.5, 3.5)
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			got := m.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("m.Element(%d, %d) = %f, want %f", i, j, got, want)
			}
		}
	}
}

func TestColorMTranslateAndScale(t *testing.T) {
	expected := [4][5]float64{
		{1, 0, 0, 0, 0},
		{0, 1, 0, 0, 0},
		{0, 0, 1, 0, 0},
		{0, 0, 0, 0.5, 0.5},
	}
	m := ColorM{}
	m.Translate(0, 0, 0, 1)
	m.Scale(1, 1, 1, 0.5)
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			got := m.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("m.Element(%d, %d) = %f, want %f", i, j, got, want)
			}
		}
	}
}

func TestColorMMonochrome(t *testing.T) {
	expected := [4][5]float64{
		{0.2990, 0.5870, 0.1140, 0, 0},
		{0.2990, 0.5870, 0.1140, 0, 0},
		{0.2990, 0.5870, 0.1140, 0, 0},
		{0, 0, 0, 1, 0},
	}
	m := ColorM{}
	m.ChangeHSV(0, 0, 1)
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			got := m.Element(i, j)
			want := expected[i][j]
			if math.Abs(want-got) > 0.0001 {
				t.Errorf("m.Element(%d, %d) = %f, want %f", i, j, got, want)
			}
		}
	}
}

func TestColorMConcatSelf(t *testing.T) {
	expected := [4][5]float64{
		{30, 40, 30, 25, 30},
		{40, 54, 43, 37, 37},
		{30, 43, 51, 39, 34},
		{25, 37, 39, 46, 36},
	}
	m := ColorM{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			m.SetElement(i, j, float64((i+j)%5+1))
		}
	}
	m.Concat(m)
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			got := m.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("m.Element(%d, %d) = %f, want %f", i, j, got, want)
			}
		}
	}
}

func absDiffU32(x, y uint32) uint32 {
	if x < y {
		return y - x
	}
	return x - y
}

func TestColorMApply(t *testing.T) {
	mono := ColorM{}
	mono.ChangeHSV(0, 0, 1)

	shiny := ColorM{}
	shiny.Translate(1, 1, 1, 0)

	shift := ColorM{}
	shift.Translate(0.5, 0.5, 0.5, 0.5)

	cases := []struct {
		ColorM ColorM
		In     color.Color
		Out    color.Color
		Delta  uint32
	}{
		{
			ColorM: ColorM{},
			In:     color.RGBA{1, 2, 3, 4},
			Out:    color.RGBA{1, 2, 3, 4},
			Delta:  0x101,
		},
		{
			ColorM: mono,
			In:     color.NRGBA{0xff, 0xff, 0xff, 0},
			Out:    color.Transparent,
			Delta:  0x101,
		},
		{
			ColorM: mono,
			In:     color.RGBA{0xff, 0, 0, 0xff},
			Out:    color.RGBA{0x4c, 0x4c, 0x4c, 0xff},
			Delta:  0x101,
		},
		{
			ColorM: shiny,
			In:     color.RGBA{0x80, 0x90, 0xa0, 0xb0},
			Out:    color.RGBA{0xb0, 0xb0, 0xb0, 0xb0},
			Delta:  1,
		},
		{
			ColorM: shift,
			In:     color.RGBA{0x00, 0x00, 0x00, 0x00},
			Out:    color.RGBA{0x40, 0x40, 0x40, 0x80},
			Delta:  0x101,
		},
	}
	for _, c := range cases {
		out := c.ColorM.Apply(c.In)
		r0, g0, b0, a0 := out.RGBA()
		r1, g1, b1, a1 := c.Out.RGBA()
		if absDiffU32(r0, r1) > c.Delta || absDiffU32(g0, g1) > c.Delta ||
			absDiffU32(b0, b1) > c.Delta || absDiffU32(a0, a1) > c.Delta {
			t.Errorf("%v.Apply(%v) = {%d, %d, %d, %d}, want {%d, %d, %d, %d}", c.ColorM, c.In, r0, g0, b0, a0, r1, g1, b1, a1)
		}
	}
}
