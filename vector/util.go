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
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	whiteImage    = ebiten.NewImage(3, 3)
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

var (
	cachedVertices []ebiten.Vertex
	cachedIndices  []uint16
	cacheM         sync.Mutex
)

func useCachedVerticesAndIndices(fn func([]ebiten.Vertex, []uint16) (vs []ebiten.Vertex, is []uint16)) {
	cacheM.Lock()
	defer cacheM.Unlock()
	cachedVertices, cachedIndices = fn(cachedVertices[:0], cachedIndices[:0])
}

func init() {
	b := whiteImage.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	for i := range pix {
		pix[i] = 0xff
	}
	// This is hacky, but WritePixels is better than Fill in term of automatic texture packing.
	whiteImage.WritePixels(pix)
}

func drawVerticesForUtil(dst *ebiten.Image, vs []ebiten.Vertex, is []uint16, clr color.Color, antialias bool) {
	r, g, b, a := clr.RGBA()
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = float32(r) / 0xffff
		vs[i].ColorG = float32(g) / 0xffff
		vs[i].ColorB = float32(b) / 0xffff
		vs[i].ColorA = float32(a) / 0xffff
	}

	op := &ebiten.DrawTrianglesOptions{}
	op.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
	op.AntiAlias = antialias
	dst.DrawTriangles(vs, is, whiteSubImage, op)
}

// StrokeLine strokes a line (x0, y0)-(x1, y1) with the specified width and color.
//
// clr has be to be a solid (non-transparent) color.
func StrokeLine(dst *ebiten.Image, x0, y0, x1, y1 float32, strokeWidth float32, clr color.Color, antialias bool) {
	var path Path
	path.MoveTo(x0, y0)
	path.LineTo(x1, y1)
	strokeOp := &StrokeOptions{}
	strokeOp.Width = strokeWidth

	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint16) ([]ebiten.Vertex, []uint16) {
		vs, is = path.AppendVerticesAndIndicesForStroke(vs, is, strokeOp)
		drawVerticesForUtil(dst, vs, is, clr, antialias)
		return vs, is
	})
}

// DrawFilledRect fills a rectangle with the specified width and color.
func DrawFilledRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color, antialias bool) {
	var path Path
	path.MoveTo(x, y)
	path.LineTo(x, y+height)
	path.LineTo(x+width, y+height)
	path.LineTo(x+width, y)

	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint16) ([]ebiten.Vertex, []uint16) {
		vs, is = path.AppendVerticesAndIndicesForFilling(vs, is)
		drawVerticesForUtil(dst, vs, is, clr, antialias)
		return vs, is
	})
}

// StrokeRect strokes a rectangle with the specified width and color.
//
// clr has be to be a solid (non-transparent) color.
func StrokeRect(dst *ebiten.Image, x, y, width, height float32, strokeWidth float32, clr color.Color, antialias bool) {
	var path Path
	path.MoveTo(x, y)
	path.LineTo(x, y+height)
	path.LineTo(x+width, y+height)
	path.LineTo(x+width, y)
	path.Close()

	strokeOp := &StrokeOptions{}
	strokeOp.Width = strokeWidth
	strokeOp.MiterLimit = 10

	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint16) ([]ebiten.Vertex, []uint16) {
		vs, is = path.AppendVerticesAndIndicesForStroke(vs, is, strokeOp)
		drawVerticesForUtil(dst, vs, is, clr, antialias)
		return vs, is
	})
}

// DrawFilledCircle fills a circle with the specified center position (cx, cy), the radius (r), width and color.
func DrawFilledCircle(dst *ebiten.Image, cx, cy, r float32, clr color.Color, antialias bool) {
	var path Path
	path.Arc(cx, cy, r, 0, 2*math.Pi, Clockwise)

	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint16) ([]ebiten.Vertex, []uint16) {
		vs, is = path.AppendVerticesAndIndicesForFilling(vs, is)
		drawVerticesForUtil(dst, vs, is, clr, antialias)
		return vs, is
	})
}

// StrokeCircle strokes a circle with the specified center position (cx, cy), the radius (r), width and color.
//
// clr has be to be a solid (non-transparent) color.
func StrokeCircle(dst *ebiten.Image, cx, cy, r float32, strokeWidth float32, clr color.Color, antialias bool) {
	var path Path
	path.Arc(cx, cy, r, 0, 2*math.Pi, Clockwise)
	path.Close()

	strokeOp := &StrokeOptions{}
	strokeOp.Width = strokeWidth

	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint16) ([]ebiten.Vertex, []uint16) {
		vs, is = path.AppendVerticesAndIndicesForStroke(vs, is, strokeOp)
		drawVerticesForUtil(dst, vs, is, clr, antialias)
		return vs, is
	})
}
