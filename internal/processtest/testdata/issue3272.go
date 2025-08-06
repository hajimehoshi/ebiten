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
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Game struct {
	onceRendered bool
}

func (g *Game) Update() error {
	if !g.onceRendered {
		return nil
	}
	return ebiten.Termination
}

func (g *Game) Draw(screen *ebiten.Image) {
	paths := make([]vector.Path, 100)
	for i := range paths {
		paths[i].MoveTo(0, 0)
		paths[i].LineTo(500, 0)
		paths[i].LineTo(500, 500)
		paths[i].LineTo(0, 500)
		paths[i].Close()
		op := &vector.DrawPathOptions{}
		op.AntiAlias = true
		vector.FillPath(screen, &paths[i], nil, op)
	}
	g.onceRendered = true
}

func (g *Game) Layout(width, height int) (int, int) {
	return width, height
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
