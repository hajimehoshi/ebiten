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

// +build example jsgo

// This example is just to check if Ebiten can draw fine checker pattern evenly.
// If there is something wrong in the implementation, the result might include
// uneven patterns (#459).
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/inpututil"
)

const (
	screenWidth     = 640
	screenHeight    = 480
	initScreenScale = 1
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
	screenScale := ebiten.ScreenScale()
	fullscreen := ebiten.IsFullscreen()

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
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
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		fullscreen = !fullscreen
	}
	ebiten.SetScreenScale(screenScale)
	ebiten.SetFullscreen(fullscreen)

	if ebiten.IsRunningSlowly() {
		return nil
	}

	screen.ReplacePixels(getDots(screen.Size()))
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, initScreenScale, "Moire (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
