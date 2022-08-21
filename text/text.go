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

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

var (
	monotonicClock int64
)

func now() int64 {
	return monotonicClock
}

func init() {
	hooks.AppendHookOnBeforeUpdate(func() error {
		monotonicClock++
		return nil
	})
}

func fixed26_6ToFloat64(x fixed.Int26_6) float64 {
	return float64(x>>6) + float64(x&((1<<6)-1))/float64(1<<6)
}

func drawGlyph(dst *ebiten.Image, face font.Face, r rune, img *ebiten.Image, dx, dy fixed.Int26_6, op *ebiten.DrawImageOptions) {
	if img == nil {
		return
	}

	b := getGlyphBounds(face, r)
	op2 := &ebiten.DrawImageOptions{}
	if op != nil {
		*op2 = *op
		op2.GeoM.Reset()
	}
	op2.GeoM.Translate(math.Floor(fixed26_6ToFloat64(dx+b.Min.X)), math.Floor(fixed26_6ToFloat64(dy+b.Min.Y)))
	if op != nil {
		op2.GeoM.Concat(op.GeoM)
	}
	dst.DrawImage(img, op2)
}

var (
	glyphBoundsCache = map[font.Face]map[rune]fixed.Rectangle26_6{}
)

func getGlyphBounds(face font.Face, r rune) fixed.Rectangle26_6 {
	if _, ok := glyphBoundsCache[face]; !ok {
		glyphBoundsCache[face] = map[rune]fixed.Rectangle26_6{}
	}
	if b, ok := glyphBoundsCache[face][r]; ok {
		return b
	}
	b, _, _ := face.GlyphBounds(r)
	glyphBoundsCache[face][r] = b
	return b
}

type glyphImageCacheEntry struct {
	image *ebiten.Image
	atime int64
}

var (
	glyphImageCache = map[font.Face]map[rune]*glyphImageCacheEntry{}
)

func getGlyphImage(face font.Face, r rune) *ebiten.Image {
	if _, ok := glyphImageCache[face]; !ok {
		glyphImageCache[face] = map[rune]*glyphImageCacheEntry{}
	}

	if e, ok := glyphImageCache[face][r]; ok {
		e.atime = now()
		return e.image
	}

	b := getGlyphBounds(face, r)
	w, h := (b.Max.X - b.Min.X).Ceil(), (b.Max.Y - b.Min.Y).Ceil()
	if w == 0 || h == 0 {
		glyphImageCache[face][r] = &glyphImageCacheEntry{
			image: nil,
			atime: now(),
		}
		return nil
	}

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

	img := ebiten.NewImageFromImage(rgba)
	if _, ok := glyphImageCache[face][r]; !ok {
		glyphImageCache[face][r] = &glyphImageCacheEntry{
			image: img,
			atime: now(),
		}
	}

	return img
}

var textM sync.Mutex

// Draw draws a given text on a given destination image dst.
//
// face is the font for text rendering.
// (x, y) represents a 'dot' (period) position.
// This means that if the given text consisted of a single character ".",
// it would be positioned at the given position (x, y).
// Be careful that this doesn't represent upper-left corner position.
//
// clr is the color for text rendering.
//
// If you want to adjust the position of the text, these functions are useful:
//
//   - text.BoundString:                     the rendered bounds of the given text.
//   - golang.org/x/image/font.Face.Metrics: the metrics of the face.
//
// The '\n' newline character puts the following text on the next line.
// Line height is based on Metrics().Height of the font.
//
// Glyphs used for rendering are cached in least-recently-used way.
// Then old glyphs might be evicted from the cache.
// As the cache capacity has limit, it is not guaranteed that all the glyphs for runes given at Draw are cached.
// The cache is shared with CacheGlyphs.
//
// It is OK to call Draw with a same text and a same face at every frame in terms of performance.
//
// Draw/DrawWithOptions and CacheGlyphs are implemented like this:
//
//	Draw        = Create glyphs by `(*ebiten.Image).WritePixels` and put them into the cache if necessary
//	            + Draw them onto the destination by `(*ebiten.Image).DrawImage`
//	CacheGlyphs = Create glyphs by `(*ebiten.Image).WritePixels` and put them into the cache if necessary
//
// Be careful that the passed font face is held by this package and is never released.
// This is a known issue (#498).
//
// Draw is concurrent-safe.
func Draw(dst *ebiten.Image, text string, face font.Face, x, y int, clr color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorM.ScaleWithColor(clr)
	DrawWithOptions(dst, text, face, op)
}

