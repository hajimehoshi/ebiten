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

package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	count int
}

func (g *Game) Update() error {
	g.count++
	g.count %= 240
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	cf := float32(g.count)
	vector.StrokeLine(screen, 100, 100, 300, 100, 1, color.RGBA{0xff, 0xff, 0xff, 0xff}, true)
	vector.StrokeLine(screen, 50, 150, 50, 350, 1, color.RGBA{0xff, 0xff, 0x00, 0xff}, true)
	vector.StrokeLine(screen, 50, 100+cf, 200+cf, 250, 4, color.RGBA{0x00, 0xff, 0xff, 0xff}, true)

	vector.DrawFilledRect(screen, 50+cf, 50+cf, 100+cf, 100+cf, color.RGBA{0x80, 0x80, 0x80, 0xc0}, true)
	vector.StrokeRect(screen, 300-cf, 50, 120, 120, 10+cf/4, color.RGBA{0x00, 0x80, 0x00, 0xff}, true)

	vector.DrawFilledCircle(screen, 400, 400, 100, color.RGBA{0x80, 0x00, 0x80, 0x80}, true)
	vector.StrokeCircle(screen, 400, 400, 10+cf, 10+cf/2, color.RGBA{0xff, 0x80, 0xff, 0xff}, true)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Shapes (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
