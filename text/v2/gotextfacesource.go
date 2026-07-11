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

// go-text/typesetting already imports image/png, so the side effect is acceptable (#2336).

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"slices"
	"sync"

	"github.com/go-text/typesetting/bidi"
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/tiff"
	xlanguage "golang.org/x/text/language"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/chunk"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/textutil"
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

type goTextGlyph struct {
	shapingGlyph *shaping.Glyph
	startIndex   int
	endIndex     int

	// render is a pointer to the shared render data (bounds plus the
	// lazily-realized segments and bitmap). It is nil for glyphs with
	// no outline and no bitmap (e.g. control characters).
	render *glyphRenderData
}

type goTextOutputCacheValue struct {
	outputs []shaping.Output

	// advances is lazily built by ensureAdvances from outputs. The text
	// used to derive advances is implied by the cache key, so it is passed
	// in to ensureAdvances rather than retained on the value.
	advances     []fixed.Int26_6
	advancesOnce sync.Once

	// glyphs is lazily built by ensureGlyphs from outputs plus the
	// (text, face) inputs implied by the cache key.
	glyphs     []goTextGlyph
	glyphsOnce sync.Once
}

// ensureAdvances returns the leading-edge X for each byte position in
// text, building it on first access. See [GoTextFaceSource.advances]
// for the meaning of the returned slice.
func (v *goTextOutputCacheValue) ensureAdvances(text string) []fixed.Int26_6 {
	v.advancesOnce.Do(func() {
		v.advances = buildAdvances(v.outputs, text)
	})
	return v.advances
}

// ensureGlyphs returns per-glyph data, building it on first access.
func (v *goTextOutputCacheValue) ensureGlyphs(g *GoTextFaceSource, text string, face *GoTextFace) []goTextGlyph {
	v.glyphsOnce.Do(func() {
		g.shapeMu.Lock()
		defer g.shapeMu.Unlock()
		v.glyphs = g.buildGlyphs(v.outputs, text, face)
	})
	return v.glyphs
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
	// size is 0 in outline mode and out.Size in bitmap mode. The outline
	// path is ppem-independent (GlyphData reads raw outline points and
	// variable-font axes route through SetVariations), so 0 is used for
	// every face size and they share an entry. Bitmap glyph data depends
	// on the face's currently-set ppem, so each size needs its own entry.
	//
	// The outline-uses-0 invariant assumes the underlying typesetting
	// library does not apply hinting at GlyphData time: today it reads
	// raw outline points from glyf/CFF/CFF2 and variable-font axes
	// (including opsz) route through SetVariations. If a hinting
	// interpreter is added in the future, outlines may vary per ppem
	// and the outline case will need a non-zero size here too.
	size fixed.Int26_6
}

// glyphRenderDataCacheKey identifies the render-data bundle for one glyph.
// Scaled segments depend on all four key fields; bitmap depends only on
// (gid, size) but is bundled here so a single cache entry yields every
// renderable form of the glyph. For fonts exercising multiple variations
// or sideways modes at the same size, the same bitmap image may be
// referenced from multiple entries — the image data is still shared via
// the interface's pointer, only the entry headers are duplicated.
//
// Bundling segments and bitmap together means [AppendVectorPath] retains
// access to the outline even for glyphs whose bitmap is also populated.
type glyphRenderDataCacheKey struct {
	gid        font.GID
	variations string
	sideways   bool
	size       fixed.Int26_6
}

// glyphRenderData bundles the data needed to render one glyph. bounds
// is computed eagerly so layout decisions (image rectangle, culling,
// ImageBounds) don't pay for the GlyphData fetch. useBitmap is true
// when the glyph renders from a bitmap strike; layout consults it to
// pick the positioning convention and realize consults it to select
// bitmap-mode glyph data.
//
// The actual render forms (segments, bitmap, SVG document, COLRv0
// layers) are produced on first call to any of the accessors below,
// which fetch glyph data via the source's glyphDataCache.
type glyphRenderData struct {
	bounds    fixed.Rectangle26_6
	useBitmap bool

	realizeOnce sync.Once

	// Captured at build time for use by realize.
	source      *GoTextFaceSource
	gid         font.GID
	size        fixed.Int26_6
	sideways    bool
	yOffset     fixed.Int26_6
	variations  []font.Variation
	bitmapXPpem uint16
	bitmapYPpem uint16

	// Populated by realize. Accessed via segments / bitmap / svg /
	// colrV0Layers so callers don't have to remember to drive the lazy
	// initialization.
	realizedSegments     []opentype.Segment
	realizedBitmap       image.Image
	realizedSVG          *svgGlyphData
	realizedCOLRV0Layers []colrV0Layer
}

// segments returns the scaled outline segments, realizing them on
// first call.
func (r *glyphRenderData) segments() []opentype.Segment {
	r.realizeOnce.Do(r.realize)
	return r.realizedSegments
}

// bitmap returns the decoded bitmap image, realizing it on first
// call. It returns nil when the glyph has no bitmap data.
func (r *glyphRenderData) bitmap() image.Image {
	r.realizeOnce.Do(r.realize)
	return r.realizedBitmap
}

