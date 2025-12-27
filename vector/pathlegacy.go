// Copyright 2025 The Ebitengine Authors
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
	"github.com/hajimehoshi/ebiten/v2"
)

// AppendVerticesAndIndicesForFilling appends vertices and indices to fill this path and returns them.
//
// AppendVerticesAndIndicesForFilling works in a similar way to the built-in append function.
// If the arguments are nils, AppendVerticesAndIndicesForFilling returns new slices.
//
// The returned vertice's SrcX and SrcY are 0, and ColorR, ColorG, ColorB, and ColorA are 1.
//
// The returned values are intended to be passed to DrawTriangles or DrawTrianglesShader with FileRuleNonZero or FillRuleEvenOdd
// in order to render a complex polygon like a concave polygon, a polygon with holes, or a self-intersecting polygon.
//
// The returned vertices and indices should be rendered with a solid (non-transparent) color with the default Blend (source-over).
// Otherwise, there is no guarantee about the rendering result.
//
// Deprecated: as of v2.9. Use [FillPath] instead.
func (p *Path) AppendVerticesAndIndicesForFilling(vertices []ebiten.Vertex, indices []uint16) ([]ebiten.Vertex, []uint16) {
	base := uint16(len(vertices))
	for _, flatPath := range p.ensureFlatPaths() {
		if flatPath.pointCount() < 3 {
			continue
		}
		for i, pt := range flatPath.points {
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
		base += uint16(flatPath.pointCount())
	}
	return vertices, indices
}

// AppendVerticesAndIndicesForStroke appends vertices and indices to render a stroke of this path and returns them.
// AppendVerticesAndIndicesForStroke works in a similar way to the built-in append function.
// If the arguments are nils, AppendVerticesAndIndicesForStroke returns new slices.
//
// The returned vertice's SrcX and SrcY are 0, and ColorR, ColorG, ColorB, and ColorA are 1.
//
// The returned values are intended to be passed to DrawTriangles or DrawTrianglesShader with a solid (non-transparent) color
// with FillRuleFillAll or FillRuleNonZero, not FileRuleEvenOdd.
//
// Deprecated: as of v2.9. Use [StrokePath] or [Path.AddStroke] instead.
func (p *Path) AppendVerticesAndIndicesForStroke(vertices []ebiten.Vertex, indices []uint16, op *StrokeOptions) ([]ebiten.Vertex, []uint16) {
	if op == nil {
		return vertices, indices
	}

	var stroke Path
	sOp := &AddStrokeOptions{}
	sOp.StrokeOptions = *op
	stroke.AddStroke(p, sOp)
	return stroke.AppendVerticesAndIndicesForFilling(vertices, indices)
}

// flatPath is a flattened sub-path of a path.
// A flatPath consists of points for line segments.
type flatPath struct {
	points []point
	closed bool
}

// reset resets the flatPath.
// reset doesn't release the allocated memory so that the memory can be reused.
func (f *flatPath) reset() {
	f.points = f.points[:0]
	f.closed = false
}

func (f flatPath) pointCount() int {
	return len(f.points)
}

func (f flatPath) lastPoint() point {
	return f.points[len(f.points)-1]
}

func (f *flatPath) appendPoint(pt point) {
	if f.closed {
		panic("vector: a closed flatPath cannot append a new point")
	}

	if len(f.points) > 0 {
		// Do not add a too close point to the last point.
		// This can cause unexpected rendering results.
		if lp := f.lastPoint(); abs(lp.x-pt.x) < 1e-2 && abs(lp.y-pt.y) < 1e-2 {
			return
		}
	}

	f.points = append(f.points, pt)
}

func (f *flatPath) close() {
	f.closed = true
}

func (p *Path) resetFlatPaths() {
	for _, fp := range p.flatPaths {
		fp.reset()
	}
	p.flatPaths = p.flatPaths[:0]
}

func (p *Path) ensureFlatPaths() []flatPath {
	if len(p.flatPaths) > 0 || len(p.subPaths) == 0 {
		return p.flatPaths
	}

	for _, subPath := range p.subPaths {
		p.appendNewFlatPath(subPath.start)
		cur := subPath.start
		for _, op := range subPath.ops {
			switch op.typ {
			case opTypeLineTo:
				p.appendFlatPathPointsForLine(op.p1)
				cur = op.p1
			case opTypeQuadTo:
				p.appendFlatPathPointsForQuad(cur, op.p1, op.p2, 0)
				cur = op.p2
			}
		}
		if subPath.closed {
			p.closeFlatPath()
		}
	}

	return p.flatPaths
}

func (p *Path) appendNewFlatPath(pt point) {
	if cap(p.flatPaths) > len(p.flatPaths) {
		// Reuse the last flat path since the last flat path might have an already allocated slice.
		p.flatPaths = p.flatPaths[:len(p.flatPaths)+1]
		p.flatPaths[len(p.flatPaths)-1].reset()
		p.flatPaths[len(p.flatPaths)-1].appendPoint(pt)
		return
	}
	p.flatPaths = append(p.flatPaths, flatPath{
		points: []point{pt},
	})
}

func (p *Path) appendFlatPathPointsForLine(pt point) {
	if len(p.flatPaths) == 0 || p.flatPaths[len(p.flatPaths)-1].closed {
		p.appendNewFlatPath(pt)
		return
	}
	p.flatPaths[len(p.flatPaths)-1].appendPoint(pt)
}

func (p *Path) appendFlatPathPointsForQuad(p0, p1, p2 point, level int) {
	if level > 10 {
		return
	}

	if isPointCloseToSegment(p1, p0, p2, 0.5) {
		p.appendFlatPathPointsForLine(p2)
		return
	}

	p01 := point{
		x: (p0.x + p1.x) / 2,
		y: (p0.y + p1.y) / 2,
	}
	p12 := point{
		x: (p1.x + p2.x) / 2,
		y: (p1.y + p2.y) / 2,
	}
	p012 := point{
		x: (p01.x + p12.x) / 2,
		y: (p01.y + p12.y) / 2,
	}
	p.appendFlatPathPointsForQuad(p0, p01, p012, level+1)
	p.appendFlatPathPointsForQuad(p012, p12, p2, level+1)
}

func (p *Path) closeFlatPath() {
	if len(p.flatPaths) == 0 {
		return
	}
	p.flatPaths[len(p.flatPaths)-1].close()
}
