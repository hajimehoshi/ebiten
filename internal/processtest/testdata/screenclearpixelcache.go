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

type Game struct {
	screen *ebiten.Image
	state  int
}

func (g *Game) Update() error {
	switch g.state {
	case 1:
		g.screen.Set(1, 0, color.White)
		g.state++
	case 3:
		g.screen.Set(3, 0, color.White)
		g.state++
	case 5:
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.state {
	case 0:
		g.screen = screen
		screen.Set(0, 0, color.White)
		g.state++
	case 2:
		checkTransparent(screen, 1, 0, "a pixel set before the clear")
		checkTransparent(screen, 0, 0, "a pixel set in the previous frame")
		screen.Set(2, 0, color.White)
		if got, want := screen.At(2, 0).(color.RGBA), (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}); got != want {
			panic(fmt.Sprintf("At(2, 0): got: %v, want: %v", got, want))
		}
		g.state++
	case 4:
		checkTransparent(screen, 2, 0, "a pixel cached in the previous frame")
		checkTransparent(screen, 3, 0, "a pixel set on the cached pixels before the clear")
		g.state++
	}
}

func checkTransparent(screen *ebiten.Image, x, y int, name string) {
	if got, want := screen.At(x, y).(color.RGBA), (color.RGBA{}); got != want {
		panic(fmt.Sprintf("At(%d, %d), %s, must be cleared: got: %v, want: %v", x, y, name, got, want))
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 16, 16
}

func main() {
	ebiten.SetWindowTitle("Test")

	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