// svg returns the OpenType SVG glyph description, realizing it on first
// call. It returns nil when the glyph has no SVG data.
func (r *glyphRenderData) svg() *svgGlyphData {
	r.realizeOnce.Do(r.realize)
	return r.realizedSVG
}

// colrV0Layers returns the COLRv0 color layers, realizing them on first
// call. It returns nil when the glyph has no COLRv0 data.
func (r *glyphRenderData) colrV0Layers() []colrV0Layer {
	r.realizeOnce.Do(r.realize)
	return r.realizedCOLRV0Layers
}

// realize delegates to [GoTextFaceSource.realizeRenderData], where
// the lock and the work both live.
func (r *glyphRenderData) realize() {
	r.source.realizeRenderData(r)
}

// GoTextFaceSource is a source of a GoTextFace. This can be shared by multiple GoTextFace objects.
type GoTextFaceSource struct {
	f        *font.Face
	metadata Metadata

	outputCache     *cache[goTextOutputCacheKey, *goTextOutputCacheValue]
	glyphImageCache map[float64]*cache[goTextGlyphImageCacheKey, *ebiten.Image]
	hasGlyphCache   runeToBoolMap

	unscaledMetrics     Metrics
	unscaledMetricsOnce sync.Once

	addr *GoTextFaceSource

	shaper shaping.HarfbuzzShaper
	seg    shaping.Segmenter
	// bidiPara runs the Unicode Bidirectional Algorithm to obtain
	// per-run levels. buildOutputs uses them to reorder runs into
	// visual order (UAX #9 rule L2); rules L3 and L4 are applied by
	// HarfBuzz during shaping.
	bidiPara bidi.Paragraph

	runes           []rune
	glyphDataCache  *cache[glyphDataCacheKey, font.GlyphData]
	renderDataCache *cache[glyphRenderDataCacheKey, *glyphRenderData]
	chunkPlanCache  *cache[chunkPlanKey, []chunk.Chunk]

	// svgDocIndexes caches per-document indexes for OpenType SVG documents
	// shared by multiple glyphs, keyed by the document's backing array.
	// Uncompressed SVG documents are subslices of the font data, so the
	// address of the first byte identifies the document; a gzipped
	// document is decompressed into a fresh buffer per fetch and only
	// misses the cache. Guarded by shapeMu.
	svgDocIndexes map[*byte]*svgDocIndex

	// shapeMu serializes mutations of the shared font state (g.f) during shaping
	// and per-glyph data lookups. Lazy glyph builds happen outside of the
	// outputCache mutex, so this mutex is needed to keep the font state consistent.
	shapeMu sync.Mutex

	// lastVariationsString and lastXPpem/lastYPpem mirror the state
	// last pushed to g.f via SetVariations and SetPpem, so callers
	// can skip the Set call when it would be a no-op. Typesetting's
	// Set methods unconditionally reset the per-Face extents cache
	// on every call (an O(nGlyphs) reset each), so without this
	// elision a long shape + draw cycle would discard the cache
	// once per realized glyph. Guarded by shapeMu.
	lastVariationsString string
	lastXPpem            uint16
	lastYPpem            uint16

	bidiLevelsBuf  []bidi.Level
	visualOrderBuf []int
	inputsBuf      []shaping.Input

	bitmapSizesResult []font.BitmapSize
	bitmapSizesOnce   sync.Once
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
	s.outputCache = newCache[goTextOutputCacheKey, *goTextOutputCacheValue](512)
	s.chunkPlanCache = newCache[chunkPlanKey, []chunk.Chunk](512)
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

	s := newGoTextFaceSource(font.NewFace(f))
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
		s := newGoTextFaceSource(font.NewFace(f))
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

// advances returns a slice of length len(text)+1 where element i is the
// visual X of the caret at logical byte i, using the leading-edge
// convention: the value is the leading edge of the cluster that starts
// at byte i (left edge in an LTR run, right edge in an RTL run). A byte
// inside a cluster snaps to the cluster's leading-edge X in logical
// order.
//
// The final element a[len(text)] is defined as the total visual line
// width, which is one of several reasonable end-of-text conventions
// chosen here for caret-at-line-end semantics. Under the leading-edge
// rule alone this position is otherwise ambiguous when the last logical
// character is in a run whose direction differs from the base.
//
// The leading-edge convention picks one of the two visually valid carets
// at a bidi level boundary; affinity-aware positioning is not exposed.
func (g *GoTextFaceSource) advances(text string, face *GoTextFace) []fixed.Int26_6 {
	g.copyCheck()
	return g.outputCacheValue(text, face).ensureAdvances(text)
}

// buildAdvances derives the leading-edge X for each byte position in
// text from already-shaped outputs. See [GoTextFaceSource.advances] for
// the meaning of the returned slice.
func buildAdvances(outputs []shaping.Output, text string) []fixed.Int26_6 {
	// Rune index → byte index, plus a final entry at len(text).
	runeToByte := make([]int, 0, len(text)+1)
	for i := range text {
		runeToByte = append(runeToByte, i)
	}
	runeToByte = append(runeToByte, len(text))

	// buildOutputs returns outputs in visual left-to-right order, so
	// walking the slice and accumulating each run's Advance yields
	// each run's left-edge X. Within a run HarfBuzz emits glyphs in
	// visual order (leftmost first) regardless of run direction; the
	// leading edge of a cluster is on the left side of its visual
	// glyph for an LTR run and on the right side for an RTL run.
	a := make([]fixed.Int26_6, len(text)+1)
	set := make([]bool, len(text)+1)
	var x fixed.Int26_6
	for oi := range outputs {
		out := &outputs[oi]
		runLeftX := x
		rtl := out.Direction.Progression() == di.TowardTopLeft
		var cum fixed.Int26_6 // sum of advances of earlier glyphs in this run
		for gi := range out.Glyphs {
			gl := &out.Glyphs[gi]
			startByte := runeToByte[gl.ClusterIndex]
			var leading fixed.Int26_6
			if rtl {
				leading = runLeftX + cum + gl.Advance
			} else {
				leading = runLeftX + cum
			}
			// The first glyph of a multi-glyph cluster wins; later
			// glyphs in the same cluster share the leading edge.
			if !set[startByte] {
				a[startByte] = leading
				set[startByte] = true
			}
			cum += gl.Advance
		}
		x += out.Advance
	}
	totalWidth := x

	// Bytes that don't start a cluster (interior bytes of a
	// multi-byte rune or a multi-rune cluster) snap to the cluster's
	// leading-edge X in logical order.
	for i := 1; i < len(a); i++ {
		if !set[i] {
			a[i] = a[i-1]
		}
	}
	// The end-of-text entry is the total visual line width.
	a[len(text)] = totalWidth
	return a
}

// advanceAt returns the advance from the start of text to indexInBytes.
// indexInBytes that falls inside a glyph cluster snaps to the cluster's start.
func (g *GoTextFaceSource) advanceAt(text string, face *GoTextFace, indexInBytes int) fixed.Int26_6 {
	g.copyCheck()

	if indexInBytes <= 0 {
		return 0
	}

	chunks := g.chunks(text, face)
	if len(chunks) == 1 {
		// Single-chunk path: index into the chunk's advance slice
		// directly. The chunk may end before len(text) when a line
		// break trimmed the first line shorter than the input.
		ch := chunks[0]
		chunkText := text[ch.Start:ch.End]
		a := g.outputCacheValue(chunkText, face).ensureAdvances(chunkText)
		local := indexInBytes - ch.Start
		if local >= len(a) {
			return a[len(a)-1]
		}
		return a[local]
	}

	// Stack arrays rather than reusable struct fields: this path
	// runs without shapeMu, so a shared buffer would race.
	var levelsArr [16]bidi.Level
	var orderArr [16]int
	order := appendChunksVisualOrder(orderArr[:0], levelsArr[:0], chunks)

	targetLogical := -1
	for li, ch := range chunks {
		if indexInBytes < ch.End {
			targetLogical = li
			break
		}
	}

	var x fixed.Int26_6
	for _, li := range order {
		ch := chunks[li]
		chunkText := text[ch.Start:ch.End]
		chunkAdv := g.outputCacheValue(chunkText, face).ensureAdvances(chunkText)
		if li == targetLogical {
			local := max(indexInBytes-ch.Start, 0)
			if local >= len(chunkAdv) {
				local = len(chunkAdv) - 1
			}
			return x + chunkAdv[local]
		}
		x += chunkAdv[len(chunkAdv)-1]
	}
	// indexInBytes is past every chunk: return the total visual width.
	return x
}

// glyphs returns per-glyph wrappers for text, computed from the cached
// shape outputs.
func (g *GoTextFaceSource) glyphs(text string, face *GoTextFace) []goTextGlyph {
	g.copyCheck()

	chunks := g.chunks(text, face)
	if len(chunks) == 1 {
		ch := chunks[0]
		chunkText := text[ch.Start:ch.End]
		return g.outputCacheValue(chunkText, face).ensureGlyphs(g, chunkText, face)
	}

	// Stack arrays rather than reusable struct fields: this path
	// runs without shapeMu, so a shared buffer would race.
	var levelsArr [16]bidi.Level
	var orderArr [16]int
	order := appendChunksVisualOrder(orderArr[:0], levelsArr[:0], chunks)

	chunkGlyphs := make([][]goTextGlyph, len(chunks))
	var total int
	for i, ch := range chunks {
		chunkText := text[ch.Start:ch.End]
		chunkGlyphs[i] = g.outputCacheValue(chunkText, face).ensureGlyphs(g, chunkText, face)
		total += len(chunkGlyphs[i])
	}
	result := make([]goTextGlyph, 0, total)
	for _, li := range order {
		ch := chunks[li]
		for _, gl := range chunkGlyphs[li] {
			translated := gl
			translated.startIndex += ch.Start
			translated.endIndex += ch.Start
			result = append(result, translated)
		}
	}
	return result
}

// outputCacheValue returns the cached shape-output bundle for the
// (text, face) pair, shaping it on first lookup. Both
// [GoTextFaceSource.advances] and [GoTextFaceSource.glyphs] derive
// their results from this single shaped form, so one piece of text is
// shaped at most once regardless of which path arrives first.
func (g *GoTextFaceSource) outputCacheValue(text string, face *GoTextFace) *goTextOutputCacheValue {
	key := face.outputCacheKey(text)
	return g.outputCache.getOrCreate(key, func() (*goTextOutputCacheValue, bool) {
		g.shapeMu.Lock()
		defer g.shapeMu.Unlock()
		return &goTextOutputCacheValue{
			outputs: g.buildOutputs(text, face),
		}, true
	})
}

// faceBitmapState describes how bitmap strikes apply to a face size.
type faceBitmapState struct {
	// xPpem and yPpem are the pixel-per-em values pushed to the font,
	// used to select a bitmap strike. Both are zero when the font has
	// no bitmap strikes.
	xPpem, yPpem uint16

	// useBitmap indicates that the font has bitmap strikes and bitmap
	// glyph data should be considered for rendering.
	useBitmap bool

	// exact indicates that face.Size matches a strike's pixel size, so
	// bitmaps render pixel-perfect without scaling. When false, a
	// glyph's outline takes precedence over its scaled bitmap.
	exact bool
}

// applyFaceState updates the shared font state to reflect face and
// returns the resulting bitmap state.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) applyFaceState(face *GoTextFace) faceBitmapState {
	if s := face.ensureVariationsString(); s != g.lastVariationsString {
		g.f.SetVariations(face.variations)
		g.lastVariationsString = s
	}

	var bm faceBitmapState
	if sizes := g.bitmapSizes(); len(sizes) > 0 {
		for _, bs := range sizes {
			if float64(bs.YPpem) == face.Size {
				bm = faceBitmapState{xPpem: bs.XPpem, yPpem: bs.YPpem, useBitmap: true, exact: true}
				break
			}
		}
		if !bm.exact && face.Size > 0 {
			// No strike matches the size exactly. Request the size as
			// the ppem so that the font selects the best strike (the
			// smallest strike not smaller than the request, if any),
			// whose bitmap is scaled at rendering (#2649).
			p := uint16(min(math.Ceil(face.Size), math.MaxUint16))
			bm = faceBitmapState{xPpem: p, yPpem: p, useBitmap: true}
		}
	}
	if bm.xPpem != g.lastXPpem || bm.yPpem != g.lastYPpem {
		g.f.SetPpem(bm.xPpem, bm.yPpem)
		g.lastXPpem, g.lastYPpem = bm.xPpem, bm.yPpem
	}
	return bm
}

