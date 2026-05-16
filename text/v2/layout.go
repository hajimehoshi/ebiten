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
	"math"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/textutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Align is the alignment that determines how to put a text.
type Align int

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
)

// DrawOptions represents options for the Draw function.
//
// DrawOption embeds ebiten.DrawImageOptions.
// DrawImageOptions.GeoM is an additional geometry transformation
// after putting the rendering region along with the specified alignments.
// DrawImageOptions.ColorScale scales the text color.
type DrawOptions struct {
	ebiten.DrawImageOptions
	LayoutOptions
}

// LayoutOptions represents options for layouting texts.
//
// PrimaryAlign and SecondaryAlign determine where to put the text in the given region at Draw.
// Draw might render the text outside of the specified image bounds, so you might have to specify GeoM to make the text visible.
type LayoutOptions struct {
	// LineSpacing is a distance between two adjacent lines's baselines.
	// The unit is in pixels.
	LineSpacing float64

	// PrimaryAlign is an alignment of the primary direction, in which a text in one line is rendered.
	// The primary direction is the horizontal direction for a horizontal-direction face,
	// and the vertical direction for a vertical-direction face.
	// The meaning of the start and the end depends on the face direction.
	PrimaryAlign Align

	// SecondaryAlign is an alignment of the secondary direction, in which multiple lines are rendered.
	// The secondary direction is the vertical direction for a horizontal-direction face,
	// and the horizontal direction for a vertical-direction face.
	// The meaning of the start and the end depends on the face direction.
	SecondaryAlign Align
}

var theDrawGlyphsPool = sync.Pool{
	New: func() any {
		// 64 is an arbitrary number for the initial capacity.
		s := make([]LazyGlyph, 0, 64)
		// Return a pointer instead of a slice, or go-vet warns at Put.
		return &s
	},
}

// drawGlyphEntry is a realized glyph image plus its destination-space
// translation, used by [Draw] to defer DrawImage calls until after all glyph
// images have been rasterized.
type drawGlyphEntry struct {
	img *ebiten.Image
	x   float64
	y   float64
}

var theDrawGlyphEntriesPool = sync.Pool{
	New: func() any {
		s := make([]drawGlyphEntry, 0, 64)
		return &s
	},
}

// Draw draws a given text on a given destination image dst.
// face is the font for text rendering.
//
// New line characters like '\n' put the following text on the next line.
// The next line starts at the position shifted by LayoutOptions.LineSpacing.
// By default, LayoutOptions.LineSpacing is 0, so you need to specify LineSpacing explicitly if you want to put multiple lines.
//
// Glyphs used for rendering are cached in least-recently-used way.
// Then old glyphs might be evicted from the cache.
// As the cache capacity has limit, it is not guaranteed that all the glyphs for runes given at Draw are cached.
//
// It is OK to call Draw with a same text and a same face at every frame in terms of performance.
//
// Draw is concurrent-safe.
//
// # Rendering region
//
// A rectangle region where a text is put is called a 'rendering region'.
// The position of the text in the rendering region is determined by the specified primary and secondary alignments.
//
// The actual rendering position of the rendering region depends on the alignments in DrawOptions.
// By default, if the face's primary direction is left-to-right, the rendering region's upper-left position is (0, 0).
// Note that this is different from text v1. In text v1, (0, 0) is always the origin position.
//
// # Alignments
//
// For horizontal directions, the start and end depends on the face.
// If the face is GoTextFace, the start and the end depend on the Direction property.
// If the face is GoXFace, the start and the end are always left and right respectively.
//
// For vertical directions, the start and end are top and bottom respectively.
//
// If the horizontal alignment is left, the rendering region's left X comes to the destination image's origin (0, 0).
// If the horizontal alignment is center, the rendering region's middle X comes to the origin.
// If the horizontal alignment is right, the rendering region's right X comes to the origin.
//
// If the vertical alignment is top, the rendering region's top Y comes to the destination image's origin (0, 0).
// If the vertical alignment is center, the rendering region's middle Y comes to the origin.
// If the vertical alignment is bottom, the rendering region's bottom Y comes to the origin.
func Draw(dst *ebiten.Image, text string, face Face, options *DrawOptions) {
	var layoutOp LayoutOptions
	var drawOp ebiten.DrawImageOptions

	if options != nil {
		layoutOp = options.LayoutOptions
		drawOp = options.DrawImageOptions
	}

	geoM := drawOp.GeoM

	glyphs := theDrawGlyphsPool.Get().(*[]LazyGlyph)
	defer func() {
		// Clear the content to avoid memory leaks.
		// The capacity is kept so that the next call to Draw can reuse it.
		*glyphs = slices.Delete(*glyphs, 0, len(*glyphs))
		theDrawGlyphsPool.Put(glyphs)
	}()
	*glyphs = AppendLazyGlyphs((*glyphs)[:0], text, face, &layoutOp)

	entries := theDrawGlyphEntriesPool.Get().(*[]drawGlyphEntry)
	defer func() {
		*entries = slices.Delete(*entries, 0, len(*entries))
		theDrawGlyphEntriesPool.Put(entries)
	}()

	// Realize images, then draw, in two passes: interleaving DrawImage
	// with glyph realization flushes the pending atlas write-pixels batch
	// on each glyph and fragments draw calls (#3455).
	dstBounds := dst.Bounds()
	for _, g := range *glyphs {
		if g.ImageBounds.Empty() {
			continue
		}
		// Cull glyphs whose transformed bounding box does not overlap dst.
		// This avoids rasterizing offscreen glyphs.
		if !transformedRectOverlaps(geoM, g.ImageBounds, dstBounds) {
			continue
		}
		img := g.Image()
		if img == nil {
			continue
		}
		*entries = append(*entries, drawGlyphEntry{
			img: img,
			x:   float64(g.ImageBounds.Min.X),
			y:   float64(g.ImageBounds.Min.Y),
		})
	}

	for _, e := range *entries {
		drawOp.GeoM.Reset()
		drawOp.GeoM.Translate(e.x, e.y)
		drawOp.GeoM.Concat(geoM)
		dst.DrawImage(e.img, &drawOp)
	}
}

