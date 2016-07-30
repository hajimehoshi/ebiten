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

package twenty48

import (
	"errors"
	"math/rand"
	"sort"
)

type Tile struct {
	value int
	x     int
	y     int
}

func NewTile(value int, x, y int) *Tile {
	return &Tile{
		value: value,
		x:     x,
		y:     y,
	}
}

func (t *Tile) Value() int {
	return t.value
}

func (t *Tile) Pos() (int, int) {
	return t.x, t.y
}

func tileAt(tiles map[*Tile]struct{}, x, y int) *Tile {
	for t := range tiles {
		if t.x == x && t.y == y {
			return t
		}
	}
	return nil
}

func MoveTiles(tiles map[*Tile]struct{}, size int, dir Dir) (map[*Tile]struct{}, bool) {
	vx, vy := dir.Vector()
	tx := []int{}
	ty := []int{}
	for i := 0; i < size; i++ {
		tx = append(tx, i)
		ty = append(ty, i)
	}
	if vx > 0 {
		sort.Sort(sort.Reverse(sort.IntSlice(tx)))
	}
	if vy > 0 {
		sort.Sort(sort.Reverse(sort.IntSlice(ty)))
	}

	nextTiles := map[*Tile]struct{}{}
	merged := map[*Tile]bool{}
	moved := false
	for _, j := range ty {
		for _, i := range tx {
			t := tileAt(tiles, i, j)
			if t == nil {
				continue
			}
			ii := i
			jj := j
			for {
				ni := ii + vx
				nj := jj + vy
				if ni < 0 || ni >= size || nj < 0 || nj >= size {
					break
				}
				tt := tileAt(nextTiles, ni, nj)
				if tt == nil {
					ii = ni
					jj = nj
					moved = true
					continue
				}
				if t.value != tt.value {
					break
				}
				if !merged[tt] {
					ii = ni
					jj = nj
					moved = true
				}
				break
			}
			if tt := tileAt(tiles, ii, jj); tt != t && tt != nil {
				t.value += tt.value
				merged[t] = true
				delete(nextTiles, tt)
			}
			t.x = ii
			t.y = jj
			nextTiles[t] = struct{}{}
		}
	}
	return nextTiles, moved
}

func addRandomTile(tiles map[*Tile]struct{}, size int) error {
	cells := make([]bool, size*size)
	for t := range tiles {
		i := t.x + t.y*size
		cells[i] = true
	}
	availableCells := []int{}
	for i, b := range cells {
		if b {
			continue
		}
		availableCells = append(availableCells, i)
	}
	if len(availableCells) == 0 {
		return errors.New("twenty48: there is no space to add a new tile")
	}
	c := availableCells[rand.Intn(len(availableCells))]
	v := 2
	if rand.Intn(10) == 0 {
		v = 4
	}
	x := c % size
	y := c / size
	t := NewTile(v, x, y)
	tiles[t] = struct{}{}
	return nil
}
