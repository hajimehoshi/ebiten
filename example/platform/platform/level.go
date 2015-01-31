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

package platform

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	ScreenWidth  = 256
	ScreenHeight = 240
)

var tileSet *ebiten.Image

func init() {
	var err error
	tileSet, _, err = ebitenutil.NewImageFromFile("images/platform/tileset.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
}

const (
	TileSize    = 16
	TileSrcXNum = 8
	TileDstXNum = ScreenWidth / TileSize
	TileDstYNum = ScreenHeight / TileSize
)

type levelRects struct {
	*Level
}

func (l *levelRects) Len() int {
	return TileDstXNum * TileDstYNum
}

func tileRect(i int) (x0, y0, x1, y1 int) {
	x := (i % TileSrcXNum) * TileSize
	y := (i / TileSrcXNum) * TileSize
	return x, y, x + TileSize, y + TileSize
}

func (l *levelRects) Src(i int) (x0, y0, x1, y1 int) {
	if (i / TileDstXNum) < 13 {
		return tileRect(1)
	}
	return tileRect(2)
}

func (l *levelRects) Dst(i int) (x0, y0, x1, y1 int) {
	x := (i % TileDstXNum) * TileSize
	y := (i / TileDstXNum) * TileSize
	return x, y, x + TileSize, y + TileSize
}

type Level struct {
}

func (l *Level) Draw(screen *ebiten.Image) error {
	return screen.DrawImage(tileSet, &ebiten.DrawImageOptions{
		ImageParts: &levelRects{l},
	})
}
