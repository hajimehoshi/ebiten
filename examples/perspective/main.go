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

// +build example jsgo

package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	gophersImage *ebiten.Image
)

func update(screen *ebiten.Image) error {
	if ebiten.IsRunningSlowly() {
		return nil
	}

	// Split the image into horizontal lines and draw them with different scales.
	op := &ebiten.DrawImageOptions{}
	w, h := gophersImage.Size()
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

		r := image.Rect(0, i, w, i+1)
		op.SourceRect = &r
		screen.DrawImage(gophersImage, op)
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
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Perspective (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
