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

	// render is a pointer to the shared render data (scaled segments,
	// bounds, and decoded bitmap). It is nil for glyphs with no outline
	// and no bitmap (e.g. control characters).
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

// glyphRenderData bundles all data needed to render one glyph: scaled
// outline segments, their bounding rectangle, and (when bitmap mode is
// active) the decoded bitmap image.
type glyphRenderData struct {
	segments []opentype.Segment
	bounds   fixed.Rectangle26_6
	bitmap   image.Image
}

// GoTextFaceSource is a source of a GoTextFace. This can be shared by multiple GoTextFace objects.
type GoTextFaceSource struct {
	f        *font.Face
	metadata Metadata

	outputCache     *cache[goTextOutputCacheKey, *goTextOutputCacheValue]
	advanceCache    *cache[goTextOutputCacheKey, fixed.Int26_6]
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
	s.advanceCache = newCache[goTextOutputCacheKey, fixed.Int26_6](512)
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

// advance returns the total advance of text. It uses ShapeNoExtents to skip
// computing glyph extents, which are not needed when only advance is required.
func (g *GoTextFaceSource) advance(text string, face *GoTextFace) fixed.Int26_6 {
	g.copyCheck()

	key := face.outputCacheKey(text)
	return g.advanceCache.getOrCreate(key, func() (fixed.Int26_6, bool) {
		g.shapeMu.Lock()
		defer g.shapeMu.Unlock()
		var a fixed.Int26_6
		for _, out := range g.buildOutputs(text, face, true) {
			a += out.Advance
		}
		return a, true
	})
}

func (g *GoTextFaceSource) shape(text string, face *GoTextFace) ([]shaping.Output, []goTextGlyph) {
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
	return e.outputs, e.ensureGlyphs(g)
}

// applyFaceState updates the shared font state to reflect face. It returns
// whether bitmap glyph data should be used for face.Size.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) applyFaceState(face *GoTextFace) bool {
	g.f.SetVariations(face.variations)

	g.f.SetPpem(0, 0)
	for _, bs := range g.bitmapSizes() {
		if float64(bs.YPpem) == face.Size {
			g.f.SetPpem(bs.XPpem, bs.YPpem)
			return true
		}
	}
	return false
}

// buildOutputs runs HarfBuzz shaping on text and returns the per-segment outputs.
// When skipExtents is true, glyph extents are not queried, which is cheaper and
// suitable when only advance is required.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) buildOutputs(text string, face *GoTextFace, skipExtents bool) []shaping.Output {
	g.applyFaceState(face)

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

// buildGlyphs converts already-shaped outputs into per-glyph segment data.
// It always returns a non-nil slice so that callers can use a nil check to
// distinguish unbuilt entries from entries that built to zero glyphs.
//
// The caller must hold g.shapeMu.
func (g *GoTextFaceSource) buildGlyphs(outputs []shaping.Output, text string, face *GoTextFace) []goTextGlyph {
	useBitmap := g.applyFaceState(face)

	var indices []int
	for i := range text {
		indices = append(indices, i)
	}
	indices = append(indices, len(text))

	variations := face.ensureVariationsString()
	gs := []goTextGlyph{}
	for _, out := range outputs {
		sideways := out.Direction.IsSideways()
		for _, gl := range out.Glyphs {
			// Fetch glyph data. The size field of the cache key is 0 for
			// outline glyphs and out.Size for bitmap glyphs; see the
			// comment on glyphDataCacheKey for the rationale.
			//
			// The outline path assumes the underlying typesetting library
			// does not apply hinting at GlyphData time: today it reads raw
			// outline points from glyf/CFF/CFF2 and variable-font axes
			// (including opsz) route through SetVariations and are
			// captured by variations above. If a hinting interpreter is
			// added in the future, outlines may vary per ppem and the
			// outline path will also need a non-zero size in the key.
			if g.glyphDataCache == nil {
				g.glyphDataCache = newCache[glyphDataCacheKey, font.GlyphData](512)
			}
			var keySize fixed.Int26_6
			if useBitmap {
				keySize = out.Size
			}
			key := glyphDataCacheKey{
				gid:        gl.GlyphID,
				variations: variations,
				sideways:   sideways,
				size:       keySize,
			}
			data := g.glyphDataCache.getOrCreate(key, func() (font.GlyphData, bool) {
				data := g.f.GlyphData(gl.GlyphID)
				if data == nil {
					return nil, false
				}
				if d, ok := data.(font.GlyphOutline); ok && sideways {
					d.Sideways(fixed26_6ToFloat32(-gl.YOffset) / fixed26_6ToFloat32(out.Size) * float32(g.f.Upem()))
				}
				return data, true
			})

			// Extract the raw (unscaled) outline segments and, if bitmap
			// mode is active, the embedded bitmap data.
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
				if useBitmap {
					rawBitmap = d
					hasRawBitmap = true
				}
			}

			// Cache the render data (segments + bounds + bitmap) across
			// glyph instances of the same (gid, variations, sideways, size).
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
				if rawSegs == nil && !hasRawBitmap {
					return nil, false
				}
				rd := &glyphRenderData{}
				if rawSegs != nil {
					segs := make([]opentype.Segment, len(rawSegs))
					scale := float32(g.scale(fixed26_6ToFloat64(out.Size)))
					for i, seg := range rawSegs {
						segs[i] = seg
						for j := range seg.Args {
							segs[i].Args[j].X *= scale
							segs[i].Args[j].Y *= -scale
						}
					}
					rd.segments = segs
					rd.bounds = segmentsToBounds(segs)
				}
				if hasRawBitmap {
					rd.bitmap = decodeBitmapGlyph(rawBitmap)
				}
				return rd, true
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
