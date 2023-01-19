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

// This example is just to check if Ebitengine can draw fine checker pattern evenly.
// If there is something wrong in the implementation, the result might include
// uneven patterns (#459).
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth     = 640
	screenHeight    = 480
	initScreenScale = 1
)

var (
	dots       []byte
	dotsWidth  int
	dotsHeight int
)

func getDots(width, height int) []byte {
	if dotsWidth == width && dotsHeight == height {
		return dots
	}
	dotsWidth = width
	dotsHeight = height
	dots = make([]byte, width*height*4)
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

type game struct {
	scale float64
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *game) Update() error {
	fullscreen := ebiten.IsFullscreen()

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		switch g.scale {
		case 0.5:
			g.scale = 1
		case 1:
			g.scale = 1.5
		case 1.5:
			g.scale = 2
		case 2:
			g.scale = 0.5
		default:
			panic("not reached")
		}
		ebiten.SetWindowSize(int(screenWidth*g.scale), int(screenHeight*g.scale))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		fullscreen = !fullscreen
		ebiten.SetFullscreen(fullscreen)
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	s := screen.Bounds().Size()
	screen.WritePixels(getDots(s.X, s.Y))
}

func main() {
	g := &game{
		scale: initScreenScale,
	}
	ebiten.SetWindowSize(screenWidth*initScreenScale, screenHeight*initScreenScale)
	ebiten.SetWindowTitle("Moire (Ebitengine Demo)")
	ebiten.SetWindowResizable(true)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
