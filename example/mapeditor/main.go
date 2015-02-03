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

package main

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/example/mapeditor/mapeditor"
	"image/color"
	"log"
)

const (
	ScreenWidth  = 1024
	ScreenHeight = 768
)

var editor *mapeditor.MainEditor

func init() {
	tileSetImg, _, err := ebitenutil.NewImageFromFile("images/platform/tileset.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	tileSet := mapeditor.NewTileSet(tileSetImg)
	editor = mapeditor.NewMainEditor(tileSet)
}

func update(screen *ebiten.Image) error {
	if err := editor.Update(); err != nil {
		return err
	}

	backgroundColor := color.RGBA{0x80, 0x80, 0x80, 0xff}
	screen.Fill(backgroundColor)
	return editor.Draw(screen)
}

func main() {
	if err := ebiten.Run(update, ScreenWidth, ScreenHeight, 1, "Map Editor (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
