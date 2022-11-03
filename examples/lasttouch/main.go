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

package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type pos struct {
	x int
	y int
}

type Game struct {
	releasedTouchIDs   []ebiten.TouchID
	lastTouchPositions []pos
}

func (g *Game) Update() error {
	// In general, Append* function allocates a new slice if the given slice doesn't have enough capacity,
	// or otherwise, just extends the length of the given slice.
	//
	// This example passes an empty slice that might have a capacity in order to reduce the chances of slice allocations.
	// You can also pass 'nil' to AppendJustReleasedTouchIDs if you don't care the cost of creating slices.
	// In this case, AppendJustReleasedTouchIDs would always create a new slice.
	g.releasedTouchIDs = inpututil.AppendJustReleasedTouchIDs(g.releasedTouchIDs[:0])

	for _, id := range g.releasedTouchIDs {
		// Get the last position of the touch.
		// ebiten.TouchPosition would not work as the touch has already gone in this tick.
		x, y := inpututil.TouchPositionInPreviousTick(id)
		g.lastTouchPositions = append(g.lastTouchPositions, pos{x: x, y: y})
	}

	const n = 10
	if len(g.lastTouchPositions) > n {
		copy(g.lastTouchPositions, g.lastTouchPositions[len(g.lastTouchPositions)-n:])
		g.lastTouchPositions = g.lastTouchPositions[:n]
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	msg := "Touch the screen and release your finger from it.\n\nLast Positions:\n"
	for _, p := range g.lastTouchPositions {
		msg += fmt.Sprintf("  (%d, %d)\n", p.x, p.y)
	}
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Last Touch Positions (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
