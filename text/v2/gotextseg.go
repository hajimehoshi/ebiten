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

	"github.com/go-text/typesetting/opentype/api"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"github.com/hajimehoshi/ebiten/v2"
)

func segmentsToBounds(segs []api.Segment) fixed.Rectangle26_6 {
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
		case api.SegmentOpQuadTo:
			n = 2
		case api.SegmentOpCubeTo:
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

func segmentsToImage(segs []api.Segment, subpixelOffset fixed.Point26_6, glyphBounds fixed.Rectangle26_6) *ebiten.Image {
	if len(segs) == 0 {
		return nil
	}

	w, h := (glyphBounds.Max.X - glyphBounds.Min.X).Ceil(), (glyphBounds.Max.Y - glyphBounds.Min.Y).Ceil()
	if w == 0 || h == 0 {
		return nil
	}

	if glyphBounds.Min.X&((1<<6)-1) != 0 {
		w++
	}
	if glyphBounds.Min.Y&((1<<6)-1) != 0 {
		h++
	}

	biasX := fixed26_6ToFloat32(-glyphBounds.Min.X + subpixelOffset.X)
	biasY := fixed26_6ToFloat32(-glyphBounds.Min.Y + subpixelOffset.Y)

	rast := vector.NewRasterizer(w, h)
	rast.DrawOp = draw.Src
	for _, seg := range segs {
		switch seg.Op {
		case api.SegmentOpMoveTo:
			rast.MoveTo(seg.Args[0].X+biasX, seg.Args[0].Y+biasY)
		case api.SegmentOpLineTo:
			rast.LineTo(seg.Args[0].X+biasX, seg.Args[0].Y+biasY)
		case api.SegmentOpQuadTo:
			rast.QuadTo(
				seg.Args[0].X+biasX, seg.Args[0].Y+biasY,
				seg.Args[1].X+biasX, seg.Args[1].Y+biasY,
			)
		case api.SegmentOpCubeTo:
			rast.CubeTo(
				seg.Args[0].X+biasX, seg.Args[0].Y+biasY,
				seg.Args[1].X+biasX, seg.Args[1].Y+biasY,
				seg.Args[2].X+biasX, seg.Args[2].Y+biasY,
			)
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	rast.Draw(dst, dst.Bounds(), image.Opaque, image.Point{})
	return ebiten.NewImageFromImage(dst)
}
