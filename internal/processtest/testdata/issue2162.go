// Copyright 2022 The Ebitengine Authors
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

import "github.com/hajimehoshi/ebiten/v2"

func init() {
	s, err := ebiten.NewShader([]byte(`
package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(1)
}`))
	if err != nil {
		panic(err)
	}
	s.Deallocate()
}

type Game struct {
	counter int
}

func (g *Game) Update() error {
	g.counter++
	if g.counter > 1 {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	// Run a game loop at least for one frame to ensure the shader disposed.
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
