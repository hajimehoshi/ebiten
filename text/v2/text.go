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

// Package text offers functions to draw texts on an Ebitengine's image.
//
// For the example using a TrueType font, see examples in the examples directory.
package text

import (
	"math"
	"strings"

	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
)

type faceCacheKey uint64

// Face is an interface representing a font face. The implementations are only GoTextFace and StdFace.
type Face interface {
	// Metrics returns the metrics for this Face.
	Metrics() Metrics

	// UnsafeInternal returns the internal object for this face.
	// The returned value is either a semi-standard font.Face or go-text's font.Face.
	// This is unsafe since this might make internal cache states out of sync.
	UnsafeInternal() any

	faceCacheKey() faceCacheKey

	advance(text string) float64

	appendGlyphs(glyphs []Glyph, text string, originX, originY float64) []Glyph

	direction() Direction

	// private is an unexported function preventing being implemented by other packages.
	private()
}

// Metrics holds the metrics for a Face.
// A visual depiction is at https://developer.apple.com/library/mac/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png
type Metrics struct {
	// Height is the recommended amount of vertical space between two lines of text in pixels.
	Height float64

	// HAscent is the distance in pixels from the top of a line to its baseline for horizontal lines.
	HAscent float64

	// HDescent is the distance in pixels from the bottom of a line to its baseline for horizontal lines.
	// The value is typically positive, even though a descender goes below the baseline.
	HDescent float64

	// VAscent is the distance in pixels from the top of a line to its baseline for vertical lines.
	// If the face is StdFace or the font dosen't support a vertical direction, VAscent is 0.
	VAscent float64

	// VDescent is the distance in pixels from the top of a line to its baseline for vertical lines.
	// If the face is StdFace or the font dosen't support a vertical direction, VDescent is 0.
	VDescent float64
}

func fixed26_6ToFloat64(x fixed.Int26_6) float64 {
	return float64(x>>6) + float64(x&((1<<6)-1))/float64(1<<6)
}

func float64ToFixed26_6(x float64) fixed.Int26_6 {
	i := math.Floor(x)
	frac := x - i
	return fixed.Int26_6(i)<<6 + fixed.Int26_6(frac*(1<<6))
}

const glyphVariationCount = 4

func adjustOffsetGranularity(x fixed.Int26_6) fixed.Int26_6 {
	return x / ((1 << 6) / glyphVariationCount) * ((1 << 6) / glyphVariationCount)
}

// Glyph represents one glyph to render.
type Glyph struct {
	// Rune is a character for this glyph.
	Rune rune

	// IndexInBytes is an index in bytes for the given string at AppendGlyphs.
	IndexInBytes int

	// Image is a rasterized glyph image.
	// Image is a grayscale image i.e. RGBA values are the same.
	// Image should be used as a render source and should not be modified.
	Image *ebiten.Image

	// X is the X position to render this glyph.
	// The position is determined in a sequence of characters given at AppendGlyphs.
	// The position's origin is the first character's origin position.
	X float64

	// Y is the Y position to render this glyph.
	// The position is determined in a sequence of characters given at AppendGlyphs.
	// The position's origin is the first character's origin position.
	Y float64

	// GID is an ID for a glyph of TrueType or OpenType font. GID is valid when the font is GoTextFont.
	GID uint32
}

// AppendGlyphs appends glyphs to the given slice and returns a slice.
//
// AppendGlyphs is a low-level API, and you can use AppendGlyphs to have more control than Draw.
// AppendGlyphs is also available to precache glyphs.
//
// AppendGlyphs doesn't treat multiple lines.
//
// AppendGlyphs is concurrent-safe.
func AppendGlyphs(glyphs []Glyph, text string, face Face, originX, originY float64) []Glyph {
	return face.appendGlyphs(glyphs, text, originX, originY)
}

