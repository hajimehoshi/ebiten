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

// +build example

package main

import (
	"image"
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
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
	op := &ebiten.DrawImageOptions{}
	w, h := gophersImage.Size()
	for i := 0; i < h; i++ {
		op.GeoM.Reset()
		width := w + i*3/4
		x := ((h - i) * 3 / 4) / 2
		op.GeoM.Scale(float64(width)/float64(w), 1)
		op.GeoM.Translate(float64(x), float64(i))
		maxWidth := float64(w) + float64(h)*3/4
		op.GeoM.Translate(-maxWidth/2, -float64(h)/2)
		op.GeoM.Translate(screenWidth/2, screenHeight/2)
		p := image.Rect(0, i, w, i+1)
		op.SourceRect = &p
		screen.DrawImage(gophersImage, op)
	}
	return nil
}

func main() {
	var err error
	gophersImage, _, err = ebitenutil.NewImageFromFile("_resources/images/gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Perspective (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
