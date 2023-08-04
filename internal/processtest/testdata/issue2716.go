// Copyright 2023 The Ebitengine Authors
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
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	count int

	imgs []*ebiten.Image
}

func (g *Game) Update() error {
	const (
		w = 16
		h = 16
	)

	g.count++
	if g.count >= 16 {
		for c, img := range g.imgs {
			for j := 0; j < h; j++ {
				for i := 0; i < h; i++ {
					got := img.At(i, j).(color.RGBA)
					want := color.RGBA{byte(c), byte(c), byte(c), byte(c)}
					if got != want {
						return fmt.Errorf("index: %d, got: %v, want: %v", c, got, want)
					}
				}
			}
		}
		return ebiten.Termination
	}

	pix := make([]byte, 4*w*h)
	c := byte(len(g.imgs))
	for i := range pix {
		pix[i] = c
	}

	img := ebiten.NewImage(w, h)
	img.WritePixels(pix)

	g.imgs = append(g.imgs, img)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(width, height int) (int, int) {
	return width, height
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