// buildOutputs runs HarfBuzz shaping on text and returns the per-segment
// outputs, including per-glyph extents.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) buildOutputs(text string, face *GoTextFace) []shaping.Output {
	_ = g.applyFaceState(face)

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

	inputs := g.seg.Split(input, &singleFontmap{face: g.f})

	// Reorder inputs into visual (left-to-right) order by running the
	// Unicode Bidirectional Algorithm to obtain per-run levels and
	// applying rule L2. The segmenter already splits the text at bidi
	// boundaries, so every input is at a single uniform level. L2 is
	// skipped for vertical faces. Rule L1 is applied by
	// bidi.Paragraph.Segment itself.
	if !face.diDirection().IsVertical() && len(inputs) > 1 {
		defaultBidiDir := bidi.LeftToRight
		if face.diDirection().Progression() == di.TowardTopLeft {
			defaultBidiDir = bidi.RightToLeft
		}
		bidiRuns := g.bidiPara.Segment(g.runes, defaultBidiDir)

		g.bidiLevelsBuf = slices.Grow(g.bidiLevelsBuf[:0], len(inputs))
		var bi int
		for _, in := range inputs {
			for bi+1 < bidiRuns.NumRuns() && bidiRuns.Run(bi).End <= in.RunStart {
				bi++
			}
			var level bidi.Level
			if bidiRuns.NumRuns() > 0 {
				level = bidiRuns.Run(bi).Level
			}
			g.bidiLevelsBuf = append(g.bidiLevelsBuf, level)
		}

		g.visualOrderBuf = appendL2VisualOrder(g.visualOrderBuf, g.bidiLevelsBuf)
		g.inputsBuf = slices.Grow(g.inputsBuf[:0], len(inputs))
		for _, li := range g.visualOrderBuf {
			g.inputsBuf = append(g.inputsBuf, inputs[li])
		}
		inputs = g.inputsBuf
	}

	outputs := make([]shaping.Output, len(inputs))
	for i, input := range inputs {
		out := g.shaper.Shape(input)
		outputs[i] = out

		(shaping.Line{out}).AdjustBaselines()
	}
	return outputs
}

