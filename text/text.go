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
	"github.com/hajimehoshi/ebiten/internal/colormcache"
)

var (
	monotonicClock int64
)

func now() int64 {
	monotonicClock++
	return monotonicClock
}

func fixed26_6ToFloat64(x fixed.Int26_6) float64 {
	return float64(x>>6) + float64(x&((1<<6)-1))/float64(1<<6)
}

const (
	cacheLimit = 512 // This is an arbitrary number.
)

func drawGlyph(dst *ebiten.Image, face font.Face, r rune, img *ebiten.Image, x, y fixed.Int26_6, clr ebiten.ColorM) {
	if img == nil {
		return
	}

	b := getGlyphBounds(face, r)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64((x+b.Min.X)>>6), float64((y+b.Min.Y)>>6))
	op.ColorM = clr
	_ = dst.DrawImage(img, op)
}

var (
	// Use pointers as copying is expensive on GopherJS.
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

type glyphImageCacheEntry struct {
	image *ebiten.Image
	atime int64
}

var (
	glyphImageCache = map[font.Face]map[rune]*glyphImageCacheEntry{}
	emptyGlyphs     = map[font.Face]map[rune]struct{}{}
)

func getGlyphImages(face font.Face, runes []rune) []*ebiten.Image {
	if _, ok := emptyGlyphs[face]; !ok {
		emptyGlyphs[face] = map[rune]struct{}{}
	}
	if _, ok := glyphImageCache[face]; !ok {
		glyphImageCache[face] = map[rune]*glyphImageCacheEntry{}
	}

	imgs := make([]*ebiten.Image, len(runes))
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
		for i, r := range neededGlyphIndices {
			b := glyphBounds[r]
			w, h := (b.Max.X - b.Min.X).Ceil(), (b.Max.Y - b.Min.Y).Ceil()
			if b.Min.X&((1<<6)-1) != 0 {
				w++
			}
			if b.Min.Y&((1<<6)-1) != 0 {
				h++
			}
			rgba := image.NewRGBA(image.Rect(0, 0, w, h))

			d := font.Drawer{
				Dst:  rgba,
				Src:  image.White,
				Face: face,
			}
			x, y := -b.Min.X, -b.Min.Y
			x, y = fixed.I(x.Ceil()), fixed.I(y.Ceil())
			d.Dot = fixed.Point26_6{X: x, Y: y}
			d.DrawString(string(r))

			img, _ := ebiten.NewImageFromImage(rgba, ebiten.FilterDefault)
			if _, ok := glyphImageCache[face][r]; !ok {
				glyphImageCache[face][r] = &glyphImageCacheEntry{
					image: img,
					atime: now(),
				}
			}
			imgs[i] = img
		}
	}
	return imgs
}

var textM sync.Mutex

// Draw draws a given text on a given destination image dst.
//
// face is the font for text rendering.
// (x, y) represents a 'dot' (period) position.
// This means that if the given text consisted of a single character ".",
// it would be positioned at the given position (x, y).
// Be careful that this doesn't represent left-upper corner position.
//
// clr is the color for text rendering.
//
// If you want to adjust the position of the text, these functions are useful:
//
//     * text.BoundString:                     the rendered bounds of the given text.
//     * golang.org/x/image/font.Face.Metrics: the metrics of the face.
//
// The '\n' newline character puts the following text on the next line.
// Line height is based on Metrics().Height of the font.
//
// Glyphs used for rendering are cached in least-recently-used way.
// It is OK to call Draw with a same text and a same face at every frame in terms of performance.
//
// Be careful that the passed font face is held by this package and is never released.
// This is a known issue (#498).
//
// Draw is concurrent-safe.
func Draw(dst *ebiten.Image, text string, face font.Face, x, y int, clr color.Color) {
	textM.Lock()
	defer textM.Unlock()

	fx, fy := fixed.I(x), fixed.I(y)
	prevR := rune(-1)

	faceHeight := face.Metrics().Height

	runes := []rune(text)
	glyphImgs := getGlyphImages(face, runes)
	colorm := colormcache.ColorToColorM(clr)

	for i, r := range runes {
		if prevR >= 0 {
			fx += face.Kern(prevR, r)
		}
		if r == '\n' {
			fx = fixed.I(x)
			fy += faceHeight
			prevR = rune(-1)
			continue
		}

		drawGlyph(dst, face, r, glyphImgs[i], fx, fy, colorm)
		fx += glyphAdvance(face, r)

		prevR = r
	}
}

// BoundString returns the measured size of a given string using a given font.
// This method will return the exact size in pixels that a string drawn by Draw will be.
// The bound's origin point indicates the dot (period) position.
// This means that if the text consists of one character '.', this dot is rendered at (0, 0).
//
// This is very similar to golang.org/x/image/font's BoundString,
// but this BoundString calculates the actual rendered area considering multiple lines and space characters.
//
// face is the font for text rendering.
// text is the string that's being measured.
//
// Be careful that the passed font face is held by this package and is never released.
// This is a known issue (#498).
//
// BoundString is concurrent-safe.
func BoundString(face font.Face, text string) image.Rectangle {
	textM.Lock()
	defer textM.Unlock()

	m := face.Metrics()
	faceHeight := m.Height

	fx, fy := fixed.I(0), fixed.I(0)
	prevR := rune(-1)

	var bounds fixed.Rectangle26_6
	for _, r := range []rune(text) {
		if prevR >= 0 {
			fx += face.Kern(prevR, r)
		}
		if r == '\n' {
			fx = fixed.I(0)
			fy += faceHeight
			prevR = rune(-1)
			continue
		}

		bp := getGlyphBounds(face, r)
		b := *bp
		b.Min.X += fx
		b.Max.X += fx
		b.Min.Y += fy
		b.Max.Y += fy
		bounds = bounds.Union(b)

		fx += glyphAdvance(face, r)
		prevR = r
	}

	return image.Rect(
		int(math.Floor(fixed26_6ToFloat64(bounds.Min.X))),
		int(math.Floor(fixed26_6ToFloat64(bounds.Min.Y))),
		int(math.Ceil(fixed26_6ToFloat64(bounds.Max.X))),
		int(math.Ceil(fixed26_6ToFloat64(bounds.Max.Y))),
	)
}
