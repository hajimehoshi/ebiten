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
	"image/color"
	"math/rand"
	"sort"
	"strconv"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/common"
)

type TileData struct {
	value int
	x     int
	y     int
}

type Tile struct {
	current TileData
	next    TileData
	moving  bool
}

func NewTile(value int, x, y int) *Tile {
	return &Tile{
		current: TileData{
			value: value,
			x:     x,
			y:     y,
		},
	}
}

func (t *Tile) Value() int {
	return t.current.value
}

func (t *Tile) Pos() (int, int) {
	return t.current.x, t.current.y
}

func (t *Tile) NextValue() int {
	return t.next.value
}

func (t *Tile) NextPos() (int, int) {
	return t.next.x, t.next.y
}

func tileAt(tiles map[*Tile]struct{}, x, y int) *Tile {
	var result *Tile
	for t := range tiles {
		if t.current.x != x || t.current.y != y {
			continue
		}
		if result != nil {
			panic("not reach")
		}
		result = t
	}
	return result
}

func nextTileAt(tiles map[*Tile]struct{}, x, y int) *Tile {
	var result *Tile
	for t := range tiles {
		if t.next.x != x || t.next.y != y || t.next.value == 0 {
			continue
		}
		if result != nil {
			panic("not reach")
		}
		result = t
	}
	return result
}

func MoveTiles(tiles map[*Tile]struct{}, size int, dir Dir) bool {
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
				tt := nextTileAt(tiles, ni, nj)
				if tt == nil {
					ii = ni
					jj = nj
					moved = true
					continue
				}
				if t.current.value != tt.next.value {
					break
				}
				if tt.current.value != tt.next.value {
					// already merged
					break
				}
				ii = ni
				jj = nj
				moved = true
				break
			}
			if tt := nextTileAt(tiles, ii, jj); tt != t && tt != nil {
				t.next.value = t.current.value + tt.next.value
				tt.next = TileData{}
			} else {
				t.next.value = t.current.value
			}
			t.next.x = ii
			t.next.y = jj
			t.moving = true
		}
	}
	return moved
}

func addRandomTile(tiles map[*Tile]struct{}, size int) error {
	cells := make([]bool, size*size)
	for t := range tiles {
		i := 0
		if t.moving {
			i = t.next.x + t.next.y*size
		} else {
			i = t.current.x + t.current.y*size
		}
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

func (t *Tile) Update() error {
	if !t.moving {
		return nil
	}
	t.current = t.next
	t.next = TileData{}
	t.moving = false
	return nil
}

func colorToScale(clr color.Color) (float64, float64, float64, float64) {
	r, g, b, a := clr.RGBA()
	rf := float64(r) / 0xffff
	gf := float64(g) / 0xffff
	bf := float64(b) / 0xffff
	af := float64(a) / 0xffff
	// Convert to non-premultiplied alpha components.
	if 0 < af {
		rf /= af
		gf /= af
		bf /= af
	}
	return rf, gf, bf, af
}

func (t *Tile) Draw(boardImage *ebiten.Image) error {
	i, j := t.current.x, t.current.y
	v := t.current.value
	op := &ebiten.DrawImageOptions{}
	x := i*tileSize + (i+1)*tileMargin
	y := j*tileSize + (j+1)*tileMargin
	op.GeoM.Translate(float64(x), float64(y))
	r, g, b, a := colorToScale(tileBackgroundColor(v))
	op.ColorM.Scale(r, g, b, a)
	if err := boardImage.DrawImage(tileImage, op); err != nil {
		return err
	}
	str := strconv.Itoa(v)
	scale := 2
	if 2 < len(str) {
		scale = 1
	}
	w := common.ArcadeFont.TextWidth(str) * scale
	h := common.ArcadeFont.TextHeight(str) * scale
	x = x + (tileSize-w)/2
	y = y + (tileSize-h)/2
	common.ArcadeFont.DrawText(boardImage, str, x, y, scale, tileColor(v))
	return nil
}
