// Copyright 2024 The Ebitengine Authors
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

// This test confirms that deallocation of a shader works correctly.

type Game struct {
	count int

	img *ebiten.Image
}

func (g *Game) Update() error {
	if g.img == nil {
		g.img = ebiten.NewImage(1, 1)
	}

	g.count++

	s, err := ebiten.NewShader([]byte(fmt.Sprintf(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(%d/255.0)
}
`, g.count)))
	if err != nil {
		return err
	}

	// Use the shader to ensure that the shader is actually allocated.
	g.img.DrawRectShader(1, 1, s, nil)

	s.Deallocate()

	if g.count == 60 {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(w, h int) (int, int) {
	return 320, 240
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
