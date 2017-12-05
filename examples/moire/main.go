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
	"log"

	"github.com/hajimehoshi/ebiten"
)

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
	}
)

var (
	dots       []uint8
	dotsWidth  int
	dotsHeight int
)

func getDots(width, height int) []uint8 {
	if dotsWidth == width && dotsHeight == height {
		return dots
	}
	dotsWidth = width
	dotsHeight = height
	dots = make([]uint8, width*height*4)
	for j := 0; j < height; j++ {
		for i := 0; i < width; i++ {
			if (i+j)%2 == 0 {
				dots[(i+j*width)*4+0] = 0xff
				dots[(i+j*width)*4+1] = 0xff
				dots[(i+j*width)*4+2] = 0xff
				dots[(i+j*width)*4+3] = 0xff
			}
		}
	}
	return dots
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
	ebiten.SetScreenSize(screenWidth, screenHeight)
	ebiten.SetScreenScale(screenScale)
	ebiten.SetFullscreen(fullscreen)

	if ebiten.IsRunningSlowly() {
		return nil
	}

	screen.ReplacePixels(getDots(screen.Size()))
	return nil
}

func main() {
	if err := ebiten.Run(update, initScreenWidth, initScreenHeight, initScreenScale, "Moire (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
