// Copyright 2015 Hajime Hoshi
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
	"image/color"
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	initScreenWidth  = 320
	initScreenHeight = 240
	initScreenScale  = 2
)

var (
	gophersImage *ebiten.Image
	screenWidth  = initScreenWidth
	screenHeight = initScreenHeight
	screenScale  = initScreenScale
	keyStates    = map[ebiten.Key]int{
		ebiten.KeyUp:    0,
		ebiten.KeyDown:  0,
		ebiten.KeyLeft:  0,
		ebiten.KeyRight: 0,
		ebiten.KeyS:     0,
	}
)

func update(screen *ebiten.Image) error {
	for key, _ := range keyStates {
		if !ebiten.IsKeyPressed(key) {
			keyStates[key] = 0
			continue
		}
		keyStates[key]++
	}
	d := 0
	switch screenScale {
	case 1:
		d = 32
	case 2:
		d = 16
	}

	if keyStates[ebiten.KeyUp] == 1 {
		screenHeight += d
	}
	if keyStates[ebiten.KeyDown] == 1 {
		if 16 < screenHeight {
			screenHeight -= d
		}
	}
	if keyStates[ebiten.KeyLeft] == 1 {
		if 16 < screenWidth {
			screenWidth -= d
		}
	}
	if keyStates[ebiten.KeyRight] == 1 {
		screenWidth += d
	}
	if keyStates[ebiten.KeyS] == 1 {
		screenScale = 3 - screenScale // Swap 1 and 2
	}
	ebiten.SetScreenSize(screenWidth, screenHeight)
	ebiten.SetScreenScale(screenScale)

	screen.Fill(color.RGBA{0x80, 0x80, 0xc0, 0xff})
	w, h := gophersImage.Size()
	w2, h2 := screen.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(-w+w2)/2, float64(-h+h2)/2)
	if err := screen.DrawImage(gophersImage, op); err != nil {
		return err
	}

	x, y := ebiten.CursorPosition()
	msg := fmt.Sprintf(`Press arrow keys to change the window size
Press S key to change the window scale
Cursor: (%d, %d)
FPS: %0.2f`, x, y, ebiten.CurrentFPS())
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	var err error
	gophersImage, _, err = ebitenutil.NewImageFromFile("images/gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, initScreenWidth, initScreenHeight, initScreenScale, "Window Size (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
