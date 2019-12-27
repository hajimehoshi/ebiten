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
	"image/color"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/vector/internal/math"
)

var emptyImage *ebiten.Image

func init() {
	emptyImage, _ = ebiten.NewImage(1, 1, ebiten.FilterDefault)
	emptyImage.Fill(color.White)
}

// Path represents a collection of path segments.
type Path struct {
	segs [][]math.Point
	cur  math.Point
}

// MoveTo skips the current position of the path to the given position (x, y) without adding any strokes.
func (p *Path) MoveTo(x, y float32) {
	p.cur = math.Point{X: x, Y: y}
	p.segs = append(p.segs, []math.Point{p.cur})
}

// LineTo adds a line segument to the path, which starts from the current position and ends to the given position (x, y).
//
// LineTo updates the current position to (x, y).
func (p *Path) LineTo(x, y float32) {
	if len(p.segs) == 0 {
		p.segs = append(p.segs, []math.Point{{X: 0, Y: 0}})
	}
	p.segs[len(p.segs)-1] = append(p.segs[len(p.segs)-1], math.Point{X: x, Y: y})
	p.cur = math.Point{X: x, Y: y}
}

func (p *Path) Fill(dst *ebiten.Image, clr color.Color) {
	var vertices []ebiten.Vertex
	var indices []uint16

	r, g, b, a := clr.RGBA()
	var rf, gf, bf, af float32
	if a > 0 {
		rf = float32(r) / float32(a)
		gf = float32(g) / float32(a)
		bf = float32(b) / float32(a)
		af = float32(a) / 0xffff
	}

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
		for _, idx := range math.Triangulate(seg) {
			indices = append(indices, idx+base)
		}
		base += uint16(len(seg))
	}
	dst.DrawTriangles(vertices, indices, emptyImage, nil)
}
