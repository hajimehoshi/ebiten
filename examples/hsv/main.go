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

// +build example jsgo

package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	hue128        = 0
	saturation128 = 128
	value128      = 128

	inverted = false

	prevPressedI = false
	gophersImage *ebiten.Image
)

// clamp clamps v to the range [min, max].
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
	// Adjust HSV values along with the user's input.
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		hue128--
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		hue128++
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		saturation128--
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		saturation128++
	}
	if ebiten.IsKeyPressed(ebiten.KeyZ) {
		value128--
	}
	if ebiten.IsKeyPressed(ebiten.KeyX) {
		value128++
	}

	hue128 = clamp(hue128, -256, 256)
	saturation128 = clamp(saturation128, 0, 256)
	value128 = clamp(value128, 0, 256)

	pressedI := ebiten.IsKeyPressed(ebiten.KeyI)
	if pressedI && !prevPressedI {
		inverted = !inverted
	}
	prevPressedI = pressedI

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	// Center the image on the screen.
	w, h := gophersImage.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth-w)/2, float64(screenHeight-h)/2)

	// Change HSV.
	hue := float64(hue128) * 2 * math.Pi / 128
	saturation := float64(saturation128) / 128
	value := float64(value128) / 128
	op.ColorM.ChangeHSV(hue, saturation, value)

	// Invert the color.
	if inverted {
		op.ColorM.Scale(-1, -1, -1, 1)
		op.ColorM.Translate(1, 1, 1, 0)
	}

	screen.DrawImage(gophersImage, op)

	// Draw the text of the current status.
	msgInverted := "false"
	if inverted {
		msgInverted = "true"
	}
	msg := fmt.Sprintf(`Hue:        %0.2f [Q][W]
Saturation: %0.2f [A][S]
Value:      %0.2f [Z][X]
Inverted:   %s [I]`, hue, saturation, value, msgInverted)
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	// Decode image from a byte slice instead of a file so that
	// this example works in any working directory.
	// If you want to use a file, there are some options:
	// 1) Use os.Open and pass the file to the image decoder.
	//    This is a very regular way, but doesn't work on browsers.
	// 2) Use ebitenutil.OpenFile and pass the file to the image decoder.
	//    This works even on browsers.
	// 3) Use ebitenutil.NewImageFromFile to create an ebiten.Image directly from a file.
	//    This also works on browsers.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "HSV (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
