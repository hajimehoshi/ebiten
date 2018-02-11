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
// For the example using a TTF font, see font package in the examples.
package text

import (
	"image"
	"image/color"
	"math"
	"reflect"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten"
	emath "github.com/hajimehoshi/ebiten/internal/math"
	"github.com/hajimehoshi/ebiten/internal/sync"
)

var (
	monotonicClock int64
)

func now() int64 {
	monotonicClock++
	return monotonicClock
}

type face struct {
	f font.Face
}

var (
	faces = map[font.Face]face{}
)

func fontFaceToFace(f font.Face) face {
	if fa, ok := faces[f]; ok {
		return fa
	}
	// If the (DeepEqual-ly) same font exists,
	// reuse this to avoid to consume a lot of cache (#498).
	for key, value := range faces {
		if reflect.DeepEqual(key, f) {
			return value
		}
	}
	fa := face{f}
	faces[f] = fa
	return fa
}

var (
	charBounds = map[font.Face]map[rune]fixed.Rectangle26_6{}
)

type char struct {
	face face
	rune rune
}

func (c *char) bounds() (minX, minY, maxX, maxY fixed.Int26_6) {
	if m, ok := charBounds[c.face.f]; ok {
		if b, ok := m[c.rune]; ok {
			return b.Min.X, b.Min.Y, b.Max.X, b.Max.Y
		}
	} else {
		charBounds[c.face.f] = map[rune]fixed.Rectangle26_6{}
	}
	b, _, _ := c.face.f.GlyphBounds(c.rune)
	charBounds[c.face.f][c.rune] = b
	return b.Min.X, b.Min.Y, b.Max.X, b.Max.Y
}

func (c *char) size() (fixed.Int26_6, fixed.Int26_6) {
	minX, minY, maxX, maxY := c.bounds()
	return maxX - minX, maxY - minY
}

func (c *char) empty() bool {
	x, y := c.size()
	return x == 0 || y == 0
}

func (c *char) atlasGroup() int {
	x, y := c.size()
	w, h := x.Ceil(), y.Ceil()
	t := w
	if t < h {
		t = h
	}

	// Different images for small runes are inefficient.
	// Let's use a same texture atlas for typical character sizes.
	if t < 32 {
		return 32
	}
	return emath.NextPowerOf2Int(t)
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

	minX, minY, _, _ := g.char.bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(fixed26_6ToFloat64(x+minX), fixed26_6ToFloat64(y+minY))

	rf := float64(cr) / float64(ca)
	gf := float64(cg) / float64(ca)
	bf := float64(cb) / float64(ca)
	af := float64(ca) / 0xffff
	op.ColorM.Scale(rf, gf, bf, af)

	a := atlases[g.char.face.f][g.char.atlasGroup()]
	sx, sy := a.at(g)
	r := image.Rect(sx, sy, sx+a.glyphSize, sy+a.glyphSize)
	op.SourceRect = &r

	dst.DrawImage(a.image, op)
}

var (
	atlases = map[font.Face]map[int]*atlas{}
)

type atlas struct {
	// image is the back-end image to hold glyph cache.
	image *ebiten.Image

	// tmpImage is the temporary image as a renderer source for glyph.
	tmpImage *ebiten.Image

	// glyphSize is the size of one glyph in the cache.
	// This value is always power of 2.
	glyphSize int

	runeToGlyph map[rune]*glyph
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

func (a *atlas) appendGlyph(face face, rune rune, now int64) *glyph {
	g := &glyph{
		char:  char{face, rune},
		atime: now,
	}
	if len(a.runeToGlyph) == a.maxGlyphNum() {
		var oldest *glyph
		t := int64(math.MaxInt64)
		for _, g := range a.runeToGlyph {
			if g.atime < t {
				t = g.atime
				oldest = g
			}
		}
		if oldest == nil {
			panic("not reached")
		}
		idx := oldest.index
		delete(a.runeToGlyph, oldest.char.rune)

		g.index = idx
	} else {
		g.index = len(a.runeToGlyph)
	}
	a.runeToGlyph[g.char.rune] = g
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
		Face: glyph.char.face.f,
	}
	minX, minY, _, _ := glyph.char.bounds()
	d.Dot = fixed.Point26_6{-minX, -minY}
	d.DrawString(string(glyph.char.rune))
	a.tmpImage.ReplacePixels(dst.Pix)

	op := &ebiten.DrawImageOptions{}
	x, y := a.at(glyph)
	op.GeoM.Translate(float64(x), float64(y))
	op.CompositeMode = ebiten.CompositeModeCopy
	a.image.DrawImage(a.tmpImage, op)

	a.tmpImage.Clear()
}

func getGlyphFromCache(face face, r rune, now int64) *glyph {
	ch := char{face, r}
	var at *atlas
	if m, ok := atlases[face.f]; ok {
		a, ok := m[ch.atlasGroup()]
		if ok {
			g, ok := a.runeToGlyph[r]
			if ok {
				g.atime = now
				return g
			}
		}
		at = a
	} else {
		atlases[face.f] = map[int]*atlas{}
	}

	if ch.empty() {
		// The glyph doesn't have its size but might have valid 'advance' parameter
		// when ch is e.g. space (U+0020).
		return &glyph{
			char:  ch,
			atime: now,
		}
	}

	if at == nil {
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
		at = &atlas{
			image:       i,
			glyphSize:   ch.atlasGroup(),
			runeToGlyph: map[rune]*glyph{},
		}
		atlases[face.f][ch.atlasGroup()] = at
	}

	return at.appendGlyph(ch.face, ch.rune, now)
}

var textM sync.Mutex

// Draw draws a given text on a given destination image dst.
//
// face is the font for text rendering.
// (x, y) represents a 'dot' (period) position.
// Be careful that this doesn't represent left-upper corner position.
// clr is the color for text rendering.
//
// Glyphs used for rendering are cached in least-recently-used way.
// It is OK to call this function with a same text and a same face at every frame in terms of performance.
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
		fa := fontFaceToFace(face)
		if g := getGlyphFromCache(fa, c, n); g != nil {
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
