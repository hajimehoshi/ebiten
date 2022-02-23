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

//go:build example
// +build example

package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	gophersImage *ebiten.Image
)

type Game struct{}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 0)
	screen.DrawImage(gophersImage, op)

	// Box blur (7x7)
	// https://en.wikipedia.org/wiki/Box_blur
	//
	// Note that this is a fixed function implementation of a box blur - more
	// efficiency can be gained by using a separable blur
	// (blurring horizontally and vertically separately, or for large blurs,
	// even multiple horizontal or vertical passes), ideally combined with
	// doing the summing up in a fragment shader (Kage can be used here).
	//
	// So this implementation only serves to demonstrate use of alpha blending.
	layers := 0
	for j := -3; j <= 3; j++ {
		for i := -3; i <= 3; i++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(i), 244+float64(j))
			// This is a blur based on the CompositerModeSourceOver composition mode,
			// which is basically (GL_ONE, GL_ONE_MINUS_SRC_ALPHA). ColorM acts
			// on unpremultiplied colors, but all Ebiten internal colors are
			// premultiplied, meaning this mode is regular alpha blending,
			// computing each destination pixel as srcPix * alpha + dstPix * (1 - alpha).
			//
			// This means that the final color is affected by the destination color when CompositeModeSourceOver is used.
			// This composite mode is the default mode. See how this is calculated at the doc:
			// https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#CompositeMode
			//
			// So if using the same alpha every time, the end result will sure be biased towards the last layer.
			//
			// Correct averaging works based on
			//   Let A_n := (a_1 + ... + a_n) / n
			//   A_{n+1} = (a_1 + ... + a_{n+1}) / (n + 1)
			//   A_{n+1} = (n * A_n + a_{n+1)) / (n + 1)
			//   A_{n+1} = A_n * (1 - 1/(n+1)) + a_{n+1} * 1/(n+1)
			// which is precisely what an alpha blend with alpha 1/(n+1) does.
			layers++
			op.ColorM.Scale(1, 1, 1, 1.0/float64(layers))
			screen.DrawImage(gophersImage, op)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Decode an image from the image file's byte slice.
	// Now the byte slice is generated with //go:generate for Go 1.15 or older.
	// If you use Go 1.16 or newer, it is strongly recommended to use //go:embed to embed the image file.
	// See https://pkg.go.dev/embed for more details.
	img, _, err := image.Decode(bytes.NewReader(images.FiveYears_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage = ebiten.NewImageFromImage(img)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Blur (Ebiten Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