// transformedRectOverlaps reports whether rect's image under geoM overlaps
// dst. The transformed rectangle is expanded to integer pixel boundaries
// before the overlap check.
func transformedRectOverlaps(geoM ebiten.GeoM, rect, dst image.Rectangle) bool {
	x0f, y0f := float64(rect.Min.X), float64(rect.Min.Y)
	x1f, y1f := float64(rect.Max.X), float64(rect.Max.Y)
	// Transform the four corners of the rectangle into destination space.
	ax, ay := geoM.Apply(x0f, y0f)
	bx, by := geoM.Apply(x1f, y0f)
	cx, cy := geoM.Apply(x0f, y1f)
	dx, dy := geoM.Apply(x1f, y1f)

	minX := min(ax, bx, cx, dx)
	minY := min(ay, by, cy, dy)
	maxX := max(ax, bx, cx, dx)
	maxY := max(ay, by, cy, dy)

	transformed := image.Rect(
		int(math.Floor(minX)),
		int(math.Floor(minY)),
		int(math.Ceil(maxX)),
		int(math.Ceil(maxY)),
	)
	return transformed.Overlaps(dst)
}

// AppendGlyphs appends glyphs to the given slice and returns a slice.
//
// AppendGlyphs is a low-level API, and you can use AppendGlyphs to have more control than Draw.
// AppendGlyphs is also available to precache glyphs.
//
// AppendGlyphs rasterizes every glyph eagerly. If you do not need the
// rasterized images for every glyph (for example, for hit testing or
// visibility culling), consider [AppendLazyGlyphs] instead, which defers
// rasterization until [LazyGlyph.Image] is called.
//
// For the details of options, see Draw function.
//
// AppendGlyphs is concurrent-safe.
func AppendGlyphs(glyphs []Glyph, text string, face Face, options *LayoutOptions) []Glyph {
	lazyBufP := theAppendGlyphsLazyBufPool.Get().(*[]LazyGlyph)
	lazyBuf := (*lazyBufP)[:0]
	defer func() {
		*lazyBufP = slices.Delete(lazyBuf, 0, len(lazyBuf))
		theAppendGlyphsLazyBufPool.Put(lazyBufP)
	}()
	forEachLine(text, face, options, func(line string, indexOffset int, originX, originY float64) {
		before := len(lazyBuf)
		lazyBuf = face.appendLazyGlyphsForLine(lazyBuf, line, indexOffset, originX, originY)
		for i := before; i < len(lazyBuf); i++ {
			lg := lazyBuf[i]
			glyphs = append(glyphs, Glyph{
				StartIndexInBytes: lg.StartIndexInBytes,
				EndIndexInBytes:   lg.EndIndexInBytes,
				GID:               lg.GID,
				Image:             lg.Image(),
				X:                 float64(lg.ImageBounds.Min.X),
				Y:                 float64(lg.ImageBounds.Min.Y),
				OriginX:           lg.OriginX,
				OriginY:           lg.OriginY,
				OriginOffsetX:     lg.OriginOffsetX,
				OriginOffsetY:     lg.OriginOffsetY,
				AdvanceX:          lg.AdvanceX,
				AdvanceY:          lg.AdvanceY,
			})
		}
	})
	return glyphs
}

