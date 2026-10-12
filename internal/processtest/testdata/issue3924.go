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
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// This test confirms that dstPos counts from the top of the final screen (#3924).

const w, h = 16, 16

type game struct {
	ref    *ebiten.Image
	shader *ebiten.Shader
	done   bool
	err    error
}

func (g *game) Update() error {
	if g.done {
		if g.err != nil {
			return g.err
		}
		return ebiten.Termination
	}
	if g.ref != nil {
		return nil
	}
	g.ref = ebiten.NewImage(w, h)
	g.ref.SubImage(image.Rect(0, 0, w, h/2)).(*ebiten.Image).Fill(color.RGBA{R: 0xff, A: 0xff})
	g.ref.SubImage(image.Rect(0, h/2, w, h)).(*ebiten.Image).Fill(color.RGBA{B: 0xff, A: 0xff})
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	if dstPos.y < 8 {
		return vec4(1, 0, 0, 1)
	}
	return vec4(0, 0, 1, 1)
}
`))
	if err != nil {
		return err
	}
	g.shader = s
	return nil
}

func (*game) Draw(*ebiten.Image) {
}

func (g *game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	if g.done {
		return
	}
	// The left half is the reference image, and the right half is the shader's result.
	// Both must be the same however the screen's pixels are read.
	screen.DrawImage(g.ref, nil)
	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(w, 0)
	screen.DrawRectShader(w, h, g.shader, op)

	sw := screen.Bounds().Dx()
	pix := make([]byte, 4*sw*screen.Bounds().Dy())
	screen.(*ebiten.Image).ReadPixels(pix)
	for j := range h {
		for i := range w {
			want := pix[4*(i+sw*j) : 4*(i+sw*j)+4]
			got := pix[4*(i+w+sw*j) : 4*(i+w+sw*j)+4]
			if string(got) != string(want) && g.err == nil {
				g.err = fmt.Errorf("pixel (%d, %d): got %v, want %v", i+w, j, got, want)
			}
		}
	}
	g.done = true
}

func (*game) Layout(int, int) (int, int) {
	return 2 * w, h
}

func main() {
	ebiten.SetWindowVisible(false)
	ebiten.SetWindowSize(2*w, h)
	if err := ebiten.RunGameWithOptions(&game{}, &ebiten.RunGameOptions{
		GraphicsLibrary: ebiten.GraphicsLibraryOpenGL,
	}); err != nil {
		panic(err)
	}
}
