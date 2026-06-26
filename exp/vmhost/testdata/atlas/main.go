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

//go:build ebitenginevm

// This guest exercises atlas-packed images. It creates several small ebiten.Images (the default
// ImageType is Regular, so the atlas packs them into a shared backend at distinct offsets, each with
// padding) and draws each at a known screen position. The colors are all distinct, so if the host
// mishandled a packed source offset, a draw would sample a neighbor image's pixels and the rendered
// color would be wrong. This is the same atlas path that glyph (text) rendering uses.
//
// It is launched by a host; see vmhost's atlas test.
package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type tile struct {
	col  color.RGBA
	x, y int
}

// tiles are the distinct-colored images and where to draw them. Distinct colors make a wrong source
// offset visible.
var tiles = []tile{
	{color.RGBA{R: 0xff, A: 0xff}, 10, 10},          // red
	{color.RGBA{G: 0xff, A: 0xff}, 40, 10},          // green
	{color.RGBA{B: 0xff, A: 0xff}, 10, 40},          // blue
	{color.RGBA{R: 0xff, G: 0xff, A: 0xff}, 40, 40}, // yellow
	{color.RGBA{R: 0xff, B: 0xff, A: 0xff}, 70, 10}, // magenta
	{color.RGBA{G: 0xff, B: 0xff, A: 0xff}, 70, 40}, // cyan
}

type game struct {
	images []*ebiten.Image
}

func (g *game) Update() error {
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	if g.images == nil {
		for _, t := range tiles {
			img := ebiten.NewImage(16, 16) // Regular: packed into the shared atlas.
			img.Fill(t.col)
			g.images = append(g.images, img)
		}
	}
	for i, t := range tiles {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(t.x), float64(t.y))
		screen.DrawImage(g.images[i], op)
	}
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
