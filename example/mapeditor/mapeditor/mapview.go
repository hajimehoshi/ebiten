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

type MapView struct {
	m       *Map
	tileSet *TileSet
	cursorX int
	cursorY int
}

func NewMapView(m *Map) *MapView {
	return &MapView{
		m:       m,
		cursorX: -1,
		cursorY: -1,
	}
}

func (m *MapView) Update(ox, oy, width, height int, tileSet *TileSet, selectedTile int) error {
	m.tileSet = tileSet

	x, y := ebiten.CursorPosition()
	x -= ox
	y -= oy
	if x < 0 || y < 0 || width <= x || height <= y {
		return nil
	}
	if m.m.width*TileWidth <= x || m.m.height*TileHeight <= y {
		return nil
	}
	m.cursorX = x / TileWidth
	m.cursorY = y / TileHeight
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		m.m.SetTile(m.cursorX, m.cursorY, selectedTile)
	}
	return nil
}

func (m *MapView) Draw(i *ebiten.Image, x, y, width, height int) error {
	if err := m.m.Draw(i, m.tileSet.image, x, y); err != nil {
		return err
	}

	if m.cursorX == -1 || m.cursorY == -1 {
		return nil
	}
	sx := x + m.cursorX*TileWidth
	sy := y + m.cursorY*TileHeight
	i.DrawRect(sx, sy, TileWidth, TileHeight, color.Black)
	i.DrawRect(sx+1, sy+1, TileWidth-2, TileHeight-2, color.White)
	i.DrawRect(sx+2, sy+2, TileWidth-4, TileHeight-4, color.Black)

	return nil
}
