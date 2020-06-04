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

const shaderSrc = `package main

// Internal_ViewportSize is a predefined uniform variable.
// TODO: Hide this and create a function for the projection matrix?

func Vertex(position vec2, texCoord vec2, color vec4) vec4 {
	return mat4(
		2.0/Internal_ViewportSize.x, 0, 0, 0,
		0, 2.0/Internal_ViewportSize.y, 0, 0,
		0, 0, 1, 0,
		-1, -1, 0, 1,
	) * vec4(position, 0, 1)
}

func Fragment(position vec4) vec4 {
	return vec4(position.x/Internal_ViewportSize.x, position.y/Internal_ViewportSize.y, 0, 1)
}`

type Game struct {
	shader *ebiten.Shader
}

func (g *Game) Update(screen *ebiten.Image) error {
	if g.shader == nil {
		var err error
		g.shader, err = ebiten.NewShader([]byte(shaderSrc))
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
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
	screen.DrawTrianglesWithShader(vs, is, g.shader, nil)
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