// Advance returns the advanced distance from the origin position when rendering the given text with the given face.
//
// Advance doesn't treat multiple lines.
//
// Advance is concurrent-safe.
func Advance(face Face, text string) float64 {
	return face.advance(text)
}

// Direction represents a direction of text rendering.
// Direction indicates both the primary direction, in which a text in one line is rendered,
// and the secondary direction, in which multiple lines are rendered.
type Direction int

const (
	// DirectionLeftToRight indicates that the primary direction is from left to right,
	// and the secondary direction is from top to bottom.
	DirectionLeftToRight Direction = iota

	// DirectionRightToLeft indicates that the primary direction is from right to left,
	// and the secondary direction is from top to bottom.
	DirectionRightToLeft

	// DirectionTopToBottomAndLeftToRight indicates that the primary direction is from top to bottom,
	// and the secondary direction is from left to right.
	// This is used e.g. for Mongolian.
	DirectionTopToBottomAndLeftToRight

	// DirectionTopToBottomAndRightToLeft indicates that the primary direction is from top to bottom,
	// and the secondary direction is from right to left.
	// This is used e.g. for Japanese.
	DirectionTopToBottomAndRightToLeft
)

func (d Direction) isHorizontal() bool {
	switch d {
	case DirectionLeftToRight, DirectionRightToLeft:
		return true
	}
	return false
}

// Measure measures the boundary size of the text.
// With a horizontal direction face, the width is the longest line's advance, and the height is the total of line heights.
// With a vertical direction face, the width and the height are calculated in an opposite manner.
//
// Measure is concurrent-safe.
func Measure(text string, face Face, lineHeightInPixels float64) (width, height float64) {
	if text == "" {
		return 0, 0
	}

	var primary float64
	var lineCount int
	for t := text; ; {
		lineCount++
		line, rest, found := strings.Cut(t, "\n")
		a := face.advance(line)
		if primary < a {
			primary = a
		}
		if !found {
			break
		}
		t = rest
	}

	m := face.Metrics()

	if face.direction().isHorizontal() {
		secondary := float64(lineCount-1)*lineHeightInPixels + m.HAscent + m.HDescent
		return primary, secondary
	}
	secondary := float64(lineCount-1)*lineHeightInPixels + m.VAscent + m.VDescent
	return secondary, primary
}

// CacheGlyphs pre-caches the glyphs for the given text and the given font face into the cache.
//
// CacheGlyphs doesn't treat multiple lines.
//
// Glyphs used for rendering are cached in the least-recently-used way.
// Then old glyphs might be evicted from the cache.
// As the cache capacity has limitations, it is not guaranteed that all the glyphs for runes given at CacheGlyphs are cached.
// The cache is shared with Draw and AppendGlyphs.
//
// One rune can have multiple variations of glyphs due to sub-pixels in X or Y direction.
// CacheGlyphs creates all such variations for one rune, while Draw and AppendGlyphs create only necessary glyphs.
//
// Draw and AppendGlyphs automatically create and cache necessary glyphs, so usually you don't have to call CacheGlyphs explicitly.
// However, for example, when you call Draw for each rune of one big text, Draw tries to create the glyph cache and render it for each rune.
// This is very inefficient because creating a glyph image and rendering it are different operations
// (`(*ebiten.Image).WritePixels` and `(*ebiten.Image).DrawImage`) and can never be merged as one draw call.
// CacheGlyphs creates necessary glyphs without rendering them so that these operations are likely merged into one draw call regardless of the size of the text.
//
// CacheGlyphs is concurrent-safe.
func CacheGlyphs(text string, face Face) {
	var x, y float64

	var buf []Glyph
	// Create all the possible variations (#2528).
	for i := 0; i < 4; i++ {
		buf = AppendGlyphs(buf, text, face, x, y)
		buf = buf[:0]

		if face.direction().isHorizontal() {
			x += 1.0 / glyphVariationCount
		} else {
			y += 1.0 / glyphVariationCount
		}
	}
}
