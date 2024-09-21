// Copyright 2024 The Ebitengine Authors
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
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	count int
}

func (g *Game) Update() error {
	if g.count > 0 {
		g.count--
		if g.count == 0 {
			ebiten.RequestAttention()
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.count = ebiten.TPS() * 3
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.count > 0 {
		c := int(math.Ceil(float64(g.count) / float64(ebiten.TPS())))
		msg := fmt.Sprintf("Requesting attention in %d seconds...", c)
		if ebiten.IsFocused() {
			msg += "\nPlease unfocus this window to see the effect."
		}
		ebitenutil.DebugPrint(screen, msg)
		return
	}
	ebitenutil.DebugPrint(screen, "Press R to request attention after 3 seconds.")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Request Attention (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
