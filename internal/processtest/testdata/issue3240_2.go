// Copyright 2025 The Ebitengine Authors
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
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	counter int
}

func (g *Game) Update() error {
	g.counter++
	if g.counter >= 60 {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	panic("panic from Draw")
}

func (g *Game) Layout(w, h int) (int, int) {
	return 320, 240
}

func main() {
	defer func() {
		r := recover()
		if r == nil {
			fmt.Fprintf(os.Stderr, "Expected a panic, but got none\n")
			os.Exit(1)
		}
	}()
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
