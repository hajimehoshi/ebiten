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

const mosaicRatio = 16

var (
	gophersImage        *ebiten.Image
	gophersRenderTarget *ebiten.Image
)

func init() {
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
}

func update(screen *ebiten.Image) error {
	if ebiten.IsRunningSlowly() {
		return nil
	}

	// Shrink the image once.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0/mosaicRatio, 1.0/mosaicRatio)
	gophersRenderTarget.DrawImage(gophersImage, op)

	// Enlarge the shrunk image.
	// The filter is the nearest filter, so the result will be mosaic.
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(mosaicRatio, mosaicRatio)
	screen.DrawImage(gophersRenderTarget, op)
	return nil
}

func main() {
	w, h := gophersImage.Size()
	gophersRenderTarget, _ = ebiten.NewImage(w/mosaicRatio, h/mosaicRatio, ebiten.FilterDefault)
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Mosaic (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
