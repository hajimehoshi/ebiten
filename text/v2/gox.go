// Copyright 2023 The Ebitengine Authors
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

package text

import (
	"image"
	"slices"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var _ Face = (*GoXFace)(nil)

type goXFaceGlyphImageCacheKey struct {
	rune    rune
	xoffset fixed.Int26_6

	// yoffset is always the same if the rune is the same, so this doesn't have to be a key.
}

// GoXFace is a Face implementation for a semi-standard font.Face (golang.org/x/image/font).
// GoXFace is useful to transit from existing codebase with text v1, or to use some bitmap fonts defined as font.Face.
// GoXFace must not be copied by value.
//
// Unlike GoTextFace, one GoXFace instance has its own glyph image cache.
// You should reuse the same GoXFace instance as much as possible.
type GoXFace struct {
	f *faceWithCache

	glyphImageCache *cache[goXFaceGlyphImageCacheKey, *ebiten.Image]

	cachedMetrics Metrics

	originXCache *cache[string, []fixed.Int26_6]

	addr *GoXFace
}

// NewGoXFace creates a new GoXFace from a semi-standard font.Face.
func NewGoXFace(face font.Face) *GoXFace {
	g := &GoXFace{
		f: &faceWithCache{
			f: face,
		},
	}
	// Set addr as early as possible. This is necessary for glyphVariationCount.
	g.addr = g
	g.glyphImageCache = newCache[goXFaceGlyphImageCacheKey, *ebiten.Image](128 * glyphVariationCount(g))
	g.originXCache = newCache[string, []fixed.Int26_6](512)
	return g
}

func (g *GoXFace) copyCheck() {
	if g.addr != g {
		panic("text: illegal use of non-zero GoXFace copied by value")
	}
}

// Metrics implements Face.
func (g *GoXFace) Metrics() Metrics {
	g.copyCheck()

	if g.cachedMetrics != (Metrics{}) {
		return g.cachedMetrics
	}

	fm := g.f.Metrics()
	m := Metrics{
		HLineGap:  fixed26_6ToFloat64(fm.Height - fm.Ascent - fm.Descent),
		HAscent:   fixed26_6ToFloat64(fm.Ascent),
		HDescent:  fixed26_6ToFloat64(fm.Descent),
		XHeight:   fixed26_6ToFloat64(fm.XHeight),
		CapHeight: fixed26_6ToFloat64(fm.CapHeight),
	}

	// There is an issue that XHeight and CapHeight are negative for some old fonts (golang/go#69378).
	if fm.XHeight < 0 {
		m.XHeight *= -1
	}
	if fm.CapHeight < 0 {
		m.CapHeight *= -1
	}
	g.cachedMetrics = m
	return m
}

// UnsafeInternal returns its internal font.Face.
//
// UnsafeInternal is unsafe since this might make internal cache states out of sync.
//
// UnsafeInternal might have breaking changes even in the same major version.
func (g *GoXFace) UnsafeInternal() font.Face {
	g.copyCheck()
	return g.f.f
}

// advance implements Face.
func (g *GoXFace) advance(text string) float64 {
	xs := g.originXs(text)
	if len(xs) == 0 {
		return 0
	}
	return fixed26_6ToFloat64(xs[len(xs)-1])
}

func (g *GoXFace) originXs(text string) []fixed.Int26_6 {
	return g.originXCache.getOrCreate(text, func() ([]fixed.Int26_6, bool) {
		if len(text) == 0 {
			return nil, false
		}

		var originXs []fixed.Int26_6
		prevR := rune(-1)
		var originX fixed.Int26_6
		for _, r := range text {
			if prevR >= 0 {
				originX += g.f.Kern(prevR, r)
				originXs = append(originXs, originX)
			}
			a, _ := g.f.GlyphAdvance(r)
			originX += a
			prevR = r
		}
		originXs = append(originXs, originX)
		return originXs, true
	})
}

// hasGlyph implements Face.
func (g *GoXFace) hasGlyph(r rune) bool {
	_, ok := g.f.GlyphAdvance(r)
	return ok
}

// appendLazyGlyphsForLine implements Face.
func (g *GoXFace) appendLazyGlyphsForLine(glyphs []LazyGlyph, line string, indexOffset int, originX, originY float64, keepGlyph func(originX, originY float64) bool) []LazyGlyph {
	g.copyCheck()

	origin := fixed.Point26_6{
		X: float64ToFixed26_6(originX),
		Y: float64ToFixed26_6(originY),
	}
	ox := origin.X

	originXs := g.originXs(line)
	var advanceIndex int
	granularity := granularityFactor(g)

	glyphs = slices.Grow(glyphs, len(line))

	// imager is allocated lazily on the first glyph that produces an image.
	var imager *goXLineImager

	for i, r := range line {
		if i > 0 {
			origin.X = ox + originXs[advanceIndex]
			advanceIndex++
		}

		if keepGlyph != nil && !keepGlyph(fixed26_6ToFloat64(origin.X), fixed26_6ToFloat64(origin.Y)) {
			continue
		}

		// The image position is integer so that the nearest filter can be used.
		bounds, args, hasImage := goXGlyphImageInfo(g, r, origin, granularity)

		// Adjust the position to the integers.
		// The current glyph images assume that they are rendered on integer positions so far.
		// Do not use utf8.RuneLen here, as r may be U+FFFD (replacement character)
		// when the line contains invalid UTF-8 sequences (#3284).
		_, size := utf8.DecodeRuneInString(line[i:])
		if size < 0 {
			// A string for-loop iterator advances by 1 byte when it encounters an invalid UTF-8 sequence.
			size = 1
		}

		var prevOriginX fixed.Int26_6
		if advanceIndex > 0 {
			prevOriginX = originXs[advanceIndex-1]
		}
		advanceX := fixed26_6ToFloat64(originXs[advanceIndex] - prevOriginX)

		lg := LazyGlyph{
			StartIndexInBytes: indexOffset + i,
			EndIndexInBytes:   indexOffset + i + size,
			ImageBounds:       bounds,
			OriginX:           fixed26_6ToFloat64(origin.X),
			OriginY:           fixed26_6ToFloat64(origin.Y),
			OriginOffsetX:     0,
			OriginOffsetY:     0,
			AdvanceX:          advanceX,
			AdvanceY:          0,
		}
		if hasImage {
			if imager == nil {
				imager = &goXLineImager{face: g}
			}
			imager.args = append(imager.args, args)
			lg.imager = imager
			lg.imageIndex = len(imager.args) - 1
		}
		// Append a glyph even if it has no image (control characters etc.).
		// This is necessary to return index information for control characters.
		glyphs = append(glyphs, lg)
	}

	return glyphs
}

// goXLineImager owns a per-call slice of per-glyph args for a single
// invocation of [GoXFace.appendLazyGlyphsForLine]. It satisfies
// [glyphImager].
type goXLineImager struct {
	face *GoXFace
	args []goXGlyphImageArgs
}

// goXGlyphImageArgs is the per-glyph data needed by
// [goXLineImager.glyphImage]. Unlike GoTextFace there is no pre-existing
// per-glyph cache to point into, so the args are inlined.
type goXGlyphImageArgs struct {
	rune           rune
	subpixelOffset fixed.Point26_6
	bounds         fixed.Rectangle26_6
}

// glyphImage implements glyphImager.
func (im *goXLineImager) glyphImage(index int) *ebiten.Image {
	args := &im.args[index]
	face := im.face
	key := goXFaceGlyphImageCacheKey{
		rune:    args.rune,
		xoffset: args.subpixelOffset.X,
	}
	return face.glyphImageCache.getOrCreate(key, func() (*ebiten.Image, bool) {
		img := face.glyphImageImpl(args.rune, args.subpixelOffset, args.bounds)
		return img, img != nil
	})
}

// goXGlyphImageInfo returns the image bounds in layout space and the
// per-glyph args needed to realize the image. The bounds are computed
// without rasterizing. hasImage is false for runes that produce no image;
// in that case bounds is empty and args is the zero value.
//
// granularity is taken as a parameter rather than derived from face
// because the caller is expected to compute it once per layout pass
// (see granularityFactor) and share it across every glyph.
func goXGlyphImageInfo(face *GoXFace, r rune, origin fixed.Point26_6, granularity fixed.Int26_6) (bounds image.Rectangle, args goXGlyphImageArgs, hasImage bool) {
	// Assume that GoXFace's direction is always horizontal.
	origin.X = adjustGranularity(origin.X, granularity)
	origin.Y &^= ((1 << 6) - 1)

	b, _, _ := face.f.GlyphBounds(r)
	subpixelOffset := fixed.Point26_6{
		X: (origin.X + b.Min.X) & ((1 << 6) - 1),
		Y: (origin.Y + b.Min.Y) & ((1 << 6) - 1),
	}

	imgX := (origin.X + b.Min.X).Floor()
	imgY := (origin.Y + b.Min.Y).Floor()

	rw := (b.Max.X - b.Min.X).Ceil()
	rh := (b.Max.Y - b.Min.Y).Ceil()
	if rw == 0 || rh == 0 {
		return
	}
	bounds = image.Rect(imgX, imgY, imgX+rw+1, imgY+rh+1)

	args = goXGlyphImageArgs{
		rune:           r,
		subpixelOffset: subpixelOffset,
		bounds:         b,
	}
	hasImage = true
	return
}

func (g *GoXFace) glyphImageImpl(r rune, subpixelOffset fixed.Point26_6, glyphBounds fixed.Rectangle26_6) *ebiten.Image {
	w, h := (glyphBounds.Max.X - glyphBounds.Min.X).Ceil(), (glyphBounds.Max.Y - glyphBounds.Min.Y).Ceil()
	if w == 0 || h == 0 {
		return nil
	}

	// Add always 1 to the size.
	// In theory, it is possible to determine whether +1 is necessary or not, but the calculation is pretty complicated.
	w++
	h++

	rgba := newPooledRGBA(w, h)
	defer releasePooledRGBA(rgba)

	d := font.Drawer{
		Dst:  rgba,
		Src:  image.White,
		Face: g.f,
		Dot: fixed.Point26_6{
			X: -glyphBounds.Min.X + subpixelOffset.X,
			Y: -glyphBounds.Min.Y + subpixelOffset.Y,
		},
	}
	d.DrawString(string(r))

	return ebiten.NewImageFromImage(rgba)
}

// direction implements Face.
func (g *GoXFace) direction() Direction {
	return DirectionLeftToRight
}

// appendVectorPathForLine implements Face.
func (g *GoXFace) appendVectorPathForLine(path *vector.Path, line string, originX, originY float64) {
}

// Metrics implements Face.
func (g *GoXFace) private() {
}
