// Copyright 2020 The Ebiten Authors
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

// +build example jsgo

package main

import (
	"log"

	"github.com/hajimehoshi/ebiten"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var shaderSrcs = [][]byte{
	default_go,
}

type Game struct {
	shaders map[int]*ebiten.Shader
	idx     int
	time    int
}

func (g *Game) Update(screen *ebiten.Image) error {
	g.time++
	if g.shaders == nil {
		g.shaders = map[int]*ebiten.Shader{}
	}
	if _, ok := g.shaders[g.idx]; !ok {
		s, err := ebiten.NewShader([]byte(shaderSrcs[g.idx]))
		if err != nil {
			return err
		}
		g.shaders[g.idx] = s
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	s, ok := g.shaders[g.idx]
	if !ok {
		return
	}

	w, h := screen.Size()
	vs := []ebiten.Vertex{
		{
			DstX: 0,
			DstY: 0,
		},
		{
			DstX: float32(w),
			DstY: 0,
		},
		{
			DstX: 0,
			DstY: float32(h),
		},
		{
			DstX: float32(w),
			DstY: float32(h),
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}

	cx, cy := ebiten.CursorPosition()

	op := &ebiten.DrawTrianglesWithShaderOptions{}
	op.Uniforms = []interface{}{
		float32(g.time) / 60,                // time
		[]float32{float32(cx), float32(cy)}, // cursor
	}
	screen.DrawTrianglesWithShader(vs, is, s, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Shader (Ebiten Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
