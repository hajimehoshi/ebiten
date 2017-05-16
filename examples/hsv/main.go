// Copyright 2016 Hajime Hoshi
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
	"fmt"
	_ "image/jpeg"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	hueInt        = 0
	saturationInt = 128
	valueInt      = 128
	gophersImage  *ebiten.Image
)

func clamp(v, min, max int) int {
	if min > max {
		panic("min must <= max")
	}
	if v < min {
		return min
	}
	if max < v {
		return max
	}
	return v
}

func update(screen *ebiten.Image) error {
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		hueInt--
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		hueInt++
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		saturationInt--
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		saturationInt++
	}
	if ebiten.IsKeyPressed(ebiten.KeyZ) {
		valueInt--
	}
	if ebiten.IsKeyPressed(ebiten.KeyX) {
		valueInt++
	}
	if ebiten.IsRunningSlowly() {
		return nil
	}
	hueInt = clamp(hueInt, -256, 256)
	saturationInt = clamp(saturationInt, 0, 256)
	valueInt = clamp(valueInt, 0, 256)

	w, h := gophersImage.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth-w)/2, float64(screenHeight-h)/2)
	hue := float64(hueInt) * 2 * math.Pi / 128
	saturation := float64(saturationInt) / 128
	value := float64(valueInt) / 128
	op.ColorM.ChangeHSV(hue, saturation, value)
	screen.DrawImage(gophersImage, op)

	msg := fmt.Sprintf(`Hue:        %0.2f [Q][W]
Saturation: %0.2f [A][S]
Value:      %0.2f [Z][X]`, hue, saturation, value)
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	var err error
	gophersImage, _, err = ebitenutil.NewImageFromFile("_resources/images/gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "HSV (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
