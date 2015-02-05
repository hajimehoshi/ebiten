// Copyright 2015 Hajime Hoshi
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

package mapeditor

import (
	"github.com/hajimehoshi/ebiten"
	"image/color"
)

type TileSetView struct {
	tileSet      *TileSet
	selectedTile int
	dragging     bool
}

func NewTileSetView(tileSet *TileSet) *TileSetView {
	return &TileSetView{
		tileSet: tileSet,
	}
}

func (t *TileSetView) Update(input *Input, ox, oy, width, height int) error {
	x, y := ebiten.CursorPosition()
	x -= ox
	y -= oy
	if x < 0 || y < 0 || width <= x || height <= y {
		return nil
	}
	if TileWidth*TileSetXNum <= x && TileHeight*TileSetYNum <= y {
		return nil
	}
	if input.MouseButtonState(ebiten.MouseButtonLeft) == 0 {
		t.dragging = false
		return nil
	}
	if input.MouseButtonState(ebiten.MouseButtonLeft) == 1 {
		t.dragging = true
	}
	// TODO: Implement dragging to select multiple tiles.

	if input.MouseButtonState(ebiten.MouseButtonLeft) == 1 {
		tile, err := t.tileSet.TileAt(x, y)
		if err != nil {
			return err
		}
		t.selectedTile = tile
	}
	return nil
}

func (t *TileSetView) Draw(i *ebiten.Image, x, y, width, height int) error {
	t.tileSet.Draw(i, x, y, width, height)

	s := t.selectedTile
	sx := x + s%TileSetXNum*TileWidth
	sy := y + s/TileSetXNum*TileHeight
	i.DrawRect(sx, sy, TileWidth, TileHeight, color.Black)
	i.DrawRect(sx+1, sy+1, TileWidth-2, TileHeight-2, color.White)
	i.DrawRect(sx+2, sy+2, TileWidth-4, TileHeight-4, color.Black)

	return nil
}
