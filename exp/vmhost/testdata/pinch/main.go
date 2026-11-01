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

//go:build ebitenginevmguest

// This is a guest that verifies the pinch gestures the host forwards. The host injects a fixed
// sequence of gestures; during each tick this reads the amount through the public ebiten API and
// compares it against the expectation for that tick, filling the screen green when it matches and
// red otherwise (logging each mismatch), so the outcome is observable in the rendered screen.
//
// It is launched by a host; see vmhost's pinch test.
package main

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// wantByTick is the expected pinch amount per tick; keep it in sync with the host's injected gestures.
// Tick 0: two gestures, whose amounts add up. Tick 1: none, so the amount of tick 0 must be gone.
// Tick 2: a gesture that shrinks.
var wantByTick = []float64{0.25, 0, -0.125}

// eps is the tolerance for an amount that is the sum of several injected ones.
const eps = 1e-9

type game struct {
	tick int
	ok   bool
}

func (g *game) Update() error {
	g.ok = g.check()
	g.tick++
	return nil
}

func (g *game) check() bool {
	if g.tick >= len(wantByTick) {
		// The host runs exactly len(wantByTick) ticks; keep the last result for any extra tick.
		return g.ok
	}
	if got, want := ebiten.PinchDelta(), wantByTick[g.tick]; math.Abs(got-want) > eps {
		log.Printf("tick %d: PinchDelta() = %v; want %v", g.tick, got, want)
		return false
	}
	return true
}

func (g *game) Draw(screen *ebiten.Image) {
	if g.ok {
		screen.Fill(color.RGBA{R: 0x00, G: 0xff, B: 0x00, A: 0xff})
		return
	}
	screen.Fill(color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff})
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
