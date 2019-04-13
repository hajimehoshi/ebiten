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
	"math"

	"github.com/hajimehoshi/ebiten"
)

var emptyImage *ebiten.Image

func init() {
	const w, h = 16, 16
	emptyImage, _ = ebiten.NewImage(w, h, ebiten.FilterDefault)
	pix := make([]byte, 4*w*h)
	for i := range pix {
		pix[i] = 0xff
	}
	emptyImage.ReplacePixels(pix)
}

type point struct {
	x float32
	y float32
}

type segment struct {
	p0 point
	p1 point
}

func (s *segment) atan2() float64 {
	return math.Atan2(float64(s.p1.y-s.p0.y), float64(s.p1.x-s.p0.x))
}

// Path represents a collection of paths.
type Path struct {
	segs []segment
	cur  point
}

// MoveTo skips the current position of the path to the given position (x, y) without adding any strokes.
func (p *Path) MoveTo(x, y float32) {
	p.cur = point{x, y}
}

// LineTo adds a segment to the path, which starts from the current position and ends to the given position (x, y).
//
// LineTo updates the current position to (x, y).
func (p *Path) LineTo(x, y float32) {
	p.segs = append(p.segs, segment{p.cur, point{x, y}})
	p.cur = point{x, y}
}

func (p *Path) strokeVertices(lineWidth float32, clr color.Color) (vertices []ebiten.Vertex, indices []uint16) {
	if len(p.segs) == 0 {
		return nil, nil
	}

	sw, sh := emptyImage.Size()
	r, g, b, a := clr.RGBA()
	rf, gf, bf, af := float32(r)/0xffff, float32(g)/0xffff, float32(b)/0xffff, float32(a)/0xffff
	for i, s := range p.segs {
		si, co := math.Sincos(s.atan2() + math.Pi/2)
		dx, dy := float32(co)*lineWidth/2, float32(si)*lineWidth/2
		v0 := point{s.p0.x + dx, s.p0.y + dy}
		v1 := point{s.p0.x - dx, s.p0.y - dy}
		v2 := point{s.p1.x + dx, s.p1.y + dy}
		v3 := point{s.p1.x - dx, s.p1.y - dy}

		vertices = append(vertices,
			ebiten.Vertex{
				DstX:   v0.x,
				DstY:   v0.y,
				SrcX:   0,
				SrcY:   0,
				ColorR: rf,
				ColorG: gf,
				ColorB: bf,
				ColorA: af,
			},
			ebiten.Vertex{
				DstX:   v1.x,
				DstY:   v1.y,
				SrcX:   float32(sw),
				SrcY:   0,
				ColorR: rf,
				ColorG: gf,
				ColorB: bf,
				ColorA: af,
			},
			ebiten.Vertex{
				DstX:   v2.x,
				DstY:   v2.y,
				SrcX:   0,
				SrcY:   float32(sh),
				ColorR: rf,
				ColorG: gf,
				ColorB: bf,
				ColorA: af,
			},
			ebiten.Vertex{
				DstX:   v3.x,
				DstY:   v3.y,
				SrcX:   float32(sw),
				SrcY:   float32(sh),
				ColorR: rf,
				ColorG: gf,
				ColorB: bf,
				ColorA: af,
			},
		)
		idx := uint16(4 * i)
		indices = append(indices, idx, idx+1, idx+2, idx+1, idx+2, idx+3)
	}

	return
}

// DrawPathOptions is the options specified at (*Path).Draw.
type DrawPathOptions struct {
	LineWidth   float32
	StrokeColor color.Color
}

// Draw draws the path by rendering its stroke or filling.
func (p *Path) Draw(target *ebiten.Image, op *DrawPathOptions) {
	if op == nil {
		return
	}

	// TODO: Implement filling
	if op.StrokeColor != nil {
		vs, is := p.strokeVertices(op.LineWidth, op.StrokeColor)
		target.DrawTriangles(vs, is, emptyImage, nil)
	}
}
