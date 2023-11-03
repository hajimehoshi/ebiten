// Copyright 2022 The Ebiten Authors
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
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	count int
}

func (g *Game) Update() error {
	g.count++
	if g.count >= 2 {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(width, height int) (int, int) {
	// Ebitengine's image function should be able to be called even in Layout.
	done := make(chan struct{})
	timeout := time.After(time.Second)
	go func() {
		i := ebiten.NewImage(width, height)
		i.Fill(color.White)
		i.Deallocate()
		close(done)
	}()

	select {
	case <-done:
	case <-timeout:
		panic("timeout")
	}
	return width, height
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
