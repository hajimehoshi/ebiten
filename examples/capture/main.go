// Copyright 2023 The Ebitengine Authors
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
	"log"
	"math"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var pointerImage = ebiten.NewImage(8, 8)

func init() {
	pointerImage.Fill(color.RGBA{0xff, 0, 0, 0xff})
}

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	mouseX int
	mouseY int
	x      int
	y      int
}

func (g *Game) Update() error {
	if ebiten.CursorMode() == ebiten.CursorModeCaptured {
		cursorX, cursorY := ebiten.CursorPosition()

		if g.mouseX == math.MinInt32 && g.mouseY == math.MinInt32 {
			// Initialize first position to establish delta.
			if cursorX != 0 && cursorY != 0 {
				g.mouseX, g.mouseY = cursorX, cursorY
			}
		} else {
			deltaX, deltaY := cursorX-g.mouseX, cursorY-g.mouseY
			g.mouseX, g.mouseY = cursorX, cursorY

			if deltaX != 0 {
				g.x += deltaX
			}

			if deltaY != 0 {
				g.y += deltaY
			}
		}

		// Constrain red dot within screen view.
		if g.x < 0 {
			g.x = 0
		} else if g.x > screenWidth-8 {
			g.x = screenWidth - 8
		}

		if g.y < 0 {
			g.y = 0
		} else if g.y > screenHeight-8 {
			g.y = screenHeight - 8
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if ebiten.CursorMode() == ebiten.CursorModeCaptured {
			// Release mouse cursor capture.
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
		} else {
			// Recapture mouse cursor.
			ebiten.SetCursorMode(ebiten.CursorModeCaptured)

			// Reset mouse cursor position for its return.
			g.mouseX, g.mouseY = math.MinInt32, math.MinInt32
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(g.x), float64(g.y))
	screen.DrawImage(pointerImage, op)

	var message string
	if ebiten.CursorMode() == ebiten.CursorModeCaptured {
		message = fmt.Sprintf("Move the red point with mouse captured\nPress Space to release mouse capture\n(%d, %d)", g.x, g.y)
	} else {
		message = fmt.Sprintf("The red point can only move when mouse captured\nPress Space to capture mouse\n(%d, %d)", g.x, g.y)
	}
	ebitenutil.DebugPrint(screen, message)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowTitle("Mouse Capture (Ebitengine Demo)")
	ebiten.SetWindowSize(screenWidth, screenHeight)

	if runtime.GOOS == "js" {
		// Web browsers allow cursor mode capture only with user interaction.
		// Start without cursor captured with message indicating how to capture.
	} else {
		ebiten.SetCursorMode(ebiten.CursorModeCaptured)
	}

	g := &Game{
		mouseX: math.MinInt32,
		mouseY: math.MinInt32,
		x:      screenWidth / 2,
		y:      screenHeight / 2,
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
