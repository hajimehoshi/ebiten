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
	_ "image/jpeg"
	"log"
	"math"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	count           int
	horizontalCount int
	verticalCount   int
	gophersImage    *ebiten.Image
)

func update(screen *ebiten.Image) error {
	count++
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		horizontalCount--
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		horizontalCount++
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		verticalCount--
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		verticalCount++
	}

	w, h := gophersImage.Size()
	geo := ebiten.TranslateGeometry(-float64(w)/2, -float64(h)/2)
	scaleX := math.Pow(1.05, float64(horizontalCount))
	scaleY := math.Pow(1.05, float64(verticalCount))
	geo.Concat(ebiten.ScaleGeometry(scaleX, scaleY))
	geo.Concat(ebiten.RotateGeometry(float64(count%720) * 2 * math.Pi / 720))
	geo.Concat(ebiten.TranslateGeometry(screenWidth/2, screenHeight/2))
	if err := screen.DrawImage(gophersImage, &ebiten.ImageDrawOption{
		GeometryMatrix: &geo,
	}); err != nil {
		return err
	}
	return nil
}

func main() {
	var err error
	gophersImage, _, err = ebitenutil.NewImageFromFile("images/gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Image (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
