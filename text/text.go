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
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten"
)

var (
	monotonicClock int64
)

func now() int64 {
	monotonicClock++
	return monotonicClock
}

func fixed26_6ToFloat64(x fixed.Int26_6) float64 {
	return float64(x) / (1 << 6)
}

const (
	cacheLimit = 512 // This is an arbitrary number.
)

type colorMCacheKey uint32

type colorMCacheEntry struct {
	m     ebiten.ColorM
	atime int64
}

func drawGlyph(dst *ebiten.Image, face font.Face, r rune, img *glyphImage, x, y fixed.Int26_6, clr ebiten.ColorM) {
	if img == nil {
		return
	}

	b := getGlyphBounds(face, r)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(fixed26_6ToFloat64(x+b.Min.X), fixed26_6ToFloat64(y+b.Min.Y))

	op.ColorM = clr
	re := image.Rect(img.x, img.y, img.x+img.width, img.y+img.height)
	op.SourceRect = &re

	_ = dst.DrawImage(img.image, op)
}

var (
	// Use pointers to avoid copying on browsers.
	glyphBoundsCache = map[font.Face]map[rune]*fixed.Rectangle26_6{}
)

func getGlyphBounds(face font.Face, r rune) *fixed.Rectangle26_6 {
	if _, ok := glyphBoundsCache[face]; !ok {
		glyphBoundsCache[face] = map[rune]*fixed.Rectangle26_6{}
	}
	if b, ok := glyphBoundsCache[face][r]; ok {
		return b
	}
	b, _, _ := face.GlyphBounds(r)
	glyphBoundsCache[face][r] = &b
	return &b
}

type glyphImage struct {
	image  *ebiten.Image
	x      int
	y      int
	width  int
	height int
}

type glyphImageCacheEntry struct {
	image *glyphImage
	atime int64
}

var (
	glyphImageCache = map[font.Face]map[rune]*glyphImageCacheEntry{}
	emptyGlyphs     = map[font.Face]map[rune]struct{}{}
)

func getGlyphImages(face font.Face, runes []rune) []*glyphImage {
	if _, ok := emptyGlyphs[face]; !ok {
		emptyGlyphs[face] = map[rune]struct{}{}
	}
	if _, ok := glyphImageCache[face]; !ok {
		glyphImageCache[face] = map[rune]*glyphImageCacheEntry{}
	}

	imgs := make([]*glyphImage, len(runes))
	glyphBounds := map[rune]*fixed.Rectangle26_6{}
	neededGlyphIndices := map[int]rune{}
	for i, r := range runes {
		if _, ok := emptyGlyphs[face][r]; ok {
			continue
		}

		if e, ok := glyphImageCache[face][r]; ok {
			e.atime = now()
			imgs[i] = e.image
			continue
		}

		b := getGlyphBounds(face, r)
		w, h := (b.Max.X - b.Min.X).Ceil(), (b.Max.Y - b.Min.Y).Ceil()
		if w == 0 || h == 0 {
			emptyGlyphs[face][r] = struct{}{}
			continue
		}

		// TODO: What if len(runes) > cacheLimit?
		if len(glyphImageCache[face]) > cacheLimit {
			oldest := int64(math.MaxInt64)
			oldestKey := rune(-1)
			for r, e := range glyphImageCache[face] {
				if e.atime < oldest {
					oldestKey = r
					oldest = e.atime
				}
			}
			delete(glyphImageCache[face], oldestKey)
		}

		glyphBounds[r] = b
		neededGlyphIndices[i] = r
	}

	if len(neededGlyphIndices) > 0 {
		// TODO: What if w2 is too big (e.g. > 4096)?
		w2 := 0
		h2 := 0
		for _, b := range glyphBounds {
			w, h := (b.Max.X - b.Min.X).Ceil(), (b.Max.Y - b.Min.Y).Ceil()
			w2 += w
			if h2 < h {
				h2 = h
			}
		}
		rgba := image.NewRGBA(image.Rect(0, 0, w2, h2))

		x := 0
		xs := map[rune]int{}
		for r, b := range glyphBounds {
			w := (b.Max.X - b.Min.X).Ceil()

			d := font.Drawer{
				Dst:  rgba,
				Src:  image.White,
				Face: face,
			}
			d.Dot = fixed.Point26_6{fixed.I(x) - b.Min.X, -b.Min.Y}
			d.DrawString(string(r))
			xs[r] = x

			x += w
		}

		img, _ := ebiten.NewImageFromImage(rgba, ebiten.FilterDefault)
		for i, r := range neededGlyphIndices {
			b := glyphBounds[r]
			w, h := (b.Max.X - b.Min.X).Ceil(), (b.Max.Y - b.Min.Y).Ceil()

			g := &glyphImage{
				image:  img,
				x:      xs[r],
				y:      0,
				width:  w,
				height: h,
			}
			if _, ok := glyphImageCache[face][r]; !ok {
				glyphImageCache[face][r] = &glyphImageCacheEntry{
					image: g,
					atime: now(),
				}
			}
			imgs[i] = g
		}
	}
	return imgs
}

var textM sync.Mutex

var (
	colorMCache = map[colorMCacheKey]*colorMCacheEntry{}
	emptyColorM ebiten.ColorM
)

func init() {
	emptyColorM.Scale(0, 0, 0, 0)
}

func colorToColorM(clr color.Color) ebiten.ColorM {
	// RGBA() is in [0 - 0xffff]. Adjust them in [0 - 0xff].
	cr, cg, cb, ca := clr.RGBA()
	cr >>= 8
	cg >>= 8
	cb >>= 8
	ca >>= 8
	if ca == 0 {
		return emptyColorM
	}
	key := colorMCacheKey(uint32(cr) | (uint32(cg) << 8) | (uint32(cb) << 16) | (uint32(ca) << 24))
	e, ok := colorMCache[key]
	if ok {
		e.atime = now()
		return e.m
	}
	if len(colorMCache) > cacheLimit {
		oldest := int64(math.MaxInt64)
		oldestKey := colorMCacheKey(0)
		for key, c := range colorMCache {
			if c.atime < oldest {
				oldestKey = key
				oldest = c.atime
			}
		}
		delete(colorMCache, oldestKey)
	}

	cm := ebiten.ColorM{}
	rf := float64(cr) / float64(ca)
	gf := float64(cg) / float64(ca)
	bf := float64(cb) / float64(ca)
	af := float64(ca) / 0xff
	cm.Scale(rf, gf, bf, af)
	e = &colorMCacheEntry{
		m:     cm,
		atime: now(),
	}
	colorMCache[key] = e

	return e.m
}

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
// Be careful that the passed font face is held by this package and is never released.
// This is a known issue (#498).
//
// This function is concurrent-safe.
func Draw(dst *ebiten.Image, text string, face font.Face, x, y int, clr color.Color) {
	textM.Lock()

	fx := fixed.I(x)
	prevR := rune(-1)

	runes := []rune(text)
	glyphImgs := getGlyphImages(face, runes)
	colorm := colorToColorM(clr)

	for i, r := range runes {
		if prevR >= 0 {
			fx += face.Kern(prevR, r)
		}
		drawGlyph(dst, face, r, glyphImgs[i], fx, fixed.I(y), colorm)
		fx += glyphAdvance(face, r)

		prevR = r
	}

	textM.Unlock()
}
