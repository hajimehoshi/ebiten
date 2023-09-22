// Copyright 2023 The Ebitengine Authors
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
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct {
	count int
	x     int
	y     int
}

func delta(x0, x1 int) int {
	if x0 < x1 {
		return x1 - x0
	}
	return x0 - x1
}

func (g *Game) Update() error {
	switch g.count {
	case 0:
		g.x, g.y = ebiten.CursorPosition()
	case 20:
		ebiten.SetFullscreen(true)
	case 40:
		ebiten.SetFullscreen(false)
	case 60:
		return ebiten.Termination
	default:
		// Allow some numerical errors (±1).
		if x, y := ebiten.CursorPosition(); delta(g.x, x) > 1 || delta(g.y, y) > 1 {
			return fmt.Errorf("cursor position changed: got: (%d, %d), want: (%d, %d)", x, y, g.x, g.y)
		}
	}
	g.count++
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	x, y := ebiten.CursorPosition()
	ebitenutil.DebugPrint(screen, fmt.Sprintf("%d, %d", x, y))
}

func (g *Game) Layout(width, height int) (int, int) {
	// Using a fixed size matters.
	// If a window size is changed or fullscreened, the cursor position calculation considers the current screen scale, and
	// a fixed size changes the scale.
	return 320, 240
}

func main() {
	// Mouse is not supported on mobiles.
	// Capturing a cursor requires a user gesture on browsers.
	// Skip the test in these environments.
	if runtime.GOOS == "android" || runtime.GOOS == "ios" || runtime.GOOS == "js" {
		return
	}

	// This test is flaky on Windows (especially on GitHub Actions). Skip this.
	if runtime.GOOS == "windows" {
		return
	}

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
