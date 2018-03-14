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
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	bgImage        *ebiten.Image
	fgImage        *ebiten.Image
	maskedFgImage  *ebiten.Image
	spotLightImage *ebiten.Image
	spotLightX     = 0
	spotLightY     = 0
	spotLightVX    = 1
	spotLightVY    = 1
)

func init() {
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	bgImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	img, _, err = image.Decode(bytes.NewReader(images.FiveYears_jpg))
	if err != nil {
		log.Fatal(err)
	}
	fgImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	maskedFgImage, _ = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterDefault)

	// Initialize the spot light image.
	const r = 64
	alphas := image.Point{r * 2, r * 2}
	a := image.NewAlpha(image.Rectangle{image.ZP, alphas})
	for j := 0; j < alphas.Y; j++ {
		for i := 0; i < alphas.X; i++ {
			// d is the distance between (i, j) and the (circle) center.
			d := math.Sqrt(float64((i-r)*(i-r) + (j-r)*(j-r)))
			// Alphas around the center are 0 and values outside of the circle are 0xff.
			b := uint8(max(0, min(0xff, int(3*d*0xff/r)-2*0xff)))
			a.SetAlpha(i, j, color.Alpha{b})
		}
	}
	spotLightImage, _ = ebiten.NewImageFromImage(a, ebiten.FilterDefault)
}

func update(screen *ebiten.Image) error {
	spotLightX += spotLightVX
	spotLightY += spotLightVY
	if spotLightX < 0 {
		spotLightX = -spotLightX
		spotLightVX = -spotLightVX
	}
	if spotLightY < 0 {
		spotLightY = -spotLightY
		spotLightVY = -spotLightVY
	}
	w, h := spotLightImage.Size()
	maxX, maxY := screenWidth-w, screenHeight-h
	if maxX < spotLightX {
		spotLightX = -spotLightX + 2*maxX
		spotLightVX = -spotLightVX
	}
	if maxY < spotLightY {
		spotLightY = -spotLightY + 2*maxY
		spotLightVY = -spotLightVY
	}

	if ebiten.IsRunningSlowly() {
		return nil
	}

	// Reset the maskedFgImage.
	maskedFgImage.Fill(color.White)
	op := &ebiten.DrawImageOptions{}
	op.CompositeMode = ebiten.CompositeModeCopy
	op.GeoM.Translate(float64(spotLightX), float64(spotLightY))
	maskedFgImage.DrawImage(spotLightImage, op)

	// Use 'source-in' composite mode so that the source image (fgImage) is used but the alpha
	// is determined by the destination image (maskedFgImage).
	//
	// The result image is the source image with the destination alpha. In maskedFgImage, alpha
	// values in the hole is 0 and alpha values in other places are 0xff. As a result, the
	// maskedFgImage draws the source image with a hole that shape is spotLightImage. Note that
	// RGB values in the destination image are ignored.
	//
	// See also https://www.w3.org/TR/compositing-1/#porterduffcompositingoperators_srcin.
	op = &ebiten.DrawImageOptions{}
	op.CompositeMode = ebiten.CompositeModeSourceIn
	maskedFgImage.DrawImage(fgImage, op)

	screen.Fill(color.RGBA{0x00, 0x00, 0x80, 0xff})
	screen.DrawImage(bgImage, &ebiten.DrawImageOptions{})
	screen.DrawImage(maskedFgImage, &ebiten.DrawImageOptions{})

	return nil
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Masking (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
