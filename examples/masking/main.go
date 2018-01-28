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
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
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
	var err error
	bgImage, _, err = ebitenutil.NewImageFromFile(ebitenutil.JoinStringsIntoFilePath("_resources", "images", "gophers.jpg"), ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}

	fgImage, _, err = ebitenutil.NewImageFromFile(ebitenutil.JoinStringsIntoFilePath("_resources", "images", "fiveyears.jpg"), ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}

	maskedFgImage, _ = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterNearest)

	// Initialize the spot light image.
	const r = 64
	alphas := image.Point{r * 2, r * 2}
	a := image.NewAlpha(image.Rectangle{image.ZP, alphas})
	for j := 0; j < alphas.Y; j++ {
		for i := 0; i < alphas.X; i++ {
			// d is the distance between (i, j) and the (circle) center.
			d := math.Sqrt(float64((i-r)*(i-r) + (j-r)*(j-r)))
			// Alpha at the center is 0xff and the outside of the circle is 0.
			b := uint8(max(0, min(0xff, 3*(0xff-int(d*0xff)/r))))
			a.SetAlpha(i, j, color.Alpha{b})
		}
	}
	// Note that alpha values matter an other RGB values don't matter in the spot light image.
	spotLightImage, _ = ebiten.NewImageFromImage(a, ebiten.FilterNearest)
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

	maskedFgImage.Clear()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(spotLightX), float64(spotLightY))
	maskedFgImage.DrawImage(spotLightImage, op)

	// Use 'source-out' composite mode so that the source image (fgImage) is used but the alpha
	// is determined by the destination image (maskedFgImage). With source-out, the destination
	// image values are not used at the result image.
	//
	// If the alpha value of the destination is 0xff, the source at the point is not adopted.
	// In the opposite way, if the alpha value of the destination is 0, the source at the point
	// is fully adopted. As alpha values outside of the spot light image are 0, the source
	// values are fully adopted there. As a result, the maskedFgImage draws the source image
	// with a hole that shape is spotLightImage.
	//
	// See also https://www.w3.org/TR/compositing-1/#porterduffcompositingoperators_srcout.
	op = &ebiten.DrawImageOptions{}
	op.CompositeMode = ebiten.CompositeModeSourceOut
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
