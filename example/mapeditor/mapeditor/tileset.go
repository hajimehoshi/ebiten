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
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"image/color"
)

const (
	TileLogicWidth  = 16
	TileLogicHeight = 16
	TileWidth       = 32
	TileHeight      = 32
	TileSetXNum     = 8
	TileSetYNum     = 16
)

var tilesBackground *ebiten.Image

const tilesBackgroundWidth = 32
const tilesBackgroundHeight = 32

func init() {
	var err error
	tilesBackground, err = ebiten.NewImage(tilesBackgroundWidth, tilesBackgroundHeight, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	blue := color.RGBA{0x00, 0x00, 0x80, 0xff}
	black := color.Black
	tilesBackground.DrawFilledRect(0, 0, 16, 16, blue)
	tilesBackground.DrawFilledRect(16, 0, 16, 16, black)
	tilesBackground.DrawFilledRect(0, 16, 16, 16, black)
	tilesBackground.DrawFilledRect(16, 16, 16, 16, blue)
}

type TilesBackgroundRects struct {
	xNum int
	yNum int
}

func (t *TilesBackgroundRects) Len() int {
	return t.xNum * t.yNum
}

func (t *TilesBackgroundRects) Src(i int) (x0, y0, x1, y1 int) {
	return 0, 0, tilesBackgroundWidth, tilesBackgroundHeight
}

func (t *TilesBackgroundRects) Dst(i int) (x0, y0, x1, y1 int) {
	x := i % t.xNum * tilesBackgroundWidth
	y := i / t.xNum * tilesBackgroundHeight
	return x, y, x + tilesBackgroundWidth, y + tilesBackgroundHeight
}

type TileSet struct {
	image *ebiten.Image
}

func NewTileSet(img *ebiten.Image) *TileSet {
	return &TileSet{
		image: img,
	}
}

func (t *TileSet) TileAt(x, y int) (int, error) {
	if x < 0 || y < 0 || TileWidth*TileSetXNum <= x || TileHeight*TileSetYNum <= y {
		return 0, fmt.Errorf("out of range: (%d, %d)", x, y)
	}
	return x/TileWidth + y/TileHeight*TileSetXNum, nil
}

func (t *TileSet) Draw(i *ebiten.Image, x, y, width, height int) error {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ImageParts = &TilesBackgroundRects{8, 16}
	if err := i.DrawImage(tilesBackground, op); err != nil {
		return err
	}

	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(x), float64(y))
	if err := i.DrawImage(t.image, op); err != nil {
		return err
	}

	/*sx := x + s%TileSetXNum*TileWidth
	sy := y + s/TileSetXNum*TileHeight
	i.DrawRect(sx, sy, TileWidth, TileHeight, color.Black)
	i.DrawRect(sx+1, sy+1, TileWidth-2, TileHeight-2, color.White)
	i.DrawRect(sx+2, sy+2, TileWidth-4, TileHeight-4, color.Black)*/

	return nil
}
