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

// +build example jsgo

package main

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
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

func update(screen *ebiten.Image) error {
	if ebiten.IsRunningSlowly() {
		return nil
	}

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
		op := &ebiten.DrawImageOptions{}
		x := i % 4
		y := i/4 + 1
		op.GeoM.Translate(ox+float64(dx*x), oy+float64(dy*y))

		// Reset RGB (not Alpha) 0 forcibly
		op.ColorM.Scale(0, 0, 0, 1)

		// Set color
		r := float64(c.R) / 0xff
		g := float64(c.G) / 0xff
		b := float64(c.B) / 0xff
		op.ColorM.Translate(r, g, b, 0)
		screen.DrawImage(ebitenImage, op)
	}
	return nil
}

func main() {
	// Decode image from a byte slice instead of a file so that
	// this example works in any working directory.
	// If you want to use a file, there are some options:
	// 1) Use os.Open and pass the file to the image decoder.
	//    This is a very regular way, but doesn't work on browsers.
	// 2) Use ebitenutil.OpenFile and pass the file to the image decoder.
	//    This works even on browsers.
	// 3) Use ebitenutil.NewImageFromFile to create an ebiten.Image directly from a file.
	//    This also works on browsers.
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}
	ebitenImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Flood fill with solid colors (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
