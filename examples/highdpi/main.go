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

var (
	count        int
	gophersImage *ebiten.Image
)

func update(screen *ebiten.Image) error {
	count++
	if ebiten.IsRunningSlowly() {
		return nil
	}
	scale := ebiten.DeviceScaleFactor()
	sw, sh := screen.Size()

	w, h := gophersImage.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Rotate(float64(count%360) * 2 * math.Pi / 360)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(sw)/2, float64(sh)/2)
	screen.DrawImage(gophersImage, op)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("Scale: %0.2f", scale))
	return nil
}

func main() {
	const (
		screenWidth  = 320
		screenHeight = 240
	)

	var err error
	gophersImage, _, err = ebitenutil.NewImageFromFile(ebitenutil.JoinStringsIntoFilePath("_resources", "images", "gophers.jpg"), ebiten.FilterLinear)
	if err != nil {
		log.Fatal(err)
	}
	s := ebiten.DeviceScaleFactor()
	// Pass the invert of scale so that Ebiten's auto scaling by device scale is disabled.
	if err := ebiten.Run(update, int(screenWidth*s), int(screenHeight*s), 1/s, "High DPI (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
