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

package triangulate_test

import (
	"math"
	"reflect"
	"testing"

	. "github.com/hajimehoshi/ebiten/v2/vector/internal/triangulate"
)

func TestIsInTriangle(t *testing.T) {
	tests := []struct {
		Tri []Point
		Pt  Point
		Out bool
	}{
		{
			Tri: []Point{
				{0, 0},
				{0, 10},
				{10, 10},
			},
			Pt:  Point{1, 9},
			Out: true,
		},
		{
			Tri: []Point{
				{0, 0},
				{0, 10},
				{10, 10},
			},
			Pt:  Point{8, 9},
			Out: true,
		},
		{
			Tri: []Point{
				{0, 0},
				{0, 10},
				{10, 10},
			},
			Pt:  Point{10, 9},
			Out: false,
		},
		{
			Tri: []Point{
				{3, 5},
				{2, 7},
				{7, 7},
			},
			Pt:  Point{3, 6},
			Out: true,
		},
		{
			Tri: []Point{
				{3, 5},
				{2, 7},
				{7, 7},
			},
			Pt:  Point{7, 6},
			Out: false,
		},
	}
	for _, tc := range tests {
		got := InTriangle(tc.Pt, tc.Tri[0], tc.Tri[1], tc.Tri[2])
		want := tc.Out
		if got != want {
			t.Errorf("InTriangle(%v, %v, %v, %v): got: %t, want: %t", tc.Pt, tc.Tri[0], tc.Tri[1], tc.Tri[2], got, want)
		}
	}
}

func TestTriangulate(t *testing.T) {
	tests := []struct {
		In  []Point
		Out []uint16
	}{
		{
			In:  []Point{},
			Out: nil,
		},
		{
			In: []Point{
				{0, 0},
			},
			Out: nil,
		},
		{
			In: []Point{
				{0, 0},
				{0, 1},
			},
			Out: nil,
		},
		{
			In: []Point{
				{0, 0},
				{0, 0},
				{1, 1},
			},
			Out: nil,
		},
		{
			In: []Point{
				{0, 0},
				{0.5, 0.5},
				{1, 1},
			},
			Out: nil,
		},
		{
			In: []Point{
				{0, 0},
				{0.5, 0.5},
				{1.5, 1.5},
				{1, 1},
			},
			Out: nil,
		},
		{
			In: []Point{
				{0, 0},
				{0, 1},
				{1, 1},
			},
			Out: []uint16{2, 0, 1},
		},
		{
			In: []Point{
				{0, 0},
				{1, 1},
				{0, 1},
			},
			Out: []uint16{2, 0, 1},
		},
		{
			In: []Point{
				{0, 0},
				{1, 1},
				{0, 1},
				{0, 0.5},
			},
			Out: []uint16{2, 0, 1},
		},
		{
			In: []Point{
				{0, 0},
				{0, 1},
				{1, 1},
				{1, 0},
			},
			Out: []uint16{3, 0, 1, 3, 1, 2},
		},
		{
			In: []Point{
				{2, 2},
				{2, 7},
				{7, 7},
				{7, 6},
				{3, 6},
				{3, 5},
			},
			Out: []uint16{5, 0, 1, 1, 2, 3, 1, 3, 4, 5, 1, 4},
		},
		{
			In: []Point{
				{2, 2},
				{2, 7},
				{7, 7},
				{7, 6},
				{3, 6},
				{3, 5},
				{7, 5},
				{7, 4},
				{3, 4},
				{3, 3},
			},
			Out: []uint16{9, 0, 1, 1, 2, 3, 1, 3, 4, 9, 1, 4, 8, 5, 6, 8, 6, 7},
		},
		{
			In: []Point{
				{0, 0},
				{0, 5},
				{2, 5},
				{3, 3},
				{2, 2},
				{3, 1},
				{2, 0},
			},
			Out: []uint16{6, 0, 1, 6, 1, 2, 6, 2, 3, 4, 5, 6, 6, 3, 4},
		},
		{
			In: []Point{
				{0, 0},
				{0, 5},
				{2, 5},
				{2, 5},
				{3, 3},
				{2, 2},
				{2, 2},
				{3, 1},
				{2, 0},
			},
			Out: []uint16{6, 7, 8, 6, 8, 0, 4, 0, 1, 4, 1, 3},
		},
		{
			In: []Point{
				{0, 0},
				{0, 1},
				{1, 1},
				{1, 0},
				{2, 0},
			},
			Out: []uint16{3, 0, 1, 3, 1, 2},
		},
		{
			In: []Point{
				{2, 0},
				{0, 0},
				{1, 0},
				{1, 1},
				{2, 1},
			},
			Out: []uint16{4, 0, 2, 4, 2, 3},
		},
		{
			In: []Point{
				{2, 0},
				{2, 1},
				{1, 1},
				{1, 0},
				{0, 0},
			},
			Out: []uint16{3, 0, 1, 3, 1, 2},
		},
		{
			// Butterfly
			In: []Point{
				{0, 2},
				{1, 1},
				{2, 2},
				{2, 0},
				{1, 1},
				{0, 0},
			},
			Out: []uint16{3, 1, 2, 0, 4, 5},
		},
		{
			In: []Point{
				{0, 6},
				{0, 9},
				{6, 6},
				{6, 3},
				{9, 3},
				{8, 0},
				{6, 0},
				{6, 3},
			},
			Out: []uint16{6, 3, 4, 6, 4, 5, 2, 7, 0, 2, 0, 1},
		},
		{
			In: []Point{
				{0, 4},
				{0, 6},
				{2, 6},
				{2, 5},
				{3, 5},
				{3, 4},
				{4, 4},
				{4, 2},
				{6, 2},
				{6, 1},
				{5, 1},
				{5, 0},
				{4, 0},
				{4, 2},
				{2, 2},
				{2, 3},
				{1, 3},
				{1, 4},
			},
			Out: []uint16{7, 8, 9, 7, 9, 10, 12, 7, 10, 12, 10, 11, 6, 13, 14, 6, 14, 15, 6, 15, 16, 6, 16, 17, 5, 0, 1, 1, 2, 3, 5, 1, 3, 5, 3, 4},
		},
	}
	for _, tc := range tests {
		got := Triangulate(tc.In)
		want := tc.Out
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Triangulate(%v): got: %v, want: %v", tc.In, got, want)
		}
	}
}

var benchmarkPath []Point

func init() {
	const (
		w = 640
		h = 480
	)
	var p []Point
	for i := 0; i < w; i++ {
		x := float32(i)
		y := h/2 + 80*float32(math.Sin(2*math.Pi*float64(i)/40))
		p = append(p, Point{X: x, Y: y})
	}
	p = append(p, Point{w, h}, Point{0, h})
	benchmarkPath = p
}

func BenchmarkTriangulate(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Triangulate(benchmarkPath)
	}
}
