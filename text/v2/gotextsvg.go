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
	"bytes"

	"github.com/go-text/typesetting/font"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/oksvg"
)

// svgGlyphData is an OpenType SVG glyph description: the SVG document and its
// resolved viewBox.
type svgGlyphData struct {
	source  []byte
	viewBox font.SVGViewBox
}

// svgToImage rasterizes an OpenType SVG glyph document into a color image.
// sizeInPixels is the em size in pixels. subpixelOffset and glyphBounds
// follow the same conventions as segmentsToImage, and the returned image
// has the same dimensions as segmentsToImage would produce, so the two
// are interchangeable at the draw position derived from glyphBounds.
//
// svgToImage returns nil if the glyph produces no image or the SVG
// document cannot be parsed.
//
// The whole document is rasterized: a document shared by multiple
// glyphs is reduced to a single glyph's description beforehand (see
// [GoTextFaceSource.svgGlyphData]).
func svgToImage(svg *svgGlyphData, sizeInPixels float64, subpixelOffset fixed.Point26_6, glyphBounds fixed.Rectangle26_6) *ebiten.Image {
	if svg.viewBox.Width <= 0 || svg.viewBox.Height <= 0 {
		return nil
	}

	w, h := (glyphBounds.Max.X - glyphBounds.Min.X).Ceil(), (glyphBounds.Max.Y - glyphBounds.Min.Y).Ceil()
	if w == 0 || h == 0 {
		return nil
	}

	// Add always 1 to the size, following segmentsToImage.
	w++
	h++

	icon, err := oksvg.ReadIconStream(bytes.NewReader(svg.source), oksvg.IgnoreErrorMode)
	if err != nil {
		return nil
	}

	// The OpenType SVG coordinate system has its origin at the glyph origin with
	// the Y axis pointing down, and the viewBox maps onto the em square.
	// This matches the display space of the scaled outline segments, so the
	// glyph pixel position of an SVG user-space point is
	// (point - viewBoxMin) * (emSizeInPixels / viewBoxSize).
	//
	// The transform is set directly instead of via SvgIcon.SetTarget:
	// SetTarget applies the viewBox offset after scaling, which misplaces
	// documents with a non-zero viewBox origin.
	sx := sizeInPixels / float64(svg.viewBox.Width)
	sy := sizeInPixels / float64(svg.viewBox.Height)
	biasX := fixed26_6ToFloat64(-glyphBounds.Min.X + subpixelOffset.X)
	biasY := fixed26_6ToFloat64(-glyphBounds.Min.Y + subpixelOffset.Y)
	icon.Transform = rasterx.Matrix2D{
		A: sx,
		D: sy,
		E: biasX - float64(svg.viewBox.MinX)*sx,
		F: biasY - float64(svg.viewBox.MinY)*sy,
	}

	dst := newPooledRGBA(w, h)
	defer releasePooledRGBA(dst)
	scanner := rasterx.NewScannerGV(w, h, dst, dst.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	icon.Draw(dasher, 1)
	return ebiten.NewImageFromImage(dst)
}
