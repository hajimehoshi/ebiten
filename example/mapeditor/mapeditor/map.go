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

type Map struct {
	width  int
	height int
	tiles  []int
}

func NewMap(width, height int) *Map {
	return &Map{
		width:  width,
		height: height,
		tiles:  make([]int, width*height),
	}
}

func (m *Map) TileAt(x, y int) (int, error) {
	/*if x < 0 || y < 0 || TileWidth*TileSetXNum <= x || TileHeight*TileSetYNum <= y {
		return 0, fmt.Errorf("out of range: (%d, %d)", x, y)
	}*/
	return x/TileWidth + y/TileHeight, nil
}

func (m *Map) SetTile(x, y int, tile int) {
	i := x + y*m.width
	m.tiles[i] = tile
}

type TileRects struct {
	m *Map
}

func (t *TileRects) Len() int {
	return t.m.width * t.m.height
}

func (t *TileRects) Src(i int) (x0, y0, x1, y1 int) {
	tile := t.m.tiles[i]
	x := tile % TileSetXNum * TileLogicWidth
	y := tile / TileSetXNum * TileLogicHeight
	return x, y, x + TileLogicWidth, y + TileLogicHeight
}

func (t *TileRects) Dst(i int) (x0, y0, x1, y1 int) {
	x := i % t.m.width * TileLogicWidth
	y := i / t.m.width * TileLogicHeight
	return x, y, x + TileLogicWidth, y + TileLogicHeight
}

func (m *Map) Draw(i *ebiten.Image, tileSetImg *ebiten.Image, x, y int) error {
	i.Fill(color.RGBA{0x80, 0x80, 0x80, 0xff})

	op := &ebiten.DrawImageOptions{
		ImageParts: &TilesBackgroundRects{m.width, m.height},
	}
	op.GeoM.Translate(float64(x), float64(y))
	if err := i.DrawImage(tilesBackground, op); err != nil {
		return err
	}

	op = &ebiten.DrawImageOptions{
		ImageParts: &TileRects{m},
	}
	op.GeoM.Translate(float64(x), float64(y))
	op.GeoM.Scale(2, 2)
	if err := i.DrawImage(tileSetImg, op); err != nil {
		return err
	}
	return nil
}
