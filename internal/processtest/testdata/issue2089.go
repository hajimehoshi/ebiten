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

package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	count int
}

func (g *Game) Update() error {
	g.count++
	if g.count == 16 {
		return ebiten.Termination
	}

	w, h := 256+g.count, 256+g.count
	img := ebiten.NewImage(w, h)
	img.Fill(color.White)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if (i+j)%3 == 0 {
				img.Set(i, j, color.Black)
			}
		}
	}

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if (i+j)%3 == 0 {
				want = color.RGBA{0, 0, 0, 0xff}
			}
			got := img.At(i, j)
			if got != want {
				panic(fmt.Sprintf("At(%d, %d): got: %v, want: %v", i, j, got, want))
			}
		}
	}

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
