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

// This example is an experiment to minify images with various filters.
// When linear filter is used, mipmap images should be used for high-quality rendering (#578).

package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
)

const (
	screenWidth  = 640
	screenHeight = 800
)

var (
	gophersImage *ebiten.Image
)

func update(screen *ebiten.Image) error {
	if ebiten.IsDrawingSkipped() {
		return nil
	}

	ebitenutil.DebugPrint(screen, "Minifying images (Nearest filter vs Linear filter)")

	for i, f := range []ebiten.Filter{ebiten.FilterNearest, ebiten.FilterLinear} {
		y := 0
		for j := 0; j < 12; j++ {
			w, h := gophersImage.Size()

			op := &ebiten.DrawImageOptions{}
			s := 1.0 / math.Pow(2, float64(j)/2)
			op.GeoM.Scale(s, s)
			op.GeoM.Translate(32+float64(i*(w+4)), 32+float64(y))
			op.Filter = f
			screen.DrawImage(gophersImage, op)

			y += int(float64(h)*s) + 4
		}
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

	// Specifying filter on NewImage[FromImage] is just for backward compatibility.
	// Now specifying filter at DrawImageOptions is recommended.
	// Specify FilterDefault here, that means to prefer filter specified at DrawImageOptions.
	gophersImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Minify (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
