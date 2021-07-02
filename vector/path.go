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
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type point struct {
	x float32
	y float32
}

// Path represents a collection of path segments.
type Path struct {
	segs [][]point
	cur  point
}

// MoveTo skips the current position of the path to the given position (x, y) without adding any strokes.
func (p *Path) MoveTo(x, y float32) {
	p.cur = point{x: x, y: y}
	p.segs = append(p.segs, []point{p.cur})
}

// LineTo adds a line segument to the path, which starts from the current position and ends to the given position (x, y).
//
// LineTo updates the current position to (x, y).
func (p *Path) LineTo(x, y float32) {
	if len(p.segs) == 0 {
		p.segs = append(p.segs, []point{p.cur})
	}
	p.segs[len(p.segs)-1] = append(p.segs[len(p.segs)-1], point{x: x, y: y})
	p.cur = point{x: x, y: y}
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
	// TODO: Split more appropriate number of segments.
	c := p.cur
	num := nseg(c.x, c.y, x, y)
	for t := float32(0.0); t <= 1; t += 1.0 / float32(num) {
		xf := (1-t)*(1-t)*c.x + 2*t*(1-t)*cpx + t*t*x
		yf := (1-t)*(1-t)*c.y + 2*t*(1-t)*cpy + t*t*y
		p.LineTo(xf, yf)
	}
}

// CubicTo adds a cubic Bézier curve to the path.
func (p *Path) CubicTo(cp0x, cp0y, cp1x, cp1y, x, y float32) {
	// TODO: Split more appropriate number of segments.
	c := p.cur
	num := nseg(c.x, c.y, x, y)
	for t := float32(0.0); t <= 1; t += 1.0 / float32(num) {
		xf := (1-t)*(1-t)*(1-t)*c.x + 3*(1-t)*(1-t)*t*cp0x + 3*(1-t)*t*t*cp1x + t*t*t*x
		yf := (1-t)*(1-t)*(1-t)*c.y + 3*(1-t)*(1-t)*t*cp0y + 3*(1-t)*t*t*cp1y + t*t*t*y
		p.LineTo(xf, yf)
	}
}

// AppendVerticesAndIndices appends vertices and indices for this path and returns them.
// AppendVerticesAndIndices works in a similar way to the built-in append function.
// If the arguments are nils, AppendVerticesAndIndices returns new slices.
//
// The returned vertice's SrcX and SrcY are 0, and ColorR, ColorG, ColorB, and ColorA are 1.
//
// The returned values are intended to be passed to DrawTriangles or DrawTrianglesShader with EvenOdd option
// in order to render a complex polygon like a concave polygon, a polygon with holes, or a self-intersecting polygon.
func (p *Path) AppendVerticesAndIndices(vertices []ebiten.Vertex, indices []uint16) ([]ebiten.Vertex, []uint16) {
	// TODO: Add tests.

	var base uint16
	for _, seg := range p.segs {
		if len(seg) < 3 {
			continue
		}
		for i, pt := range seg {
			vertices = append(vertices, ebiten.Vertex{
				DstX:   pt.x,
				DstY:   pt.y,
				SrcX:   0,
				SrcY:   0,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			})
			if i < 2 {
				continue
			}
			indices = append(indices, base, base+uint16(i-1), base+uint16(i))
		}
		base += uint16(len(seg))
	}
	return vertices, indices
}
