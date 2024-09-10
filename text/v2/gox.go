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
// Unlike GoFontFace, one GoXFace instance has its own glyph image cache.
// You should reuse the same GoXFace instance as much as possible.
type GoXFace struct {
	f *faceWithCache

	glyphImageCache glyphImageCache[goXFaceGlyphImageCacheKey]

	cachedMetrics Metrics

	addr *GoXFace
}

// NewGoXFace creates a new GoXFace from a semi-standard font.Face.
func NewGoXFace(face font.Face) *GoXFace {
	s := &GoXFace{
		f: &faceWithCache{
			f: face,
		},
	}
	s.addr = s
	return s
}

func (s *GoXFace) copyCheck() {
	if s.addr != s {
		panic("text: illegal use of non-zero GoXFace copied by value")
	}
}

// Metrics implements Face.
func (s *GoXFace) Metrics() Metrics {
	s.copyCheck()

	if s.cachedMetrics != (Metrics{}) {
		return s.cachedMetrics
	}

	fm := s.f.Metrics()
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
	s.cachedMetrics = m
	return m
}

// UnsafeInternal returns its internal font.Face.
//
// UnsafeInternal is unsafe since this might make internal cache states out of sync.
//
// UnsafeInternal might have breaking changes even in the same major version.
func (s *GoXFace) UnsafeInternal() font.Face {
	s.copyCheck()
	return s.f.f
}

// advance implements Face.
func (s *GoXFace) advance(text string) float64 {
	return fixed26_6ToFloat64(font.MeasureString(s.f, text))
}

// hasGlyph implements Face.
func (s *GoXFace) hasGlyph(r rune) bool {
	_, ok := s.f.GlyphAdvance(r)
	return ok
}

// appendGlyphsForLine implements Face.
func (s *GoXFace) appendGlyphsForLine(glyphs []Glyph, line string, indexOffset int, originX, originY float64) []Glyph {
	s.copyCheck()

	origin := fixed.Point26_6{
		X: float64ToFixed26_6(originX),
		Y: float64ToFixed26_6(originY),
	}
	prevR := rune(-1)

	for i, r := range line {
		if prevR >= 0 {
			origin.X += s.f.Kern(prevR, r)
		}
		img, imgX, imgY, a := s.glyphImage(r, origin)

		// Adjust the position to the integers.
		// The current glyph images assume that they are rendered on integer positions so far.
		_, size := utf8.DecodeRuneInString(line[i:])

		// Append a glyph even if img is nil.
		// This is necessary to return index information for control characters.
		glyphs = append(glyphs, Glyph{
			StartIndexInBytes: indexOffset + i,
			EndIndexInBytes:   indexOffset + i + size,
			Image:             img,
			X:                 float64(imgX),
			Y:                 float64(imgY),
			OriginX:           fixed26_6ToFloat64(origin.X),
			OriginY:           fixed26_6ToFloat64(origin.Y),
			OriginOffsetX:     0,
			OriginOffsetY:     0,
		})
		origin.X += a
		prevR = r
	}

	return glyphs
}

func (s *GoXFace) glyphImage(r rune, origin fixed.Point26_6) (*ebiten.Image, int, int, fixed.Int26_6) {
	// Assume that GoXFace's direction is always horizontal.
	origin.X = adjustGranularity(origin.X, s)
	origin.Y &^= ((1 << 6) - 1)

	b, a, _ := s.f.GlyphBounds(r)
	subpixelOffset := fixed.Point26_6{
		X: (origin.X + b.Min.X) & ((1 << 6) - 1),
		Y: (origin.Y + b.Min.Y) & ((1 << 6) - 1),
	}
	key := goXFaceGlyphImageCacheKey{
		rune:    r,
		xoffset: subpixelOffset.X,
	}
	img := s.glyphImageCache.getOrCreate(s, key, func() *ebiten.Image {
		return s.glyphImageImpl(r, subpixelOffset, b)
	})
	imgX := (origin.X + b.Min.X).Floor()
	imgY := (origin.Y + b.Min.Y).Floor()
	return img, imgX, imgY, a
}

func (s *GoXFace) glyphImageImpl(r rune, subpixelOffset fixed.Point26_6, glyphBounds fixed.Rectangle26_6) *ebiten.Image {
	w, h := (glyphBounds.Max.X - glyphBounds.Min.X).Ceil(), (glyphBounds.Max.Y - glyphBounds.Min.Y).Ceil()
	if w == 0 || h == 0 {
		return nil
	}

	// Add always 1 to the size.
	// In theory, it is possible to determine whether +1 is necessary or not, but the calculation is pretty complicated.
	w++
	h++

	rgba := image.NewRGBA(image.Rect(0, 0, w, h))

	d := font.Drawer{
		Dst:  rgba,
		Src:  image.White,
		Face: s.f,
		Dot: fixed.Point26_6{
			X: -glyphBounds.Min.X + subpixelOffset.X,
			Y: -glyphBounds.Min.Y + subpixelOffset.Y,
		},
	}
	d.DrawString(string(r))

	return ebiten.NewImageFromImage(rgba)
}

// direction implements Face.
func (s *GoXFace) direction() Direction {
	return DirectionLeftToRight
}

// appendVectorPathForLine implements Face.
func (s *GoXFace) appendVectorPathForLine(path *vector.Path, line string, originX, originY float64) {
}

// Metrics implements Face.
func (s *GoXFace) private() {
}
