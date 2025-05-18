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
	"io"
	"slices"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
	xlanguage "golang.org/x/text/language"

	"github.com/hajimehoshi/ebiten/v2"
)

type goTextOutputCacheKey struct {
	text       string
	direction  Direction
	size       float64
	language   xlanguage.Tag
	script     xlanguage.Script
	variations string
	features   string
}

type glyph struct {
	shapingGlyph   *shaping.Glyph
	startIndex     int
	endIndex       int
	scaledSegments []opentype.Segment
	bounds         fixed.Rectangle26_6
}

type goTextOutputCacheValue struct {
	outputs []shaping.Output
	glyphs  []glyph
}

type goTextGlyphImageCacheKey struct {
	gid        opentype.GID
	xoffset    fixed.Int26_6
	yoffset    fixed.Int26_6
	variations string
}

// runeToBoolMap is a map from rune to bool with performance optimizations.
type runeToBoolMap struct {
	m []uint64
}

func (r *runeToBoolMap) get(rune rune) (value bool, ok bool) {
	index := rune / 32
	if len(r.m) <= int(index) {
		return false, false
	}
	shift := 2 * (rune % 32)
	v := r.m[index] >> shift
	return v&0b10 != 0, v&0b01 != 0
}

func (r *runeToBoolMap) set(rune rune, value bool) {
	index := rune / 32
	if len(r.m) <= int(index) {
		r.m = slices.Grow(r.m, int(index)+1)[:index+1]
	}
	shift := 2 * (rune % 32)
	if value {
		r.m[index] |= 0b11 << shift
	} else {
		r.m[index] |= 0b01 << shift
		r.m[index] &^= 0b10 << shift
	}
}

type glyphDataCacheKey struct {
	gid        font.GID
	variations string
	sideways   bool
}

// GoTextFaceSource is a source of a GoTextFace. This can be shared by multiple GoTextFace objects.
type GoTextFaceSource struct {
	f        *font.Face
	metadata Metadata

	outputCache     *cache[goTextOutputCacheKey, goTextOutputCacheValue]
	glyphImageCache map[float64]*cache[goTextGlyphImageCacheKey, *ebiten.Image]
	hasGlyphCache   runeToBoolMap

	unscaledMetrics Metrics

	addr *GoTextFaceSource

	shaper shaping.HarfbuzzShaper

	runes          []rune
	glyphDataCache *cache[glyphDataCacheKey, font.GlyphData]
}

func toFontResource(source io.Reader) (font.Resource, error) {
	// font.Resource has io.Seeker and io.ReaderAt in addition to io.Reader.
	// If source has it, use it as it is.
	if s, ok := source.(font.Resource); ok {
		return s, nil
	}

	// Read all the bytes and convert this to bytes.Reader.
	// This is a very rough solution, but it works.
	// TODO: Implement io.ReaderAt in a more efficient way when source is io.Seeker.
	bs, err := io.ReadAll(source)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(bs), nil
}

func newGoTextFaceSource(face *font.Face) *GoTextFaceSource {
	s := &GoTextFaceSource{
		f: face,
	}
	s.addr = s
	s.metadata = metadataFromFace(face)
	s.outputCache = newCache[goTextOutputCacheKey, goTextOutputCacheValue](512)
	// 4 is an arbitrary number, which should not cause troubles.
	s.shaper.SetFontCacheSize(4)
	return s
}

// NewGoTextFaceSource parses an OpenType or TrueType font and returns a GoTextFaceSource object.
func NewGoTextFaceSource(source io.Reader) (*GoTextFaceSource, error) {
	src, err := toFontResource(source)
	if err != nil {
		return nil, err
	}

	l, err := opentype.NewLoader(src)
	if err != nil {
		return nil, err
	}

	f, err := font.NewFont(l)
	if err != nil {
		return nil, err
	}

	s := newGoTextFaceSource(&font.Face{Font: f})
	return s, nil
}

// NewGoTextFaceSourcesFromCollection parses an OpenType or TrueType font collection and returns a slice of GoTextFaceSource objects.
func NewGoTextFaceSourcesFromCollection(source io.Reader) ([]*GoTextFaceSource, error) {
	src, err := toFontResource(source)
	if err != nil {
		return nil, err
	}

	ls, err := opentype.NewLoaders(src)
	if err != nil {
		return nil, err
	}

	sources := make([]*GoTextFaceSource, len(ls))
	for i, l := range ls {
		f, err := font.NewFont(l)
		if err != nil {
			return nil, err
		}
		s := newGoTextFaceSource(&font.Face{Font: f})
		sources[i] = s
	}
	return sources, nil
}

func (g *GoTextFaceSource) copyCheck() {
	if g.addr != g {
		panic("text: illegal use of non-zero GoTextFaceSource copied by value")
	}
}

// Metadata returns its metadata.
func (g *GoTextFaceSource) Metadata() Metadata {
	return g.metadata
}

// UnsafeInternal returns its font.Face.
// The return value type is any since github.com/go-text/typesettings's API is now unstable.
//
// UnsafeInternal is unsafe since this might make internal cache states out of sync.
//
// UnsafeInternal might have breaking changes even in the same major version.
func (g *GoTextFaceSource) UnsafeInternal() any {
	return g.f
}

func (g *GoTextFaceSource) shape(text string, face *GoTextFace) ([]shaping.Output, []glyph) {
	g.copyCheck()

	key := face.outputCacheKey(text)
	e := g.outputCache.getOrCreate(key, func() (goTextOutputCacheValue, bool) {
		outputs, gs := g.shapeImpl(text, face)
		return goTextOutputCacheValue{
			outputs: outputs,
			glyphs:  gs,
		}, true
	})
	return e.outputs, e.glyphs
}

