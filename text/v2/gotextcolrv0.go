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
	"image/draw"

	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
)

// colrV0Layer is one layer of a COLRv0 color glyph: an outline in the same
// scaled coordinates as glyphRenderData's realized segments, filled with
// a solid color.
type colrV0Layer struct {
	segments []opentype.Segment
	color    color.Color
}

// appendCOLRV0Layers converts COLRv0 layer records into colrV0Layers with
// scaled outline segments, appending to dst. Layers whose outline is
// missing or whose palette entry is invalid are skipped.
//
// The caller must hold g.shapeMu, and the shared font state must
// reflect the target variations.
func (g *GoTextFaceSource) appendCOLRV0Layers(dst []colrV0Layer, layers tables.PaintColrLayersResolved, scale float32) []colrV0Layer {
	palette := g.f.CPAL
	for _, layer := range layers {
		o, ok := g.f.GlyphDataOutline(opentype.GID(layer.GlyphID))
		if !ok || len(o.Segments) == 0 {
			continue
		}

		// A palette index of 0xFFFF means the text foreground color.
		// White is used so that the usual color scaling applies.
		var clr color.Color = color.White
		if layer.PaletteIndex != 0xffff {
			if len(palette) == 0 || int(layer.PaletteIndex) >= len(palette[0]) {
				continue
			}
			c := palette[0][layer.PaletteIndex]
			clr = color.NRGBA{R: c.Red, G: c.Green, B: c.Blue, A: c.Alpha}
		}

		segs := make([]opentype.Segment, len(o.Segments))
		for i, seg := range o.Segments {
			segs[i] = seg
			for j := range seg.Args {
				segs[i].Args[j].X *= scale
				segs[i].Args[j].Y *= -scale
			}
		}
		dst = append(dst, colrV0Layer{
			segments: segs,
			color:    clr,
		})
	}
	return dst
}

// colrV0LayersToImage rasterizes COLRv0 layers into a color image.
// subpixelOffset and glyphBounds follow the same conventions as
// segmentsToImage, and the returned image has the same dimensions as
// segmentsToImage would produce.
func colrV0LayersToImage(layers []colrV0Layer, subpixelOffset fixed.Point26_6, glyphBounds fixed.Rectangle26_6) *ebiten.Image {
	if len(layers) == 0 {
		return nil
	}

	w, h := (glyphBounds.Max.X - glyphBounds.Min.X).Ceil(), (glyphBounds.Max.Y - glyphBounds.Min.Y).Ceil()
	if w == 0 || h == 0 {
		return nil
	}

	// Add always 1 to the size, following segmentsToImage.
	w++
	h++

	dst := newPooledRGBA(w, h)
	defer releasePooledRGBA(dst)
	src := &image.Uniform{}
	for _, l := range layers {
		rast := newGlyphRasterizer(w, h, l.segments, subpixelOffset, glyphBounds)
		rast.DrawOp = draw.Over
		src.C = l.color
		rast.Draw(dst, dst.Bounds(), src, image.Point{})
	}
	return ebiten.NewImageFromImage(dst)
}
