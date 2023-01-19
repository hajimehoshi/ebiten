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

package main

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	bgImage        *ebiten.Image
	fgImage        *ebiten.Image
	maskedFgImage  = ebiten.NewImage(screenWidth, screenHeight)
	spotLightImage *ebiten.Image
)

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	bgImage = ebiten.NewImageFromImage(img)

	img, _, err = image.Decode(bytes.NewReader(images.FiveYears_jpg))
	if err != nil {
		log.Fatal(err)
	}
	fgImage = ebiten.NewImageFromImage(img)

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
	spotLightImage = ebiten.NewImageFromImage(a)
}

type Game struct {
	spotLightX  int
	spotLightY  int
	spotLightVX int
	spotLightVY int
}

func NewGame() *Game {
	return &Game{
		spotLightX:  0,
		spotLightY:  0,
		spotLightVX: 1,
		spotLightVY: 1,
	}
}

func (g *Game) Update() error {
	if g.spotLightVX == 0 {
		g.spotLightVX = 1
	}
	if g.spotLightVY == 0 {
		g.spotLightVY = 1
	}

	g.spotLightX += g.spotLightVX
	g.spotLightY += g.spotLightVY
	if g.spotLightX < 0 {
		g.spotLightX = -g.spotLightX
		g.spotLightVX = -g.spotLightVX
	}
	if g.spotLightY < 0 {
		g.spotLightY = -g.spotLightY
		g.spotLightVY = -g.spotLightVY
	}
	s := spotLightImage.Bounds().Size()
	maxX, maxY := screenWidth-s.X, screenHeight-s.Y
	if maxX < g.spotLightX {
		g.spotLightX = -g.spotLightX + 2*maxX
		g.spotLightVX = -g.spotLightVX
	}
	if maxY < g.spotLightY {
		g.spotLightY = -g.spotLightY + 2*maxY
		g.spotLightVY = -g.spotLightVY
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Reset the maskedFgImage.
	maskedFgImage.Fill(color.White)
	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendCopy
	op.GeoM.Translate(float64(g.spotLightX), float64(g.spotLightY))
	maskedFgImage.DrawImage(spotLightImage, op)

	// Use 'source-in' blend mode so that the source image (fgImage) is used but the alpha
	// is determined by the destination image (maskedFgImage).
	//
	// The result image is the source image with the destination alpha. In maskedFgImage, alpha
	// values in the hole is 0 and alpha values in other places are 0xff. As a result, the
	// maskedFgImage draws the source image with a hole that shape is spotLightImage. Note that
	// RGB values in the destination image are ignored.
	//
	// See also https://www.w3.org/TR/compositing-1/#porterduffcompositingoperators_srcin.
	op = &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceIn
	maskedFgImage.DrawImage(fgImage, op)

	screen.Fill(color.RGBA{0x00, 0x00, 0x80, 0xff})
	screen.DrawImage(bgImage, &ebiten.DrawImageOptions{})
	screen.DrawImage(maskedFgImage, &ebiten.DrawImageOptions{})
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
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
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Masking (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
