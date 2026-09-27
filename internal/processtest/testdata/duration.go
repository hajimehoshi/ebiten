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

package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type game struct {
	updates      int
	pausedFrames int
	last         ebiten.Duration
}

func (g *game) Update() error {
	now := ebiten.DurationTime()
	switch {
	case g.updates <= 3:
		want := ebiten.Duration(g.updates) * ebiten.DurationSecond / 60
		if now != want {
			return fmt.Errorf("update %d: got %d, want %d", g.updates, now, want)
		}
	case g.updates <= 6:
		want := 3*ebiten.DurationSecond/60 + ebiten.Duration(g.updates-3)*ebiten.DurationSecond/120
		if now != want {
			return fmt.Errorf("update %d: got %d, want %d", g.updates, now, want)
		}
	default:
		if ebiten.TPS() == 0 {
			if now != g.last {
				return fmt.Errorf("forced update while paused: got %d, want %d", now, g.last)
			}
			return nil
		}
		if now < g.last {
			return fmt.Errorf("time moved backwards: got %d, previous %d", now, g.last)
		}
	}
	switch g.updates {
	case 3:
		ebiten.SetTPS(120)
	case 6:
		ebiten.SetTPS(0)
	case 10:
		return ebiten.Termination
	}
	if got := ebiten.DurationTime(); got != now {
		return fmt.Errorf("time changed within Update: got %d, want %d", got, now)
	}
	g.last = now
	g.updates++
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	if got := ebiten.DurationTime(); got != g.last {
		panic(fmt.Sprintf("Draw: got %d, want last Update time %d", got, g.last))
	}
	if ebiten.TPS() == 0 {
		g.pausedFrames++
		switch g.pausedFrames {
		case 1:
			ebiten.SetWindowSize(321, 241)
		case 3:
			ebiten.SetTPS(ebiten.SyncWithFPS)
		}
	}
}

func (*game) Layout(width, height int) (int, int) {
	return width, height
}

func main() {
	if got := ebiten.DurationTime(); got != 0 {
		log.Fatalf("before RunGame: got %d, want 0", got)
	}
	ebiten.SetWindowVisible(false)
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
