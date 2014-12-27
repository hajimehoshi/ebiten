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
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"image"
	_ "image/jpeg"
	"log"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	gophersImage *ebiten.Image
)

func update(screen *ebiten.Image) error {
	parts := []ebiten.ImagePart{}
	w, h := gophersImage.Size()
	for i := 0; i < h; i++ {
		width := w + i*3/4
		x := ((h - i) * 3 / 4) / 2
		parts = append(parts, ebiten.ImagePart{
			Dst: image.Rect(x, i, x+width, i+1),
			Src: image.Rect(0, i, w, i+1),
		})
	}
	maxWidth := float64(w) + float64(h)*0.75
	geo := ebiten.TranslateGeo(-maxWidth/2, -float64(h)/2)
	geo.Concat(ebiten.TranslateGeo(screenWidth/2, screenHeight/2))
	screen.DrawImage(gophersImage, &ebiten.DrawImageOptions{
		Parts: parts,
		GeoM:  geo,
	})
	return nil
}

func main() {
	var err error
	gophersImage, _, err = ebitenutil.NewImageFromFile("images/gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Perspective (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
