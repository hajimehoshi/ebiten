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

//go:build ignore

package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var (
	baseImage    *ebiten.Image
	derivedImage *ebiten.Image
)

func init() {
	const (
		w = 36
		h = 40
	)

	baseImage = ebiten.NewImage(w, h)
	derivedImage = ebiten.NewImage(w, h)

	baseImage.Fill(color.White)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			baseImage.Set(i, j, color.Black)
		}
	}
	derivedImage.DrawImage(baseImage, nil)
}

type Game struct {
	count int
}

func (g *Game) Update() error {
	g.count++
	if g.count == 16 {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)

	screen.DrawImage(derivedImage, nil)
	if g.count >= 8 {
		if got, want := screen.At(0, 0), (color.RGBA{0, 0, 0, 0xff}); got != want {
			panic(fmt.Sprintf("got: %v, want: %v", got, want))
		}
	}

	// The blow 3 line matters to reproduce #2154.
	mx, my := ebiten.CursorPosition()
	msg := fmt.Sprintf("TPS: %.01f; FPS: %.01f; cursor: (%d, %d)", ebiten.ActualTPS(), ebiten.ActualFPS(), mx, my)
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}

func main() {
	ebiten.SetWindowTitle("Test")

	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
