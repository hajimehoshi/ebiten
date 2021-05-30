// Copyright 2017 The Ebiten Authors
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

package ebitenutil

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/colormcache"
)

var (
	emptyImage    = ebiten.NewImage(3, 3)
	emptySubImage = emptyImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() {
	emptyImage.Fill(color.White)
}

// DrawLine draws a line segment on the given destination dst.
//
// DrawLine is intended to be used mainly for debugging or prototyping purpose.
//
// DrawLine is not concurrent-safe.
func DrawLine(dst *ebiten.Image, x1, y1, x2, y2 float64, clr color.Color) {
	DrawLineWidth(dst, x1, y1, x2, y2, 1, clr)
}

// DrawLineWidth draws a line segment on the given destination dst with a variable width
// in pixels.
func DrawLineWidth(dst *ebiten.Image, x1, y1, x2, y2, width float64, clr color.Color) {
	x1f, y1f := float32(x1), float32(y1)
	x2f, y2f := float32(x2), float32(y2)
	var xOffset, yOffset float32
	if x1f > x2f {
		xOffset = -float32(width) / 2
	} else {
		xOffset = float32(width) / 2
	}
	if y1f > y2f {
		yOffset = float32(width) / 2
	} else {
		yOffset = -float32(width) / 2
	}

	r, g, b, a := clr.RGBA()
	r32 := float32(r) / 0xffff
	g32 := float32(g) / 0xffff
	b32 := float32(b) / 0xffff
	a32 := float32(a) / 0xffff

	dst.DrawTriangles([]ebiten.Vertex{
		{
			DstX: x1f - xOffset, DstY: y1f - yOffset,
			SrcX: 0, SrcY: 0,
			ColorR: r32, ColorG: g32, ColorB: b32, ColorA: a32,
		},
		{
			DstX: x1f + xOffset, DstY: y1f + yOffset,
			SrcX: 0, SrcY: 1,
			ColorR: r32, ColorG: g32, ColorB: b32, ColorA: a32,
		},
		{
			DstX: x2f - xOffset, DstY: y2f - yOffset,
			SrcX: 1, SrcY: 1,
			ColorR: r32, ColorG: g32, ColorB: b32, ColorA: a32,
		},
		{
			DstX: x2f - xOffset, DstY: y2f - yOffset,
			SrcX: 0, SrcY: 0,
			ColorR: r32, ColorG: g32, ColorB: b32, ColorA: a32,
		},
		{
			DstX: x2f + xOffset, DstY: y2f + yOffset,
			SrcX: 0, SrcY: 1,
			ColorR: r32, ColorG: g32, ColorB: b32, ColorA: a32,
		},
		{
			DstX: x1f + xOffset, DstY: y1f + yOffset,
			SrcX: 1, SrcY: 1,
			ColorR: r32, ColorG: g32, ColorB: b32, ColorA: a32,
		},
	},
		[]uint16{0, 1, 2, 3, 4, 5}, emptySubImage, nil,
	)
}

// DrawRect draws a rectangle on the given destination dst.
//
// DrawRect is intended to be used mainly for debugging or prototyping purpose.
//
// DrawRect is not concurrent-safe.
func DrawRect(dst *ebiten.Image, x, y, width, height float64, clr color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(width, height)
	op.GeoM.Translate(x, y)
	op.ColorM = colormcache.ColorToColorM(clr)
	// Filter must be 'nearest' filter (default).
	// Linear filtering would make edges blurred.
	dst.DrawImage(emptyImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image), op)
}
