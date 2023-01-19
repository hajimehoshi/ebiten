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
	// Split the image into horizontal lines and draw them with different scales.
	op := &ebiten.DrawImageOptions{}
	w, h := gophersImage.Bounds().Dx(), gophersImage.Bounds().Dy()
	for i := 0; i < h; i++ {
		op.GeoM.Reset()

		// Move the image's center to the upper-left corner.
		op.GeoM.Translate(-float64(w)/2, -float64(h)/2)

		// Scale each lines and adjust the position.
		lineW := w + i*3/4
		x := -float64(lineW) / float64(w) / 2
		op.GeoM.Scale(float64(lineW)/float64(w), 1)
		op.GeoM.Translate(x, float64(i))

		// Move the image's center to the screen's center.
		op.GeoM.Translate(screenWidth/2, screenHeight/2)

		screen.DrawImage(gophersImage.SubImage(image.Rect(0, i, w, i+1)).(*ebiten.Image), op)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage = ebiten.NewImageFromImage(img)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Perspective (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
