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

func BenchmarkGeoM(b *testing.B) {
	var m GeoM
	for i := 0; i < b.N; i++ {
		m = GeoM{}
		m.Translate(10, 20)
		m.Scale(2, 3)
		m.Rotate(math.Pi / 2)
	}
}