func (g *GoTextFaceSource) shapeImpl(text string, face *GoTextFace) ([]shaping.Output, []glyph) {
	g.f.SetVariations(face.variations)

	g.runes = g.runes[:0]
	for _, r := range text {
		g.runes = append(g.runes, r)
	}
	input := shaping.Input{
		Text:         g.runes,
		RunStart:     0,
		RunEnd:       len(g.runes),
		Direction:    face.diDirection(),
		Face:         g.f,
		FontFeatures: face.features,
		Size:         float64ToFixed26_6(face.Size),
		Script:       face.gScript(),
		Language:     language.Language(face.Language.String()),
	}

	var seg shaping.Segmenter
	inputs := seg.Split(input, &singleFontmap{face: g.f})

	// Reverse the input for RTL texts.
	if face.Direction == DirectionRightToLeft {
		slices.Reverse(inputs)
	}

	outputs := make([]shaping.Output, len(inputs))
	var gs []glyph
	for i, input := range inputs {
		out := g.shaper.Shape(input)
		outputs[i] = out

		(shaping.Line{out}).AdjustBaselines()

		var indices []int
		for i := range text {
			indices = append(indices, i)
		}
		indices = append(indices, len(text))

		for _, gl := range out.Glyphs {
			gl := gl
			var segs []opentype.Segment
			if g.glyphDataCache == nil {
				g.glyphDataCache = newCache[glyphDataCacheKey, font.GlyphData](512)
			}
			key := glyphDataCacheKey{
				gid:        gl.GlyphID,
				variations: face.ensureVariationsString(),
				sideways:   out.Direction.IsSideways(),
			}
			data := g.glyphDataCache.getOrCreate(key, func() (font.GlyphData, bool) {
				data := g.f.GlyphData(gl.GlyphID)
				if data == nil {
					return nil, false
				}
				if data, ok := data.(font.GlyphOutline); ok && out.Direction.IsSideways() {
					data.Sideways(fixed26_6ToFloat32(-gl.YOffset) / fixed26_6ToFloat32(out.Size) * float32(g.f.Upem()))
				}
				return data, true
			})
			switch data := data.(type) {
			case font.GlyphOutline:
				segs = data.Segments
			case font.GlyphSVG:
				segs = data.Outline.Segments
			case font.GlyphBitmap:
				if data.Outline != nil {
					segs = data.Outline.Segments
				}
			}

			scaledSegs := make([]opentype.Segment, len(segs))
			scale := float32(g.scale(fixed26_6ToFloat64(out.Size)))
			for i, seg := range segs {
				scaledSegs[i] = seg
				for j := range seg.Args {
					scaledSegs[i].Args[j].X *= scale
					scaledSegs[i].Args[j].Y *= -scale
				}
			}

			gs = append(gs, glyph{
				shapingGlyph:   &gl,
				startIndex:     indices[gl.ClusterIndex],
				endIndex:       indices[gl.ClusterIndex+gl.RuneCount],
				scaledSegments: scaledSegs,
				bounds:         segmentsToBounds(scaledSegs),
			})
		}
	}
	return outputs, gs
}

func (g *GoTextFaceSource) scale(size float64) float64 {
	return size / float64(g.f.Upem())
}

func (g *GoTextFaceSource) getOrCreateGlyphImage(goTextFace *GoTextFace, key goTextGlyphImageCacheKey, create func() (*ebiten.Image, bool)) *ebiten.Image {
	if g.glyphImageCache == nil {
		g.glyphImageCache = map[float64]*cache[goTextGlyphImageCacheKey, *ebiten.Image]{}
	}
	if _, ok := g.glyphImageCache[goTextFace.Size]; !ok {
		g.glyphImageCache[goTextFace.Size] = newCache[goTextGlyphImageCacheKey, *ebiten.Image](128 * glyphVariationCount(goTextFace))
	}
	return g.glyphImageCache[goTextFace.Size].getOrCreate(key, create)
}

func (g *GoTextFaceSource) metrics(size float64) Metrics {
	um := g.unscaledMetrics
	if um.HAscent == 0 && um.HDescent == 0 && um.VAscent == 0 && um.VDescent == 0 {
		if h, ok := g.f.FontHExtents(); ok {
			um.HLineGap = float64(h.LineGap)
			um.HAscent = float64(h.Ascender)
			um.HDescent = float64(-h.Descender)
		}
		if v, ok := g.f.FontVExtents(); ok {
			um.VLineGap = float64(v.LineGap)
			um.VAscent = float64(v.Ascender)
			um.VDescent = float64(-v.Descender)
		}
		um.XHeight = float64(g.f.LineMetric(font.XHeight))
		um.CapHeight = float64(g.f.LineMetric(font.CapHeight))
	}

	scale := g.scale(size)
	return Metrics{
		HLineGap:  um.HLineGap * scale,
		HAscent:   um.HAscent * scale,
		HDescent:  um.HDescent * scale,
		VLineGap:  um.VLineGap * scale,
		VAscent:   um.VAscent * scale,
		VDescent:  um.VDescent * scale,
		XHeight:   um.XHeight * scale,
		CapHeight: um.CapHeight * scale,
	}
}

func (g *GoTextFaceSource) hasGlyph(r rune) bool {
	if has, ok := g.hasGlyphCache.get(r); ok {
		return has
	}
	_, ok := g.f.Cmap.Lookup(r)
	g.hasGlyphCache.set(r, ok)
	return ok
}

type singleFontmap struct {
	face *font.Face
}

func (s *singleFontmap) ResolveFace(r rune) *font.Face {
	return s.face
}
