// Copyright 2022 The Ebitengine Authors
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
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	cx = 32
	cy = 32
)

type Game struct {
	offscreen *ebiten.Image
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.offscreen == nil {
		s := screen.Bounds().Size()
		g.offscreen = ebiten.NewImage(s.X, s.Y)
	}

	// Use various sub-images as rendering destination.
	// This is a proof-of-concept of efficient rendering with sub-images (#2232).
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	cw := sw / cx
	ch := sh / cy
	for j := 0; j < cy; j++ {
		for i := 0; i < cx; i++ {
			r := image.Rect(cw*i, ch*j, cw*(i+1), ch*(j+1))
			img := g.offscreen.SubImage(r).(*ebiten.Image)

			// Rendering onto a sub image should be efficient.
			clr := color.RGBA{byte(0xff * float64(i) / cx), byte(0xff * float64(j) / cx), 0, 0xff}
			img.Fill(clr)
		}
	}

	screen.DrawImage(g.offscreen, nil)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f, TPS: %0.2f", ebiten.ActualFPS(), ebiten.ActualTPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}

func main() {
	ebiten.SetWindowTitle("Sub-images as rendering destinations (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
