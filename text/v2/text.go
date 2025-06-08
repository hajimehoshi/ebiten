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
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/textutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Face is an interface representing a font face.
// The implementations are only faces defined in this package, like GoTextFace and GoXFace.
type Face interface {
	// Metrics returns the metrics for this Face.
	Metrics() Metrics

	advance(text string) float64

	hasGlyph(r rune) bool

	appendGlyphsForLine(glyphs []Glyph, line string, indexOffset int, originX, originY float64) []Glyph
	appendVectorPathForLine(path *vector.Path, line string, originX, originY float64)

	direction() Direction

	// private is an unexported function preventing being implemented by other packages.
	private()
}

// Metrics holds the metrics for a Face.
// A visual depiction is at https://developer.apple.com/library/mac/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png
type Metrics struct {
	// HLineGap is the recommended amount of vertical space between two lines of text in pixels.
	HLineGap float64

	// HAscent is the distance in pixels from the top of a line to its baseline for horizontal lines.
	HAscent float64

	// HDescent is the distance in pixels from the bottom of a line to its baseline for horizontal lines.
	// The value is typically positive, even though a descender goes below the baseline.
	HDescent float64

	// VLineGap is the recommended amount of horizontal space between two lines of text in pixels.
	// If the face is GoXFace or the font doesn't support a vertical direction, VLineGap is 0.
	VLineGap float64

	// VAscent is the distance in pixels from the top of a line to its baseline for vertical lines.
	// If the face is GoXFace or the font doesn't support a vertical direction, VAscent is 0.
	VAscent float64

	// VDescent is the distance in pixels from the top of a line to its baseline for vertical lines.
	// If the face is GoXFace or the font doesn't support a vertical direction, VDescent is 0.
	VDescent float64

	// XHeight is the distance in pixels from the baseline to the top of the lower case letters.
	XHeight float64

	// CapHeight is the distance in pixels from the baseline to the top of the capital letters.
	CapHeight float64
}

func fixed26_6ToFloat32(x fixed.Int26_6) float32 {
	return float32(x) / (1 << 6)
}

func fixed26_6ToFloat64(x fixed.Int26_6) float64 {
	return float64(x) / (1 << 6)
}

func float32ToFixed26_6(x float32) fixed.Int26_6 {
	return fixed.Int26_6(x * (1 << 6))
}

func float64ToFixed26_6(x float64) fixed.Int26_6 {
	return fixed.Int26_6(x * (1 << 6))
}

func glyphVariationCount(face Face) int {
	var s float64
	if m := face.Metrics(); face.direction().isHorizontal() {
		s = m.HAscent + m.HDescent
	} else {
		s = m.VAscent + m.VDescent
	}
	// The threshold is decided based on the rendering result of the examples (e.g. examples/text, examples/ui).
	if s < 20 {
		return 8
	}
	if s < 40 {
		return 4
	}
	if s < 80 {
		return 2
	}
	return 1
}

func adjustGranularity(x fixed.Int26_6, face Face) fixed.Int26_6 {
	c := glyphVariationCount(face)
	factor := (1 << 6) / fixed.Int26_6(c)
	return x / factor * factor
}

// Glyph represents one glyph to render.
type Glyph struct {
	// StartIndexInBytes is the start index in bytes for the given string at AppendGlyphs.
	StartIndexInBytes int

	// EndIndexInBytes is the end index in bytes for the given string at AppendGlyphs.
	EndIndexInBytes int

	// GID is an ID for a glyph of TrueType or OpenType font. GID is valid when the face is GoTextFace.
	GID uint32

	// Image is a rasterized glyph image.
	// Image is a grayscale image i.e. RGBA values are the same.
	//
	// Image should be used as a render source and must not be modified.
	//
	// Image can be nil.
	Image *ebiten.Image

	// X is the X position to render this glyph.
	// The position is determined in a sequence of characters given at AppendGlyphs.
	// The position's origin is the first character's origin position.
	X float64

	// Y is the Y position to render this glyph.
	// The position is determined in a sequence of characters given at AppendGlyphs.
	// The position's origin is the first character's origin position.
	Y float64

	// OriginX is the X position of the origin of this glyph.
	OriginX float64

	// OriginY is the Y position of the origin of this glyph.
	OriginY float64

	// OriginOffsetX is the adjustment value to the X position of the origin of this glyph.
	// OriginOffsetX is usually 0, but can be non-zero for some special glyphs or glyphs in the vertical text layout.
	OriginOffsetX float64

	// OriginOffsetY is the adjustment value to the Y position of the origin of this glyph.
	// OriginOffsetY is usually 0, but can be non-zero for some special glyphs or glyphs in the vertical text layout.
	OriginOffsetY float64
}

// Advance returns the advanced distance from the origin position when rendering the given text with the given face.
//
// Advance doesn't treat multiple lines.
//
// Advance is concurrent-safe.
func Advance(text string, face Face) float64 {
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
func Measure(text string, face Face, lineSpacingInPixels float64) (width, height float64) {
	if text == "" {
		return 0, 0
	}

	var primary float64
	var lineCount int
	for line := range textutil.Lines(text) {
		primary = max(primary, face.advance(textutil.TrimTailingLineBreak(line)))
		lineCount++
	}

	m := face.Metrics()

	if face.direction().isHorizontal() {
		secondary := float64(lineCount-1)*lineSpacingInPixels + m.HAscent + m.HDescent
		return primary, secondary
	}
	secondary := float64(lineCount-1)*lineSpacingInPixels + m.VAscent + m.VDescent
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
// If you really care about the performance, CacheGlyphs might be useful.
//
// CacheGlyphs is pretty heavy since it creates all the possible variations of glyphs.
// Call CacheGlyphs only when you really need it.
//
// CacheGlyphs is concurrent-safe.
func CacheGlyphs(text string, face Face) {
	var x, y float64

	c := glyphVariationCount(face)

	var buf []Glyph
	// Create all the possible variations (#2528).
	for i := 0; i < c; i++ {
		buf = appendGlyphs(buf, text, face, x, y, nil)
		buf = buf[:0]

		if face.direction().isHorizontal() {
			x += 1.0 / float64(c)
		} else {
			y += 1.0 / float64(c)
		}
	}
}
