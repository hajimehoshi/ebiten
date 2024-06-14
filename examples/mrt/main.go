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

package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	dstSize      = 128
	screenWidth  = dstSize * 2
	screenHeight = dstSize * 2
)

var (
	dsts = [8]*ebiten.Image{
		ebiten.NewImageWithOptions(image.Rect(0, 0, dstSize, dstSize), &ebiten.NewImageOptions{
			Unmanaged: true,
		}),
		ebiten.NewImageWithOptions(image.Rect(0, 0, dstSize, dstSize), &ebiten.NewImageOptions{
			Unmanaged: true,
		}),
		ebiten.NewImageWithOptions(image.Rect(0, 0, dstSize, dstSize), &ebiten.NewImageOptions{
			Unmanaged: true,
		}),
		ebiten.NewImageWithOptions(image.Rect(0, 0, dstSize, dstSize), &ebiten.NewImageOptions{
			Unmanaged: true,
		}),
	}

	shaderSrc = []byte(
		`
//kage:units pixels

package main

func Fragment(dst vec4, src vec2, color vec4) (vec4, vec4, vec4, vec4) {
	return vec4(1,0,0,1), vec4(0,1,0,1), vec4(0,0,1,1), vec4(1,0,1,1)
}
`)
	s *ebiten.Shader
)

func init() {
	var err error

	s, err = ebiten.NewShader(shaderSrc)
	if err != nil {
		log.Fatal(err)
	}
}

type Game struct {
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	vertices := []ebiten.Vertex{
		{
			DstX: 0,
			DstY: 0,
		},
		{
			DstX: dstSize,
			DstY: 0,
		},
		{
			DstX: 0,
			DstY: dstSize,
		},
		{
			DstX: dstSize,
			DstY: dstSize,
		},
	}
	indices := []uint16{0, 1, 2, 1, 2, 3}
	ebiten.DrawTrianglesShaderMRT(dsts, vertices, indices, s, nil)
	// Dst 0
	screen.DrawImage(dsts[0], nil)
	// Dst 1
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(dstSize, 0)
	screen.DrawImage(dsts[1], opts)
	// Dst 2
	opts.GeoM.Reset()
	opts.GeoM.Translate(0, dstSize)
	screen.DrawImage(dsts[2], opts)
	// Dst 3
	opts.GeoM.Reset()
	opts.GeoM.Translate(dstSize, dstSize)
	screen.DrawImage(dsts[3], opts)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %.2f", ebiten.ActualFPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetWindowTitle("MRT (Ebitengine Demo)")
	if err := ebiten.RunGameWithOptions(&Game{}, &ebiten.RunGameOptions{
		GraphicsLibrary: ebiten.GraphicsLibraryDirectX,
	}); err != nil {
		log.Fatal(err)
	}
}