// appendL2VisualOrder appends to dst[:0] a permutation of
// [0, len(levels)) that lists the input indices in visual
// left-to-right order, per the Unicode Bidirectional Algorithm rule
// L2: from the highest level in the line down to the lowest odd
// level, reverse each contiguous run of indices whose level is at
// least the current pass level.
func appendL2VisualOrder(dst []int, levels []bidi.Level) []int {
	dst = slices.Grow(dst[:0], len(levels))
	for i := range levels {
		dst = append(dst, i)
	}
	if len(levels) <= 1 {
		return dst
	}

	var maxLevel bidi.Level
	minOddLevel := bidi.Level(127)
	for _, l := range levels {
		if l > maxLevel {
			maxLevel = l
		}
		if l%2 == 1 && l < minOddLevel {
			minOddLevel = l
		}
	}
	if minOddLevel > maxLevel {
		return dst
	}

	for level := maxLevel; level >= minOddLevel; level-- {
		for i := 0; i < len(dst); {
			if levels[dst[i]] < level {
				i++
				continue
			}
			j := i
			for j < len(dst) && levels[dst[j]] >= level {
				j++
			}
			slices.Reverse(dst[i:j])
			i = j
		}
	}
	return dst
}

// appendChunksVisualOrder appends to dstOrder[:0] a permutation of
// [0, len(chunks)) that lists the chunks in UAX #9 L2 visual order.
// levelsBuf is used as scratch space for the per-chunk level slice;
// pass nil if no buffer is available.
func appendChunksVisualOrder(dstOrder []int, levelsBuf []bidi.Level, chunks []chunk.Chunk) []int {
	levelsBuf = slices.Grow(levelsBuf[:0], len(chunks))
	for _, ch := range chunks {
		levelsBuf = append(levelsBuf, ch.Level)
	}
	return appendL2VisualOrder(dstOrder, levelsBuf)
}

