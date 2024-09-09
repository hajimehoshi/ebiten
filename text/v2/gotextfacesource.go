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
	"sync"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
)

type goTextOutputCacheKey struct {
	text       string
	direction  Direction
	size       float64
	language   string
	script     string
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
	atime   int64
}

type goTextGlyphImageCacheKey struct {
	gid        opentype.GID
	xoffset    fixed.Int26_6
	yoffset    fixed.Int26_6
	variations string
}

// GoTextFaceSource is a source of a GoTextFace. This can be shared by multiple GoTextFace objects.
type GoTextFaceSource struct {
	f        *font.Face
	metadata Metadata

	outputCache     map[goTextOutputCacheKey]*goTextOutputCacheValue
	glyphImageCache map[float64]*glyphImageCache[goTextGlyphImageCacheKey]

	addr *GoTextFaceSource

	shaper shaping.HarfbuzzShaper

	m sync.Mutex
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

	s := &GoTextFaceSource{
		f: font.NewFace(f),
	}
	s.addr = s
	s.metadata = metadataFromLoader(l)

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
		s := &GoTextFaceSource{
			f: &font.Face{Font: f},
		}
		s.addr = s
		s.metadata = metadataFromLoader(l)
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

	g.m.Lock()
	defer g.m.Unlock()

	key := face.outputCacheKey(text)
	if out, ok := g.outputCache[key]; ok {
		out.atime = now()
		return out.outputs, out.glyphs
	}

	f := face.Source.f
	f.SetVariations(face.variations)

	runes := []rune(text)
	input := shaping.Input{
		Text:         runes,
		RunStart:     0,
		RunEnd:       len(runes),
		Direction:    face.diDirection(),
		Face:         f,
		FontFeatures: face.features,
		Size:         float64ToFixed26_6(face.Size),
		Script:       face.gScript(),
		Language:     language.Language(face.Language.String()),
	}

	var seg shaping.Segmenter
	inputs := seg.Split(input, &singleFontmap{face: f})

	if face.Direction == DirectionRightToLeft {
		// Reverse the input for RTL texts.
		for i, j := 0, len(inputs)-1; i < j; i, j = i+1, j-1 {
			inputs[i], inputs[j] = inputs[j], inputs[i]
		}
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
			switch data := g.f.GlyphData(gl.GlyphID).(type) {
			case font.GlyphOutline:
				if out.Direction.IsSideways() {
					data.Sideways(fixed26_6ToFloat32(-gl.YOffset) / fixed26_6ToFloat32(out.Size) * float32(f.Upem()))
				}
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

	if g.outputCache == nil {
		g.outputCache = map[goTextOutputCacheKey]*goTextOutputCacheValue{}
	}
	g.outputCache[key] = &goTextOutputCacheValue{
		outputs: outputs,
		glyphs:  gs,
		atime:   now(),
	}

	const cacheSoftLimit = 512
	if len(g.outputCache) > cacheSoftLimit {
		for key, e := range g.outputCache {
			// 60 is an arbitrary number.
			if e.atime >= now()-60 {
				continue
			}
			delete(g.outputCache, key)
		}
	}

	return outputs, gs
}

func (g *GoTextFaceSource) scale(size float64) float64 {
	return size / float64(g.f.Upem())
}

func (g *GoTextFaceSource) getOrCreateGlyphImage(goTextFace *GoTextFace, key goTextGlyphImageCacheKey, create func() *ebiten.Image) *ebiten.Image {
	if g.glyphImageCache == nil {
		g.glyphImageCache = map[float64]*glyphImageCache[goTextGlyphImageCacheKey]{}
	}
	if _, ok := g.glyphImageCache[goTextFace.Size]; !ok {
		g.glyphImageCache[goTextFace.Size] = &glyphImageCache[goTextGlyphImageCacheKey]{}
	}
	return g.glyphImageCache[goTextFace.Size].getOrCreate(goTextFace, key, create)
}

type singleFontmap struct {
	face *font.Face
}

func (s *singleFontmap) ResolveFace(r rune) *font.Face {
	return s.face
}
