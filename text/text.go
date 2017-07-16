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

type char struct {
	face font.Face
	rune rune
}

type glyph struct {
	char   char
	index  int
	bounds fixed.Rectangle26_6
	atime  int64
}

func (g *glyph) size() (int, int) {
	p := g.bounds.Max.Sub(g.bounds.Min)
	return p.X.Ceil(), p.Y.Ceil()
}

func (g *glyph) empty() bool {
	w, h := g.size()
	return w == 0 || h == 0
}

func (g *glyph) atlasGroup() int {
	w, h := g.size()
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

func (g *glyph) draw(dst *ebiten.Image, x, y int, clr color.Color) {
	cr, cg, cb, ca := clr.RGBA()
	if ca == 0 {
		return
	}

	a := atlases[g.atlasGroup()]
	sx, sy := a.at(g)
	ox := g.bounds.Min.X.Ceil()
	oy := g.bounds.Min.Y.Ceil()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.GeoM.Translate(float64(ox), float64(oy))

	rf := float64(cr) / float64(ca)
	gf := float64(cg) / float64(ca)
	bf := float64(cb) / float64(ca)
	af := float64(ca) / 0xffff
	op.ColorM.Scale(rf, gf, bf, af)

	r := image.Rect(sx, sy, sx+a.size, sy+a.size)
	op.SourceRect = &r

	dst.DrawImage(a.image, op)
}

var (
	glyphs  = map[char]*glyph{}
	atlases = map[int]*atlas{}
)

type atlas struct {
	// image is the back-end image to hold glyph cache.
	image *ebiten.Image

	// tmpImage is the temporary image as a renderer source for glyph.
	tmpImage *ebiten.Image

	// size is the size of one glyph in the cache.
	// This value is always power of 2.
	size int

	// glyphs is the set of glyph information.
	glyphs []*glyph

	// num is the number of glyphs the atlas holds.
	num int
}

func (a *atlas) at(glyph *glyph) (int, int) {
	if a.size != glyph.atlasGroup() {
		panic("not reached")
	}
	w, _ := a.image.Size()
	xnum := w / a.size
	x, y := glyph.index%xnum, glyph.index/xnum
	return x * a.size, y * a.size
}

func (a *atlas) append(glyph *glyph) {
	if a.num == len(a.glyphs) {
		idx := -1
		t := int64(math.MaxInt64)
		for i, g := range a.glyphs {
			if g.atime < t {
				t = g.atime
				idx = i
			}
		}
		if idx < 0 {
			panic("not reached")
		}
		oldest := a.glyphs[idx]
		delete(glyphs, oldest.char)

		glyph.index = idx
		a.glyphs[idx] = glyph
		a.draw(glyph)
		return
	}
	idx := -1
	for i, g := range a.glyphs {
		if g == nil {
			idx = i
			break
		}
	}
	if idx < 0 {
		panic("not reached")
	}
	a.num++
	glyph.index = idx
	a.glyphs[idx] = glyph
	a.draw(glyph)
}

func (a *atlas) draw(glyph *glyph) {
	if a.tmpImage == nil {
		a.tmpImage, _ = ebiten.NewImage(a.size, a.size, ebiten.FilterNearest)
	}

	dst := image.NewRGBA(image.Rect(0, 0, a.size, a.size))
	d := font.Drawer{
		Dst:  dst,
		Src:  image.White,
		Face: glyph.char.face,
	}
	ox := -glyph.bounds.Min.X.Ceil()
	oy := -glyph.bounds.Min.Y.Ceil()
	d.Dot = fixed.P(ox, oy)
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
	g, ok := glyphs[ch]
	if ok {
		g.atime = now
		return g
	}

	b, _, ok := face.GlyphBounds(r)
	if !ok {
		return nil
	}
	g = &glyph{
		char:   ch,
		bounds: b,
		atime:  now,
	}
	if g.empty() {
		return g
	}

	a, ok := atlases[g.atlasGroup()]
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
			image: i,
			size:  g.atlasGroup(),
		}
		w, h := a.image.Size()
		xnum := w / a.size
		ynum := h / a.size
		a.glyphs = make([]*glyph, xnum*ynum)
		atlases[g.atlasGroup()] = a
	}

	a.append(g)
	glyphs[g.char] = g
	return g
}

var textM sync.Mutex

// Draw draws a given text on a give destination image dst.
//
// face is the font for text rendering.
// (x, y) represents a 'dot' position. Be careful that this doesn't represent left-upper corner position.
// lineHeight is the Y offset for line spacing.
// clr is the color for text rendering.
//
// Glyphs used for rendering are cached in least-recently-used way.
// It is OK to call this function with a same text and a same face at every frame.
//
// This function is concurrent-safe.
func Draw(dst *ebiten.Image, face font.Face, text string, x, y int, lineHeight int, clr color.Color) {
	textM.Lock()

	n := now()
	fx := fixed.I(x)
	ofx := fx
	prevC := rune(-1)

	runes := []rune(text)
	for _, c := range runes {
		// TODO: What if c is '\r'?
		if c == '\n' {
			fx = ofx
			y += lineHeight
			prevC = rune(-1)
			continue
		}

		if prevC >= 0 {
			fx += face.Kern(prevC, c)
		}

		if g := getGlyphFromCache(face, c, n); g != nil {
			if !g.empty() {
				g.draw(dst, fx.Ceil(), y, clr)
			}
			a, _ := face.GlyphAdvance(c)
			fx += a
		}
		prevC = c
	}

	textM.Unlock()
}
