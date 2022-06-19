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
// +build ignore

package main

import (
	"errors"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var regularTermination = errors.New("regular termination")

var src *ebiten.Image

func init() {
	const (
		w = 2
		h = 2
	)

	src0 := ebiten.NewImage(w, h)
	src0.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			src0.Set(i, j, color.RGBA{0, 0, 0, 0xff})
		}
	}

	src1 := ebiten.NewImage(w, h)
	src1.DrawImage(src0, nil)

	src = src1
}

type Game struct {
	count int
	dst   *ebiten.Image
}

func (g *Game) Update() error {
	g.count++
	if g.count == 16 {
		return regularTermination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	screen.DrawImage(src, nil)

	if g.dst == nil {
		g.dst = ebiten.NewImage(screen.Size())
		return
	}

	// Get the pixel at the next frame.
	g.dst.DrawImage(screen, nil)
	got := g.dst.At(0, 0)
	want := color.RGBA{0, 0, 0, 0xff}
	if got != want {
		panic(fmt.Sprintf("got: %v, want: %v", got, want))
	}
}

func (g *Game) Layout(width, height int) (int, int) {
	return width, height
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil && err != regularTermination {
		panic(err)
	}
}
