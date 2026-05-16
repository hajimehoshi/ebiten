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
	"slices"
	"sync"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/tiff"
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

	// text and face are the inputs that produced outputs. They are retained so
	// per-glyph data can be built lazily by ensureGlyphs.
	text string
	face *GoTextFace

	// glyphs is lazily built by ensureGlyphs. A nil slice means glyphs have not
	// been built yet; a non-nil (possibly empty) slice means they have.
	// Protected by GoTextFaceSource.shapeMu.
	glyphs []goTextGlyph
}

// ensureGlyphs returns per-glyph data, building it on first access.
func (v *goTextOutputCacheValue) ensureGlyphs(g *GoTextFaceSource) []goTextGlyph {
	g.shapeMu.Lock()
	defer g.shapeMu.Unlock()
	if v.glyphs == nil {
		v.glyphs = g.buildGlyphs(v.outputs, v.text, v.face)
		// The inputs are no longer needed; release the face reference.
		v.text = ""
		v.face = nil
	}
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
// ImageBounds) don't pay for the GlyphData fetch. hasBitmap is an
// eager heuristic — true when the face is at a bitmap-strike size —
// used only by layout to pick the positioning convention.
//
// The actual segments and bitmap are produced on first call to
// [glyphRenderData.segments] or [glyphRenderData.bitmap], which fetch
// glyph data via the source's glyphDataCache.
type glyphRenderData struct {
	bounds    fixed.Rectangle26_6
	hasBitmap bool

	realizeOnce sync.Once

	// Captured at build time for use by realize.
	source      *GoTextFaceSource
	gid         font.GID
	size        fixed.Int26_6
	sideways    bool
	yOffset     fixed.Int26_6
	variations  []font.Variation
	useBitmap   bool
	bitmapXPpem uint16
	bitmapYPpem uint16

	// Populated by realize. Accessed via segments / bitmap so callers
	// don't have to remember to drive the lazy initialization.
	realizedSegments []opentype.Segment
	realizedBitmap   image.Image
}

// segments returns the scaled outline segments, realizing them on
// first call.
func (rd *glyphRenderData) segments() []opentype.Segment {
	rd.realizeOnce.Do(rd.realize)
	return rd.realizedSegments
}

// bitmap returns the decoded bitmap image, realizing it on first
// call. It returns nil when the glyph has no bitmap data.
func (rd *glyphRenderData) bitmap() image.Image {
	rd.realizeOnce.Do(rd.realize)
	return rd.realizedBitmap
}

// realize delegates to [GoTextFaceSource.realizeRenderData], where
// the lock and the work both live.
func (rd *glyphRenderData) realize() {
	rd.source.realizeRenderData(rd)
}

// GoTextFaceSource is a source of a GoTextFace. This can be shared by multiple GoTextFace objects.
type GoTextFaceSource struct {
	f        *font.Face
	metadata Metadata

	outputCache     *cache[goTextOutputCacheKey, *goTextOutputCacheValue]
	advanceCache    *cache[goTextOutputCacheKey, []fixed.Int26_6]
	glyphImageCache map[float64]*cache[goTextGlyphImageCacheKey, *ebiten.Image]
	hasGlyphCache   runeToBoolMap

	unscaledMetrics     Metrics
	unscaledMetricsOnce sync.Once

	addr *GoTextFaceSource

	shaper shaping.HarfbuzzShaper
	seg    shaping.Segmenter

	runes           []rune
	glyphDataCache  *cache[glyphDataCacheKey, font.GlyphData]
	renderDataCache *cache[glyphRenderDataCacheKey, *glyphRenderData]

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
	s.advanceCache = newCache[goTextOutputCacheKey, []fixed.Int26_6](512)
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

// advances returns a cumulative-advance slice of length len(text)+1 such that
// element i is the advance from the start of text to byte index i. A byte
// index that lands inside a glyph cluster gets the same advance as the
// cluster's start (snap-to-prev), since the cluster's advance is recorded
// only at its end-byte position.
func (g *GoTextFaceSource) advances(text string, face *GoTextFace) []fixed.Int26_6 {
	g.copyCheck()

	key := face.outputCacheKey(text)
	return g.advanceCache.getOrCreate(key, func() ([]fixed.Int26_6, bool) {
		g.shapeMu.Lock()
		defer g.shapeMu.Unlock()
		outputs := g.buildOutputs(text, face, true)

		// Rune index → byte index, plus a final entry at len(text).
		runeToByte := make([]int, 0, len(text)+1)
		for i := range text {
			runeToByte = append(runeToByte, i)
		}
		runeToByte = append(runeToByte, len(text))

		// Accumulate each glyph's advance at its cluster end-byte, then
		// prefix-sum to get cumulative advance at every byte position.
		a := make([]fixed.Int26_6, len(text)+1)
		for _, out := range outputs {
			for i := range out.Glyphs {
				gl := &out.Glyphs[i]
				endByte := runeToByte[gl.ClusterIndex+gl.RunesCount()]
				a[endByte] += gl.Advance
			}
		}
		for i := 1; i < len(a); i++ {
			a[i] += a[i-1]
		}
		return a, true
	})
}

// advanceAt returns the advance from the start of text to indexInBytes.
// indexInBytes that falls inside a glyph cluster snaps to the cluster's start.
func (g *GoTextFaceSource) advanceAt(text string, face *GoTextFace, indexInBytes int) fixed.Int26_6 {
	if indexInBytes <= 0 {
		return 0
	}
	a := g.advances(text, face)
	if indexInBytes >= len(a) {
		return a[len(a)-1]
	}
	return a[indexInBytes]
}

// glyphs returns per-glyph wrappers for text, computed from the cached shape
// outputs.
func (g *GoTextFaceSource) glyphs(text string, face *GoTextFace) []goTextGlyph {
	g.copyCheck()

	key := face.outputCacheKey(text)
	e := g.outputCache.getOrCreate(key, func() (*goTextOutputCacheValue, bool) {
		g.shapeMu.Lock()
		defer g.shapeMu.Unlock()
		return &goTextOutputCacheValue{
			outputs: g.buildOutputs(text, face, false),
			text:    text,
			face:    face,
		}, true
	})
	return e.ensureGlyphs(g)
}

// applyFaceState updates the shared font state to reflect face and
// returns the bitmap-strike ppem that was applied. xPpem and yPpem are
// zero — and useBitmap is false — when no strike matches face.Size;
// otherwise they hold the strike's pixel size and useBitmap is true.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) applyFaceState(face *GoTextFace) (xPpem, yPpem uint16, useBitmap bool) {
	if s := face.ensureVariationsString(); s != g.lastVariationsString {
		g.f.SetVariations(face.variations)
		g.lastVariationsString = s
	}

	for _, bs := range g.bitmapSizes() {
		if float64(bs.YPpem) == face.Size {
			xPpem, yPpem, useBitmap = bs.XPpem, bs.YPpem, true
			break
		}
	}
	if xPpem != g.lastXPpem || yPpem != g.lastYPpem {
		g.f.SetPpem(xPpem, yPpem)
		g.lastXPpem, g.lastYPpem = xPpem, yPpem
	}
	return xPpem, yPpem, useBitmap
}

// buildOutputs runs HarfBuzz shaping on text and returns the per-segment outputs.
// When skipExtents is true, glyph extents are not queried, which is cheaper and
// suitable when only advance is required.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) buildOutputs(text string, face *GoTextFace, skipExtents bool) []shaping.Output {
	_, _, _ = g.applyFaceState(face)

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

	// Reverse the input for RTL texts.
	if face.Direction == DirectionRightToLeft {
		slices.Reverse(inputs)
	}

	outputs := make([]shaping.Output, len(inputs))
	for i, input := range inputs {
		var out shaping.Output
		if skipExtents {
			out = g.shaper.ShapeNoExtents(input)
		} else {
			out = g.shaper.Shape(input)
		}
		outputs[i] = out

		(shaping.Line{out}).AdjustBaselines()
	}
	return outputs
}

