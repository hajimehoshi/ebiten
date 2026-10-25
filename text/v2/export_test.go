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

	"github.com/go-text/typesetting/bidi"
	"github.com/go-text/typesetting/font"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
)

func L2VisualOrder(levels []bidi.Level) []int {
	return appendL2VisualOrder(nil, levels)
}

func Fixed26_6ToFloat32(x fixed.Int26_6) float32 {
	return fixed26_6ToFloat32(x)
}

func Fixed26_6ToFloat64(x fixed.Int26_6) float64 {
	return fixed26_6ToFloat64(x)
}

func Float32ToFixed26_6(x float32) fixed.Int26_6 {
	return float32ToFixed26_6(x)
}

func Float64ToFixed26_6(x float64) fixed.Int26_6 {
	return float64ToFixed26_6(x)
}

type RuneToBoolMap = runeToBoolMap

func (rtb *RuneToBoolMap) Set(rune rune, value bool) {
	rtb.set(rune, value)
}

func (rtb *RuneToBoolMap) Get(rune rune) (bool, bool) {
	return rtb.get(rune)
}

// SVGGlyphDocument extracts the SVG document describing only the glyph
// gid from an OpenType SVG document.
func SVGGlyphDocument(source []byte, gid uint32) []byte {
	return newSVGDocIndex(source).glyphDocument(gid)
}

// GlyphImageCacheCount returns the number of the glyph image caches of the source, one per face size.
func GlyphImageCacheCount(source *GoTextFaceSource) int {
	c := &source.glyphImageCache
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.caches)
}

// HasGlyphImageCache reports whether the source has a glyph image cache for the given size.
func HasGlyphImageCache(source *GoTextFaceSource, size float64) bool {
	c := &source.glyphImageCache
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.caches[size]
	return ok
}

// TouchGlyphImageCache creates or refreshes the glyph image cache for the size of the given face,
// as if the face were used at the given tick.
func TouchGlyphImageCache(face *GoTextFace, now int64) {
	face.Source.glyphImageCache.cacheForSize(face.Size, glyphVariationCount(face), now)
}

func NewCacheForTesting[Key comparable, Value any](softLimit int) *cache[Key, Value] {
	return newCache[Key, Value](softLimit)
}

func (c *cache[Key, Value]) GetOrCreateAt(key Key, tick int64, create func() (Value, bool)) Value {
	return c.getOrCreateAt(key, tick, create)
}

// SVGAcceptanceGlyph prepares an SVG record through the text cache, bypassing format selection.
func SVGAcceptanceGlyph(face *GoTextFace, gid uint16, offset float64, realize bool) (*ebiten.Image, image.Rectangle, error) {
	s := face.Source.svgFont
	document, ok := s.documentIndex(font.GID(gid))
	if !ok {
		return nil, image.Rectangle{}, opentypesvg.ErrGlyphNotFound
	}
	colors := face.svgColors()
	key := svgGlyphKey{
		document: document,
		gid:      gid,
		size:     float64ToFixed26_6(face.Size),
		xoffset:  adjustGranularity(float64ToFixed26_6(offset), granularityFactor(face)) & 63,
		colors:   colors.key,
	}
	layout := s.prepare(key, colors, float64(face.Source.f.Upem()))
	if layout.err != nil || layout.bounds.Empty() || !realize {
		return nil, layout.bounds, layout.err
	}
	return s.image(key, layout, float64(face.Source.f.Upem())), layout.bounds, nil
}

// SVGAcceptanceOffset returns the horizontal fractional origin used by text layout.
func SVGAcceptanceOffset(face *GoTextFace, offset float64) float64 {
	return fixed26_6ToFloat64(adjustGranularity(float64ToFixed26_6(offset), granularityFactor(face)) & 63)
}
