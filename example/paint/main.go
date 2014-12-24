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
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"image/color"
	"log"
	"math"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	count              int
	brushRenderTarget  *ebiten.Image
	canvasRenderTarget *ebiten.Image
)

func Update(screen *ebiten.Image) error {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		count++
	}

	mx, my := ebiten.CursorPosition()

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		clr := ebiten.ScaleColor(1.0, 0.25, 0.25, 1.0)
		theta := 2.0 * math.Pi * float64(count%60) / 60.0
		clr.Concat(ebiten.RotateHue(theta))
		op := ebiten.At(mx, my)
		op.ColorMatrix = &clr
		canvasRenderTarget.DrawImage(brushRenderTarget, op)
	}

	screen.DrawImage(canvasRenderTarget, nil)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("(%d, %d)", mx, my))
	return nil
}

func main() {
	var err error
	brushRenderTarget, err = ebiten.NewImage(4, 4, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	brushRenderTarget.Fill(color.White)

	canvasRenderTarget, err = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	canvasRenderTarget.Fill(color.White)

	if err := ebiten.Run(Update, screenWidth, screenHeight, 2, "Paint (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
