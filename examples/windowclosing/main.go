// Copyright 2021 The Ebiten Authors
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
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	windowClosingHandled bool
}

func (g *Game) Update() error {
	if ebiten.IsWindowBeingClosed() {
		g.windowClosingHandled = true
	}
	if g.windowClosingHandled {
		if inpututil.IsKeyJustPressed(ebiten.KeyY) {
			return ebiten.Termination
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.windowClosingHandled = false
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if !g.windowClosingHandled {
		ebitenutil.DebugPrint(screen, "Try to close this window. This works only on desktops.")
		return
	}
	ebitenutil.DebugPrint(screen, "Do you really want to close this window? [y/n]")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetWindowTitle("Window Closing (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
