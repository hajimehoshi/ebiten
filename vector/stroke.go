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
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

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
	// Line caps are not rendered when the sub-path is marked as closed.
	//
	// The default (zero) value is [LineCapButt].
	LineCap LineCap

	// LineJoin is the way in which how two segments are joined.
	//
	// The default (zero) value is [LineJoinMiter].
	LineJoin LineJoin

	// MiterLimit is the miter limit for [LineJoinMiter].
	// For details, see https://developer.mozilla.org/en-US/docs/Web/SVG/Attribute/stroke-miterlimit.
	//
	// The default (zero) value is 0.
	MiterLimit float32
}

// AddStrokeOptions is options for [Path.AddStroke].
type AddStrokeOptions struct {
	// StrokeOptions is options for the stroke.
	StrokeOptions

	// GeoM is a geometry matrix to apply to the path.
	//
	// The default (zero) value is an identity matrix.
	GeoM ebiten.GeoM
}

// AddStroke adds a stroke path to the path p.
//
// The added stroke path must be rendered with FileRuleNonZero.
func (p *Path) AddStroke(src *Path, options *AddStrokeOptions) {
	if options == nil {
		return
	}
	if options.Width <= 0 {
		return
	}

	// Normalize the source path to simplify the logic to generate a stroke path.
	src.normalize()

	origN := len(p.subPaths)
	// p might be the same as src. Use srcN to avoid modifying the overlapped region.
	srcN := len(src.subPaths)
	for _, subPath := range src.subPaths[:srcN] {
		_, sp1, sp2, sp3, sp4 := strokeStartControlPositions(&subPath, options.Width/2)
		p.MoveTo(sp4.x, sp4.y)

		appendParalleledPathFromSubPath(p, &subPath, &options.StrokeOptions)
		_, ep1, ep2, ep3, ep4 := strokeEndControlPositions(&subPath, options.Width/2)
		if subPath.closed {
			p.Close()
			p.MoveTo(ep4.x, ep4.y)
		} else {
			switch options.LineCap {
			case LineCapButt:
				p.LineTo(ep4.x, ep4.y)
			case LineCapRound:
				p.ArcTo(ep1.x, ep1.y, ep2.x, ep2.y, options.Width/2)
				p.ArcTo(ep3.x, ep3.y, ep4.x, ep4.y, options.Width/2)
			case LineCapSquare:
				p.LineTo(ep1.x, ep1.y)
				p.LineTo(ep3.x, ep3.y)
				p.LineTo(ep4.x, ep4.y)
			}
		}
		appendParalleledPathFromSubPathReversed(p, &subPath, &options.StrokeOptions)
		if !subPath.closed {
			switch options.LineCap {
			case LineCapButt:
				p.LineTo(sp4.x, sp4.y)
			case LineCapRound:
				p.ArcTo(sp1.x, sp1.y, sp2.x, sp2.y, options.Width/2)
				p.ArcTo(sp3.x, sp3.y, sp4.x, sp4.y, options.Width/2)
			case LineCapSquare:
				p.LineTo(sp1.x, sp1.y)
				p.LineTo(sp3.x, sp3.y)
				p.LineTo(sp4.x, sp4.y)
			}
		}
		p.Close()
	}

	if options.GeoM != (ebiten.GeoM{}) {
		for i, subPath := range p.subPaths[origN:] {
			x, y := options.GeoM.Apply(float64(subPath.start.x), float64(subPath.start.y))
			p.subPaths[origN+i].start = point{x: float32(x), y: float32(y)}
			for j, op := range subPath.ops {
				switch op.typ {
				case opTypeLineTo:
					x1, y1 := options.GeoM.Apply(float64(op.p1.x), float64(op.p1.y))
					p.subPaths[origN+i].ops[j].p1 = point{x: float32(x1), y: float32(y1)}
				case opTypeQuadTo:
					x1, y1 := options.GeoM.Apply(float64(op.p1.x), float64(op.p1.y))
					x2, y2 := options.GeoM.Apply(float64(op.p2.x), float64(op.p2.y))
					p.subPaths[origN+i].ops[j].p1 = point{x: float32(x1), y: float32(y1)}
					p.subPaths[origN+i].ops[j].p2 = point{x: float32(x2), y: float32(y2)}
				}
			}
		}
	}
}

func strokeStartControlPositions(subPath *subPath, dist float32) (point, point, point, point, point) {
	p := subPath.startAtOp(0)
	dir := subPath.startDir(0).inv().norm().mul(dist)
	dirPerp := dir.perp()
	// TODO: These values are a little tricky. Refactor this.
	return p.add(dirPerp), p.add(dir).add(dirPerp), p.add(dir), p.add(dir).add(dirPerp.inv()), p.add(dirPerp.inv())
}

func strokeEndControlPositions(subPath *subPath, dist float32) (point, point, point, point, point) {
	p := subPath.endAtOp(len(subPath.ops) - 1)
	dir := subPath.endDir(len(subPath.ops) - 1).norm().mul(dist)
	dirPerp := dir.perp()
	// TODO: These values are a little tricky. Refactor this.
	return p.add(dirPerp), p.add(dir).add(dirPerp), p.add(dir), p.add(dir).add(dirPerp.inv()), p.add(dirPerp.inv())
}

