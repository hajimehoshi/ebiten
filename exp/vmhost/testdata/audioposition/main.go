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

// This guest exercises the player position under the virtual audio device: it plays an infinite
// source at tick 1 and asserts the position is exactly what the host has pulled — 0 through tick 4,
// 100ms at tick 5. A failed assertion surfaces as the Update error.
//
// It is launched by a host; see vmhost's audio position test.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sampleRate = 48000

// silence endlessly yields zero samples.
type silence struct{}

func (silence) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

type game struct {
	tick   int
	player *audio.Player
}

func (g *game) Update() error {
	g.tick++
	switch {
	case g.tick == 1:
		g.player.Play()
	case g.tick <= 4:
		if pos := g.player.Position(); pos != 0 {
			return fmt.Errorf("Position() at tick %d = %v; want 0: the host has pulled nothing", g.tick, pos)
		}
	case g.tick == 5:
		if pos := g.player.Position(); pos != 100*time.Millisecond {
			return fmt.Errorf("Position() at tick 5 = %v; want exactly 100ms: the host has pulled 100ms of samples", pos)
		}
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	ctx := audio.NewContext(sampleRate)
	player, err := ctx.NewPlayerF32(silence{})
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.RunGame(&game{player: player}); err != nil {
		log.Fatal(err)
	}
}