// buildGlyphs converts already-shaped outputs into per-glyph render
// data entries (each carrying eager bounds plus the parameters needed
// for a later realize).
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) buildGlyphs(outputs []shaping.Output, text string, face *GoTextFace) []goTextGlyph {
	bm := g.applyFaceState(face)

	var indices []int
	for i := range text {
		indices = append(indices, i)
	}
	indices = append(indices, len(text))

	variations := face.ensureVariationsString()

	// Snapshot the variations slice once; every glyphRenderData built
	// in this call shares the same underlying copy. The user-facing
	// face.variations may be replaced by a later SetVariation, so each
	// buildGlyphs call needs its own copy, but all glyphs within one
	// call can safely point at the same snapshot.
	var variationsSnapshot []font.Variation
	if len(face.variations) > 0 {
		variationsSnapshot = append([]font.Variation(nil), face.variations...)
	}

	var gs []goTextGlyph
	for _, out := range outputs {
		sideways := out.Direction.IsSideways()
		for _, gl := range out.Glyphs {
			if g.renderDataCache == nil {
				g.renderDataCache = newCache[glyphRenderDataCacheKey, *glyphRenderData](512)
			}
			renderKey := glyphRenderDataCacheKey{
				gid:        gl.GlyphID,
				variations: variations,
				sideways:   sideways,
				size:       out.Size,
			}
			render := g.renderDataCache.getOrCreate(renderKey, func() (*glyphRenderData, bool) {
				return g.buildRenderData(gl, out.Size, sideways, variationsSnapshot, variations, bm)
			})

			gs = append(gs, goTextGlyph{
				shapingGlyph: &gl,
				startIndex:   indices[gl.TextIndex()],
				endIndex:     indices[gl.TextIndex()+gl.RunesCount()],
				render:       render,
			})
		}
	}
	return gs
}