func appendParalleledPathFromSubPath(strokePath *Path, subPath *subPath, options *StrokeOptions) {
	if len(subPath.ops) == 0 {
		panic("not reached")
	}

	// As the source path is normalized, every operation is guaranteed to be valid.
	// A line operation must have a different point from the start point.
	// A quadratic curve operation must have create a curve, not a line.

	cur := subPath.start

	for i, op := range subPath.ops {
		switch op.typ {
		case opTypeLineTo:
			appendParalleledLine(strokePath, cur, op.p1, options.Width/2)
			cur = op.p1
		case opTypeQuadTo:
			appendParalleledQuad(strokePath, cur, op.p1, op.p2, options.Width/2)
			cur = op.p2
		}
		addJoint(strokePath, subPath, i, false, options)
	}
}

func appendParalleledPathFromSubPathReversed(strokePath *Path, subPath *subPath, options *StrokeOptions) {
	if len(subPath.ops) == 0 {
		panic("not reached")
	}

	// As the source path is normalized, every operation is guaranteed to be valid.
	// A line operation must have a different point from the start point.
	// A quadratic curve operation must have create a curve, not a line.

	for i := len(subPath.ops) - 1; i >= 0; i-- {
		op := subPath.ops[i]
		nextP := subPath.startAtOp(i)
		switch op.typ {
		case opTypeLineTo:
			appendParalleledLine(strokePath, op.p1, nextP, options.Width/2)
		case opTypeQuadTo:
			appendParalleledQuad(strokePath, op.p2, op.p1, nextP, options.Width/2)
		}
		addJoint(strokePath, subPath, i, true, options)
	}
}

func appendParalleledLine(path *Path, p0, p1 point, dist float32) {
	if p0 == p1 {
		panic("not reached")
	}

	dir := vec2{x: p1.x - p0.x, y: p1.y - p0.y}
	v := dir.perp().norm().mul(dist)
	pp1 := p1.add(v)
	path.LineTo(pp1.x, pp1.y)
}

// appendParalleledLineForQuadIfNeeded appends a paralleled line for a quadratic curve if the quadratic curve is just a line.
func appendParalleledLineForQuadIfNeeded(path *Path, p0, p1, p2 point, dist float32) bool {
	if p0 == p1 && p0 == p2 {
		panic("not reached")
	}
	// This curve is empty as the start and the end points are the same.
	if p0 == p2 {
		return true
	}
	// This curve is a line as the control point is the same as the start point.
	if p0 == p1 || p1 == p2 {
		appendParalleledLine(path, p0, p2, dist)
		return true
	}
	// This curve is a line as p0, p1, and p2 are on the same line.
	if (p1.x-p0.x)*(p2.y-p0.y)-(p2.x-p0.x)*(p1.y-p0.y) == 0 {
		appendParalleledLine(path, p0, p2, dist)
		return true
	}
	return false
}

func appendParalleledQuad(path *Path, p0, p1, p2 point, dist float32) {
	if appendParalleledLineForQuadIfNeeded(path, p0, p1, p2, dist) {
		return
	}
	doAppendParalleledQuad(path, p0, p1, p2, dist, 0)
}

