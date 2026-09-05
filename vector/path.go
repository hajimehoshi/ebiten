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
	"math"
	"slices"

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
	opTypeLineTo opType = iota
	opTypeQuadTo
)

type op struct {
	typ opType
	p1  point
	p2  point
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

const epsilon = 1e-6

type point struct {
	x float32
	y float32
}

func (p point) add(v vec2) point {
	return point{x: p.x + v.x, y: p.y + v.y}
}

type vec2 struct {
	x, y float32
}

func (v vec2) perp() vec2 {
	return vec2{x: -v.y, y: v.x}
}

func (v vec2) inv() vec2 {
	return vec2{x: -v.x, y: -v.y}
}

func (v vec2) len() float32 {
	return float32(math.Hypot(float64(v.x), float64(v.y)))
}

func (v vec2) norm() vec2 {
	len := v.len()
	if len == 0 {
		return vec2{float32(math.NaN()), float32(math.NaN())}
	}
	return vec2{v.x / len, v.y / len}
}

func (v vec2) mul(s float32) vec2 {
	return vec2{x: s * v.x, y: s * v.y}
}

type subPath struct {
	ops    []op
	start  point
	closed bool

	// invalid indicates that the sub-path has a non-finite coordinate.
	// The operations modifying the sub-path keep this up to date so that reading a path never writes to it.
	invalid bool
}

func (s *subPath) reset() {
	s.ops = s.ops[:0]
	s.start = point{}
	s.closed = false
	s.invalid = false
}

func isRegularF32(x float32) bool {
	return !math.IsNaN(float64(x)) && !math.IsInf(float64(x), 0)
}

func isRegularPoint(p point) bool {
	return isRegularF32(p.x) && isRegularF32(p.y)
}

func (s *subPath) isValid() bool {
	return !s.invalid
}

// setStart sets the start position of the sub-path, which must not have any operations.
func (s *subPath) setStart(pt point) {
	s.start = pt
	s.invalid = !isRegularPoint(pt)
}

// appendOp adds the operation o to the sub-path.
func (s *subPath) appendOp(o op) {
	s.ops = append(s.ops, o)
	if s.invalid {
		return
	}
	switch o.typ {
	case opTypeLineTo:
		s.invalid = !isRegularPoint(o.p1)
	case opTypeQuadTo:
		s.invalid = !isRegularPoint(o.p1) || !isRegularPoint(o.p2)
	}
}

// updateValidity updates the validity state, and must be called after the coordinates are modified directly.
func (s *subPath) updateValidity() {
	s.invalid = true
	if !isRegularPoint(s.start) {
		return
	}
	for _, op := range s.ops {
		switch op.typ {
		case opTypeLineTo:
			if !isRegularPoint(op.p1) {
				return
			}
		case opTypeQuadTo:
			if !isRegularPoint(op.p1) || !isRegularPoint(op.p2) {
				return
			}
		}
	}
	s.invalid = false
}

func (s *subPath) startAtOp(index int) point {
	if index == 0 {
		return s.start
	}
	return s.endAtOp(index - 1)
}

func (s *subPath) endAtOp(index int) point {
	op := s.ops[index]
	switch op.typ {
	case opTypeLineTo:
		return op.p1
	case opTypeQuadTo:
		return op.p2
	}
	panic("not reached")
}

func (s *subPath) startDir(index int) vec2 {
	p := s.startAtOp(index)
	op := s.ops[index]
	return vec2{x: op.p1.x - p.x, y: op.p1.y - p.y}
}

func (s *subPath) endDir(index int) vec2 {
	switch op := s.ops[index]; op.typ {
	case opTypeLineTo:
		return s.startDir(index)
	case opTypeQuadTo:
		return vec2{x: op.p2.x - op.p1.x, y: op.p2.y - op.p1.y}
	}
	panic("not reached")
}

// Path represents a collection of vector graphics operations.
//
// All the coordinates in a path must be finite.
// A path with a non-finite coordinate produces an undefined result.
type Path struct {
	subPaths []subPath

	// flatPaths is a cached actual rendering positions.
	// flatPaths is used only for deprecated functions. Do not use this for new functions.
	flatPaths []flatPath

	// opsBuf is a buffer of operations, which AddStroke uses to normalize the source sub-paths.
	opsBuf []op
}

// Reset resets the path.
// Reset doesn't release the allocated memory so that the memory can be reused.
func (p *Path) Reset() {
	p.resetSubPaths()
	p.resetFlatPaths()
}

func (p *Path) resetSubPaths() {
	for i := range p.subPaths {
		p.subPaths[i].reset()
	}
	p.subPaths = p.subPaths[:0]
}

func (p *Path) addSubPaths(n int) {
	// Use slices.Grow instead of append to reuse the underlying sub path object.
	p.subPaths = slices.Grow(p.subPaths, n)[:len(p.subPaths)+n]
}

// MoveTo starts a new sub-path with the given position (x, y), without adding any line segments.
// If the last sub-path is still empty, MoveTo updates its start position instead of adding a new one.
func (p *Path) MoveTo(x, y float32) {
	p.resetFlatPaths()

	// Always update the start position.
	if len(p.subPaths) == 0 || len(p.subPaths[len(p.subPaths)-1].ops) > 0 {
		p.addSubPaths(1)
	}
	p.subPaths[len(p.subPaths)-1].setStart(point{x: x, y: y})
	p.subPaths[len(p.subPaths)-1].closed = false
}

// LineTo adds a line segment to the path, which starts from the last position of the current sub-path
// and ends to the given position (x, y).
// If p doesn't have any sub-paths, LineTo sets (x, y) as the start position of a new sub-path.
// If the last sub-path is closed, LineTo creates a new sub-path whose start position is the same as the closed sub-path's.
func (p *Path) LineTo(x, y float32) {
	p.resetFlatPaths()

	if len(p.subPaths) == 0 {
		p.addSubPaths(1)
		p.subPaths[len(p.subPaths)-1].setStart(point{x: x, y: y})
	} else if p.subPaths[len(p.subPaths)-1].closed {
		p.addSubPaths(1)
		p.subPaths[len(p.subPaths)-1].setStart(p.subPaths[len(p.subPaths)-2].start)
	}
	if cur, ok := p.currentPosition(); ok {
		if cur.x == x && cur.y == y {
			return
		}
	}
	p.subPaths[len(p.subPaths)-1].appendOp(op{
		typ: opTypeLineTo,
		p1:  point{x: x, y: y},
	})
}

// QuadTo adds a quadratic Bézier curve to the path.
// (x1, y1) is the control point, and (x2, y2) is the destination.
func (p *Path) QuadTo(x1, y1, x2, y2 float32) {
	p.resetFlatPaths()

	if len(p.subPaths) == 0 {
		p.addSubPaths(1)
		p.subPaths[len(p.subPaths)-1].setStart(point{x: x1, y: y1})
	} else if p.subPaths[len(p.subPaths)-1].closed {
		p.addSubPaths(1)
		p.subPaths[len(p.subPaths)-1].setStart(p.subPaths[len(p.subPaths)-2].start)
	}
	if cur, ok := p.currentPosition(); ok {
		// A quadratic whose start, control, and end points all coincide is a single point and can be dropped.
		// Even if the start and end points coincide, it is not a single point when the control point differs:
		// it is a cusp, which goes to the midpoint and comes back, so it must be kept.
		if cur.x == x1 && cur.y == y1 && cur.x == x2 && cur.y == y2 {
			return
		}
	}
	p.subPaths[len(p.subPaths)-1].appendOp(op{
		typ: opTypeQuadTo,
		p1:  point{x: x1, y: y1},
		p2:  point{x: x2, y: y2},
	})
}

// CubicTo adds a cubic Bézier curve to the path.
// (x1, y1) and (x2, y2) are the control points, and (x3, y3) is the destination.
func (p *Path) CubicTo(x1, y1, x2, y2, x3, y3 float32) {
	cur, ok := p.currentPosition()
	if !ok {
		cur = point{x: x1, y: y1}
	}
	minX := min(cur.x, x1, x2, x3)
	maxX := max(cur.x, x1, x2, x3)
	minY := min(cur.y, y1, y2, y3)
	maxY := max(cur.y, y1, y2, y3)
	allowance := max(maxX-minX, maxY-minY) / 1024
	p.cubicTo(x1, y1, x2, y2, x3, y3, 0, allowance)
}

func (p *Path) cubicTo(x1, y1, x2, y2, x3, y3 float32, level int, allowance float32) {
	cur, ok := p.currentPosition()
	if !ok {
		cur = point{x: x1, y: y1}
	}

	// Approximate a cubic Bézier curve to a quadratic Bézier curve.
	// Assume that P0, P1, P2, and P3 are the control points of the cubic Bézier curve C.
	// mid is the middle control point of the quadratic Bézier curve Q.
	// mid equals to 2 * Q(0.5) - (1/2)*(P0 + P3).
	// If Q(0.5) = C(0.5) = (1/8)*(P0 + 3*P1 + 3*P2 + P3), mid will be (1/4)*(-P0 + 3*P1 + 3*P2 + -P3).
	p0 := cur
	p1 := point{x: x1, y: y1}
	p2 := point{x: x2, y: y2}
	p3 := point{x: x3, y: y3}
	m := point{
		x: -0.25*p0.x + 0.75*p1.x + 0.75*p2.x - 0.25*p3.x,
		y: -0.25*p0.y + 0.75*p1.y + 0.75*p2.y - 0.25*p3.y,
	}
	if level > 5 || isQuadraticCloseEnoughToCubic(p0, p3, m, p1, p2, allowance) {
		p.QuadTo(m.x, m.y, p3.x, p3.y)
		return
	}

	// If any of the points is not a regular float32, do not call this function recursively.
	if !isRegularF32(cur.x) || !isRegularF32(cur.y) || !isRegularF32(x1) || !isRegularF32(y1) || !isRegularF32(x2) || !isRegularF32(y2) || !isRegularF32(x3) || !isRegularF32(y3) {
		p.QuadTo(m.x, m.y, p3.x, p3.y)
		return
	}

	// Split the cubic Bézier curve into two by De Casteljau's algorithm.
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
	p.cubicTo(p01.x, p01.y, p012.x, p012.y, p0123.x, p0123.y, level+1, allowance)
	p.cubicTo(p123.x, p123.y, p23.x, p23.y, p3.x, p3.y, level+1, allowance)
}

func isQuadraticCloseEnoughToCubic(start, end, qc1, cc1, cc2 point, allowance float32) bool {
	for _, t := range []float32{0.25, 0.75} {
		q := point{
			x: (1-t)*(1-t)*start.x + 2*(1-t)*t*qc1.x + t*t*end.x,
			y: (1-t)*(1-t)*start.y + 2*(1-t)*t*qc1.y + t*t*end.y,
		}
		c := point{
			x: (1-t)*(1-t)*(1-t)*start.x + 3*(1-t)*(1-t)*t*cc1.x + 3*(1-t)*t*t*cc2.x + t*t*t*end.x,
			y: (1-t)*(1-t)*(1-t)*start.y + 3*(1-t)*(1-t)*t*cc1.y + 3*(1-t)*t*t*cc2.y + t*t*t*end.y,
		}
		if !arePointsInRange(q, c, 0, allowance) {
			return false
		}
	}
	return true
}

func arePointsInRange(p0, p1 point, allowanceMin, allowanceMax float32) bool {
	d := (p0.x-p1.x)*(p0.x-p1.x) + (p0.y-p1.y)*(p0.y-p1.y)
	return d >= allowanceMin*allowanceMin && d <= allowanceMax*allowanceMax
}

// Close adds a new line from the last position of the current sub-path to the first position of the current sub-path,
// and marks the current sub-path closed.
// Following operations for this path will start with a new sub-path.
func (p *Path) Close() {
	p.resetFlatPaths()

	if len(p.subPaths) == 0 {
		return
	}
	if len(p.subPaths[len(p.subPaths)-1].ops) > 0 {
		subPath := &p.subPaths[len(p.subPaths)-1]
		start := subPath.start
		p.LineTo(start.x, start.y)
	}
	p.subPaths[len(p.subPaths)-1].closed = true
}

// lineForTwoPoints returns parameters for a line passing through p0 and p1.
func lineForTwoPoints(p0, p1 point) (a, b, c float32) {
	// Line passing through p0 and p1 in the form of ax + by + c = 0
	a = p1.y - p0.y
	b = -(p1.x - p0.x)
	c = (p1.x-p0.x)*p0.y - (p1.y-p0.y)*p0.x
	return
}

// isPointCloseToSegment detects the distance between the point p and the line passing through p0 and p1 is less than allow.
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

	// If det is close to 0, the two lines are almost parallel.
	if abs(det) < epsilon {
		return point{
			x: float32(math.NaN()),
			y: float32(math.NaN()),
		}
	}

	return point{
		x: (b0*c1 - b1*c0) / det,
		y: (a1*c0 - a0*c1) / det,
	}
}

