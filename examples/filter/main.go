// Copyright 2018 The Ebiten Authors
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
	"bytes"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	ebitenImage *ebiten.Image
)

type Game struct {
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Nearest Filter (default) VS Linear Filter")

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(4, 4)
	op.GeoM.Translate(64, 64)
	// By default, nearest filter is used.
	screen.DrawImage(ebitenImage, op)

	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(4, 4)
	op.GeoM.Translate(64, 64+240)
	// Specify linear filter.
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(ebitenImage, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	var err error

	ebitenImage, _, err = ebitenutil.NewImageFromReader(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Filter (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
