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

//go:build ebitenginevm

// This is a guest fixture for vmhost's tests. It is an ordinary Ebitengine program: it just calls
// RunGame. Built with -tags ebitenginevm and run with EBITENGINE_VM_ENDPOINT set, RunGame dials that
// host and serves it, instead of opening a window. It draws a sprite at the cursor so the host's render test
// can assert exact pixels; see vmhost/render_test.go.
package main

import (
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

type game struct {
	sprite *ebiten.Image
	tick   int
}

func (g *game) Update() error {
	g.tick++
	x, y := ebiten.CursorPosition()
	// The guest's state lives in the guest process; log it to stderr (inherited by the host) so the
	// cross-process driving is observable.
	fmt.Fprintf(os.Stderr, "[guest] tick=%d IsKeyPressed(A)=%t cursor=(%d,%d)\n",
		g.tick, ebiten.IsKeyPressed(ebiten.KeyA), x, y)
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	if g.sprite == nil {
		g.sprite = ebiten.NewImage(16, 16)
		g.sprite.Fill(color.White)
	}
	screen.Fill(color.RGBA{R: 0x20, G: 0x40, B: 0x80, A: 0xff})

	// Draw the sprite at the (host-injected) cursor position.
	x, y := ebiten.CursorPosition()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(g.sprite, op)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
