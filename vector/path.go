// Copyright 2019 The Ebiten Authors
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

// Package vector provides functions for vector graphics rendering.
//
// This package is under experiments and the API might be changed with breaking backward compatibility.
package vector

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector/internal/triangulate"
)

var (
	emptyImage    = ebiten.NewImage(3, 3)
	emptySubImage = emptyImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() {
	emptyImage.Fill(color.White)
}

// Path represents a collection of path segments.
type Path struct {
	segs [][]triangulate.Point
	cur  triangulate.Point
}

// MoveTo skips the current position of the path to the given position (x, y) without adding any strokes.
func (p *Path) MoveTo(x, y float32) {
	p.cur = triangulate.Point{X: x, Y: y}
	p.segs = append(p.segs, []triangulate.Point{p.cur})
}

// LineTo adds a line segument to the path, which starts from the current position and ends to the given position (x, y).
//
// LineTo updates the current position to (x, y).
func (p *Path) LineTo(x, y float32) {
	if len(p.segs) == 0 {
		p.segs = append(p.segs, []triangulate.Point{p.cur})
	}
	p.segs[len(p.segs)-1] = append(p.segs[len(p.segs)-1], triangulate.Point{X: x, Y: y})
	p.cur = triangulate.Point{X: x, Y: y}
}

// nseg returns a number of segments based on the given two points (x0, y0) and (x1, y1).
func nseg(x0, y0, x1, y1 float32) int {
	distx := x1 - x0
	if distx < 0 {
		distx = -distx
	}
	disty := y1 - y0
	if disty < 0 {
		disty = -disty
	}
	dist := distx
	if dist < disty {
		dist = disty
	}

	return int(math.Ceil(float64(dist)))
}

// QuadTo adds a quadratic Bézier curve to the path.
func (p *Path) QuadTo(cpx, cpy, x, y float32) {
	c := p.cur
	num := nseg(c.X, c.Y, x, y)
	for t := float32(0.0); t <= 1; t += 1.0 / float32(num) {
		xf := (1-t)*(1-t)*c.X + 2*t*(1-t)*cpx + t*t*x
		yf := (1-t)*(1-t)*c.Y + 2*t*(1-t)*cpy + t*t*y
		p.LineTo(xf, yf)
	}
}

// CubicTo adds a cubic Bézier curve to the path.
func (p *Path) CubicTo(cp0x, cp0y, cp1x, cp1y, x, y float32) {
	c := p.cur
	num := nseg(c.X, c.Y, x, y)
	for t := float32(0.0); t <= 1; t += 1.0 / float32(num) {
		xf := (1-t)*(1-t)*(1-t)*c.X + 3*(1-t)*(1-t)*t*cp0x + 3*(1-t)*t*t*cp1x + t*t*t*x
		yf := (1-t)*(1-t)*(1-t)*c.Y + 3*(1-t)*(1-t)*t*cp0y + 3*(1-t)*t*t*cp1y + t*t*t*y
		p.LineTo(xf, yf)
	}
}

// FillOptions represents options to fill a path.
type FillOptions struct {
	// Color is a color to fill with.
	Color color.Color
}

// Fill fills the region of the path with the given options op.
func (p *Path) Fill(dst *ebiten.Image, op *FillOptions) {
	var vertices []ebiten.Vertex
	var indices []uint16

	if op.Color == nil {
		return
	}

	r, g, b, a := op.Color.RGBA()
	var rf, gf, bf, af float32
	if a == 0 {
		return
	}
	rf = float32(r) / float32(a)
	gf = float32(g) / float32(a)
	bf = float32(b) / float32(a)
	af = float32(a) / 0xffff

	var base uint16
	for _, seg := range p.segs {
		for _, pt := range seg {
			vertices = append(vertices, ebiten.Vertex{
				DstX:   pt.X,
				DstY:   pt.Y,
				SrcX:   0,
				SrcY:   0,
				ColorR: rf,
				ColorG: gf,
				ColorB: bf,
				ColorA: af,
			})
		}
		for _, idx := range triangulate.Triangulate(seg) {
			indices = append(indices, idx+base)
		}
		base += uint16(len(seg))
	}
	dst.DrawTriangles(vertices, indices, emptySubImage, nil)
}
