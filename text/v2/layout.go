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
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
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

// Draw draws a given text on a given destination image dst.
// face is the font for text rendering.
//
// The '\n' newline character puts the following text on the next line.
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

	for _, g := range AppendGlyphs(nil, text, face, &layoutOp) {
		if g.Image == nil {
			continue
		}
		drawOp.GeoM.Reset()
		drawOp.GeoM.Translate(g.X, g.Y)
		drawOp.GeoM.Concat(geoM)
		dst.DrawImage(g.Image, &drawOp)
	}
}

// AppendGlyphs appends glyphs to the given slice and returns a slice.
//
// AppendGlyphs is a low-level API, and you can use AppendGlyphs to have more control than Draw.
// AppendGlyphs is also available to precache glyphs.
//
// For the details of options, see Draw function.
//
// AppendGlyphs is concurrent-safe.
func AppendGlyphs(glyphs []Glyph, text string, face Face, options *LayoutOptions) []Glyph {
	return appendGlyphs(glyphs, text, face, 0, 0, options)
}

// AppndVectorPath appends a vector path for glyphs to the given path.
//
// AppendVectorPath works only when the face is *GoTextFace or a composite face using *GoTextFace so far.
// For other types, AppendVectorPath does nothing.
func AppendVectorPath(path *vector.Path, text string, face Face, options *LayoutOptions) {
	forEachLine(text, face, options, func(line string, indexOffset int, originX, originY float64) {
		face.appendVectorPathForLine(path, line, originX, originY)
	})
}

// appendGlyphs appends glyphs to the given slice and returns a slice.
//
// appendGlyphs assumes the text is rendered with the position (x, y).
// (x, y) might affect the subpixel rendering results.
func appendGlyphs(glyphs []Glyph, text string, face Face, x, y float64, options *LayoutOptions) []Glyph {
	forEachLine(text, face, options, func(line string, indexOffset int, originX, originY float64) {
		glyphs = face.appendGlyphsForLine(glyphs, line, indexOffset, originX+x, originY+y)
	})
	return glyphs
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
	for t := text; ; {
		lineCount++
		line, rest, found := strings.Cut(t, "\n")
		a := face.advance(line)
		advances = append(advances, a)
		if longestAdvance < a {
			longestAdvance = a
		}
		if !found {
			break
		}
		t = rest
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
	for t := text; ; {
		line, rest, found := strings.Cut(t, "\n")

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

		f(line, indexOffset, originX+offsetX, originY+offsetY)

		if !found {
			break
		}
		t = rest
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
