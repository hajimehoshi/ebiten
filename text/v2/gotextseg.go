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
	"image/draw"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"golang.org/x/image/math/fixed"
	gvector "golang.org/x/image/vector"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// glyphExtentsToBounds returns the bounding rectangle of a glyph in the
// same display-space coordinates as its scaled outline segments. scale
// converts font units to pixels. When sideways is true, the same
// rotation that [font.GlyphOutline.Sideways] applies to segments is
// applied here, with yOffsetFontUnits as the post-rotation offset in
// font units.
func glyphExtentsToBounds(ext font.GlyphExtents, scale float32, sideways bool, yOffsetFontUnits float32) fixed.Rectangle26_6 {
	// Font-unit corners. X grows right; Y grows up. Height is negative
	// because YBearing is the top of the glyph and YBearing + Height is
	// the bottom.
	x0, x1 := ext.XBearing, ext.XBearing+ext.Width
	y0, y1 := ext.YBearing+ext.Height, ext.YBearing // y0 < y1
	if sideways {
		// (x, y) -> (y, -x + yOff). The new x range is the old y range;
		// the new y range is -old-x + yOff.
		x0, x1 = y0, y1
		y0, y1 = -ext.XBearing-ext.Width+yOffsetFontUnits, -ext.XBearing+yOffsetFontUnits
	}
	// Scale and flip Y to match the segment-coordinate transform. After
	// the flip the most-positive font Y becomes the most-negative display
	// Y (the top edge of the box).
	return fixed.Rectangle26_6{
		Min: fixed.Point26_6{
			X: float32ToFixed26_6(x0 * scale),
			Y: float32ToFixed26_6(-y1 * scale),
		},
		Max: fixed.Point26_6{
			X: float32ToFixed26_6(x1 * scale),
			Y: float32ToFixed26_6(-y0 * scale),
		},
	}
}

func segmentsToImage(segs []opentype.Segment, subpixelOffset fixed.Point26_6, glyphBounds fixed.Rectangle26_6) *ebiten.Image {
	if len(segs) == 0 {
		return nil
	}

	w, h := (glyphBounds.Max.X - glyphBounds.Min.X).Ceil(), (glyphBounds.Max.Y - glyphBounds.Min.Y).Ceil()
	if w == 0 || h == 0 {
		return nil
	}

	// Add always 1 to the size.
	// In theory, it is possible to determine whether +1 is necessary or not, but the calculation is pretty complicated.
	w++
	h++

	biasX := fixed26_6ToFloat32(-glyphBounds.Min.X + subpixelOffset.X)
	biasY := fixed26_6ToFloat32(-glyphBounds.Min.Y + subpixelOffset.Y)

	rast := gvector.NewRasterizer(w, h)
	rast.DrawOp = draw.Src
	for _, seg := range segs {
		switch seg.Op {
		case opentype.SegmentOpMoveTo:
			rast.MoveTo(seg.Args[0].X+biasX, seg.Args[0].Y+biasY)
		case opentype.SegmentOpLineTo:
			rast.LineTo(seg.Args[0].X+biasX, seg.Args[0].Y+biasY)
		case opentype.SegmentOpQuadTo:
			rast.QuadTo(
				seg.Args[0].X+biasX, seg.Args[0].Y+biasY,
				seg.Args[1].X+biasX, seg.Args[1].Y+biasY,
			)
		case opentype.SegmentOpCubeTo:
			rast.CubeTo(
				seg.Args[0].X+biasX, seg.Args[0].Y+biasY,
				seg.Args[1].X+biasX, seg.Args[1].Y+biasY,
				seg.Args[2].X+biasX, seg.Args[2].Y+biasY,
			)
		}
	}

	// Explicit closing is necessary especially for some OpenType fonts like
	// NotoSansJP-VF.otf in https://github.com/notofonts/noto-cjk/releases/tag/Sans2.004.
	// See also https://github.com/go-text/typesetting/issues/122.
	rast.ClosePath()

	dst := newPooledRGBA(w, h)
	defer releasePooledRGBA(dst)
	rast.Draw(dst, dst.Bounds(), image.Opaque, image.Point{})
	return ebiten.NewImageFromImage(dst)
}

func appendVectorPathFromSegments(path *vector.Path, segs []opentype.Segment, x, y float32) {
	for _, seg := range segs {
		switch seg.Op {
		case opentype.SegmentOpMoveTo:
			path.MoveTo(seg.Args[0].X+x, seg.Args[0].Y+y)
		case opentype.SegmentOpLineTo:
			path.LineTo(seg.Args[0].X+x, seg.Args[0].Y+y)
		case opentype.SegmentOpQuadTo:
			path.QuadTo(
				seg.Args[0].X+x, seg.Args[0].Y+y,
				seg.Args[1].X+x, seg.Args[1].Y+y,
			)
		case opentype.SegmentOpCubeTo:
			path.CubicTo(
				seg.Args[0].X+x, seg.Args[0].Y+y,
				seg.Args[1].X+x, seg.Args[1].Y+y,
				seg.Args[2].X+x, seg.Args[2].Y+y,
			)
		}
	}
	path.Close()
}
