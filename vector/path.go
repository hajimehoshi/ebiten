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
	const w, h = 16, 16
	emptyImage, _ = ebiten.NewImage(w, h, ebiten.FilterDefault)
	pix := make([]byte, 4*w*h)
	for i := range pix {
		pix[i] = 0xff
	}
	emptyImage.ReplacePixels(pix)
}

// Path represents a collection of paths.
type Path struct {
	segs [][]math.Segment
	cur  math.Point
}

// MoveTo skips the current position of the path to the given position (x, y) without adding any strokes.
func (p *Path) MoveTo(x, y float32) {
	p.cur = math.Point{x, y}
	if len(p.segs) > 0 && len(p.segs[len(p.segs)-1]) == 0 {
		return
	}
	p.segs = append(p.segs, []math.Segment{})
}

// LineTo adds a math.Segment to the path, which starts from the current position and ends to the given position (x, y).
//
// LineTo updates the current position to (x, y).
func (p *Path) LineTo(x, y float32) {
	if len(p.segs) == 0 {
		p.segs = append(p.segs, []math.Segment{})
	}
	p.segs[len(p.segs)-1] = append(p.segs[len(p.segs)-1], math.Segment{p.cur, math.Point{x, y}})
	p.cur = math.Point{x, y}
}

func (p *Path) strokeVertices(lineWidth float32, clr color.Color) (vertices []ebiten.Vertex, indices []uint16) {
	if len(p.segs) == 0 {
		return nil, nil
	}

	sw, sh := emptyImage.Size()
	r, g, b, a := clr.RGBA()
	rf, gf, bf, af := float32(r)/0xffff, float32(g)/0xffff, float32(b)/0xffff, float32(a)/0xffff
	idx := uint16(0)
	for _, ss := range p.segs {
		for i, s := range ss {
			s0 := s.Translate(-lineWidth / 2)
			s1 := s.Translate(lineWidth / 2)

			v0 := s0.P0
			v1 := s1.P0
			v2 := s0.P1
			v3 := s1.P1
			if i != 0 {
				ps := ss[i-1]
				v0 = ps.Translate(-lineWidth / 2).IntersectionAsLines(s0)
				v1 = ps.Translate(lineWidth / 2).IntersectionAsLines(s1)
			}
			if i != len(ss)-1 {
				ns := ss[i+1]
				v2 = ns.Translate(-lineWidth / 2).IntersectionAsLines(s0)
				v3 = ns.Translate(lineWidth / 2).IntersectionAsLines(s1)
			}

			vertices = append(vertices,
				ebiten.Vertex{
					DstX:   v0.X,
					DstY:   v0.Y,
					SrcX:   0,
					SrcY:   0,
					ColorR: rf,
					ColorG: gf,
					ColorB: bf,
					ColorA: af,
				},
				ebiten.Vertex{
					DstX:   v1.X,
					DstY:   v1.Y,
					SrcX:   float32(sw),
					SrcY:   0,
					ColorR: rf,
					ColorG: gf,
					ColorB: bf,
					ColorA: af,
				},
				ebiten.Vertex{
					DstX:   v2.X,
					DstY:   v2.Y,
					SrcX:   0,
					SrcY:   float32(sh),
					ColorR: rf,
					ColorG: gf,
					ColorB: bf,
					ColorA: af,
				},
				ebiten.Vertex{
					DstX:   v3.X,
					DstY:   v3.Y,
					SrcX:   float32(sw),
					SrcY:   float32(sh),
					ColorR: rf,
					ColorG: gf,
					ColorB: bf,
					ColorA: af,
				},
			)
			indices = append(indices, idx, idx+1, idx+2, idx+1, idx+2, idx+3)
			idx += 4
		}
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
