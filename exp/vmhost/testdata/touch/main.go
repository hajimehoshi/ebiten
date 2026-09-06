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

// This is a guest that verifies the touch events the host forwards. The host drives a fixed sequence
// of press, move, and release events; during each tick this reads the current touches through the
// public ebiten API and compares them against the expectation for that tick, filling the screen green
// when they match and red otherwise (logging each mismatch), so the outcome is observable in the
// rendered screen.
//
// It is launched by a host; see vmhost's touch test.
package main

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type wantTouch struct {
	id   ebiten.TouchID
	x, y float64
}

// wantByTick is the expected set of touches per tick; keep it in sync with the host's injected events.
// The guest fills the whole window, so the device-independent positions the host injects map to the
// same logical positions regardless of the device scale factor. Tick 0: two touches begin. Tick 1:
// touch 1 moves and touch 2 ends. The positions have fractional parts well away from an integer
// boundary, so that both the truncated and the high-precision accessors are pinned down.
var wantByTick = [][]wantTouch{
	{{id: 1, x: 3.75, y: 4.5}, {id: 2, x: 30.25, y: 20.5}},
	{{id: 1, x: 5.5, y: 6.25}},
}

// eps is the tolerance for a position, which is scaled by the device scale factor and back and so is
// not necessarily bit-exact.
const eps = 1e-6

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
	want := wantByTick[g.tick]

	ok := true
	fail := func(format string, args ...any) {
		log.Printf("tick %d: "+format, append([]any{g.tick}, args...)...)
		ok = false
	}

	got := map[ebiten.TouchID][2]float64{}
	for _, id := range ebiten.AppendTouchIDs(nil) {
		x, y := ebiten.TouchPositionF(id)
		got[id] = [2]float64{x, y}
	}
	if len(got) != len(want) {
		fail("touch count = %d; want %d", len(got), len(want))
	}
	for _, w := range want {
		pos, present := got[w.id]
		if !present {
			fail("touch %d missing", w.id)
			continue
		}
		if math.Abs(pos[0]-w.x) > eps || math.Abs(pos[1]-w.y) > eps {
			fail("TouchPositionF(%d) = (%v, %v); want (%v, %v)", w.id, pos[0], pos[1], w.x, w.y)
		}
		if x, y := ebiten.TouchPosition(w.id); x != int(w.x) || y != int(w.y) {
			fail("TouchPosition(%d) = (%d, %d); want (%d, %d)", w.id, x, y, int(w.x), int(w.y))
		}
	}

	// The previous tick's touches are still readable, including the ones released since.
	if g.tick > 0 {
		for _, w := range wantByTick[g.tick-1] {
			x, y := inpututil.TouchPositionFInPreviousTick(w.id)
			if math.Abs(x-w.x) > eps || math.Abs(y-w.y) > eps {
				fail("TouchPositionFInPreviousTick(%d) = (%v, %v); want (%v, %v)", w.id, x, y, w.x, w.y)
			}
			if x, y := inpututil.TouchPositionInPreviousTick(w.id); x != int(w.x) || y != int(w.y) {
				fail("TouchPositionInPreviousTick(%d) = (%d, %d); want (%d, %d)", w.id, x, y, int(w.x), int(w.y))
			}
		}
	}

	return ok
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
