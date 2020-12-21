// Copyright 2016 The Ebiten Authors
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

package twenty48_test

import (
	. "github.com/hajimehoshi/ebiten/v2/examples/2048/2048"

	"fmt"
	"testing"
)

func cellsToTiles(cells []int, size int) map[*Tile]struct{} {
	tiles := map[*Tile]struct{}{}
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			c := cells[i+j*size]
			if c == 0 {
				continue
			}
			t := NewTile(c, i, j)
			tiles[t] = struct{}{}
		}
	}
	return tiles
}

func tilesToCells(tiles map[*Tile]struct{}, size int) ([]int, []int) {
	cells := make([]int, size*size)
	nextCells := make([]int, size*size)
	for t := range tiles {
		x, y := t.Pos()
		cells[x+y*size] = t.Value()
		if t.IsMoving() {
			if t.NextValue() == 0 {
				continue
			}
			nx, ny := t.NextPos()
			nextCells[nx+ny*size] = t.NextValue()
		} else {
			nextCells[x+y*size] = t.Value()
		}
	}
	return cells, nextCells
}

func TestMoveTiles(t *testing.T) {
	const size = 4
	testCases := []struct {
		Dir   Dir
		Input []int
		Want  []int
	}{
		{
			Dir: DirUp,
			Input: []int{
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
			},
			Want: []int{
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
			},
		},
		{
			Dir: DirRight,
			Input: []int{
				2, 0, 0, 0,
				0, 2, 0, 0,
				0, 0, 2, 0,
				0, 0, 0, 2,
			},
			Want: []int{
				0, 0, 0, 2,
				0, 0, 0, 2,
				0, 0, 0, 2,
				0, 0, 0, 2,
			},
		},
		{
			Dir: DirUp,
			Input: []int{
				2, 0, 0, 0,
				0, 2, 0, 0,
				0, 0, 2, 0,
				0, 0, 0, 2,
			},
			Want: []int{
				2, 2, 2, 2,
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
			},
		},
		{
			Dir: DirLeft,
			Input: []int{
				0, 2, 2, 2,
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
			},
			Want: []int{
				4, 2, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
			},
		},
		{
			Dir: DirRight,
			Input: []int{
				0, 0, 0, 2,
				0, 0, 2, 2,
				0, 2, 2, 2,
				2, 2, 2, 2,
			},
			Want: []int{
				0, 0, 0, 2,
				0, 0, 0, 4,
				0, 0, 2, 4,
				0, 0, 4, 4,
			},
		},
		{
			Dir: DirLeft,
			Input: []int{
				0, 0, 0, 2,
				0, 0, 2, 2,
				0, 2, 2, 2,
				2, 2, 2, 2,
			},
			Want: []int{
				2, 0, 0, 0,
				4, 0, 0, 0,
				4, 2, 0, 0,
				4, 4, 0, 0,
			},
		},
		{
			Dir: DirRight,
			Input: []int{
				4, 8, 8, 4,
				8, 8, 4, 4,
				4, 4, 8, 8,
				8, 4, 4, 8,
			},
			Want: []int{
				0, 4, 16, 4,
				0, 0, 16, 8,
				0, 0, 8, 16,
				0, 8, 8, 8,
			},
		},
		{
			Dir: DirDown,
			Input: []int{
				4, 8, 8, 4,
				8, 8, 4, 4,
				4, 4, 8, 8,
				8, 4, 4, 8,
			},
			Want: []int{
				4, 0, 8, 0,
				8, 0, 4, 0,
				4, 16, 8, 8,
				8, 8, 4, 16,
			},
		},
		{
			Dir: DirLeft,
			Input: []int{
				4, 8, 8, 4,
				8, 8, 4, 4,
				4, 4, 8, 8,
				8, 4, 4, 8,
			},
			Want: []int{
				4, 16, 4, 0,
				16, 8, 0, 0,
				8, 16, 0, 0,
				8, 8, 8, 0,
			},
		},
		{
			Dir: DirUp,
			Input: []int{
				4, 8, 8, 4,
				8, 8, 4, 4,
				4, 4, 8, 8,
				8, 4, 4, 8,
			},
			Want: []int{
				4, 16, 8, 8,
				8, 8, 4, 16,
				4, 0, 8, 0,
				8, 0, 4, 0,
			},
		},
		{
			Dir: DirUp,
			Input: []int{
				2, 4, 2, 4,
				4, 2, 4, 2,
				2, 4, 2, 4,
				4, 2, 4, 2,
			},
			Want: []int{
				2, 4, 2, 4,
				4, 2, 4, 2,
				2, 4, 2, 4,
				4, 2, 4, 2,
			},
		},
	}
	for _, test := range testCases {
		want, _ := tilesToCells(cellsToTiles(test.Want, size), size)
		tiles := cellsToTiles(test.Input, size)
		moved := MoveTiles(tiles, size, test.Dir)
		input, got := tilesToCells(tiles, size)
		if !moved {
			got = input
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("dir: %s, input: %v, got %v; want %v", test.Dir.String(), test.Input, got, want)
		}
	}
}