// buildRenderData computes the eager parts of a glyph's render data —
// its bounding rectangle and the bitmap-mode discriminator — and
// captures the parameters needed to defer everything else (the
// GlyphData fetch, segment scaling, bitmap decoding) to a later
// [GoTextFaceSource.realizeRenderData] call. It returns (nil, false)
// for glyphs that produce no bounds (control characters, glyphs
// absent from the font) when not in bitmap mode; in bitmap mode a
// render data is returned even if GlyphExtents fails, in case the
// realize step finds a usable bitmap entry for the glyph.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) buildRenderData(gl shaping.Glyph, size fixed.Int26_6, sideways bool, variations []font.Variation, variationsString string, bm faceBitmapState) (*glyphRenderData, bool) {
	useBitmap := bm.useBitmap
	if useBitmap && !bm.exact {
		// At a size without a pixel-perfect strike, decide per glyph
		// whether to scale the strike's bitmap. Color bitmaps (color
		// emoji fonts) are always used since these glyphs have no other
		// renderable form. A monochrome bitmap is dropped in favor of
		// the glyph's outline, which renders with better fidelity at
		// arbitrary sizes; color bitmap fonts typically have only stub
		// outlines, which must not win here.
		switch d := g.fetchGlyphData(gl.GlyphID, variationsString, sideways, size, gl.YOffset, true).(type) {
		case font.GlyphBitmap:
			switch d.Format {
			case font.BlackAndWhite, font.BlackAndWhiteByteAligned:
				if d.Outline != nil && len(d.Outline.Segments) > 0 {
					useBitmap = false
				}
			}
		default:
			useBitmap = false
		}
	}
	// bounds is the source of truth for the glyph's rendered
	// rectangle on both the outline and bitmap render paths. In
	// outline mode [font.Face.GlyphExtents] resolves through glyf or
	// CFF and matches the bounds of the eventually-realized segments.
	// In bitmap mode it resolves through sbix or CBDT/EBDT and
	// matches the dimensions of the bitmap that realize will decode.
	var bounds fixed.Rectangle26_6
	if ext, ok := g.glyphExtents(gl.GlyphID); ok {
		var yOffset float32
		if sideways {
			yOffset = fixed26_6ToFloat32(-gl.YOffset) / fixed26_6ToFloat32(size) * float32(g.f.Upem())
		}
		scale := float32(g.scale(fixed26_6ToFloat64(size)))
		bounds = glyphExtentsToBounds(ext, scale, sideways, yOffset)
	}

	if bounds.Empty() && !useBitmap {
		return nil, false
	}

	// variations is a caller-owned snapshot — taken by buildGlyphs
	// once per call and shared across every glyph it builds. Storing
	// the slice header by value is safe because the caller never
	// mutates the underlying array.
	return &glyphRenderData{
		bounds:      bounds,
		useBitmap:   useBitmap,
		source:      g,
		gid:         gl.GlyphID,
		size:        size,
		sideways:    sideways,
		yOffset:     gl.YOffset,
		variations:  variations,
		bitmapXPpem: bm.xPpem,
		bitmapYPpem: bm.yPpem,
	}, true
}

// svgGlyphData returns the rasterization data for gid's OpenType SVG glyph:
// the document reduced to the glyph's own description, plus the
// resolved viewBox. It returns nil when the glyph cannot be extracted
// from a document shared by multiple glyphs.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) svgGlyphData(svg font.GlyphSVG, gid font.GID) *svgGlyphData {
	if len(svg.Source) == 0 {
		return nil
	}
	key := &svg.Source[0]
	idx, ok := g.svgDocIndexes[key]
	if !ok {
		idx = newSVGDocIndex(svg.Source)
		if g.svgDocIndexes == nil {
			g.svgDocIndexes = map[*byte]*svgDocIndex{}
		}
		// The cache grows by one entry per document for uncompressed
		// SVG tables. Gzipped documents miss the cache on every fetch;
		// the reset keeps such fonts from accumulating entries forever.
		if len(g.svgDocIndexes) >= 512 {
			clear(g.svgDocIndexes)
		}
		g.svgDocIndexes[key] = idx
	}
	source := idx.glyphDocument(uint32(gid))
	if source == nil {
		return nil
	}
	return &svgGlyphData{
		source:  source,
		viewBox: svg.ViewBox,
	}
}

// glyphExtents returns the extents of gid. A COLRv0 color glyph
// resolves to the union of its layers' extents: the layers, not the
// base glyph's own outline, are what is rendered, and they may extend
// beyond it. A COLRv1 color glyph resolves to its clip box, if any:
// such glyphs typically have an empty base outline, and the clip box
// also bounds the SVG fallback used to render them.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) glyphExtents(gid font.GID) (font.GlyphExtents, bool) {
	if g.f.COLR != nil {
		if c, ok := g.f.GlyphDataColor(gid); ok {
			if layers, ok := c.Paint.(tables.PaintColrLayersResolved); ok {
				if ext, ok := g.colrV0LayersExtents(layers); ok {
					return ext, true
				}
			} else if ext, ok := colrClipBoxExtents(g.f.COLR.ClipList, gid); ok {
				return ext, true
			}
		}
	}
	return g.f.GlyphExtents(gid)
}

// colrV0LayersExtents returns the union of the extents of COLRv0 layers.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) colrV0LayersExtents(layers tables.PaintColrLayersResolved) (font.GlyphExtents, bool) {
	var x0, x1, y0, y1 float32
	var found bool
	for _, l := range layers {
		e, ok := g.f.GlyphExtents(font.GID(l.GlyphID))
		if !ok {
			continue
		}
		// YBearing is the top edge and Height is negative.
		ex0, ex1 := e.XBearing, e.XBearing+e.Width
		ey0, ey1 := e.YBearing+e.Height, e.YBearing
		if !found {
			x0, x1, y0, y1 = ex0, ex1, ey0, ey1
			found = true
			continue
		}
		x0, x1 = min(x0, ex0), max(x1, ex1)
		y0, y1 = min(y0, ey0), max(y1, ey1)
	}
	if !found {
		return font.GlyphExtents{}, false
	}
	return font.GlyphExtents{
		XBearing: x0,
		YBearing: y1,
		Width:    x1 - x0,
		Height:   y0 - y1,
	}, true
}

