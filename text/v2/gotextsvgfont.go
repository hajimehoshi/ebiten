// Copyright 2026 The Ebitengine Authors
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
	"image/color"
	"sync"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg/svgrender"
)

type svgRecord struct {
	first, last uint16
	document    int
}

type svgDocument struct {
	document *opentypesvg.RenderDocument
	err      error
}

type svgGlyphKey struct {
	document         int
	gid              uint16
	size             fixed.Int26_6
	xoffset, yoffset fixed.Int26_6
	colors           string
}

type svgGlyphLayout struct {
	bounds image.Rectangle
	err    error
}

type svgPreparedGlyph struct {
	layout svgGlyphLayout
	plan   *opentypesvg.RenderPlan
}

// svgMemo has a hard entry limit even when ticks do not advance. Values held
// by a caller remain live independently of cache membership.
type svgMemo[K comparable, V any] struct {
	entries map[K]svgMemoEntry[V]
	clock   uint64
}

type svgMemoEntry[V any] struct {
	value V
	used  uint64
}

func (m *svgMemo[K, V]) get(key K, limit int, create func() V) V {
	m.clock++
	if e, ok := m.entries[key]; ok {
		e.used = m.clock
		m.entries[key] = e
		return e.value
	}
	if m.entries == nil {
		m.entries = make(map[K]svgMemoEntry[V])
	}
	if len(m.entries) >= limit {
		var oldest K
		age := m.clock
		for k, e := range m.entries {
			if e.used < age {
				oldest, age = k, e.used
			}
		}
		delete(m.entries, oldest)
	}
	value := create()
	m.entries[key] = svgMemoEntry[V]{
		value: value,
		used:  m.clock,
	}
	return value
}

type svgFont struct {
	records []svgRecord
	sources [][]byte
	palette []color.NRGBA
	// Each plan is bounded by the interpreter's geometry, image and surface
	// budgets. Two documents and four plans cap retention independently of the
	// number of glyphs or color combinations visited in a frame.
	documents svgMemo[int, svgDocument]
	layouts   svgMemo[svgGlyphKey, *svgGlyphLayout]
	plans     svgMemo[svgGlyphKey, *svgPreparedGlyph]
	images    svgMemo[svgGlyphKey, *ebiten.Image]
	mu        sync.Mutex
}

func loadSVGFont(loader *opentype.Loader) *svgFont {
	raw, err := loader.RawTable(opentype.MustNewTag("SVG "))
	if err != nil {
		return nil
	}
	table, _, err := tables.ParseSVG(raw)
	if err != nil {
		return nil
	}
	s := &svgFont{}
	data := table.SVGDocumentList.SVGRawData
	type span struct{ offset, length uint32 }
	documents := map[span]int{}
	var previous uint16
	for i, r := range table.SVGDocumentList.DocumentRecords {
		start, length := uint64(r.SvgDocOffset), uint64(r.SvgDocLength)
		if r.StartGlyphID > r.EndGlyphID || i > 0 && uint16(r.StartGlyphID) <= previous || start+length > uint64(len(data)) {
			return nil
		}
		previous = uint16(r.EndGlyphID)
		key := span{
			offset: uint32(start),
			length: uint32(length),
		}
		id, ok := documents[key]
		if !ok {
			id = len(s.sources)
			documents[key] = id
			s.sources = append(s.sources, data[start:start+length])
		}
		s.records = append(s.records, svgRecord{
			first:    uint16(r.StartGlyphID),
			last:     uint16(r.EndGlyphID),
			document: id,
		})
	}
	raw, err = loader.RawTable(opentype.MustNewTag("CPAL"))
	if err == nil {
		cpal, _, err := tables.ParseCPAL(raw)
		if err == nil && len(cpal.ColorRecordIndices) > 0 {
			start := int(cpal.ColorRecordIndices[0])
			for i := start; i < start+int(cpal.NumPaletteEntries) && i < len(cpal.ColorRecordsArray); i++ {
				c := cpal.ColorRecordsArray[i]
				s.palette = append(s.palette, color.NRGBA{R: c.Red, G: c.Green, B: c.Blue, A: c.Alpha})
			}
		}
	}
	return s
}

func (s *svgFont) documentIndex(gid font.GID) (int, bool) {
	if s == nil || gid > 65535 {
		return 0, false
	}
	lo, hi := 0, len(s.records)
	for lo < hi {
		mid := lo + (hi-lo)/2
		r := s.records[mid]
		if uint16(gid) < r.first {
			hi = mid
		} else if uint16(gid) > r.last {
			lo = mid + 1
		} else {
			return r.document, true
		}
	}
	return 0, false
}

type svgColors struct {
	foreground color.NRGBA
	palette    []color.NRGBA
	key        string
}

