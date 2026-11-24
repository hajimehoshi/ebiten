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
	"errors"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
)

// This test confirms that Update is not called again after it returned Termination in a frame rendered
// during event polling, neither during the same event polling nor after it.

const windowTitle = "Ebitengine forced frame termination"

type game struct {
	ticks      int
	resizing   atomic.Bool
	terminated bool
	err        error
}

func (g *game) Update() error {
	if g.terminated {
		g.err = errors.New("Update was called after it returned Termination")
		return g.err
	}
	if g.resizing.Load() {
		g.terminated = true
		return ebiten.Termination
	}

	g.ticks++
	if g.ticks == 1 {
		return callDuringEventPolling(g.resizeWindow)
	}
	if g.ticks > 600 {
		return errors.New("no frame was rendered during event polling")
	}
	return nil
}

func (g *game) resizeWindow(setSize func(width, height int)) {
	g.resizing.Store(true)
	defer g.resizing.Store(false)

	for i := range 2 {
		setSize(400+100*i, 300)
	}
}

func (g *game) Draw(screen *ebiten.Image) {
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowTitle(windowTitle)
	ebiten.SetWindowVisible(false)
	ebiten.SetTPS(ebiten.SyncWithFPS)

	g := &game{}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	if g.err != nil {
		panic(g.err)
	}
}
