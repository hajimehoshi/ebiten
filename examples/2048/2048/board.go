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
	"image/color"
	"math/rand"
	"sort"
	"strconv"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/common"
)

type Board struct {
	size  int
	tiles map[*Tile]struct{}
}

func NewBoard(size int) *Board {
	b := &Board{
		size:  size,
		tiles: map[*Tile]struct{}{},
	}
	for i := 0; i < 2; i++ {
		b.addRandomTile()
	}
	return b
}

func (b *Board) addRandomTile() bool {
	cells := make([]bool, b.size*b.size)
	for t := range b.tiles {
		i := t.x + t.y*b.size
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
		return false
	}
	c := availableCells[rand.Intn(len(availableCells))]
	v := 2
	if rand.Intn(10) == 0 {
		v = 4
	}
	x := c % b.size
	y := c / b.size
	t := NewTile(v, x, y)
	b.tiles[t] = struct{}{}
	return true
}

func tileAt(tiles map[*Tile]struct{}, x, y int) *Tile {
	for t := range tiles {
		if t.x == x && t.y == y {
			return t
		}
	}
	return nil
}

func (b *Board) tileAt(x, y int) *Tile {
	return tileAt(b.tiles, x, y)
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

func (b *Board) Move(dir Dir) {
	moved := false
	b.tiles, moved = MoveTiles(b.tiles, b.size, dir)
	if !moved {
		return
	}
	b.addRandomTile()
}

const (
	tileSize   = 40
	tileMargin = 2
)

var (
	tileImage *ebiten.Image
)

func init() {
	var err error
	tileImage, err = ebiten.NewImage(tileSize, tileSize, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	if err := tileImage.Fill(color.White); err != nil {
		panic(err)
	}
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

func (b *Board) Size() (int, int) {
	x := b.size*tileSize + (b.size+1)*tileMargin
	y := x
	return x, y
}

func (b *Board) Draw(screen *ebiten.Image) error {
	if err := screen.Fill(frameColor); err != nil {
		return err
	}
	for j := 0; j < b.size; j++ {
		for i := 0; i < b.size; i++ {
			t := b.tileAt(i, j)
			v := 0
			if t != nil {
				v = t.value
			}
			op := &ebiten.DrawImageOptions{}
			x := i*tileSize + (i+1)*tileMargin
			y := j*tileSize + (j+1)*tileMargin
			op.GeoM.Translate(float64(x), float64(y))
			r, g, b, a := colorToScale(tileBackgroundColor(v))
			op.ColorM.Scale(r, g, b, a)
			if err := screen.DrawImage(tileImage, op); err != nil {
				return err
			}
			if t != nil {
				str := strconv.Itoa(t.value)
				scale := 2
				if 2 < len(str) {
					scale = 1
				}
				w := common.ArcadeFont.TextWidth(str) * scale
				h := common.ArcadeFont.TextHeight(str) * scale
				x := x + (tileSize-w)/2
				y := y + (tileSize-h)/2
				common.ArcadeFont.DrawText(screen, str, x, y, scale, tileColor(t.value))
			}
		}
	}
	return nil
}