func (p *Path) currentPosition() (point, bool) {
	if len(p.subPaths) == 0 {
		return point{}, false
	}
	ops := p.subPaths[len(p.subPaths)-1].ops
	if len(ops) == 0 {
		return p.subPaths[len(p.subPaths)-1].start, true
	}
	op := ops[len(ops)-1]
	switch op.typ {
	case opTypeLineTo:
		return op.p1, true
	case opTypeQuadTo:
		return op.p2, true
	}
	return point{}, false
}

// ArcTo adds an arc curve to the path.
// The arc is tangent to the line from the current position to (x1, y1) and to the line from (x1, y1) to (x2, y2).
// (x1, y1) is the corner point where the two tangent lines meet, and (x2, y2) gives the direction of the second
// tangent line. The arc ends at the tangent point on the second tangent line, which is at the distance of
// radius / tan(θ/2) from (x1, y1), where θ is the angle at (x1, y1) between the two rays.
// radius must be non-negative. A negative radius produces an undefined result.
func (p *Path) ArcTo(x1, y1, x2, y2, radius float32) {
	p0, ok := p.currentPosition()
	if !ok {
		p0 = point{x: x1, y: y1}
	}

	// If the start and end points are too close, just add a line segment to avoid strange rendering results.
	if arePointsInRange(p0, point{x: x2, y: y2}, 0, radius) {
		p.LineTo(x2, y2)
		return
	}

	d0 := vec2{
		x: p0.x - x1,
		y: p0.y - y1,
	}
	d1 := vec2{
		x: x2 - x1,
		y: y2 - y1,
	}
	if d0 == (vec2{}) || d1 == (vec2{}) {
		p.LineTo(x1, y1)
		return
	}

	d0 = d0.norm()
	d1 = d1.norm()

	dot := min(max(float64(d0.x*d1.x+d0.y*d1.y), -1.0), 1.0)
	// theta is the angle between two vectors d0 and d1.
	theta := math.Acos(dot)
	// TODO: When theta is bigger than π/2, the arc should be split into two.
	if theta == 0 {
		p.LineTo(x2, y2)
		return
	}

	// dist is the distance between the control point and the arc's beginning and ending points.
	dist := radius / float32(math.Tan(theta/2))

	// TODO: What if dist is too big?

	// (ax0, ay0) is the start of the arc.
	ax0 := x1 + d0.x*dist
	ay0 := y1 + d0.y*dist

	var cx, cy, a0, a1 float32
	var dir Direction

	// A cross product can be calculated by d0.x*d1.y - d0.y*d1.x,
	// but this can cause a floating-point precision issue due to FMSUBS.
	// Avoid this subtraction.
	//
	// a*b - c*d can be translated into
	//     (1) x := c*d
	//     (2) y := a*b-x
	// One rounding happens at (1). The number of rounding is not determined at (2).
	// If FMSUBS is used for (2), only one rounding happens at (2) for the multiplying and the subtraction.
	// Thus, even if a*b == c*d, y can be non-zero.
	if d0.x*d1.y >= d0.y*d1.x {
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

func euclideanMod(a, b float32) float32 {
	return a - b*float32(math.Floor(float64(a)/float64(b)))
}

// Arc adds an arc to the path.
// (x, y) is the center of the arc.
// radius must be non-negative. A negative radius produces an undefined result.
func (p *Path) Arc(x, y, radius, startAngle, endAngle float32, dir Direction) {
	origStartAngle := startAngle
	origEndAngle := endAngle

	// Adjust the angles.
	var da float64
	if dir == Clockwise {
		endAngle = startAngle + float32(euclideanMod(endAngle-startAngle, 2*math.Pi))
		da = float64(endAngle - startAngle)
	} else {
		startAngle = endAngle + float32(euclideanMod(startAngle-endAngle, 2*math.Pi))
		da = float64(startAngle - endAngle)
	}

	if da == 0 && origStartAngle != origEndAngle {
		da = 2 * math.Pi
		if dir == Clockwise {
			endAngle = startAngle + 2*math.Pi
		} else {
			startAngle = endAngle + 2*math.Pi
		}
	}

	// If the angle is big, split this into multiple arcs.
	if da > math.Pi/2 {
		const delta = math.Pi / 3
		a := float64(startAngle)
		if dir == Clockwise {
			for {
				p.arc(x, y, radius, float32(a), float32(min(a+delta, float64(endAngle))), dir)
				if a+delta >= float64(endAngle) {
					break
				}
				a += delta
			}
		} else {
			for {
				p.arc(x, y, radius, float32(a), float32(max(a-delta, float64(endAngle))), dir)
				if a-delta <= float64(endAngle) {
					break
				}
				a -= delta
			}
		}
		return
	}

	p.arc(x, y, radius, startAngle, endAngle, dir)
}

// arc adds an arc to the path without splitting it.
// startAngle and endAngle must be already adjusted so that the arc sweeps from startAngle to endAngle in the direction dir.
func (p *Path) arc(x, y, radius, startAngle, endAngle float32, dir Direction) {
	var da float64
	if dir == Clockwise {
		da = float64(endAngle - startAngle)
	} else {
		da = float64(startAngle - endAngle)
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

// AddPathOptions is options for [Path.AddPath].
type AddPathOptions struct {
	// GeoM is a geometry matrix to apply to the path.
	//
	// The default (zero) value is an identity matrix.
	GeoM ebiten.GeoM
}

// AddPath adds the given path src to this path p as a sub-path.
func (p *Path) AddPath(src *Path, options *AddPathOptions) {
	p.resetFlatPaths()

	if options == nil {
		options = &AddPathOptions{}
	}

	srcN := len(src.subPaths)
	n := len(p.subPaths)
	p.addSubPaths(srcN)
	// p might be the same as src. Use srcN to avoid modifying the overlapped region.
	for i, origSubPath := range src.subPaths[:srcN] {
		sx, sy := options.GeoM.Apply(float64(origSubPath.start.x), float64(origSubPath.start.y))
		if m := len(origSubPath.ops) - len(p.subPaths[n+i].ops); m > 0 {
			p.subPaths[n+i].ops = slices.Grow(p.subPaths[n+i].ops, m)
		}
		p.subPaths[n+i].ops = p.subPaths[n+i].ops[:len(origSubPath.ops)]
		p.subPaths[n+i].start = point{x: float32(sx), y: float32(sy)}
		p.subPaths[n+i].closed = origSubPath.closed

		for j, o := range origSubPath.ops {
			switch o.typ {
			case opTypeLineTo:
				x1, y1 := options.GeoM.Apply(float64(o.p1.x), float64(o.p1.y))
				p.subPaths[n+i].ops[j] = op{
					typ: o.typ,
					p1:  point{x: float32(x1), y: float32(y1)},
				}
			case opTypeQuadTo:
				x1, y1 := options.GeoM.Apply(float64(o.p1.x), float64(o.p1.y))
				x2, y2 := options.GeoM.Apply(float64(o.p2.x), float64(o.p2.y))
				p.subPaths[n+i].ops[j] = op{
					typ: o.typ,
					p1:  point{x: float32(x1), y: float32(y1)},
					p2:  point{x: float32(x2), y: float32(y2)},
				}
			}
		}

		p.subPaths[n+i].updateValidity()
	}
}

// countCusps counts the number of cusps in subPath, which are quadratic curves whose start and end points are the same
// and whose control point differs from them.
func countCusps(subPath *subPath) int {
	var n int
	cur := subPath.start
	for _, op := range subPath.ops {
		switch op.typ {
		case opTypeLineTo:
			cur = op.p1
		case opTypeQuadTo:
			if cur == op.p2 && cur != op.p1 {
				n++
			}
			cur = op.p2
		}
	}
	return n
}

// normalizeSubPath normalizes src by removing unnecessary operations, and stores the result into dst.
// dst must not be src.
func normalizeSubPath(dst *subPath, src *subPath) {
	// A cusp is converted into two lines, which increases the number of operations.
	ops := slices.Grow(dst.ops[:0], len(src.ops)+countCusps(src))
	cur := src.start
	for _, op := range src.ops {
		switch op.typ {
		case opTypeLineTo:
			if cur == op.p1 {
				continue
			}
			cur = op.p1
		case opTypeQuadTo:
			switch {
			case cur == op.p1 && op.p1 == op.p2:
				// A single point: drop it.
				continue
			case cur == op.p2:
				// A cusp goes to the midpoint and comes back.
				// Keep this as lines, not as a curve, so that the 180-degree turn at the tip gets a joint.
				// Divide by 2 before adding so that the midpoint computation cannot overflow.
				mid := point{
					x: cur.x/2 + op.p1.x/2,
					y: cur.y/2 + op.p1.y/2,
				}
				if mid == cur {
					// The midpoint is rounded to the current point in float32, so there is nothing to keep.
					continue
				}
				first := op
				first.typ = opTypeLineTo
				first.p1 = mid
				first.p2 = point{}
				ops = append(ops, first)
				op.typ = opTypeLineTo
				op.p1 = op.p2
				op.p2 = point{}
				cur = op.p1
			case cur == op.p1, op.p1 == op.p2:
				op.typ = opTypeLineTo
				op.p1 = op.p2
				op.p2 = point{}
				cur = op.p1
			case (op.p1.x-cur.x)*(op.p2.y-cur.y)-(op.p2.x-cur.x)*(op.p1.y-cur.y) == 0:
				op.typ = opTypeLineTo
				op.p1 = op.p2
				op.p2 = point{}
				cur = op.p1
			default:
				cur = op.p2
			}
		}
		ops = append(ops, op)
	}

	dst.ops = ops
	dst.start = src.start
	dst.closed = src.closed
	dst.updateValidity()
}

func floor(x float32) int {
	return int(math.Floor(float64(x)))
}

func ceil(x float32) int {
	return int(math.Ceil(float64(x)))
}

// Bounds returns the minimum bounding rectangle of the path.
// The rectangle can have a zero width or height when the path is degenerate in an axis, like a horizontal line.
// Bounds returns the zero rectangle when the path has no drawing operations.
func (p *Path) Bounds() image.Rectangle {
	// Note that (image.Rectangle).Union doesn't work well with empty rectangles.
	totalMinX := math.MaxInt
	totalMinY := math.MaxInt
	totalMaxX := math.MinInt
	totalMaxY := math.MinInt
	var found bool

	for i := range p.subPaths {
		subPath := &p.subPaths[i]
		if !subPath.isValid() || len(subPath.ops) == 0 {
			continue
		}
		found = true

		minX := math.MaxInt
		minY := math.MaxInt
		maxX := math.MinInt
		maxY := math.MinInt
		cur := subPath.start
		for _, op := range subPath.ops {
			switch op.typ {
			case opTypeLineTo:
				minX = min(minX, floor(cur.x), floor(op.p1.x))
				minY = min(minY, floor(cur.y), floor(op.p1.y))
				maxX = max(maxX, ceil(cur.x), ceil(op.p1.x))
				maxY = max(maxY, ceil(cur.y), ceil(op.p1.y))
				cur = op.p1
			case opTypeQuadTo:
				// The candidates are the two control points on the edges (cur and op.p2), and an extremum point.
				// B(t) = (1-t)*(1-t)*p0 + 2*(1-t)*t*p1 + t*t*p2
				// B'(t) = 2*(1-t)*(p1-p0) + 2*t*(p2-p1)
				// B'(t) = 0 <=> t = (p0-p1) / (p0-2*p1+p2)
				// Avoid an extreme denominator for precision.
				if denom := cur.x - 2*op.p1.x + op.p2.x; abs(denom) >= 1.0/16.0 {
					if t := (cur.x - op.p1.x) / denom; t > 0 && t < 1 {
						ex := (1-t)*(1-t)*cur.x + 2*t*(1-t)*op.p1.x + t*t*op.p2.x
						minX = min(minX, floor(cur.x), floor(ex), floor(op.p2.x))
						maxX = max(maxX, ceil(cur.x), ceil(ex), ceil(op.p2.x))
					} else {
						minX = min(minX, floor(cur.x), floor(op.p2.x))
						maxX = max(maxX, ceil(cur.x), ceil(op.p2.x))
					}
				} else {
					// The curve is almost linear. Include all the points for safety.
					minX = min(minX, floor(cur.x), floor(op.p1.x), floor(op.p2.x))
					maxX = max(maxX, ceil(cur.x), ceil(op.p1.x), ceil(op.p2.x))
				}
				if denom := cur.y - 2*op.p1.y + op.p2.y; abs(denom) >= 1.0/16.0 {
					if t := (cur.y - op.p1.y) / denom; t > 0 && t < 1 {
						ex := (1-t)*(1-t)*cur.y + 2*t*(1-t)*op.p1.y + t*t*op.p2.y
						minY = min(minY, floor(cur.y), floor(ex), floor(op.p2.y))
						maxY = max(maxY, ceil(cur.y), ceil(ex), ceil(op.p2.y))
					} else {
						minY = min(minY, floor(cur.y), floor(op.p2.y))
						maxY = max(maxY, ceil(cur.y), ceil(op.p2.y))
					}
				} else {
					minY = min(minY, floor(cur.y), floor(op.p1.y), floor(op.p2.y))
					maxY = max(maxY, ceil(cur.y), ceil(op.p1.y), ceil(op.p2.y))
				}
				cur = op.p2
			}
		}
		totalMinX = min(totalMinX, minX)
		totalMinY = min(totalMinY, minY)
		totalMaxX = max(totalMaxX, maxX)
		totalMaxY = max(totalMaxY, maxY)
	}
	if !found {
		return image.Rectangle{}
	}
	return image.Rect(totalMinX, totalMinY, totalMaxX, totalMaxY)
}
