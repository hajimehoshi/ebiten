// Copyright 2022 The Ebitengine Authors
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

package vector

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	whiteImage    = ebiten.NewImage(3, 3)
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() {
	whiteImage.Fill(color.White)
}

func updateVerticesForUtil(vs []ebiten.Vertex, clr color.Color) {
	r, g, b, a := clr.RGBA()
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = float32(r) / 0xffff
		vs[i].ColorG = float32(g) / 0xffff
		vs[i].ColorB = float32(b) / 0xffff
		vs[i].ColorA = float32(a) / 0xffff
	}
}

// StrokeLine strokes a line (x0, y0)-(x1, y1) with the specified width and color.
func StrokeLine(dst *ebiten.Image, x0, y0, x1, y1 float32, strokeWidth float32, clr color.Color) {
	var path Path
	path.MoveTo(x0, y0)
	path.LineTo(x1, y1)
	strokeOp := &StrokeOptions{}
	strokeOp.Width = strokeWidth
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, strokeOp)

	updateVerticesForUtil(vs, clr)

	op := &ebiten.DrawTrianglesOptions{}
	op.ColorScaleFormat = ebiten.ColorScaleFormatPremultipliedAlpha
	op.AntiAlias = true
	dst.DrawTriangles(vs, is, whiteSubImage, op)
}

// FillRect fills a rectangle with the specified width and color.
func FillRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color) {
	var path Path
	path.MoveTo(x, y)
	path.LineTo(x, y+height)
	path.LineTo(x+width, y+height)
	path.LineTo(x+width, y)
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)

	updateVerticesForUtil(vs, clr)

	op := &ebiten.DrawTrianglesOptions{}
	op.ColorScaleFormat = ebiten.ColorScaleFormatPremultipliedAlpha
	op.AntiAlias = true
	dst.DrawTriangles(vs, is, whiteSubImage, op)
}

// StrokeRect strokes a rectangle with the specified width and color.
//
// clr has be to be a solid (non-transparent) color.
func StrokeRect(dst *ebiten.Image, x, y, width, height float32, strokeWidth float32, clr color.Color) {
	var path Path
	path.MoveTo(x, y)
	path.LineTo(x, y+height)
	path.LineTo(x+width, y+height)
	path.LineTo(x+width, y)
	path.Close()

	strokeOp := &StrokeOptions{}
	strokeOp.Width = strokeWidth
	strokeOp.MiterLimit = 10
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, strokeOp)

	updateVerticesForUtil(vs, clr)

	op := &ebiten.DrawTrianglesOptions{}
	op.ColorScaleFormat = ebiten.ColorScaleFormatPremultipliedAlpha
	op.AntiAlias = true
	dst.DrawTriangles(vs, is, whiteSubImage, op)
}