// AppendLazyGlyphs appends lazy glyphs to the given slice and returns a slice.
//
// AppendLazyGlyphs is similar to [AppendGlyphs] but each [LazyGlyph] defers
// rasterization until [LazyGlyph.Image] is called. This is useful when only
// glyph layout is needed (such as hit testing) or when a custom draw loop
// culls glyphs that fall outside a viewport.
//
// For the details of options, see Draw function.
//
// AppendLazyGlyphs is concurrent-safe.
func AppendLazyGlyphs(glyphs []LazyGlyph, text string, face Face, options *LayoutOptions) []LazyGlyph {
	return appendLazyGlyphs(glyphs, text, face, 0, 0, options)
}

// AppendVectorPath appends a vector path for glyphs to the given path.
//
// AppendVectorPath works only when the face is *GoTextFace or a composite face using *GoTextFace so far.
// For other types, AppendVectorPath does nothing.
func AppendVectorPath(path *vector.Path, text string, face Face, options *LayoutOptions) {
	forEachLine(text, face, options, func(line string, indexOffset int, originX, originY float64) {
		face.appendVectorPathForLine(path, line, originX, originY)
	})
}

// appendLazyGlyphs appends lazy glyphs to the given slice and returns a slice.
//
// appendLazyGlyphs assumes the text is rendered with the position (x, y).
// (x, y) might affect the subpixel rendering results.
func appendLazyGlyphs(glyphs []LazyGlyph, text string, face Face, x, y float64, options *LayoutOptions) []LazyGlyph {
	forEachLine(text, face, options, func(line string, indexOffset int, originX, originY float64) {
		glyphs = face.appendLazyGlyphsForLine(glyphs, line, indexOffset, originX+x, originY+y)
	})
	return glyphs
}

var theAppendGlyphsLazyBufPool = sync.Pool{
	New: func() any {
		s := make([]LazyGlyph, 0, 64)
		return &s
	},
}

