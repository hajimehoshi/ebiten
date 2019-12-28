// Copyright 2019 The Ebiten Authors
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
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

func drawEbitenText(screen *ebiten.Image) {
	var path vector.Path

	// E
	path.MoveTo(20, 20)
	path.LineTo(20, 70)
	path.LineTo(70, 70)
	path.LineTo(70, 60)
	path.LineTo(30, 60)
	path.LineTo(30, 50)
	path.LineTo(70, 50)
	path.LineTo(70, 40)
	path.LineTo(30, 40)
	path.LineTo(30, 30)
	path.LineTo(70, 30)
	path.LineTo(70, 20)

	// B
	path.MoveTo(80, 20)
	path.LineTo(80, 70)
	path.LineTo(100, 70)
	path.QuadraticCurveTo(150, 57.5, 100, 45)
	path.QuadraticCurveTo(150, 32.5, 100, 20)

	// I
	path.MoveTo(140, 20)
	path.LineTo(140, 70)
	path.LineTo(150, 70)
	path.LineTo(150, 20)

	// T
	path.MoveTo(160, 20)
	path.LineTo(160, 30)
	path.LineTo(180, 30)
	path.LineTo(180, 70)
	path.LineTo(190, 70)
	path.LineTo(190, 30)
	path.LineTo(210, 30)
	path.LineTo(210, 20)

	// E
	path.MoveTo(220, 20)
	path.LineTo(220, 70)
	path.LineTo(270, 70)
	path.LineTo(270, 60)
	path.LineTo(230, 60)
	path.LineTo(230, 50)
	path.LineTo(270, 50)
	path.LineTo(270, 40)
	path.LineTo(230, 40)
	path.LineTo(230, 30)
	path.LineTo(270, 30)
	path.LineTo(270, 20)

	// N
	path.MoveTo(280, 20)
	path.LineTo(280, 70)
	path.LineTo(290, 70)
	path.LineTo(290, 35)
	path.LineTo(320, 70)
	path.LineTo(330, 70)
	path.LineTo(330, 20)
	path.LineTo(320, 20)
	path.LineTo(320, 55)
	path.LineTo(290, 20)

	path.Fill(screen, color.White)
}

func drawEbitenLogo(screen *ebiten.Image, x, y int) {
	const unit = 16

	var path vector.Path
	xf, yf := float32(x), float32(y)

	path.MoveTo(xf, yf+4*unit)
	path.LineTo(xf, yf+6*unit)
	path.LineTo(xf+2*unit, yf+6*unit)
	path.LineTo(xf+2*unit, yf+5*unit)
	path.LineTo(xf+3*unit, yf+5*unit)
	path.LineTo(xf+3*unit, yf+4*unit)
	path.LineTo(xf+4*unit, yf+4*unit)
	path.LineTo(xf+4*unit, yf+2*unit)
	path.LineTo(xf+6*unit, yf+2*unit)
	path.LineTo(xf+6*unit, yf+1*unit)
	path.LineTo(xf+5*unit, yf+1*unit)
	path.LineTo(xf+5*unit, yf)
	path.LineTo(xf+4*unit, yf)
	path.LineTo(xf+4*unit, yf+2*unit)
	path.LineTo(xf+2*unit, yf+2*unit)
	path.LineTo(xf+2*unit, yf+3*unit)
	path.LineTo(xf+unit, yf+3*unit)
	path.LineTo(xf+unit, yf+4*unit)

	path.Fill(screen, color.RGBA{0xdb, 0x56, 0x20, 0xff})
}

var counter = 0

func update(screen *ebiten.Image) error {
	counter++
	if ebiten.IsDrawingSkipped() {
		return nil
	}

	drawEbitenText(screen)
	drawEbitenLogo(screen, 20, 80)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Vector (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
