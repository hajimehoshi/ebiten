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
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Game struct {
	images [2]*ebiten.Image
}

func (g *Game) Update() error {
	var path vector.Path
	path.MoveTo(1, 1)
	path.LineTo(9, 1)
	path.LineTo(9, 7)
	path.Close()
	for i := range g.images {
		g.images[i] = ebiten.NewImage(16, 16)
		op := &vector.DrawPathOptions{AntiAlias: true}
		if i == 1 {
			op.Clip = &vector.PathClip{Path: &path}
		}
		vector.FillPath(g.images[i], &path, nil, op)
	}
	return ebiten.Termination
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(width, height int) (int, int) {
	return width, height
}

func main() {
	g := &Game{}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	g.images[0].Deallocate()
	g.images[1].Deallocate()
}