func doAppendParalleledQuad(path *Path, p0, p1, p2 point, dist float32, level int) {
	if p0 == p1 && p0 == p2 {
		return
	}
	if appendParalleledLineForQuadIfNeeded(path, p0, p1, p2, dist) {
		return
	}

	// B(t) = (1-t)*(1-t)*p0 + 2*(1-t)*t*p1 + t*t*p2
	// B'(t) = 2*(1-t)*(p1-p0) + 2*t*(p2-p1)
	// B'(0) = 2*(p1-p0)
	// B'(0.5) = p2-p0
	// B'(1) = 2*(p2-p1)
	// B''(t) = 2*(p0 - 2*p1 + p2)

	// t = 0
	dir0 := vec2{x: p1.x - p0.x, y: p1.y - p0.y}
	v0 := dir0.perp().norm().mul(dist)
	pp0 := p0.add(v0)

	// t = 1
	dir2 := vec2{x: p2.x - p1.x, y: p2.y - p1.y}
	v2 := dir2.perp().norm().mul(dist)
	pp2 := p2.add(v2)

	// t = 0.5
	dir1 := vec2{x: p2.x - p0.x, y: p2.y - p0.y}
	v1 := dir1.perp().norm().mul(dist)
	mid := point{
		x: 0.25*p0.x + 0.5*p1.x + 0.25*p2.x,
		y: 0.25*p0.y + 0.5*p1.y + 0.25*p2.y,
	}.add(v1)
	// Calculate the control point P1 from B(0.5).
	pp1 := point{
		x: 2*mid.x - 0.5*(pp0.x+pp2.x),
		y: 2*mid.y - 0.5*(pp0.y+pp2.y),
	}

	if level > 5 {
		path.QuadTo(pp1.x, pp1.y, pp2.x, pp2.y)
		return
	}

	// If any of the points is not a regular float32, do not call this function recursively.
	if !isRegularF32(pp0.x) || !isRegularF32(pp0.y) || !isRegularF32(pp1.x) || !isRegularF32(pp1.y) || !isRegularF32(pp2.x) || !isRegularF32(pp2.y) {
		path.QuadTo(pp1.x, pp1.y, pp2.x, pp2.y)
		return
	}

	minAllowance := max(dist*63/64, 0)
	maxAllowance := dist * 65 / 64

	var needSplit bool
	for _, t := range []float32{0.25, 0.75} {
		gotP := point{
			x: (1-t)*(1-t)*pp0.x + 2*(1-t)*t*pp1.x + t*t*pp2.x,
			y: (1-t)*(1-t)*pp0.y + 2*(1-t)*t*pp1.y + t*t*pp2.y,
		}

		dir := vec2{
			x: (1-t)*(p1.x-p0.x) + t*(p2.x-p1.x),
			y: (1-t)*(p1.y-p0.y) + t*(p2.y-p1.y),
		}
		v := dir.perp().norm().mul(dist)
		p := point{
			x: (1-t)*(1-t)*p0.x + 2*(1-t)*t*p1.x + t*t*p2.x + v.x,
			y: (1-t)*(1-t)*p0.y + 2*(1-t)*t*p1.y + t*t*p2.y + v.y,
		}
		expectedP := p.add(v)

		if !arePointsInRange(gotP, expectedP, minAllowance, maxAllowance) {
			needSplit = true
			break
		}
	}

	if !needSplit {
		path.QuadTo(pp1.x, pp1.y, pp2.x, pp2.y)
		return
	}

	// Split a quadratic curve into two quadratic curves by De Casteljau algorithm.
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
	doAppendParalleledQuad(path, p0, p01, p012, dist, level+1)
	doAppendParalleledQuad(path, p012, p12, p2, dist, level+1)
}

func addJoint(strokePath *Path, subPath *subPath, opIndex int, reverse bool, options *StrokeOptions) {
	var p point
	var dir0, dir1 vec2
	if !reverse {
		nextOpIdx := opIndex + 1
		if nextOpIdx == len(subPath.ops) {
			if !subPath.closed {
				return
			}
			nextOpIdx = 0
		}
		p = subPath.endAtOp(opIndex)
		dir0 = subPath.endDir(opIndex).norm()
		dir1 = subPath.startDir(nextOpIdx).norm()
	} else {
		nextOpIdx := opIndex - 1
		if nextOpIdx == -1 {
			if !subPath.closed {
				return
			}
			nextOpIdx = len(subPath.ops) - 1
		}
		p = subPath.startAtOp(opIndex)
		dir0 = subPath.startDir(opIndex).inv().norm()
		dir1 = subPath.endDir(nextOpIdx).inv().norm()
	}

	if dir0 == dir1 {
		return
	}

	v1 := dir1.perp().mul(options.Width / 2)
	p1 := p.add(v1)

	// If the joint is an internal angle (< 180 degrees), the joint is not rendered. Just connect the two segments.
	// [vec2.cross] has a precision issue. Use a comparison instead.
	if dir0.x*dir1.y > dir0.y*dir1.x {
		strokePath.LineTo(p1.x, p1.y)
		return
	}

	v0 := dir0.perp().mul(options.Width / 2)
	p0 := p.add(v0)

	// Add a joint.
	switch options.LineJoin {
	case LineJoinMiter:
		theta := math.Acos(float64(dir0.x*(-dir1.x) + dir0.y*(-dir1.y)))
		exceed := float32(math.Abs(1/math.Sin(float64(theta/2)))) > options.MiterLimit
		if !exceed {
			cp := crossingPointForTwoLines(p0, p0.add(dir0), p1, p1.add(dir1))
			if isRegularF32(cp.x) && isRegularF32(cp.y) {
				strokePath.LineTo(cp.x, cp.y)
			}
		}
		strokePath.LineTo(p1.x, p1.y)
	case LineJoinBevel:
		strokePath.LineTo(p1.x, p1.y)
	case LineJoinRound:
		dir := vec2{
			x: dir0.x - dir1.x,
			y: dir0.y - dir1.y,
		}.norm()
		cp := p.add(dir.mul(options.Width / 2))
		cp0 := crossingPointForTwoLines(p0, p0.add(dir0), cp, cp.add(dir.perp()))
		cp1 := crossingPointForTwoLines(p1, p1.add(dir1), cp, cp.add(dir.perp()))
		if isRegularF32(cp.x) && isRegularF32(cp.y) && isRegularF32(cp0.x) && isRegularF32(cp0.y) && isRegularF32(cp1.x) && isRegularF32(cp1.y) {
			strokePath.ArcTo(cp0.x, cp0.y, cp.x, cp.y, options.Width/2)
			strokePath.ArcTo(cp1.x, cp1.y, p1.x, p1.y, options.Width/2)
		} else {
			strokePath.LineTo(p1.x, p1.y)
		}
	}
}
