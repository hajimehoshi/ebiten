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
// +build ignore

package main

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
)

var regularTermination = errors.New("regular termination")

type Game struct {
	shader  *ebiten.Shader
	timeout int
}

func (g *Game) Update() error {
	// A shader compilation error breaks the state of the graphics command queue, and this cannot be reused.
	// Thus, a process test is appropriated.
	g.shader, _ = ebiten.NewShader([]byte(`package main`))
	g.timeout++
	if g.timeout > 60 {
		return regularTermination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawRectShader(1, 1, g.shader, nil)
}

func (g *Game) Layout(width, height int) (int, int) {
	return width, height
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil && err != regularTermination {
		return
	}
	panic("RunGame must return an error but not")
}
