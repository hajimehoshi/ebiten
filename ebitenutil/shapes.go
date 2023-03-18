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
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	whiteImage    = ebiten.NewImage(3, 3)
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() {
	pix := make([]byte, 4*3*3)
	for i := range pix {
		pix[i] = 0xff
	}
	whiteImage.WritePixels(pix)
}

// DrawLine draws a line segment on the given destination dst.
//
// DrawLine is intended to be used mainly for debugging or prototyping purpose.
//
// Use vector.StrokeLine instead if you require rendering with higher quality.
func DrawLine(dst *ebiten.Image, x1, y1, x2, y2 float64, clr color.Color) {
	// Use ebiten.Image's DrawImage instead of vector.StrokeLine for backward compatibility (#2605)

	length := math.Hypot(x2-x1, y2-y1)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(length, 1)
	op.GeoM.Rotate(math.Atan2(y2-y1, x2-x1))
	op.GeoM.Translate(x1, y1)
	op.ColorScale.ScaleWithColor(clr)
	// Filter must be 'nearest' filter (default).
	// Linear filtering would make edges blurred.
	dst.DrawImage(whiteSubImage, op)
}

// DrawRect draws a rectangle on the given destination dst.
//
// DrawRect is intended to be used mainly for debugging or prototyping purpose.
//
// Use vector.DrawFilledRect instead if you require rendering with higher quality.
func DrawRect(dst *ebiten.Image, x, y, width, height float64, clr color.Color) {
	// Use ebiten.Image's DrawImage instead of vector.DrawFilledRect for backward compatibility (#2605)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(width, height)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	// Filter must be 'nearest' filter (default).
	// Linear filtering would make edges blurred.
	dst.DrawImage(whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image), op)
}

// DrawCircle draws a circle on given destination dst.
//
// DrawCircle is intended to be used mainly for debugging or prototyping puropose.
//
// Use vector.DrawFilledCircle instead if you require rendering with higher quality.
func DrawCircle(dst *ebiten.Image, cx, cy, r float64, clr color.Color) {
	// Use ebiten.Image's DrawTriangles instead of vector.DrawFilledCircle for backward compatibility (#2605)

	var path vector.Path
	rd, g, b, a := clr.RGBA()

	path.Arc(float32(cx), float32(cy), float32(r), 0, 2*math.Pi, vector.Clockwise)

	vertices, indices := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vertices {
		vertices[i].SrcX = 1
		vertices[i].SrcY = 1
		vertices[i].ColorR = float32(rd) / 0xffff
		vertices[i].ColorG = float32(g) / 0xffff
		vertices[i].ColorB = float32(b) / 0xffff
		vertices[i].ColorA = float32(a) / 0xffff
	}
	dst.DrawTriangles(vertices, indices, whiteSubImage, nil)
}
