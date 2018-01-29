// Copyright 2016 Hajime Hoshi
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

// +build example

package main

import (
	"image/color"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	ebitenImage *ebiten.Image
)

func update(screen *ebiten.Image) error {
	if ebiten.IsRunningSlowly() {
		return nil
	}

	const (
		// The offset point to render the image.
		ox = 10
		oy = 10
	)

	screen.Fill(color.NRGBA{0x00, 0x40, 0x80, 0xff})

	// Draw the image with 'Source Alpha' composite mode (default).
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox, oy)
	screen.DrawImage(ebitenImage, op)

	// Draw the image with 'Lighter (a.k.a Additive)' composite mode.
	op = &ebiten.DrawImageOptions{}
	w, _ := ebitenImage.Size()
	op.GeoM.Translate(ox+float64(w), oy)
	op.CompositeMode = ebiten.CompositeModeLighter
	screen.DrawImage(ebitenImage, op)

	return nil
}

func main() {
	var err error
	ebitenImage, _, err = ebitenutil.NewImageFromFile("_resources/images/ebiten.png", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Additive Blending (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
