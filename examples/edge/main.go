// Copyright 2017 The Ebiten Authors
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
	"image/color"
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
	colorImage       *ebiten.Image
	colorImageWidth  = 64
	colorImageHeight = 64
	angle            = 0
)

func update(screen *ebiten.Image) error {
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		angle--
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		angle++
	}
	angle %= 360
	if err := screen.Fill(color.White); err != nil {
		return err
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(colorImageWidth)/2, -float64(colorImageHeight)/2)
	op.GeoM.Rotate(float64(angle) * math.Pi / 180)
	op.GeoM.Scale(4, 4)
	op.GeoM.Translate(screenWidth/2, screenHeight/2)
	if err := screen.DrawImage(colorImage, op); err != nil {
		return err
	}
	if err := ebitenutil.DebugPrint(screen, fmt.Sprintf("Angle: %d [deg]", angle)); err != nil {
		return err
	}
	return nil
}

func main() {
	var err error
	colorImage, err = ebiten.NewImage(colorImageWidth, colorImageHeight, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	pixels := make([]uint8, 4*colorImageWidth*colorImageHeight)
	for j := 0; j < colorImageHeight; j++ {
		for i := 0; i < colorImageWidth; i++ {
			idx := 4 * (i + j*colorImageWidth)
			switch {
			case j < colorImageHeight/2:
				pixels[idx] = 0xff
				pixels[idx+1] = 0
				pixels[idx+2] = 0
				pixels[idx+3] = 0xff
			default:
				pixels[idx] = 0
				pixels[idx+1] = 0xff
				pixels[idx+2] = 0
				pixels[idx+3] = 0xff
			}
		}
	}
	if err := colorImage.ReplacePixels(pixels); err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Edge (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
