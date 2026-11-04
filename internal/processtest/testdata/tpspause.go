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

//go:build ignore

package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	updated bool
	draws   int
}

func (g *Game) Update() error {
	if !g.updated {
		g.updated = true
		ebiten.SetTPS(0)
		return nil
	}
	if g.draws < 2 {
		return fmt.Errorf("unexpected Update while TPS is zero")
	}
	return ebiten.Termination
}

func (g *Game) Draw(*ebiten.Image) {
	g.draws++
	if g.draws == 2 {
		ebiten.SetTPS(ebiten.SyncWithFPS)
	}
}

func (g *Game) Layout(int, int) (int, int) {
	return 64, 64
}

func main() {
	ebiten.SetWindowVisible(false)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
