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
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// This program tests that Image.Deallocate can be called after RunGame returns
// even when the image still has deferred vector fills that have never been flushed.
//
// vector.FillPath and its family do not render immediately: the paths are kept
// pending until the destination image is used, and Deallocate counts as a use
// and flushes them. The flush allocates a stencil atlas image via NewImage,
// which panics after RunGame finishes.
type Game struct {
	off *ebiten.Image
	n   int
}

func (g *Game) Update() error {
	if g.off == nil {
		g.off = ebiten.NewImage(16, 16)
	}
	// Anti-aliased vector drawing to an image that is never used as a rendering
	// source, so the fills stay pending until the image is used. This is done in
	// Update so that the test does not depend on Draw being called before
	// Termination.
	vector.FillRect(g.off, 1, 1, 8, 6, color.White, true)
	vector.StrokeLine(g.off, 0, 0, 15, 15, 1, color.White, true)
	g.n++
	if g.n >= 2 {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(width, height int) (int, int) {
	return width, height
}

func main() {
	g := &Game{}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}

	// Deallocating an image after RunGame finishes must not panic, even if the
	// image still has deferred vector fills that have never been flushed.
	if g.off != nil {
		g.off.Deallocate()
	}
}
