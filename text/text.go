// Copyright 2017 The Ebiten Authors
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

// Package text offers functions to draw texts on an Ebiten's image.
//
// Note: This package is experimental and API might be changed.
package text

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/graphics" // TODO: Move NextPowerOf2Int to a new different package
	"github.com/hajimehoshi/ebiten/internal/sync"
)

var (
	monotonicClock int64
)

func now() int64 {
	monotonicClock++
	return monotonicClock
}

var (
	charBounds = map[char]fixed.Rectangle26_6{}
)

type char struct {
	face font.Face
	rune rune
}

func (c *char) bounds() fixed.Rectangle26_6 {
	if b, ok := charBounds[*c]; ok {
		return b
	}
	b, _, _ := c.face.GlyphBounds(c.rune)
	charBounds[*c] = b
	return b
}

func (c *char) size() fixed.Point26_6 {
	b := c.bounds()
	return b.Max.Sub(b.Min)
}

func (c *char) empty() bool {
	s := c.size()
	return s.X == 0 || s.Y == 0
}

func (c *char) atlasGroup() int {
	s := c.size()
	w, h := s.X.Ceil(), s.Y.Ceil()
	t := w
	if t < h {
		t = h
	}

	// Different images for small runes are inefficient.
	// Let's use a same texture atlas for typical character sizes.
	if t < 32 {
		return 32
	}
	return graphics.NextPowerOf2Int(t)
}

type glyph struct {
	char  char
	index int
	atime int64
}

func fixed26_6ToFloat64(x fixed.Int26_6) float64 {
	return float64(x) / (1 << 6)
}

func (g *glyph) draw(dst *ebiten.Image, x, y fixed.Int26_6, clr color.Color) {
	cr, cg, cb, ca := clr.RGBA()
	if ca == 0 {
		return
	}

	b := g.char.bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(fixed26_6ToFloat64(x), fixed26_6ToFloat64(y))
	op.GeoM.Translate(fixed26_6ToFloat64(b.Min.X), fixed26_6ToFloat64(b.Min.Y))

	rf := float64(cr) / float64(ca)
	gf := float64(cg) / float64(ca)
	bf := float64(cb) / float64(ca)
	af := float64(ca) / 0xffff
	op.ColorM.Scale(rf, gf, bf, af)

	a := atlases[g.char.atlasGroup()]
	sx, sy := a.at(g)
	r := image.Rect(sx, sy, sx+a.glyphSize, sy+a.glyphSize)
	op.SourceRect = &r

	dst.DrawImage(a.image, op)
}

var (
	atlases = map[int]*atlas{}
)

type atlas struct {
	// image is the back-end image to hold glyph cache.
	image *ebiten.Image

	// tmpImage is the temporary image as a renderer source for glyph.
	tmpImage *ebiten.Image

	// glyphSize is the size of one glyph in the cache.
	// This value is always power of 2.
	glyphSize int

	charToGlyph map[char]*glyph
}

func (a *atlas) at(glyph *glyph) (int, int) {
	if a.glyphSize != glyph.char.atlasGroup() {
		panic("not reached")
	}
	w, _ := a.image.Size()
	xnum := w / a.glyphSize
	x, y := glyph.index%xnum, glyph.index/xnum
	return x * a.glyphSize, y * a.glyphSize
}

func (a *atlas) maxGlyphNum() int {
	w, h := a.image.Size()
	xnum := w / a.glyphSize
	ynum := h / a.glyphSize
	return xnum * ynum
}

func (a *atlas) appendGlyph(face font.Face, rune rune, now int64) *glyph {
	g := &glyph{
		char:  char{face, rune},
		atime: now,
	}
	if len(a.charToGlyph) == a.maxGlyphNum() {
		var oldest *glyph
		t := int64(math.MaxInt64)
		for _, g := range a.charToGlyph {
			if g.atime < t {
				t = g.atime
				oldest = g
			}
		}
		if oldest == nil {
			panic("not reached")
		}
		idx := oldest.index
		delete(a.charToGlyph, oldest.char)

		g.index = idx
	} else {
		g.index = len(a.charToGlyph)
	}
	a.charToGlyph[g.char] = g
	a.draw(g)
	return g
}

func (a *atlas) draw(glyph *glyph) {
	if a.tmpImage == nil {
		a.tmpImage, _ = ebiten.NewImage(a.glyphSize, a.glyphSize, ebiten.FilterNearest)
	}

	dst := image.NewRGBA(image.Rect(0, 0, a.glyphSize, a.glyphSize))
	d := font.Drawer{
		Dst:  dst,
		Src:  image.White,
		Face: glyph.char.face,
	}
	b := glyph.char.bounds()
	d.Dot = fixed.Point26_6{-b.Min.X, -b.Min.Y}
	d.DrawString(string(glyph.char.rune))
	a.tmpImage.ReplacePixels(dst.Pix)

	op := &ebiten.DrawImageOptions{}
	x, y := a.at(glyph)
	op.GeoM.Translate(float64(x), float64(y))
	op.CompositeMode = ebiten.CompositeModeCopy
	a.image.DrawImage(a.tmpImage, op)

	a.tmpImage.Clear()
}

func getGlyphFromCache(face font.Face, r rune, now int64) *glyph {
	ch := char{face, r}
	a, ok := atlases[ch.atlasGroup()]
	if ok {
		g, ok := a.charToGlyph[ch]
		if ok {
			g.atime = now
			return g
		}
	}

	if ch.empty() {
		// The glyph doesn't have its size but might have valid 'advance' parameter
		// when ch is e.g. space (U+0020).
		return &glyph{
			char:  ch,
			atime: now,
		}
	}

	if !ok {
		// Don't use ebiten.MaxImageSize here.
		// It's because the back-end image pixels will be restored from GPU
		// whenever a new glyph is rendered on the image, and restoring cost is
		// expensive if the image is big.
		// The back-end image is updated a temporary image, and the temporary image is
		// always cleared after used. This means that there is no clue to restore
		// the back-end image without reading from GPU
		// (see the package 'restorable' implementation).
		//
		// TODO: How about making a new function for 'flagile' image?
		const size = 1024
		i, _ := ebiten.NewImage(size, size, ebiten.FilterNearest)
		a = &atlas{
			image:       i,
			glyphSize:   ch.atlasGroup(),
			charToGlyph: map[char]*glyph{},
		}
		atlases[ch.atlasGroup()] = a
	}

	return a.appendGlyph(ch.face, ch.rune, now)
}

var textM sync.Mutex

// Draw draws a given text on a given destination image dst.
//
// face is the font for text rendering.
// (x, y) represents a 'dot' position. Be careful that this doesn't represent left-upper corner position.
// clr is the color for text rendering.
//
// Glyphs used for rendering are cached in least-recently-used way.
// It is OK to call this function with a same text and a same face at every frame.
//
// This function is concurrent-safe.
func Draw(dst *ebiten.Image, text string, face font.Face, x, y int, clr color.Color) {
	textM.Lock()

	n := now()
	fx := fixed.I(x)
	prevC := rune(-1)

	runes := []rune(text)
	for _, c := range runes {
		if prevC >= 0 {
			fx += face.Kern(prevC, c)
		}
		if g := getGlyphFromCache(face, c, n); g != nil {
			if !g.char.empty() {
				g.draw(dst, fx, fixed.I(y), clr)
			}
			a, _ := face.GlyphAdvance(c)
			fx += a
		}
		prevC = c
	}

	textM.Unlock()
}
