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
	screenWidth  = 320
	screenHeight = 240
)

const mosaicRatio = 16

var (
	gophersImage *ebiten.Image
)

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage = ebiten.NewImageFromImage(img)
}

type Game struct {
	gophersRenderTarget *ebiten.Image
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Shrink the image once.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0/mosaicRatio, 1.0/mosaicRatio)
	g.gophersRenderTarget.DrawImage(gophersImage, op)

	// Enlarge the shrunk image.
	// The filter is the nearest filter, so the result will be mosaic.
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(mosaicRatio, mosaicRatio)
	screen.DrawImage(g.gophersRenderTarget, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	w, h := gophersImage.Bounds().Dx(), gophersImage.Bounds().Dy()
	g := &Game{
		gophersRenderTarget: ebiten.NewImage(w/mosaicRatio, h/mosaicRatio),
	}
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Mosaic (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
