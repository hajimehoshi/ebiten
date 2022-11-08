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
)

type Game struct {
	drawCount int
}

func (g *Game) Update() error {
	if g.drawCount == 2 {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.drawCount {
	case 0:
		screen.Set(0, 0, color.White)
		g.drawCount++
	case 1:
		if got, want := screen.At(0, 0).(color.RGBA), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
			panic(fmt.Sprintf("got: %v, want: %v", got, want))
		}
		g.drawCount++
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}

func main() {
	ebiten.SetWindowTitle("Test")
	ebiten.SetScreenClearedEveryFrame(false)

	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
