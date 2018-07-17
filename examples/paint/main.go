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

// +build example jsgo

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

func init() {
	const (
		a0 = 0x40
		a1 = 0xc0
		a2 = 0xff
	)
	pixels := []uint8{
		a0, a1, a1, a0,
		a1, a2, a2, a1,
		a1, a2, a2, a1,
		a0, a1, a1, a0,
	}
	brushImage, _ = ebiten.NewImageFromImage(&image.Alpha{
		Pix:    pixels,
		Stride: 4,
		Rect:   image.Rect(0, 0, 4, 4),
	}, ebiten.FilterDefault)

	canvasImage, _ = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterDefault)
	canvasImage.Fill(color.White)
}

// paint draws the brush on the given canvas image at the position (x, y).
func paint(canvas *ebiten.Image, x, y int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	// Scale the color and rotate the hue so that colors vary on each frame.
	op.ColorM.Scale(1.0, 0.50, 0.125, 1.0)
	tps := ebiten.MaxTPS()
	theta := 2.0 * math.Pi * float64(count%tps) / float64(tps)
	op.ColorM.RotateHue(theta)
	canvas.DrawImage(brushImage, op)
}

func update(screen *ebiten.Image) error {
	drawn := false

	// Paint the brush by mouse dragging
	mx, my := ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		paint(canvasImage, mx, my)
		drawn = true
	}

	// Paint the brush by touches
	for _, t := range ebiten.TouchIDs() {
		x, y := ebiten.TouchPosition(t)
		paint(canvasImage, x, y)
		drawn = true
	}
	if drawn {
		count++
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	screen.DrawImage(canvasImage, nil)

	msg := fmt.Sprintf("(%d, %d)", mx, my)
	for _, t := range ebiten.TouchIDs() {
		x, y := ebiten.TouchPosition(t)
		msg += fmt.Sprintf("\n(%d, %d) touch %d", x, y, t)
	}
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Paint (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
