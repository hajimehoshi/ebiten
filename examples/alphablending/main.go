// Copyright 2014 Hajime Hoshi
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
	"image"
	"image/color"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	ebitenImage *ebiten.Image
)

type Game struct {
	count int
}

func (g *Game) Update() error {
	g.count++
	g.count %= ebiten.TPS() * 10
	return nil
}

func (g *Game) offset() float64 {
	v := float64(g.count) * 0.2
	switch {
	case 480 < g.count:
		v = 0
	case 240 < g.count:
		v = float64(480-g.count) * 0.2
	}
	return v
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{0x00, 0x00, 0x80, 0xff})

	// Draw 100 Ebitens
	v := g.offset()
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(0.5)
	for i := range 10 * 10 {
		op.GeoM.Reset()
		x := float64(i%10)*v + 15
		y := float64(i/10)*v + 20
		op.GeoM.Translate(x, y)
		screen.DrawImage(ebitenImage, op)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}
	ebitenImage = ebiten.NewImageFromImage(img)

	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Alpha Blending (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
