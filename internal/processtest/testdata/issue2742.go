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
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	ch <-chan struct{}
}

func (g *Game) Update() error {
	select {
	case <-g.ch:
		return ebiten.Termination
	default:
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	ch := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			ebiten.SetCursorMode(ebiten.CursorModeHidden)
			time.Sleep(time.Millisecond)
		}
		close(ch)
	}()
	if err := ebiten.RunGame(&Game{ch: ch}); err != nil {
		panic(err)
	}
}
