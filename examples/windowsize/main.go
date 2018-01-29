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

// +build example

package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	initScreenWidth  = 320
	initScreenHeight = 240
	initScreenScale  = 2
)

var (
	gophersImage *ebiten.Image
	keyStates    = map[ebiten.Key]int{
		ebiten.KeyUp:    0,
		ebiten.KeyDown:  0,
		ebiten.KeyLeft:  0,
		ebiten.KeyRight: 0,
		ebiten.KeyS:     0,
		ebiten.KeyF:     0,
		ebiten.KeyB:     0,
		ebiten.KeyC:     0,
		ebiten.KeyI:     0,
	}
	count = 0
)

func createRandomIconImage() image.Image {
	const size = 32

	r := uint8(rand.Intn(0x100))
	g := uint8(rand.Intn(0x100))
	b := uint8(rand.Intn(0x100))
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			img.Pix[j*img.Stride+4*i] = r
			img.Pix[j*img.Stride+4*i+1] = g
			img.Pix[j*img.Stride+4*i+2] = b
			img.Pix[j*img.Stride+4*i+3] = uint8(float64(i+j) / float64(2*size) * 0xff)
		}
	}

	return img
}

func update(screen *ebiten.Image) error {
	for key := range keyStates {
		if !ebiten.IsKeyPressed(key) {
			keyStates[key] = 0
			continue
		}
		keyStates[key]++
	}
	screenScale := ebiten.ScreenScale()
	d := int(32 / screenScale)
	screenWidth, screenHeight := screen.Size()
	fullscreen := ebiten.IsFullscreen()
	runnableInBackground := ebiten.IsRunnableInBackground()
	cursorVisible := ebiten.IsCursorVisible()

	if keyStates[ebiten.KeyUp] == 1 {
		screenHeight += d
	}
	if keyStates[ebiten.KeyDown] == 1 {
		if 16 < screenHeight && d < screenHeight {
			screenHeight -= d
		}
	}
	if keyStates[ebiten.KeyLeft] == 1 {
		if 16 < screenWidth && d < screenWidth {
			screenWidth -= d
		}
	}
	if keyStates[ebiten.KeyRight] == 1 {
		screenWidth += d
	}
	if keyStates[ebiten.KeyS] == 1 {
		switch screenScale {
		case 1:
			screenScale = 1.5
		case 1.5:
			screenScale = 2
		case 2:
			screenScale = 1
		default:
			panic("not reached")
		}
	}
	if keyStates[ebiten.KeyF] == 1 {
		fullscreen = !fullscreen
	}
	if keyStates[ebiten.KeyB] == 1 {
		runnableInBackground = !runnableInBackground
	}
	if keyStates[ebiten.KeyC] == 1 {
		cursorVisible = !cursorVisible
	}
	ebiten.SetScreenSize(screenWidth, screenHeight)
	ebiten.SetScreenScale(screenScale)
	ebiten.SetFullscreen(fullscreen)
	ebiten.SetRunnableInBackground(runnableInBackground)
	ebiten.SetCursorVisible(cursorVisible)

	if keyStates[ebiten.KeyI] == 1 {
		ebiten.SetWindowIcon([]image.Image{createRandomIconImage()})
	}

	count++

	if ebiten.IsRunningSlowly() {
		return nil
	}

	screen.Fill(color.RGBA{0x80, 0x80, 0xc0, 0xff})
	w, h := gophersImage.Size()
	w2, h2 := screen.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(-w+w2)/2, float64(-h+h2)/2)
	dx := math.Cos(2*math.Pi*float64(count)/360) * 10
	dy := math.Sin(2*math.Pi*float64(count)/360) * 10
	op.GeoM.Translate(dx, dy)
	screen.DrawImage(gophersImage, op)

	x, y := ebiten.CursorPosition()
	msg := fmt.Sprintf(`Press arrow keys to change the window size
Press S key to change the window scale
Press F key to switch the fullscreen state
Press B key to switch the run-in-background state
Press C key to switch the cursor visibility
Press I key to change the window icon
Cursor: (%d, %d)
FPS: %0.2f`, x, y, ebiten.CurrentFPS())
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	fmt.Printf("Device scale factor: %0.2f\n", ebiten.DeviceScaleFactor())

	var err error
	gophersImage, _, err = ebitenutil.NewImageFromFile("_resources/images/gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowIcon([]image.Image{createRandomIconImage()})

	if err := ebiten.Run(update, initScreenWidth, initScreenHeight, initScreenScale, "Window Size (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
