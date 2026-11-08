// Copyright 2026 The Ebitengine Authors
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

//go:build ignore

package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

const drawCountToCheck = 20

type Game struct {
	drawCount            int
	finalScreenDrawCount int
}

func (g *Game) Update() error {
	if g.drawCount < drawCountToCheck {
		return nil
	}
	if g.finalScreenDrawCount < drawCountToCheck/2 {
		panic(fmt.Sprintf("DrawFinalScreen must be called after Draw modifying the screen with Set: called %d times for %d draws", g.finalScreenDrawCount, g.drawCount))
	}
	return ebiten.Termination
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Set(g.drawCount%screen.Bounds().Dx(), 0, color.White)
	g.drawCount++
}

func (g *Game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	g.finalScreenDrawCount++
	ebiten.DefaultDrawFinalScreen(screen, offscreen, geoM)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 16, 16
}

func main() {
	ebiten.SetWindowTitle("Test")
	ebiten.SetScreenClearedEveryFrame(false)

	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
