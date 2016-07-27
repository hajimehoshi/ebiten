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
	"fmt"
	"math/rand"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
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

func (b *Board) Move(dir Dir) {
	b.addRandomTile()
}

func (b *Board) Draw(screen *ebiten.Image) error {
	posToTile := map[int]*Tile{}
	for t := range b.tiles {
		i := t.x + t.y*b.size
		posToTile[i] = t
	}
	str := ""
	for j := 0; j < b.size; j++ {
		for i := 0; i < b.size; i++ {
			t := posToTile[i+j*b.size]
			if t != nil {
				str += fmt.Sprintf("[%4d]", t.value)
			} else {
				str += "[    ]"
			}
		}
		str += "\n"
	}
	ebitenutil.DebugPrint(screen, str)
	return nil
}
