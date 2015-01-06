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
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(mx), float64(my))
		op.ColorM.Scale(1.0, 0.25, 0.25, 1.0)
		theta := 2.0 * math.Pi * float64(count%60) / 60.0
		op.ColorM.Concat(ebiten.RotateHue(theta))
		if err := canvasRenderTarget.DrawImage(brushRenderTarget, op); err != nil {
			return err
		}
	}

	if err := screen.DrawImage(canvasRenderTarget, nil); err != nil {
		return err
	}

	if err := ebitenutil.DebugPrint(screen, fmt.Sprintf("(%d, %d)", mx, my)); err != nil {
		return err
	}
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
