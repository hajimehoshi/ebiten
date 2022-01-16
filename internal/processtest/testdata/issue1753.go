// Copyright 2021 The Ebiten Authors
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
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var regularTermination = errors.New("regular termination")

type Game struct {
	dst   *ebiten.Image
	phase int
}

func (g *Game) Update() error {
	const w, h = 1, 1

	switch g.phase {
	case 0:
		s, err := ebiten.NewShader([]byte(`package main

var Color vec4

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return Color
}`))
		if err != nil {
			return err
		}

		g.dst = ebiten.NewImage(w, h)

		op := &ebiten.DrawRectShaderOptions{}
		op.CompositeMode = ebiten.CompositeModeCopy
		op.Uniforms = map[string]interface{}{
			"Color": []float32{1, 1, 1, 1},
		}
		g.dst.DrawRectShader(w, h, s, op)
		if got, want := g.dst.At(0, 0), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
			return fmt.Errorf("phase: %d, got: %v, want: %v", g.phase, got, want)
		}

		// Dispose the shader. When a new shader is created in the next phase, the underlying shader ID might be reused.
		// This test checks that the new shader works in this situation.
		// The actual disposal will happen after this frame and before the next frame in the current implmentation.
		s.Dispose()

		g.phase++

	case 1:
		s, err := ebiten.NewShader([]byte(`package main

var Dummy float
var A, B, G, R float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(R, G, B, A)
}`))
		if err != nil {
			return err
		}

		op := &ebiten.DrawRectShaderOptions{}
		op.CompositeMode = ebiten.CompositeModeCopy
		op.Uniforms = map[string]interface{}{
			"Dummy": float32(0),
			"R":     float32(0.5),
			"G":     float32(1),
			"B":     float32(0.5),
			"A":     float32(1),
		}
		g.dst.DrawRectShader(w, h, s, op)
		if got, want := g.dst.At(0, 0), (color.RGBA{0x80, 0xff, 0x80, 0xff}); got != want {
			return fmt.Errorf("phase: %d, got: %v, want: %v", g.phase, got, want)
		}

		return regularTermination
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(width, height int) (int, int) {
	return width, height
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil && err != regularTermination {
		panic(err)
	}
}