// DrawWithOptions draws a given text on a given destination image dst.
//
// face is the font for text rendering.
// op is the options to draw glyph images.
// The origin point is a 'dot' (period) position.
// Be careful that the origin point is not upper-left corner position of dst.
// The default glyph color is while. op's ColorM adjusts the color.
//
// If you want to adjust the position of the text, these functions are useful:
//
//   - text.BoundString:                     the rendered bounds of the given text.
//   - golang.org/x/image/font.Face.Metrics: the metrics of the face.
//
// The '\n' newline character puts the following text on the next line.
// Line height is based on Metrics().Height of the font.
//
// Glyphs used for rendering are cached in least-recently-used way.
// Then old glyphs might be evicted from the cache.
// As the cache capacity has limit, it is not guaranteed that all the glyphs for runes given at DrawWithOptions are cached.
// The cache is shared with CacheGlyphs.
//
// It is OK to call DrawWithOptions with a same text and a same face at every frame in terms of performance.
//
// Draw/DrawWithOptions and CacheGlyphs are implemented like this:
//
//	Draw        = Create glyphs by `(*ebiten.Image).WritePixels` and put them into the cache if necessary
//	            + Draw them onto the destination by `(*ebiten.Image).DrawImage`
//	CacheGlyphs = Create glyphs by `(*ebiten.Image).WritePixels` and put them into the cache if necessary
//
// Be careful that the passed font face is held by this package and is never released.
// This is a known issue (#498).
//
// DrawWithOptions is concurrent-safe.
func DrawWithOptions(dst *ebiten.Image, text string, face font.Face, options *ebiten.DrawImageOptions) {
	textM.Lock()
	defer textM.Unlock()

	var dx, dy fixed.Int26_6
	prevR := rune(-1)

	faceHeight := face.Metrics().Height

	for _, r := range text {
		if prevR >= 0 {
			dx += face.Kern(prevR, r)
		}
		if r == '\n' {
			dx = 0
			dy += faceHeight
			prevR = rune(-1)
			continue
		}

		img := getGlyphImage(face, r)
		drawGlyph(dst, face, r, img, dx, dy, options)
		dx += glyphAdvance(face, r)

		prevR = r
	}

	// cacheSoftLimit indicates the soft limit of the number of glyphs in the cache.
	// If the number of glyphs exceeds this soft limits, old glyphs are removed.
	// Even after clearning up the cache, the number of glyphs might still exceeds the soft limit, but
	// this is fine.
	const cacheSoftLimit = 512

	// Clean up the cache.
	if len(glyphImageCache[face]) > cacheSoftLimit {
		for r, e := range glyphImageCache[face] {
			// 60 is an arbitrary number.
			if e.atime < now()-60 {
				delete(glyphImageCache[face], r)
			}
		}
	}
}

// BoundString returns the measured size of a given string using a given font.
// This method will return the exact size in pixels that a string drawn by Draw will be.
// The bound's origin point indicates the dot (period) position.
// This means that if the text consists of one character '.', this dot is rendered at (0, 0).
//
// BoundString behaves almost exactly like golang.org/x/image/font's BoundString,
// but newline characters '\n' in the input string move the text position to the following line.
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
	for _, r := range text {
		if prevR >= 0 {
			fx += face.Kern(prevR, r)
		}
		if r == '\n' {
			fx = fixed.I(0)
			fy += faceHeight
			prevR = rune(-1)
			continue
		}

		b := getGlyphBounds(face, r)
		b.Min.X += fx
		b.Max.X += fx
		b.Min.Y += fy
		b.Max.Y += fy
		bounds = bounds.Union(b)

		fx += glyphAdvance(face, r)
		prevR = r
	}

	return image.Rect(
		bounds.Min.X.Floor(),
		bounds.Min.Y.Floor(),
		bounds.Max.X.Ceil(),
		bounds.Max.Y.Ceil(),
	)
}

