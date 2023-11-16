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
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/go-text/typesetting/di"
	glanguage "github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/opentype/api/font"
	"github.com/go-text/typesetting/opentype/loader"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/language"

	"github.com/hajimehoshi/ebiten/v2"
)

var _ Face = (*GoTextFace)(nil)

// GoTextFace is a Face implementation for go-text's font.Face (github.com/go-text/typesetting).
// With a GoTextFace, shaping.HarfBuzzShaper is always used as a shaper internally.
// GoTextFace includes the source and various options.
type GoTextFace struct {
	// Source is the font face source.
	Source *GoTextFaceSource

	// Direction is the rendering direction.
	// The default (zero) value is left-to-right horizontal.
	Direction Direction

	// Size is the font size in pixels.
	Size float64

	// Language is a hiint for a language (BCP 47).
	Language language.Tag

	// Script is a hint for a script code hint of (ISO 15924).
	// If this is empty, the script is guessed from the specified language.
	Script language.Script

	variations []font.Variation
	features   []shaping.FontFeature

	variationsString string
	featuresString   string
}

// SetVariation sets a variation value.
// For font variations, see https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_fonts/Variable_fonts_guide for more details.
func (g *GoTextFace) SetVariation(tag Tag, value float32) {
	idx := len(g.variations)
	for i, v := range g.variations {
		if uint32(v.Tag) < uint32(tag) {
			continue
		}
		if uint32(v.Tag) > uint32(tag) {
			idx = i
			break
		}
		if v.Value == value {
			return
		}
		g.variations[i].Value = value
		g.variationsString = ""
		return
	}

	// Keep the alphabetical order in order to make the cache key deterministic.
	g.variations = append(g.variations, font.Variation{})
	copy(g.variations[idx+1:], g.variations[idx:])
	g.variations[idx] = font.Variation{
		Tag:   loader.Tag(tag),
		Value: value,
	}
	g.variationsString = ""
}

// RemoveVariation removes a variation value.
func (g *GoTextFace) RemoveVariation(tag Tag) {
	for i, v := range g.variations {
		if uint32(v.Tag) < uint32(tag) {
			continue
		}
		if uint32(v.Tag) > uint32(tag) {
			return
		}

		copy(g.variations[i:], g.variations[i+1:])
		g.variations = g.variations[:len(g.variations)-1]
		g.variationsString = ""
		return
	}
}

// SetFeature sets a feature value.
// For font features, see https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_fonts/OpenType_fonts_guide for more details.
func (g *GoTextFace) SetFeature(tag Tag, value uint32) {
	idx := len(g.features)
	for i, f := range g.features {
		if uint32(f.Tag) < uint32(tag) {
			continue
		}
		if uint32(f.Tag) > uint32(tag) {
			idx = i
			break
		}
		if f.Value == value {
			return
		}
		g.features[i].Value = value
		g.featuresString = ""
		return
	}

	// Keep the alphabetical order in order to make the cache key deterministic.
	g.features = append(g.features, shaping.FontFeature{})
	copy(g.features[idx+1:], g.features[idx:])
	g.features[idx] = shaping.FontFeature{
		Tag:   loader.Tag(tag),
		Value: value,
	}
	g.featuresString = ""
}

// RemoveFeature removes a feature value.
func (g *GoTextFace) RemoveFeature(tag Tag) {
	for i, v := range g.features {
		if uint32(v.Tag) < uint32(tag) {
			continue
		}
		if uint32(v.Tag) > uint32(tag) {
			return
		}

		copy(g.features[i:], g.features[i+1:])
		g.features = g.features[:len(g.features)-1]
		g.featuresString = ""
		return
	}
}

// Tag is a tag for font variations and features.
// Tag is a 4-byte value like 'cmap'.
type Tag uint32

// String returns the Tag's string representation.
func (t Tag) String() string {
	return string([]byte{byte(t >> 24), byte(t >> 16), byte(t >> 8), byte(t)})
}

// PraseTag converts a string to Tag.
func ParseTag(str string) (Tag, error) {
	if len(str) != 4 {
		return 0, fmt.Errorf("text: a string's length must be 4 but was %d at ParseTag", len(str))
	}
	return Tag((uint32(str[0]) << 24) | (uint32(str[1]) << 16) | (uint32(str[2]) << 8) | uint32(str[3])), nil
}

// MustPraseTag converts a string to Tag.
// If parsing fails, MustParseTag panics.
func MustParseTag(str string) Tag {
	t, err := ParseTag(str)
	if err != nil {
		panic(err)
	}
	return t
}

// Metrics implements Face.
func (g *GoTextFace) Metrics() Metrics {
	scale := g.Source.scale(g.Size)

	var m Metrics
	if h, ok := g.Source.f.FontHExtents(); ok {
		m.Height = float64(h.Ascender-h.Descender+h.LineGap) * scale
		m.HAscent = float64(h.Ascender) * scale
		m.HDescent = float64(-h.Descender) * scale
	}
	if v, ok := g.Source.f.FontVExtents(); ok {
		m.Width = float64(v.Ascender-v.Descender+v.LineGap) * scale
		m.VAscent = float64(v.Ascender) * scale
		m.VDescent = float64(-v.Descender) * scale
	}

	return m
}

