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
	"image"
	"image/color"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	ebitenImage *ebiten.Image
	colors      = []color.RGBA{
		{0xff, 0xff, 0xff, 0xff},
		{0xff, 0xff, 0x0, 0xff},
		{0xff, 0x0, 0xff, 0xff},
		{0xff, 0x0, 0x0, 0xff},
		{0x0, 0xff, 0xff, 0xff},
		{0x0, 0xff, 0x0, 0xff},
		{0x0, 0x0, 0xff, 0xff},
		{0x0, 0x0, 0x0, 0xff},
	}
)

type Game struct {
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	const (
		ox = 10
		oy = 10
		dx = 60
		dy = 50
	)
	screen.Fill(color.NRGBA{0x00, 0x40, 0x80, 0xff})

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox, oy)
	screen.DrawImage(ebitenImage, op)

	// Fill with solid colors
	for i, c := range colors {
		op := &colorm.DrawImageOptions{}
		x := i % 4
		y := i/4 + 1
		op.GeoM.Translate(ox+float64(dx*x), oy+float64(dy*y))

		// Reset RGB (not Alpha) 0 forcibly
		var cm colorm.ColorM
		cm.Scale(0, 0, 0, 1)

		// Set color
		r := float64(c.R) / 0xff
		g := float64(c.G) / 0xff
		b := float64(c.B) / 0xff
		cm.Translate(r, g, b, 0)
		colorm.DrawImage(screen, ebitenImage, cm, op)
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
	ebiten.SetWindowTitle("Flood fill with solid colors (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