// buildGlyphs converts already-shaped outputs into per-glyph render
// data entries (each carrying eager bounds plus the parameters needed
// for a later realize). It always returns a non-nil slice so that
// callers can use a nil check to distinguish unbuilt entries from
// entries that built to zero glyphs.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) buildGlyphs(outputs []shaping.Output, text string, face *GoTextFace) []goTextGlyph {
	xPpem, yPpem, useBitmap := g.applyFaceState(face)

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

	gs := []goTextGlyph{}
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
				return g.buildRenderData(gl, out.Size, sideways, variationsSnapshot, useBitmap, xPpem, yPpem)
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
// absent from the font) when not in bitmap mode; in bitmap mode (the
// face is at a size matching one of the font's bitmap strikes) a
// render data is returned even if GlyphExtents fails, in case the
// realize step finds a usable bitmap entry for the glyph.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) buildRenderData(gl shaping.Glyph, size fixed.Int26_6, sideways bool, variations []font.Variation, useBitmap bool, xPpem, yPpem uint16) (*glyphRenderData, bool) {
	// bounds is the source of truth for the glyph's rendered
	// rectangle on both the outline and bitmap render paths. In
	// outline mode [font.Face.GlyphExtents] resolves through glyf or
	// CFF and matches the bounds of the eventually-realized segments.
	// In bitmap mode it resolves through sbix or CBDT/EBDT and
	// matches the dimensions of the bitmap that realize will decode.
	var bounds fixed.Rectangle26_6
	if ext, ok := g.f.GlyphExtents(gl.GlyphID); ok {
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
		hasBitmap:   useBitmap,
		source:      g,
		gid:         gl.GlyphID,
		size:        size,
		sideways:    sideways,
		yOffset:     gl.YOffset,
		variations:  variations,
		useBitmap:   useBitmap,
		bitmapXPpem: xPpem,
		bitmapYPpem: yPpem,
	}, true
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

	// Fetch GlyphData. Cached entries are reusable across glyph
	// instances sharing (gid, variations, sideways, size).
	if g.glyphDataCache == nil {
		g.glyphDataCache = newCache[glyphDataCacheKey, font.GlyphData](512)
	}
	var keySize fixed.Int26_6
	if rd.useBitmap {
		keySize = rd.size
	}
	key := glyphDataCacheKey{
		gid:        rd.gid,
		variations: variationsString,
		sideways:   rd.sideways,
		size:       keySize,
	}
	data := g.glyphDataCache.getOrCreate(key, func() (font.GlyphData, bool) {
		d := g.f.GlyphData(rd.gid)
		if d == nil {
			return nil, false
		}
		if outline, ok := d.(font.GlyphOutline); ok && rd.sideways {
			outline.Sideways(fixed26_6ToFloat32(-rd.yOffset) / fixed26_6ToFloat32(rd.size) * float32(g.f.Upem()))
		}
		return d, true
	})

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
	case font.GlyphBitmap:
		if d.Outline != nil {
			rawSegs = d.Outline.Segments
		}
		if rd.useBitmap {
			rawBitmap = d
			hasRawBitmap = true
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
		rd.realizedBitmap = decodeBitmapGlyph(rawBitmap)
	}
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