// UnsafeInternal implements Face.
func (g *GoTextFace) UnsafeInternal() any {
	return g.Source.f
}

func (g *GoTextFace) ensureVariationsString() string {
	if g.variationsString != "" {
		return g.variationsString
	}
	if len(g.variations) == 0 {
		return ""
	}
	var buf bytes.Buffer
	for _, t := range g.variations {
		_ = binary.Write(&buf, binary.LittleEndian, t.Tag)
		_ = binary.Write(&buf, binary.LittleEndian, t.Value)
	}
	g.variationsString = buf.String()
	return g.variationsString
}

func (g *GoTextFace) ensureFeaturesString() string {
	if g.featuresString != "" {
		return g.featuresString
	}
	if len(g.features) == 0 {
		return ""
	}
	var buf bytes.Buffer
	for _, t := range g.features {
		_ = binary.Write(&buf, binary.LittleEndian, t.Tag)
		_ = binary.Write(&buf, binary.LittleEndian, t.Value)
	}
	g.featuresString = buf.String()
	return g.featuresString
}

// faceCacheKey implements Face.
func (g *GoTextFace) faceCacheKey() faceCacheKey {
	return faceCacheKey{
		id:             g.Source.id,
		goTextFaceSize: g.Size,
	}
}

func (g *GoTextFace) outputCacheKey(text string) goTextOutputCacheKey {
	return goTextOutputCacheKey{
		text:       text,
		direction:  g.Direction,
		size:       g.Size,
		language:   g.Language.String(),
		script:     g.Script.String(),
		variations: g.ensureVariationsString(),
		features:   g.ensureFeaturesString(),
	}
}

func (g *GoTextFace) diDirection() di.Direction {
	switch g.Direction {
	case DirectionLeftToRight:
		return di.DirectionLTR
	case DirectionRightToLeft:
		return di.DirectionRTL
	default:
		return di.DirectionTTB
	}
}

func (g *GoTextFace) gScript() glanguage.Script {
	var str string
	if g.Script != (language.Script{}) {
		str = g.Script.String()
	} else {
		s, _ := g.Language.Script()
		str = s.String()
	}

	// ISO 15924 itself is case-insensitive [1], but go-text uses small-caps.
	// [1] https://www.unicode.org/iso15924/faq.html#17
	str = strings.ToLower(str)
	s, err := glanguage.ParseScript(str)
	if err != nil {
		panic(err)
	}
	return s
}

// advance implements Face.
func (g *GoTextFace) advance(text string) float64 {
	output, _ := g.Source.shape(text, g)
	return fixed26_6ToFloat64(output.Advance)
}

// appendGlyphs implements Face.
func (g *GoTextFace) appendGlyphs(glyphs []Glyph, text string, indexOffset int, originX, originY float64) []Glyph {
	_, gs := g.Source.shape(text, g)

	origin := fixed.Point26_6{
		X: float64ToFixed26_6(originX),
		Y: float64ToFixed26_6(originY),
	}
	for _, glyph := range gs {
		img, imgX, imgY := g.glyphImage(glyph, origin)
		if img != nil {
			glyphs = append(glyphs, Glyph{
				StartIndexInBytes: indexOffset + glyph.startIndex,
				EndIndexInBytes:   indexOffset + glyph.endIndex,
				GID:               uint32(glyph.shapingGlyph.GlyphID),
				Image:             img,
				X:                 float64(imgX),
				Y:                 float64(imgY),
			})
		}
		origin = origin.Add(fixed.Point26_6{
			X: glyph.shapingGlyph.XAdvance,
			Y: -glyph.shapingGlyph.YAdvance,
		})
	}

	return glyphs
}

func (g *GoTextFace) glyphImage(glyph glyph, origin fixed.Point26_6) (*ebiten.Image, int, int) {
	if g.direction().isHorizontal() {
		origin.X = adjustGranularity(origin.X, g)
		origin.Y &^= ((1 << 6) - 1)
	} else {
		origin.X &^= ((1 << 6) - 1)
		origin.Y = adjustGranularity(origin.Y, g)
	}

	b := segmentsToBounds(glyph.scaledSegments)
	subpixelOffset := fixed.Point26_6{
		X: (origin.X + b.Min.X) & ((1 << 6) - 1),
		Y: (origin.Y + b.Min.Y) & ((1 << 6) - 1),
	}
	key := glyphImageCacheKey{
		id:         uint32(glyph.shapingGlyph.GlyphID),
		xoffset:    subpixelOffset.X,
		yoffset:    subpixelOffset.Y,
		variations: g.ensureVariationsString(),
	}
	img := theGlyphImageCache.getOrCreate(g, key, func() *ebiten.Image {
		return segmentsToImage(glyph.scaledSegments, subpixelOffset, b)
	})

	imgX := (origin.X + b.Min.X).Floor()
	imgY := (origin.Y + b.Min.Y).Floor()
	return img, imgX, imgY
}

// direction implements Face.
func (g *GoTextFace) direction() Direction {
	return g.Direction
}

// private implements Face.
func (g *GoTextFace) private() {
}
