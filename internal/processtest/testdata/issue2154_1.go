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

package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var srcInit *ebiten.Image

func init() {
	const (
		w = 2
		h = 2
	)

	// src2 := ebiten.NewImage(1, 1)

	src0 := ebiten.NewImage(w, h)
	src0.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	src0.Set(0, 0, color.RGBA{0, 0, 0, 0xff})
	src0.Set(0, 1, color.RGBA{0, 0, 0, 0xff})
	src0.Set(1, 0, color.RGBA{0, 0, 0, 0xff})

	src1 := ebiten.NewImage(w, h)
	// Using the image as a source just after Set caused troubles on Metal.
	// For example, inserting src1.Fill(color.RGBA{0, 0xff, 0, 0xff}) here hid the error.
	src1.DrawImage(src0, nil)

	srcInit = src1
}

type Game struct {
	count int
	dst   *ebiten.Image
}

func (g *Game) Update() error {
	g.count++
	if g.count == 16 {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	screen.DrawImage(srcInit, nil)

	if g.dst == nil {
		g.dst = ebiten.NewImage(screen.Bounds().Dx(), screen.Bounds().Dy())
		return
	}

	g.dst.DrawImage(screen, nil)
	if got, want := g.dst.At(0, 0), (color.RGBA{0, 0, 0, 0xff}); got != want {
		panic(fmt.Sprintf("count: %d, got: %v, want: %v", g.count, got, want))
	}
	if got, want := g.dst.At(1, 1), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		panic(fmt.Sprintf("count: %d, got: %v, want: %v", g.count, got, want))
	}
	g.dst.Clear()

	const (
		w = 2
		h = 2
	)

	src0 := ebiten.NewImage(w, h)
	defer src0.Deallocate()
	src0.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	src0.Set(0, 0, color.RGBA{0, 0, 0, 0xff})
	src0.Set(0, 1, color.RGBA{0, 0, 0, 0xff})
	src0.Set(1, 0, color.RGBA{0, 0, 0, 0xff})

	src1 := ebiten.NewImage(w, h)
	defer src1.Deallocate()
	src1.DrawImage(src0, nil)

	screen.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	screen.DrawImage(src1, nil)

	g.dst.DrawImage(screen, nil)
	if got, want := g.dst.At(0, 0), (color.RGBA{0, 0, 0, 0xff}); got != want {
		panic(fmt.Sprintf("count: %d, got: %v, want: %v", g.count, got, want))
	}
	if got, want := g.dst.At(1, 1), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		panic(fmt.Sprintf("count: %d, got: %v, want: %v", g.count, got, want))
	}
}

func (g *Game) Layout(width, height int) (int, int) {
	return width, height
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
