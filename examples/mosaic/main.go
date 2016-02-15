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
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
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

func update(screen *ebiten.Image) error {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0/mosaicRatio, 1.0/mosaicRatio)
	gophersRenderTarget.DrawImage(gophersImage, op)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(mosaicRatio, mosaicRatio)
	screen.DrawImage(gophersRenderTarget, op)
	return nil
}

func main() {
	var err error
	gophersImage, _, err = ebitenutil.NewImageFromFile("images/gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	w, h := gophersImage.Size()
	gophersRenderTarget, err = ebiten.NewImage(w/mosaicRatio, h/mosaicRatio, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Mosaic (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
