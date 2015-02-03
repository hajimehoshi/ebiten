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
)

type MainEditor struct {
	tileSet      *TileSet
	tileSetX     int
	tileSetY     int
	selectedTile int
}

func NewMainEditor(tileSet *TileSet) *MainEditor {
	return &MainEditor{
		tileSet:  tileSet,
		tileSetX: 16,
		tileSetY: 16,
	}
}

func (m *MainEditor) Update() error {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		x -= m.tileSetX
		y -= m.tileSetY
		if 0 <= x && 0 <= y && x < TileWidth*TileSetXNum && y < TileHeight*TileSetYNum {
			tile, err := m.tileSet.TileAt(x, y)
			if err != nil {
				return err
			}
			m.selectedTile = tile
		}
	}
	return nil
}

func (m *MainEditor) Draw(screen *ebiten.Image) error {
	return m.tileSet.Draw(screen, m.selectedTile, m.tileSetX, m.tileSetY)
}
