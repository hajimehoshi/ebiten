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

// Direction represents clockwise or counterclockwise.
type Direction int

const (
	Clockwise Direction = iota
	CounterClockwise
)

type opType int

const (
	opTypeMoveTo opType = iota
	opTypeLineTo
	opTypeQuadTo
	opTypeCubicTo
	opTypeClose
)

type op struct {
	typ opType
	p1  point
	p2  point
	p3  point
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

type point struct {
	x float32
	y float32
}

type subpath struct {
	points []point
	closed bool
}

// reset resets the subpath.
// reset doesn't release the allocated memory so that the memory can be reused.
func (s *subpath) reset() {
	s.points = s.points[:0]
	s.closed = false
}

func (s subpath) pointCount() int {
	return len(s.points)
}

func (s subpath) lastPoint() point {
	return s.points[len(s.points)-1]
}

func (s *subpath) appendPoint(pt point) {
	if s.closed {
		panic("vector: a closed subpathment cannot append a new point")
	}

	if len(s.points) > 0 {
		// Do not add a too close point to the last point.
		// This can cause unexpected rendering results.
		if lp := s.lastPoint(); abs(lp.x-pt.x) < 1e-2 && abs(lp.y-pt.y) < 1e-2 {
			return
		}
	}

	s.points = append(s.points, pt)
}

func (s *subpath) close() {
	if s.closed {
		return
	}

	s.appendPoint(s.points[0])
	s.closed = true
}

// Path represents a collection of path subpathments.
type Path struct {
	ops []op

	subpaths []subpath
}

// reset resets the path.
// reset doesn't release the allocated memory so that the memory can be reused.
func (p *Path) reset() {
	p.ops = p.ops[:0]
	p.subpaths = p.subpaths[:0]
}

func (p *Path) appendNewSubpath(pt point) {
	if cap(p.subpaths) > len(p.subpaths) {
		// Reuse the last subpath since the last subpath might have an already allocated slice.
		p.subpaths = p.subpaths[:len(p.subpaths)+1]
		p.subpaths[len(p.subpaths)-1].reset()
		p.subpaths[len(p.subpaths)-1].appendPoint(pt)
		return
	}
	p.subpaths = append(p.subpaths, subpath{
		points: []point{pt},
	})
}

func (p *Path) ensureSubpaths() []subpath {
	if len(p.subpaths) > 0 || len(p.ops) == 0 {
		return p.subpaths
	}

	var cur point
	for _, op := range p.ops {
		switch op.typ {
		case opTypeMoveTo:
			p.appendNewSubpath(op.p1)
			cur = op.p1
		case opTypeLineTo:
			p.lineTo(op.p1)
			cur = op.p1
		case opTypeQuadTo:
			p.quadTo(cur, op.p1, op.p2, 0)
			cur = op.p2
		case opTypeCubicTo:
			p.cubicTo(cur, op.p1, op.p2, op.p3, 0)
			cur = op.p3
		case opTypeClose:
			p.close()
			cur = point{}
		}
	}

	return p.subpaths
}

// MoveTo starts a new subpath with the given position (x, y) without adding a subpath,
func (p *Path) MoveTo(x, y float32) {
	p.subpaths = p.subpaths[:0]
	p.ops = append(p.ops, op{
		typ: opTypeMoveTo,
		p1:  point{x: x, y: y},
	})
}

// LineTo adds a line segment to the path, which starts from the last position of the current subpath
// and ends to the given position (x, y).
// If p doesn't have any subpaths or the last subpath is closed, LineTo sets (x, y) as the start position of a new subpath.
func (p *Path) LineTo(x, y float32) {
	p.subpaths = p.subpaths[:0]
	p.ops = append(p.ops, op{
		typ: opTypeLineTo,
		p1:  point{x: x, y: y},
	})
}

// QuadTo adds a quadratic Bézier curve to the path.
// (x1, y1) is the control point, and (x2, y2) is the destination.
func (p *Path) QuadTo(x1, y1, x2, y2 float32) {
	p.subpaths = p.subpaths[:0]
	p.ops = append(p.ops, op{
		typ: opTypeQuadTo,
		p1:  point{x: x1, y: y1},
		p2:  point{x: x2, y: y2},
	})
}

// CubicTo adds a cubic Bézier curve to the path.
// (x1, y1) and (x2, y2) are the control points, and (x3, y3) is the destination.
func (p *Path) CubicTo(x1, y1, x2, y2, x3, y3 float32) {
	p.subpaths = p.subpaths[:0]
	p.ops = append(p.ops, op{
		typ: opTypeCubicTo,
		p1:  point{x: x1, y: y1},
		p2:  point{x: x2, y: y2},
		p3:  point{x: x3, y: y3},
	})
}

// Close adds a new line from the last position of the current subpath to the first position of the current subpath,
// and marks the current subpath closed.
// Following operations for this path will start with a new subpath.
func (p *Path) Close() {
	p.subpaths = p.subpaths[:0]
	p.ops = append(p.ops, op{
		typ: opTypeClose,
	})
}

func (p *Path) lineTo(pt point) {
	if len(p.subpaths) == 0 || p.subpaths[len(p.subpaths)-1].closed {
		p.appendNewSubpath(pt)
		return
	}
	p.subpaths[len(p.subpaths)-1].appendPoint(pt)
}

// lineForTwoPoints returns parameters for a line passing through p0 and p1.
func lineForTwoPoints(p0, p1 point) (a, b, c float32) {
	// Line passing through p0 and p1 in the form of ax + by + c = 0
	a = p1.y - p0.y
	b = -(p1.x - p0.x)
	c = (p1.x-p0.x)*p0.y - (p1.y-p0.y)*p0.x
	return
}

// isPointCloseToSegment detects the distance between a segment (x0, y0)-(x1, y1) and a point (x, y) is less than allow.
// If p0 and p1 are the same, isPointCloseToSegment returns true when the distance between p0 and p is less than allow.
func isPointCloseToSegment(p, p0, p1 point, allow float32) bool {
	if p0 == p1 {
		return allow*allow >= (p0.x-p.x)*(p0.x-p.x)+(p0.y-p.y)*(p0.y-p.y)
	}

	a, b, c := lineForTwoPoints(p0, p1)

	// The distance between a line ax+by+c=0 and (x0, y0) is
	//     |ax0 + by0 + c| / √(a² + b²)
	return allow*allow*(a*a+b*b) >= (a*p.x+b*p.y+c)*(a*p.x+b*p.y+c)
}

// crossingPointForTwoLines returns a crossing point for two lines.
func crossingPointForTwoLines(p00, p01, p10, p11 point) point {
	a0, b0, c0 := lineForTwoPoints(p00, p01)
	a1, b1, c1 := lineForTwoPoints(p10, p11)
	det := a0*b1 - a1*b0
	return point{
		x: (b0*c1 - b1*c0) / det,
		y: (a1*c0 - a0*c1) / det,
	}
}

func (p *Path) quadTo(p0, p1, p2 point, level int) {
	if level > 10 {
		return
	}

	if isPointCloseToSegment(p1, p0, p2, 0.5) {
		p.lineTo(p2)
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
	p.quadTo(p0, p01, p012, level+1)
	p.quadTo(p012, p12, p2, level+1)
}

func (p *Path) cubicTo(p0, p1, p2, p3 point, level int) {
	if level > 10 {
		return
	}

	if isPointCloseToSegment(p1, p0, p3, 0.5) && isPointCloseToSegment(p2, p0, p3, 0.5) {
		p.lineTo(p3)
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
	p23 := point{
		x: (p2.x + p3.x) / 2,
		y: (p2.y + p3.y) / 2,
	}
	p012 := point{
		x: (p01.x + p12.x) / 2,
		y: (p01.y + p12.y) / 2,
	}
	p123 := point{
		x: (p12.x + p23.x) / 2,
		y: (p12.y + p23.y) / 2,
	}
	p0123 := point{
		x: (p012.x + p123.x) / 2,
		y: (p012.y + p123.y) / 2,
	}
	p.cubicTo(p0, p01, p012, p0123, level+1)
	p.cubicTo(p0123, p123, p23, p3, level+1)
}

func normalize(p point) point {
	len := float32(math.Hypot(float64(p.x), float64(p.y)))
	return point{x: p.x / len, y: p.y / len}
}

func cross(p0, p1 point) float32 {
	return p0.x*p1.y - p1.x*p0.y
}

func (p *Path) currentPosition() (point, bool) {
	if len(p.ops) == 0 {
		return point{}, false
	}
	op := p.ops[len(p.ops)-1]
	switch op.typ {
	case opTypeMoveTo:
		return op.p1, true
	case opTypeLineTo:
		return op.p1, true
	case opTypeQuadTo:
		return op.p2, true
	case opTypeCubicTo:
		return op.p3, true
	case opTypeClose:
		return point{}, false
	}
	return point{}, false
}

// ArcTo adds an arc curve to the path.
// (x1, y1) is the first control point, and (x2, y2) is the second control point.
func (p *Path) ArcTo(x1, y1, x2, y2, radius float32) {
	p0, ok := p.currentPosition()
	if !ok {
		p0 = point{x: x1, y: y1}
	}
	d0 := point{
		x: p0.x - x1,
		y: p0.y - y1,
	}
	d1 := point{
		x: x2 - x1,
		y: y2 - y1,
	}
	if d0 == (point{}) || d1 == (point{}) {
		p.LineTo(x1, y1)
		return
	}

	d0 = normalize(d0)
	d1 = normalize(d1)

	// theta is the angle between two vectors d0 and d1.
	theta := math.Acos(float64(d0.x*d1.x + d0.y*d1.y))
	// TODO: When theta is bigger than π/2, the arc should be split into two.

	// dist is the distance between the control point and the arc's beginning and ending points.
	dist := radius / float32(math.Tan(theta/2))

	// TODO: What if dist is too big?

	// (ax0, ay0) is the start of the arc.
	ax0 := x1 + d0.x*dist
	ay0 := y1 + d0.y*dist

	var cx, cy, a0, a1 float32
	var dir Direction
	if cross(d0, d1) >= 0 {
		cx = ax0 - d0.y*radius
		cy = ay0 + d0.x*radius
		a0 = float32(math.Atan2(float64(-d0.x), float64(d0.y)))
		a1 = float32(math.Atan2(float64(d1.x), float64(-d1.y)))
		dir = CounterClockwise
	} else {
		cx = ax0 + d0.y*radius
		cy = ay0 - d0.x*radius
		a0 = float32(math.Atan2(float64(d0.x), float64(-d0.y)))
		a1 = float32(math.Atan2(float64(-d1.x), float64(d1.y)))
		dir = Clockwise
	}
	p.Arc(cx, cy, radius, a0, a1, dir)
}

// Arc adds an arc to the path.
// (x, y) is the center of the arc.
func (p *Path) Arc(x, y, radius, startAngle, endAngle float32, dir Direction) {
	// Adjust the angles.
	var da float64
	if dir == Clockwise {
		for startAngle > endAngle {
			endAngle += 2 * math.Pi
		}
		da = float64(endAngle - startAngle)
	} else {
		for startAngle < endAngle {
			startAngle += 2 * math.Pi
		}
		da = float64(startAngle - endAngle)
	}

	if da >= 2*math.Pi {
		da = 2 * math.Pi
		if dir == Clockwise {
			endAngle = startAngle + 2*math.Pi
		} else {
			startAngle = endAngle + 2*math.Pi
		}
	}

	// If the angle is big, splict this into multiple Arc calls.
	if da > math.Pi/2 {
		const delta = math.Pi / 3
		a := float64(startAngle)
		if dir == Clockwise {
			for {
				p.Arc(x, y, radius, float32(a), float32(math.Min(a+delta, float64(endAngle))), dir)
				if a+delta >= float64(endAngle) {
					break
				}
				a += delta
			}
		} else {
			for {
				p.Arc(x, y, radius, float32(a), float32(math.Max(a-delta, float64(endAngle))), dir)
				if a-delta <= float64(endAngle) {
					break
				}
				a -= delta
			}
		}
		return
	}

	sin0, cos0 := math.Sincos(float64(startAngle))
	x0 := x + radius*float32(cos0)
	y0 := y + radius*float32(sin0)
	sin1, cos1 := math.Sincos(float64(endAngle))
	x1 := x + radius*float32(cos1)
	y1 := y + radius*float32(sin1)

	p.LineTo(x0, y0)

	// Calculate the control points for an approximated Bézier curve.
	// See https://learn.microsoft.com/en-us/previous-versions/xamarin/xamarin-forms/user-interface/graphics/skiasharp/curves/beziers.
	l := radius * float32(math.Tan(da/4)*4/3)
	var cx0, cy0, cx1, cy1 float32
	if dir == Clockwise {
		cx0 = x0 + l*float32(-sin0)
		cy0 = y0 + l*float32(cos0)
		cx1 = x1 + l*float32(sin1)
		cy1 = y1 + l*float32(-cos1)
	} else {
		cx0 = x0 + l*float32(sin0)
		cy0 = y0 + l*float32(-cos0)
		cx1 = x1 + l*float32(-sin1)
		cy1 = y1 + l*float32(cos1)
	}
	p.CubicTo(cx0, cy0, cx1, cy1, x1, y1)
}

func (p *Path) close() {
	if len(p.subpaths) == 0 {
		return
	}
	p.subpaths[len(p.subpaths)-1].close()
}

// AppendVerticesAndIndicesForFilling appends vertices and indices to fill this path and returns them.
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
func (p *Path) AppendVerticesAndIndicesForFilling(vertices []ebiten.Vertex, indices []uint16) ([]ebiten.Vertex, []uint16) {
	// TODO: Add tests.

	base := uint16(len(vertices))
	for _, subpath := range p.ensureSubpaths() {
		if subpath.pointCount() < 3 {
			continue
		}
		for i, pt := range subpath.points {
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
		base += uint16(subpath.pointCount())
	}
	return vertices, indices
}

// LineCap represents the way in which how the ends of the stroke are rendered.
type LineCap int

const (
	LineCapButt LineCap = iota
	LineCapRound
	LineCapSquare
)

// LineJoin represents the way in which how two segments are joined.
type LineJoin int

const (
	LineJoinMiter LineJoin = iota
	LineJoinBevel
	LineJoinRound
)

// StrokeOptions is options to render a stroke.
type StrokeOptions struct {
	// Width is the stroke width in pixels.
	//
	// The default (zero) value is 0.
	Width float32

	// LineCap is the way in which how the ends of the stroke are rendered.
	// Line caps are not rendered when the subpath is marked as closed.
	//
	// The default (zero) value is LineCapButt.
	LineCap LineCap

	// LineJoin is the way in which how two segments are joined.
	//
	// The default (zero) value is LineJoiMiter.
	LineJoin LineJoin

	// MiterLimit is the miter limit for LineJoinMiter.
	// For details, see https://developer.mozilla.org/en-US/docs/Web/SVG/Attribute/stroke-miterlimit.
	//
	// The default (zero) value is 0.
	MiterLimit float32
}

// AppendVerticesAndIndicesForStroke appends vertices and indices to render a stroke of this path and returns them.
// AppendVerticesAndIndicesForStroke works in a similar way to the built-in append function.
// If the arguments are nils, AppendVerticesAndIndicesForStroke returns new slices.
//
// The returned vertice's SrcX and SrcY are 0, and ColorR, ColorG, ColorB, and ColorA are 1.
//
// The returned values are intended to be passed to DrawTriangles or DrawTrianglesShader with a solid (non-transparent) color
// with FillRuleFillAll or FillRuleNonZero, not FileRuleEvenOdd.
func (p *Path) AppendVerticesAndIndicesForStroke(vertices []ebiten.Vertex, indices []uint16, op *StrokeOptions) ([]ebiten.Vertex, []uint16) {
	if op == nil {
		return vertices, indices
	}

	var rects [][4]point
	var tmpPath Path
	for _, subpath := range p.ensureSubpaths() {
		if subpath.pointCount() < 2 {
			continue
		}

		rects = rects[:0]
		for i := 0; i < subpath.pointCount()-1; i++ {
			pt := subpath.points[i]

			nextPt := subpath.points[i+1]
			dx := nextPt.x - pt.x
			dy := nextPt.y - pt.y
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			extX := (dy) * op.Width / 2 / dist
			extY := (-dx) * op.Width / 2 / dist

			rects = append(rects, [4]point{
				{
					x: pt.x + extX,
					y: pt.y + extY,
				},
				{
					x: nextPt.x + extX,
					y: nextPt.y + extY,
				},
				{
					x: pt.x - extX,
					y: pt.y - extY,
				},
				{
					x: nextPt.x - extX,
					y: nextPt.y - extY,
				},
			})
		}

		for i, rect := range rects {
			idx := uint16(len(vertices))
			for _, pt := range rect {
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
			}
			// All the triangles are rendered in clockwise order to enable FillRuleNonZero (#2833).
			indices = append(indices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)

			// Add line joints.
			var nextRect [4]point
			if i < len(rects)-1 {
				nextRect = rects[i+1]
			} else if subpath.closed {
				nextRect = rects[0]
			} else {
				continue
			}

			// c is the center of the 'end' edge of the current rect (= the second point of the segment).
			c := point{
				x: (rect[1].x + rect[3].x) / 2,
				y: (rect[1].y + rect[3].y) / 2,
			}

			// Note that the Y direction and the angle direction are opposite from math's.
			a0 := float32(math.Atan2(float64(rect[1].y-c.y), float64(rect[1].x-c.x)))
			a1 := float32(math.Atan2(float64(nextRect[0].y-c.y), float64(nextRect[0].x-c.x)))
			da := a1 - a0
			for da < 0 {
				da += 2 * math.Pi
			}
			if da == 0 {
				continue
			}

			switch op.LineJoin {
			case LineJoinMiter:
				delta := math.Pi - da
				exceed := float32(math.Abs(1/math.Sin(float64(delta/2)))) > op.MiterLimit

				// Quadrilateral
				tmpPath.reset()
				tmpPath.MoveTo(c.x, c.y)
				if da < math.Pi {
					tmpPath.LineTo(rect[1].x, rect[1].y)
					if !exceed {
						pt := crossingPointForTwoLines(rect[0], rect[1], nextRect[0], nextRect[1])
						tmpPath.LineTo(pt.x, pt.y)
					}
					tmpPath.LineTo(nextRect[0].x, nextRect[0].y)
				} else {
					tmpPath.LineTo(rect[3].x, rect[3].y)
					if !exceed {
						pt := crossingPointForTwoLines(rect[2], rect[3], nextRect[2], nextRect[3])
						tmpPath.LineTo(pt.x, pt.y)
					}
					tmpPath.LineTo(nextRect[2].x, nextRect[2].y)
				}
				vertices, indices = tmpPath.AppendVerticesAndIndicesForFilling(vertices, indices)

			case LineJoinBevel:
				// Triangle
				tmpPath.reset()
				tmpPath.MoveTo(c.x, c.y)
				if da < math.Pi {
					tmpPath.LineTo(rect[1].x, rect[1].y)
					tmpPath.LineTo(nextRect[0].x, nextRect[0].y)
				} else {
					tmpPath.LineTo(rect[3].x, rect[3].y)
					tmpPath.LineTo(nextRect[2].x, nextRect[2].y)
				}
				vertices, indices = tmpPath.AppendVerticesAndIndicesForFilling(vertices, indices)

			case LineJoinRound:
				// Arc
				tmpPath.reset()
				tmpPath.MoveTo(c.x, c.y)
				if da < math.Pi {
					tmpPath.Arc(c.x, c.y, op.Width/2, a0, a1, Clockwise)
				} else {
					tmpPath.Arc(c.x, c.y, op.Width/2, a0+math.Pi, a1+math.Pi, CounterClockwise)
				}
				vertices, indices = tmpPath.AppendVerticesAndIndicesForFilling(vertices, indices)
			}
		}

		if len(rects) == 0 {
			continue
		}

		// If the subpath is closed, do not render line caps.
		if subpath.closed {
			continue
		}

		switch op.LineCap {
		case LineCapButt:
			// Do nothing.

		case LineCapRound:
			startR, endR := rects[0], rects[len(rects)-1]
			{
				c := point{
					x: (startR[0].x + startR[2].x) / 2,
					y: (startR[0].y + startR[2].y) / 2,
				}
				a := float32(math.Atan2(float64(startR[0].y-startR[2].y), float64(startR[0].x-startR[2].x)))
				// Arc
				tmpPath.reset()
				tmpPath.MoveTo(startR[0].x, startR[0].y)
				tmpPath.Arc(c.x, c.y, op.Width/2, a, a+math.Pi, CounterClockwise)
				vertices, indices = tmpPath.AppendVerticesAndIndicesForFilling(vertices, indices)
			}
			{
				c := point{
					x: (endR[1].x + endR[3].x) / 2,
					y: (endR[1].y + endR[3].y) / 2,
				}
				a := float32(math.Atan2(float64(endR[1].y-endR[3].y), float64(endR[1].x-endR[3].x)))
				// Arc
				tmpPath.reset()
				tmpPath.MoveTo(endR[1].x, endR[1].y)
				tmpPath.Arc(c.x, c.y, op.Width/2, a, a+math.Pi, Clockwise)
				vertices, indices = tmpPath.AppendVerticesAndIndicesForFilling(vertices, indices)
			}

		case LineCapSquare:
			startR, endR := rects[0], rects[len(rects)-1]
			{
				a := math.Atan2(float64(startR[0].y-startR[1].y), float64(startR[0].x-startR[1].x))
				s, c := math.Sincos(a)
				dx, dy := float32(c)*op.Width/2, float32(s)*op.Width/2

				// Quadrilateral
				tmpPath.reset()
				tmpPath.MoveTo(startR[0].x, startR[0].y)
				tmpPath.LineTo(startR[0].x+dx, startR[0].y+dy)
				tmpPath.LineTo(startR[2].x+dx, startR[2].y+dy)
				tmpPath.LineTo(startR[2].x, startR[2].y)
				vertices, indices = tmpPath.AppendVerticesAndIndicesForFilling(vertices, indices)
			}
			{
				a := math.Atan2(float64(endR[1].y-endR[0].y), float64(endR[1].x-endR[0].x))
				s, c := math.Sincos(a)
				dx, dy := float32(c)*op.Width/2, float32(s)*op.Width/2

				// Quadrilateral
				tmpPath.reset()
				tmpPath.MoveTo(endR[1].x, endR[1].y)
				tmpPath.LineTo(endR[1].x+dx, endR[1].y+dy)
				tmpPath.LineTo(endR[3].x+dx, endR[3].y+dy)
				tmpPath.LineTo(endR[3].x, endR[3].y)
				vertices, indices = tmpPath.AppendVerticesAndIndicesForFilling(vertices, indices)
			}
		}
	}

	return vertices, indices
}