// forEachLine interates lines.
func forEachLine(text string, face Face, options *LayoutOptions, f func(text string, indexOffset int, originX, originY float64)) {
	if text == "" {
		return
	}

	if options == nil {
		options = &LayoutOptions{}
	}

	// Calculate the advances for each line.
	var advances []float64
	var longestAdvance float64
	var lineCount int
	for line := range textutil.Lines(text) {
		lineCount++
		a := face.advance(textutil.TrimTailingLineBreak(line))
		advances = append(advances, a)
		longestAdvance = max(longestAdvance, a)
	}

	d := face.direction()
	m := face.Metrics()

	var boundaryWidth, boundaryHeight float64
	if d.isHorizontal() {
		boundaryWidth = longestAdvance
		boundaryHeight = float64(lineCount-1)*options.LineSpacing + m.HAscent + m.HDescent
	} else {
		// TODO: Perhaps HAscent and HDescent should be used for sideways glyphs.
		boundaryWidth = float64(lineCount-1)*options.LineSpacing + m.VAscent + m.VDescent
		boundaryHeight = longestAdvance
	}

	var offsetX, offsetY float64

	// Adjust the offset based on the secondary alignments.
	h, v := calcAligns(d, options.PrimaryAlign, options.SecondaryAlign)
	switch d {
	case DirectionLeftToRight, DirectionRightToLeft:
		offsetY += m.HAscent
		switch v {
		case verticalAlignTop:
		case verticalAlignCenter:
			offsetY -= boundaryHeight / 2
		case verticalAlignBottom:
			offsetY -= boundaryHeight
		}
	case DirectionTopToBottomAndLeftToRight:
		// TODO: Perhaps HDescent should be used for sideways glyphs.
		offsetX += m.VDescent
		switch h {
		case horizontalAlignLeft:
		case horizontalAlignCenter:
			offsetX -= boundaryWidth / 2
		case horizontalAlignRight:
			offsetX -= boundaryWidth
		}
	case DirectionTopToBottomAndRightToLeft:
		// TODO: Perhaps HAscent should be used for sideways glyphs.
		offsetX -= m.VAscent
		switch h {
		case horizontalAlignLeft:
			offsetX += boundaryWidth
		case horizontalAlignCenter:
			offsetX += boundaryWidth / 2
		case horizontalAlignRight:
		}
	}

	var indexOffset int
	var originX, originY float64
	var i int
	for line := range textutil.Lines(text) {
		// Adjust the origin position based on the primary alignments.
		switch d {
		case DirectionLeftToRight, DirectionRightToLeft:
			switch h {
			case horizontalAlignLeft:
				originX = 0
			case horizontalAlignCenter:
				originX = -advances[i] / 2
			case horizontalAlignRight:
				originX = -advances[i]
			}
		case DirectionTopToBottomAndLeftToRight, DirectionTopToBottomAndRightToLeft:
			switch v {
			case verticalAlignTop:
				originY = 0
			case verticalAlignCenter:
				originY = -advances[i] / 2
			case verticalAlignBottom:
				originY = -advances[i]
			}
		}

		line = textutil.TrimTailingLineBreak(line)
		f(line, indexOffset, originX+offsetX, originY+offsetY)

		indexOffset += len(line) + 1
		i++

		// Advance the origin position in the secondary direction.
		switch face.direction() {
		case DirectionLeftToRight:
			originY += options.LineSpacing
		case DirectionRightToLeft:
			originY += options.LineSpacing
		case DirectionTopToBottomAndLeftToRight:
			originX += options.LineSpacing
		case DirectionTopToBottomAndRightToLeft:
			originX -= options.LineSpacing
		}
	}
}

type horizontalAlign int

const (
	horizontalAlignLeft horizontalAlign = iota
	horizontalAlignCenter
	horizontalAlignRight
)

type verticalAlign int

const (
	verticalAlignTop verticalAlign = iota
	verticalAlignCenter
	verticalAlignBottom
)

func calcAligns(direction Direction, primaryAlign, secondaryAlign Align) (horizontalAlign, verticalAlign) {
	var h horizontalAlign
	var v verticalAlign

	switch direction {
	case DirectionLeftToRight:
		switch primaryAlign {
		case AlignStart:
			h = horizontalAlignLeft
		case AlignCenter:
			h = horizontalAlignCenter
		case AlignEnd:
			h = horizontalAlignRight
		}
		switch secondaryAlign {
		case AlignStart:
			v = verticalAlignTop
		case AlignCenter:
			v = verticalAlignCenter
		case AlignEnd:
			v = verticalAlignBottom
		}
	case DirectionRightToLeft:
		switch primaryAlign {
		case AlignStart:
			h = horizontalAlignRight
		case AlignCenter:
			h = horizontalAlignCenter
		case AlignEnd:
			h = horizontalAlignLeft
		}
		switch secondaryAlign {
		case AlignStart:
			v = verticalAlignTop
		case AlignCenter:
			v = verticalAlignCenter
		case AlignEnd:
			v = verticalAlignBottom
		}
	case DirectionTopToBottomAndLeftToRight:
		switch primaryAlign {
		case AlignStart:
			v = verticalAlignTop
		case AlignCenter:
			v = verticalAlignCenter
		case AlignEnd:
			v = verticalAlignBottom
		}
		switch secondaryAlign {
		case AlignStart:
			h = horizontalAlignLeft
		case AlignCenter:
			h = horizontalAlignCenter
		case AlignEnd:
			h = horizontalAlignRight
		}
	case DirectionTopToBottomAndRightToLeft:
		switch primaryAlign {
		case AlignStart:
			v = verticalAlignTop
		case AlignCenter:
			v = verticalAlignCenter
		case AlignEnd:
			v = verticalAlignBottom
		}
		switch secondaryAlign {
		case AlignStart:
			h = horizontalAlignRight
		case AlignCenter:
			h = horizontalAlignCenter
		case AlignEnd:
			h = horizontalAlignLeft
		}
	}

	return h, v
}
