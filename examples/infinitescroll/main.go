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

package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	bgImage *ebiten.Image
)

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Tile_png))
	if err != nil {
		log.Fatal(err)
	}
	bgImage = ebiten.NewImageFromImage(img)
}

type viewport struct {
	x16 int
	y16 int
}

func (p *viewport) Move() {
	s := bgImage.Bounds().Size()
	maxX16 := s.X * 16
	maxY16 := s.Y * 16

	p.x16 += s.X / 32
	p.y16 += s.Y / 32
	p.x16 %= maxX16
	p.y16 %= maxY16
}

func (p *viewport) Position() (int, int) {
	return p.x16, p.y16
}

type Game struct {
	viewport viewport
}

func (g *Game) Update() error {
	g.viewport.Move()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	x16, y16 := g.viewport.Position()
	offsetX, offsetY := float64(-x16)/16, float64(-y16)/16

	// Draw bgImage on the screen repeatedly.
	const repeat = 3
	w, h := bgImage.Bounds().Dx(), bgImage.Bounds().Dy()
	for j := 0; j < repeat; j++ {
		for i := 0; i < repeat; i++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(w*i), float64(h*j))
			op.GeoM.Translate(offsetX, offsetY)
			screen.DrawImage(bgImage, op)
		}
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Infinite Scroll (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