func (g *GoTextFace) svgColors() svgColors {
	c := svgColors{
		foreground: color.NRGBA{A: 255},
	}
	if g.Source.svgFont == nil {
		return c
	}
	if g.SVGForeground != nil {
		c.foreground = color.NRGBAModel.Convert(g.SVGForeground).(color.NRGBA)
	}
	palette := g.SVGPalette
	if palette == nil {
		palette = g.Source.svgFont.palette
	}
	c.palette = append([]color.NRGBA(nil), palette...)
	key := []byte{c.foreground.R, c.foreground.G, c.foreground.B, c.foreground.A}
	for _, p := range palette {
		key = append(key, p.R, p.G, p.B, p.A)
	}
	c.key = string(key)
	return c
}

func (s *svgFont) prepare(key svgGlyphKey, colors svgColors, upem float64) *svgGlyphLayout {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.layouts.get(key, 512, func() *svgGlyphLayout {
		// Copy the small result so lazy glyphs do not retain an evicted heavy plan.
		layout := s.preparePlan(key, colors, upem).layout
		return &layout
	})
}

// The caller holds s.mu.
func (s *svgFont) preparePlan(key svgGlyphKey, colors svgColors, upem float64) *svgPreparedGlyph {
	return s.plans.get(key, 4, func() *svgPreparedGlyph {
		if key.size == 0 {
			return &svgPreparedGlyph{}
		}

		doc := s.documents.get(key.document, 2, func() svgDocument {
			decoded, err := opentypesvg.Decode(s.sources[key.document])
			if err != nil {
				return svgDocument{
					err: err,
				}
			}
			d, err := opentypesvg.ParseRenderDocument(decoded)
			return svgDocument{
				document: d,
				err:      err,
			}
		})
		if doc.err != nil {
			return &svgPreparedGlyph{
				layout: svgGlyphLayout{
					err: doc.err,
				},
			}
		}
		scale := fixed26_6ToFloat64(key.size) / upem
		// Native vector antialiasing applies to paths and clips at glyph resolution.
		plan, bounds, err := doc.document.PrepareGlyph(opentypesvg.RenderOptions{
			GlyphID:    key.gid,
			UnitsPerEm: upem,
			Samples:    1,
			AntiAlias:  true,
			Transform:  [6]float64{scale, 0, 0, scale, fixed26_6ToFloat64(key.xoffset), fixed26_6ToFloat64(key.yoffset)},
			Foreground: colors.foreground,
			Palette:    colors.palette,
		})
		// Bound cached image storage independently of the render grid density.
		const maxGlyphPixels = 1 << 18
		if err == nil && !bounds.Empty() && bounds.Dx() > maxGlyphPixels/bounds.Dy() {
			plan, bounds, err = nil, image.Rectangle{}, opentypesvg.ErrLimit
		}
		return &svgPreparedGlyph{
			plan: plan,
			layout: svgGlyphLayout{
				bounds: bounds,
				err:    err,
			},
		}
	})
}

func (s *svgFont) image(key svgGlyphKey, glyph *svgGlyphLayout, upem float64) *ebiten.Image {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.images.get(key, 128, func() *ebiten.Image {
		if glyph.bounds.Empty() {
			return nil
		}
		// Color bytes are the immutable layout snapshot. Decode only on an image miss.
		colors := svgColors{
			foreground: color.NRGBA{R: key.colors[0], G: key.colors[1], B: key.colors[2], A: key.colors[3]},
		}
		for i := 4; i < len(key.colors); i += 4 {
			colors.palette = append(colors.palette, color.NRGBA{R: key.colors[i], G: key.colors[i+1], B: key.colors[i+2], A: key.colors[i+3]})
		}
		prepared := s.preparePlan(key, colors, upem)
		if prepared.layout.err != nil {
			panic(prepared.layout.err)
		}
		img := ebiten.NewImage(glyph.bounds.Dx(), glyph.bounds.Dy())
		var renderer svgrender.Renderer
		defer renderer.Deallocate()
		if err := renderer.Draw(img, prepared.plan); err != nil {
			img.Deallocate()
			// Prepared plans satisfy the renderer contract. Shader compilation errors
			// are engine failures, not a reason to disguise a glyph with an outline.
			panic(err)
		}
		return img
	})
}

// glyphDataWithoutSVG preserves go-text's format priority without entering its
// unbounded gzip decoder. SVG records are selected and decoded independently.
func (g *GoTextFaceSource) glyphDataWithoutSVG(gid font.GID) font.GlyphData {
	if d, ok := g.f.GlyphDataColor(gid); ok {
		return d
	}
	if d, ok := g.f.GlyphDataBitmap(gid); ok {
		return d
	}
	if d, ok := g.f.GlyphDataOutline(gid); ok {
		return d
	}
	return nil
}