// colrClipBoxExtents returns the extents of gid's COLRv1 clip box, if
// present in the clip list.
func colrClipBoxExtents(clips tables.ClipList, gid font.GID) (font.GlyphExtents, bool) {
	if gid > math.MaxUint16 {
		return font.GlyphExtents{}, false
	}
	box, ok := clips.Search(tables.GlyphID(gid))
	if !ok {
		return font.GlyphExtents{}, false
	}
	var xMin, yMin, xMax, yMax int16
	switch b := box.(type) {
	case tables.ClipBoxFormat1:
		xMin, yMin, xMax, yMax = b.XMin, b.YMin, b.XMax, b.YMax
	case tables.ClipBoxFormat2:
		// Variation deltas are not applied.
		xMin, yMin, xMax, yMax = b.XMin, b.YMin, b.XMax, b.YMax
	default:
		return font.GlyphExtents{}, false
	}
	return font.GlyphExtents{
		XBearing: float32(xMin),
		YBearing: float32(yMax),
		Width:    float32(xMax) - float32(xMin),
		Height:   float32(yMin) - float32(yMax),
	}, true
}

// fetchGlyphData returns the glyph data for gid via the glyph data
// cache. Cached entries are reusable across glyph instances sharing
// (gid, variations, sideways, size). useBitmap selects the bitmap-mode
// cache key; the outline path is ppem-independent and shares one entry
// across sizes (see glyphDataCacheKey).
//
// The caller must hold g.shapeMu, and the shared font state must
// reflect the given variations and the bitmap-mode ppem.
func (g *GoTextFaceSource) fetchGlyphData(gid font.GID, variationsString string, sideways bool, size, yOffset fixed.Int26_6, useBitmap bool) font.GlyphData {
	if g.glyphDataCache == nil {
		g.glyphDataCache = newCache[glyphDataCacheKey, font.GlyphData](512)
	}
	var keySize fixed.Int26_6
	if useBitmap {
		keySize = size
	}
	key := glyphDataCacheKey{
		gid:        gid,
		variations: variationsString,
		sideways:   sideways,
		size:       keySize,
	}
	return g.glyphDataCache.getOrCreate(key, func() (font.GlyphData, bool) {
		d := g.f.GlyphData(gid)
		if d == nil {
			return nil, false
		}
		if outline, ok := d.(font.GlyphOutline); ok && sideways {
			outline.Sideways(fixed26_6ToFloat32(-yOffset) / fixed26_6ToFloat32(size) * float32(g.f.Upem()))
		}
		return d, true
	})
}

// realizeRenderData performs the deferred work that buildRenderData
// captured params for: fetching glyph data, scaling outline segments,
// and decoding bitmaps, then writes the results back onto rd.
//
// realizeRenderData acquires g.shapeMu because it mutates shared
// font state (g.f variations and ppem) and reads from g.glyphDataCache.
func (g *GoTextFaceSource) realizeRenderData(rd *glyphRenderData) {
	g.shapeMu.Lock()
	defer g.shapeMu.Unlock()

	// Re-apply the face state captured at build time. The current
	// g.f state may belong to a different face configuration if
	// shaping or another realize has run since.
	variationsString := encodeVariations(rd.variations)
	if variationsString != g.lastVariationsString {
		g.f.SetVariations(rd.variations)
		g.lastVariationsString = variationsString
	}
	var wantX, wantY uint16
	if rd.useBitmap {
		wantX, wantY = rd.bitmapXPpem, rd.bitmapYPpem
	}
	if wantX != g.lastXPpem || wantY != g.lastYPpem {
		g.f.SetPpem(wantX, wantY)
		g.lastXPpem, g.lastYPpem = wantX, wantY
	}

	data := g.fetchGlyphData(rd.gid, variationsString, rd.sideways, rd.size, rd.yOffset, rd.useBitmap)
	if data == nil {
		return
	}

	var rawSegs []opentype.Segment
	var rawBitmap font.GlyphBitmap
	var hasRawBitmap bool
	switch d := data.(type) {
	case font.GlyphOutline:
		rawSegs = d.Segments
	case font.GlyphSVG:
		rawSegs = d.Outline.Segments
		// A sideways SVG glyph would need a rotated rasterization, which
		// is not supported; the fallback outline is used instead.
		if !rd.sideways {
			rd.realizedSVG = g.svgGlyphData(d, rd.gid)
		}
	case font.GlyphBitmap:
		if d.Outline != nil {
			rawSegs = d.Outline.Segments
		}
		if rd.useBitmap {
			rawBitmap = d
			hasRawBitmap = true
		}
	case font.GlyphColor:
		// Only COLRv0 layer lists are supported. A COLRv1 paint graph
		// (gradients, transforms, compositions) falls back to the SVG
		// table, if any, and then to the base glyph's outline. A
		// sideways color glyph would need a rotated rasterization,
		// which is not supported either.
		if layers, ok := d.Paint.(tables.PaintColrLayersResolved); ok && !rd.sideways {
			scale := float32(g.scale(fixed26_6ToFloat64(rd.size)))
			rd.realizedCOLRV0Layers = g.appendCOLRV0Layers(nil, layers, scale)
		}
		if len(rd.realizedCOLRV0Layers) == 0 && !rd.sideways {
			if svg, ok := g.f.GlyphDataSVG(rd.gid); ok {
				rd.realizedSVG = g.svgGlyphData(svg, rd.gid)
			}
			if rd.realizedSVG == nil {
				if o, ok := g.f.GlyphDataOutline(rd.gid); ok {
					rawSegs = o.Segments
				}
			}
		}
	}

	if len(rawSegs) > 0 {
		scale := float32(g.scale(fixed26_6ToFloat64(rd.size)))
		segs := make([]opentype.Segment, len(rawSegs))
		for i, seg := range rawSegs {
			segs[i] = seg
			for j := range seg.Args {
				segs[i].Args[j].X *= scale
				segs[i].Args[j].Y *= -scale
			}
		}
		rd.realizedSegments = segs
	}
	if hasRawBitmap {
		rd.realizedBitmap = scaleBitmapToBounds(decodeBitmapGlyph(rawBitmap), rd.bounds)
	}
}

