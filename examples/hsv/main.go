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
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
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

type Game struct {
	hue128        int
	saturation128 int
	value128      int

	inverted bool
}

func NewGame() *Game {
	return &Game{
		saturation128: 128,
		value128:      128,
	}
}

func (g *Game) Update() error {
	// Adjust HSV values along with the user's input.
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		g.hue128--
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.hue128++
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.saturation128--
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.saturation128++
	}
	if ebiten.IsKeyPressed(ebiten.KeyZ) {
		g.value128--
	}
	if ebiten.IsKeyPressed(ebiten.KeyX) {
		g.value128++
	}

	g.hue128 = clamp(g.hue128, -256, 256)
	g.saturation128 = clamp(g.saturation128, 0, 256)
	g.value128 = clamp(g.value128, 0, 256)

	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		g.inverted = !g.inverted
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Center the image on the screen.
	w, h := gophersImage.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(screenWidth)/2, float64(screenHeight)/2)

	// Change HSV.
	hue := float64(g.hue128) * 2 * math.Pi / 128
	saturation := float64(g.saturation128) / 128
	value := float64(g.value128) / 128
	op.ColorM.ChangeHSV(hue, saturation, value)

	// Invert the color.
	if g.inverted {
		op.ColorM.Scale(-1, -1, -1, 1)
		op.ColorM.Translate(1, 1, 1, 0)
	}

	screen.DrawImage(gophersImage, op)

	// Draw the text of the current status.
	msgInverted := "false"
	if g.inverted {
		msgInverted = "true"
	}
	msg := fmt.Sprintf(`Hue:        %0.2f [Q][W]
Saturation: %0.2f [A][S]
Value:      %0.2f [Z][X]
Inverted:   %s [I]`, hue, saturation, value, msgInverted)
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
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
	gophersImage = ebiten.NewImageFromImage(img)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("HSV (Ebiten Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