// CacheGlyphs precaches the glyphs for the given text and the given font face into the cache.
//
// Glyphs used for rendering are cached in least-recently-used way.
// Then old glyphs might be evicted from the cache.
// As the cache capacity has limit, it is not guaranteed that all the glyphs for runes given at CacheGlyphs are cached.
// The cache is shared with Draw.
//
// Draw/DrawWithOptions and CacheGlyphs are implemented like this:
//
//	Draw        = Create glyphs by `(*ebiten.Image).WritePixels` and put them into the cache if necessary
//	            + Draw them onto the destination by `(*ebiten.Image).DrawImage`
//	CacheGlyphs = Create glyphs by `(*ebiten.Image).WritePixels` and put them into the cache if necessary
//
// Draw automatically creates and caches necessary glyphs, so usually you don't have to call CacheGlyphs
// explicitly. However, for example, when you call Draw for each rune of one big text, Draw tries to create the glyph
// cache and render it for each rune. This is very inefficient because creating a glyph image and rendering it are
// different operations (`(*ebiten.Image).WritePixels` and `(*ebiten.Image).DrawImage`) and can never be merged as
// one draw call. CacheGlyphs creates necessary glyphs without rendering them so that these operations are likely
// merged into one draw call regardless of the size of the text.
//
// If a rune's glyph is already cached, CacheGlyphs does nothing for the rune.
func CacheGlyphs(face font.Face, text string) {
	textM.Lock()
	defer textM.Unlock()

	for _, r := range text {
		getGlyphImage(face, r)
	}
}

// FaceWithLineHeight returns a font.Face with the given lineHeight in pixels.
// The returned face will otherwise have the same glyphs and metrics as face.
func FaceWithLineHeight(face font.Face, lineHeight float64) font.Face {
	return faceWithLineHeight{
		face:       face,
		lineHeight: fixed.Int26_6(lineHeight * (1 << 6)),
	}
}

type faceWithLineHeight struct {
	face       font.Face
	lineHeight fixed.Int26_6
}

func (f faceWithLineHeight) Close() error {
	return f.face.Close()
}

func (f faceWithLineHeight) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	return f.face.Glyph(dot, r)
}

func (f faceWithLineHeight) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	return f.face.GlyphBounds(r)
}

func (f faceWithLineHeight) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	return f.face.GlyphAdvance(r)
}

func (f faceWithLineHeight) Kern(r0, r1 rune) fixed.Int26_6 {
	return f.face.Kern(r0, r1)
}

func (f faceWithLineHeight) Metrics() font.Metrics {
	m := f.face.Metrics()
	m.Height = f.lineHeight
	return m
}

// Glyphs is information to render one glyph.
type Glyph struct {
	// Rune is a character for this glyph.
	Rune rune

	// Image is an image for this glyph.
	// Image is a grayscale image i.e. RGBA values are the same.
	// Image should be used as a render source and should not be modified.
	Image *ebiten.Image

	// X is the X position to render this glyph.
	// The position is determined in a sequence of characters given at AppendGlyphs.
	// The position's origin is the first character's dot ('.') position.
	X float64

	// Y is the Y position to render this glyph.
	// The position is determined in a sequence of characters given at AppendGlyphs.
	// The position's origin is the first character's dot ('.') position.
	Y float64
}

// AppendGlyphs appends the glyph information to glyphs.
// You can render each glyphs as you like. See examples/text for an example of AppendGlyphs.
func AppendGlyphs(glyphs []Glyph, face font.Face, text string) []Glyph {
	textM.Lock()
	defer textM.Unlock()

	var pos fixed.Point26_6
	prevR := rune(-1)

	faceHeight := face.Metrics().Height

	for _, r := range text {
		if prevR >= 0 {
			pos.X += face.Kern(prevR, r)
		}
		if r == '\n' {
			pos.X = 0
			pos.Y += faceHeight
			prevR = rune(-1)
			continue
		}

		if img := getGlyphImage(face, r); img != nil {
			b := getGlyphBounds(face, r)
			glyphs = append(glyphs, Glyph{
				Rune:  r,
				Image: img,
				X:     math.Floor(fixed26_6ToFloat64(pos.X + b.Min.X)),
				Y:     math.Floor(fixed26_6ToFloat64(pos.Y + b.Min.Y)),
			})
		}
		pos.X += glyphAdvance(face, r)

		prevR = r
	}

	return glyphs
}
