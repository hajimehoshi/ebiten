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

package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type countingFace struct {
	font.Face
	calls map[rune]int
}

func (f *countingFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	f.calls[r]++
	return f.Face.Glyph(dot, 'X')
}

func (f *countingFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	return f.Face.GlyphBounds('X')
}

func (f *countingFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	return f.Face.GlyphAdvance('X')
}

type game struct {
	phase        int
	initialized  bool
	started      int64
	pausedFrames int
	paused       bool
	image        *ebiten.Image
	subimages    [3]image.Image
	legacy       *countingFace
	modern       *countingFace
	modernFace   *textv2.GoXFace
}

var rates = []int{30, 120, ebiten.SyncWithFPS, 60}

func (g *game) use(s string) {
	text.Draw(g.image, s, g.legacy, 0, 12, color.White)
	textv2.AppendGlyphs(nil, s, g.modernFace, nil)
}

func (g *game) check(r rune, want int) error {
	for name, face := range map[string]*countingFace{"legacy": g.legacy, "v2": g.modern} {
		if got := face.calls[r]; got != want {
			return fmt.Errorf("TPS %d, %s glyph %q: rasterized %d times, want %d", rates[g.phase], name, r, got, want)
		}
	}
	return nil
}

func subrect(i int) image.Rectangle {
	return image.Rect(i, 0, i+1, 1)
}

func (g *game) Update() error {
	if !g.initialized {
		ebiten.SetTPS(rates[g.phase])
		g.started = ebiten.Tick()
		g.pausedFrames = 0
		g.image = ebiten.NewImage(16, 16)
		g.legacy = &countingFace{
			Face:  basicfont.Face7x13,
			calls: map[rune]int{},
		}
		g.modern = &countingFace{
			Face:  basicfont.Face7x13,
			calls: map[rune]int{},
		}
		g.modernFace = textv2.NewGoXFace(g.modern)
		for i, r := range "abc" {
			g.use(string(r))
			g.subimages[i] = g.image.SubImage(subrect(i))
		}
		// Exceed the existing glyph-cache soft limits without using the probe glyphs.
		runes := make([]rune, 2050)
		for i := range runes {
			runes[i] = rune(0x4e00 + i)
		}
		g.use(string(runes))
		g.initialized = true
		return nil
	}
	if g.paused {
		return nil
	}
	elapsed := ebiten.Tick() - g.started
	// Keep one resource hot while the other two age.
	g.use("c")
	if err := g.check('c', 1); err != nil {
		return err
	}
	if got := g.image.SubImage(subrect(2)); got != g.subimages[2] {
		return fmt.Errorf("TPS %d: frequently used sub-image was evicted", rates[g.phase])
	}
	if elapsed == 60 {
		g.use("d")
		g.image.SubImage(subrect(3))
		g.use("a")
		if err := g.check('a', 1); err != nil {
			return err
		}
		if evicted := g.image.SubImage(subrect(0)) != g.subimages[0]; evicted {
			return fmt.Errorf("TPS %d: sub-image eviction after %d ticks: got %t, want false", rates[g.phase], elapsed, evicted)
		}
		// Draw calls without updates must not age resources.
		g.paused = true
		ebiten.SetTPS(0)
		return nil
	}
	if elapsed <= 60 {
		return nil
	}
	g.use("e")
	g.image.SubImage(subrect(4))
	g.use("b")
	if err := g.check('b', 2); err != nil {
		return err
	}
	if got := g.image.SubImage(subrect(1)); got == g.subimages[1] {
		return fmt.Errorf("TPS %d: stale sub-image was retained", rates[g.phase])
	}
	g.image.Deallocate()
	g.phase++
	if g.phase == len(rates) {
		return ebiten.Termination
	}
	g.initialized = false
	return nil
}

func (g *game) Draw(*ebiten.Image) {
	if !g.paused {
		return
	}
	g.use("a")
	if err := g.check('a', 1); err != nil {
		panic(err)
	}
	g.pausedFrames++
	if g.pausedFrames == 3 {
		ebiten.SetTPS(rates[g.phase])
		if g.phase == 3 {
			ebiten.SetTPS(ebiten.SyncWithFPS)
		}
		g.paused = false
	}
}

func (*game) Layout(int, int) (int, int) {
	return 16, 16
}

func main() {
	ebiten.SetWindowVisible(false)
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
