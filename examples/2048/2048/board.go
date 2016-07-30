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

	"github.com/hajimehoshi/ebiten"
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
		addRandomTile(b.tiles, b.size)
	}
	return b
}

func (b *Board) tileAt(x, y int) *Tile {
	return tileAt(b.tiles, x, y)
}

func (b *Board) Update(input *Input) error {
	if dir, ok := input.Dir(); ok {
		if err := b.Move(dir); err != nil {
			return err
		}
	}
	return nil
}

func (b *Board) Move(dir Dir) error {
	moved := false
	b.tiles, moved = MoveTiles(b.tiles, b.size, dir)
	if !moved {
		return nil
	}
	if err := addRandomTile(b.tiles, b.size); err != nil {
		return err
	}
	return nil
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
			v := 0
			op := &ebiten.DrawImageOptions{}
			x := i*tileSize + (i+1)*tileMargin
			y := j*tileSize + (j+1)*tileMargin
			op.GeoM.Translate(float64(x), float64(y))
			r, g, b, a := colorToScale(tileBackgroundColor(v))
			op.ColorM.Scale(r, g, b, a)
			if err := screen.DrawImage(tileImage, op); err != nil {
				return err
			}
		}
	}
	for t := range b.tiles {
		if err := t.Draw(screen); err != nil {
			return err
		}
	}
	return nil
}
