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

	"github.com/hajimehoshi/ebiten/v2"
)

type wantTouch struct {
	id   ebiten.TouchID
	x, y int
}

// wantByTick is the expected set of touches per tick; keep it in sync with the host's injected events.
// The guest fills the whole window, so the device-independent positions the host injects map to the
// same logical positions regardless of the device scale factor. Tick 0: two touches begin. Tick 1:
// touch 1 moves and touch 2 ends.
var wantByTick = [][]wantTouch{
	{{id: 1, x: 3, y: 4}, {id: 2, x: 30, y: 20}},
	{{id: 1, x: 5, y: 6}},
}

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

	got := map[ebiten.TouchID][2]int{}
	for _, id := range ebiten.AppendTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		got[id] = [2]int{x, y}
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
		if pos != [2]int{w.x, w.y} {
			fail("TouchPosition(%d) = %v; want [%d %d]", w.id, pos, w.x, w.y)
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
