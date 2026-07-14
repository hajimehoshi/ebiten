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

// This guest exercises cursor-shape forwarding: it keeps the default shape, switches to a pointer at
// tick 3, and back to the default at tick 5. The host asserts GuestSession.CursorShape tracks each
// change; see vmhost's cursor-shape test.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type game struct {
	tick int
}

func (g *game) Update() error {
	g.tick++
	switch g.tick {
	case 3:
		ebiten.SetCursorShape(ebiten.CursorShapePointer)
	case 5:
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
