/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
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

func Update(screen *ebiten.Image) error {
	parts := []ebiten.ImagePart{}
	w, h := gophersImage.Size()
	for i := 0; i < h; i++ {
		width := float64(w) + float64(i)*0.75
		x := float64(h-i) * 0.75 / 2
		part := ebiten.ImagePart{
			Dst: ebiten.Rect{x, float64(i), width, 1},
			Src: ebiten.Rect{0, float64(i), float64(w), 1},
		}
		parts = append(parts, part)
	}
	maxWidth := float64(w) + float64(h)*0.75
	geo := ebiten.TranslateGeometry(-maxWidth/2, -float64(h)/2)
	geo.Concat(ebiten.ScaleGeometry(0.4, 0.4))
	geo.Concat(ebiten.TranslateGeometry(screenWidth/2, screenHeight/2))
	screen.DrawImage(gophersImage, parts, geo, ebiten.ColorMatrixI())
	return nil
}

func main() {
	var err error
	gophersImage, _, err = ebitenutil.NewImageFromFile("images/gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(Update, screenWidth, screenHeight, 2, "Perspective (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