// scaleBitmapToBounds scales a decoded bitmap glyph image to the pixel
// dimensions of bounds, which is where layout places the glyph image.
// The image is returned as is when it already matches, in particular
// when the face size equals the strike's pixel size.
func scaleBitmapToBounds(img image.Image, bounds fixed.Rectangle26_6) image.Image {
	if img == nil {
		return nil
	}
	w, h := (bounds.Max.X - bounds.Min.X).Ceil(), (bounds.Max.Y - bounds.Min.Y).Ceil()
	if w <= 0 || h <= 0 {
		return img
	}
	if b := img.Bounds(); b.Dx() == w && b.Dy() == h {
		return img
	}
	scaled := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.BiLinear.Scale(scaled, scaled.Bounds(), img, img.Bounds(), xdraw.Over, nil)
	return scaled
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
	g.unscaledMetricsOnce.Do(func() {
		um := &g.unscaledMetrics
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
	})

	um := g.unscaledMetrics
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

// chunkPlanKey keys the memo so a Source shared by faces of different
// paragraph direction can't return one face's chunk plan to the other.
type chunkPlanKey struct {
	text  string
	level bidi.Level
}

// chunks returns the chunk plan for text under face. Vertical faces
// fall back to a single chunk covering only the first line, since the
// chunker only handles horizontal text (LTR or RTL base). The result
// is memoized so repeated calls within a frame don't re-walk the
// input.
func (g *GoTextFaceSource) chunks(text string, face *GoTextFace) []chunk.Chunk {
	if face.diDirection().IsVertical() {
		n := textutil.FirstLineLen(text)
		return []chunk.Chunk{{Start: 0, End: n}}
	}

	var paragraphLevel bidi.Level
	if face.Direction == DirectionRightToLeft {
		paragraphLevel = 1
	}
	key := chunkPlanKey{text: text, level: paragraphLevel}
	return g.chunkPlanCache.getOrCreate(key, func() ([]chunk.Chunk, bool) {
		return chunk.AppendChunks(nil, text, paragraphLevel), true
	})
}

func (g *GoTextFaceSource) bitmapSizes() []font.BitmapSize {
	g.bitmapSizesOnce.Do(func() {
		g.bitmapSizesResult = g.f.BitmapSizes()
	})
	return g.bitmapSizesResult
}

func decodeBitmapGlyph(data font.GlyphBitmap) image.Image {
	switch data.Format {
	case font.BlackAndWhite:
		img := image.NewAlpha(image.Rect(0, 0, data.Width, data.Height))
		for j := range data.Height {
			for i := range data.Width {
				idx := j*data.Width + i
				if data.Data[idx/8]&(1<<(7-idx%8)) != 0 {
					img.Pix[j*img.Stride+i] = 0xff
				}
			}
		}
		return img
	case font.BlackAndWhiteByteAligned:
		img := image.NewAlpha(image.Rect(0, 0, data.Width, data.Height))
		rowBytes := (data.Width + 7) / 8
		for j := range data.Height {
			for i := range data.Width {
				byteIdx := j*rowBytes + i/8
				if data.Data[byteIdx]&(1<<(7-i%8)) != 0 {
					img.Pix[j*img.Stride+i] = 0xff
				}
			}
		}
		return img
	case font.PNG:
		img, err := png.Decode(bytes.NewReader(data.Data))
		if err != nil {
			return nil
		}
		return img
	case font.JPG:
		img, err := jpeg.Decode(bytes.NewReader(data.Data))
		if err != nil {
			return nil
		}
		return img
	case font.TIFF:
		img, err := tiff.Decode(bytes.NewReader(data.Data))
		if err != nil {
			return nil
		}
		return img
	}
	return nil
}

type singleFontmap struct {
	face *font.Face
}

func (s *singleFontmap) ResolveFace(r rune) *font.Face {
	return s.face
}
