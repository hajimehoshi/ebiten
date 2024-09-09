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
	"math"

	"github.com/go-text/typesetting/font/opentype"
	"golang.org/x/image/math/fixed"
	gvector "golang.org/x/image/vector"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func segmentsToBounds(segs []opentype.Segment) fixed.Rectangle26_6 {
	if len(segs) == 0 {
		return fixed.Rectangle26_6{}
	}

	minX := float32(math.Inf(1))
	minY := float32(math.Inf(1))
	maxX := float32(math.Inf(-1))
	maxY := float32(math.Inf(-1))

	for _, seg := range segs {
		n := 1
		switch seg.Op {
		case opentype.SegmentOpQuadTo:
			n = 2
		case opentype.SegmentOpCubeTo:
			n = 3
		}
		for i := 0; i < n; i++ {
			x := seg.Args[i].X
			y := seg.Args[i].Y
			if minX > x {
				minX = x
			}
			if minY > y {
				minY = y
			}
			if maxX < x {
				maxX = x
			}
			if maxY < y {
				maxY = y
			}
		}
	}

	return fixed.Rectangle26_6{
		Min: fixed.Point26_6{
			X: float32ToFixed26_6(minX),
			Y: float32ToFixed26_6(minY),
		},
		Max: fixed.Point26_6{
			X: float32ToFixed26_6(maxX),
			Y: float32ToFixed26_6(maxY),
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

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
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
