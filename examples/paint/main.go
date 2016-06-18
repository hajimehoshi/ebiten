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
	"image"
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
	count       int
	brushImage  *ebiten.Image
	canvasImage *ebiten.Image
)

func paint(screen *ebiten.Image, x, y int) error {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorM.Scale(1.0, 0.50, 0.125, 1.0)
	theta := 2.0 * math.Pi * float64(count%60) / ebiten.FPS
	op.ColorM.RotateHue(theta)
	if err := canvasImage.DrawImage(brushImage, op); err != nil {
		return err
	}
	return nil
}

func update(screen *ebiten.Image) error {
	drawn := false
	mx, my := ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if err := paint(screen, mx, my); err != nil {
			return err
		}
		drawn = true
	}
	for _, t := range ebiten.Touches() {
		x, y := t.Position()
		if err := paint(screen, x, y); err != nil {
			return err
		}
		drawn = true
	}
	if drawn {
		count++
	}

	if err := screen.DrawImage(canvasImage, nil); err != nil {
		return err
	}

	msg := fmt.Sprintf("(%d, %d)", mx, my)
	for _, t := range ebiten.Touches() {
		x, y := t.Position()
		msg += fmt.Sprintf("\n(%d, %d) touch %d", x, y, t.ID())
	}
	if err := ebitenutil.DebugPrint(screen, msg); err != nil {
		return err
	}
	return nil
}

func main() {
	var err error
	const a0, a1, a2 = 0x40, 0xc0, 0xff
	pixels := []uint8{
		a0, a1, a1, a0,
		a1, a2, a2, a1,
		a1, a2, a2, a1,
		a0, a1, a1, a0,
	}
	brushImage, err = ebiten.NewImageFromImage(&image.Alpha{
		Pix:    pixels,
		Stride: 4,
		Rect:   image.Rect(0, 0, 4, 4),
	}, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}

	canvasImage, err = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	canvasImage.Fill(color.White)

	if err := ebiten.Run(update, screenWidth, screenHeight, 1.5, "Paint (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
