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

type vec2 struct {
	x, y float32
}

func (v vec2) perp() vec2 {
	return vec2{x: -v.y, y: v.x}
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

func (v vec2) cross(u vec2) float32 {
	return v.x*u.y - u.x*v.y
}

func (v vec2) mul(s float32) vec2 {
	return vec2{x: s * v.x, y: s * v.y}
}

func (p point) add(v vec2) point {
	return point{x: p.x + v.x, y: p.y + v.y}
}

// AddPathStrokeOptions is options for [Path.AddPathStroke].
type AddPathStrokeOptions struct {
	// StrokeOptions is options for the stroke.
	StrokeOptions

	// GeoM is a geometry matrix to apply to the path.
	//
	// The default (zero) value is an identity matrix.
	GeoM ebiten.GeoM
}

// AddPathStroke adds a stroke path to the path p.
//
// The added stroke path must be rendered with FileRuleNonZero.
func (p *Path) AddPathStroke(src *Path, options *AddPathStrokeOptions) {
	if options == nil {
		return
	}
	if options.Width <= 0 {
		return
	}

	origN := len(p.subPaths)
	srcN := len(src.subPaths)
	for _, subPath := range src.subPaths[:srcN] {
		if len(subPath.ops) == 0 {
			continue
		}
		appendParalleledPathFromSubPath(p, &subPath, &options.StrokeOptions)
		if subPath.closed {
			p.Close()
		} else {
			// TODO: Add a cap
		}
		appendParalleledPathFromSubPathReversed(p, &subPath, &options.StrokeOptions, !subPath.closed)
		if !subPath.closed {
			// TODO: Add a cap
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

func appendParalleledPathFromSubPath(strokePath *Path, subPath *subPath, options *StrokeOptions) {
	cur := subPath.start
	var subPathInited bool

	for _, op := range subPath.ops {
		switch op.typ {
		case opTypeLineTo:
			if cur == op.p1 {
				continue
			}

			appendParalleledLine(strokePath, cur, op.p1, options.Width/2, !subPathInited)
			subPathInited = true
			cur = op.p1
		case opTypeQuadTo:
			if cur == op.p1 && cur == op.p2 {
				continue
			}

			appendParalleledQuad(strokePath, cur, op.p1, op.p2, options.Width/2, !subPathInited)
			cur = op.p2
			subPathInited = true
		}
	}
}

func appendParalleledPathFromSubPathReversed(strokePath *Path, subPath *subPath, options *StrokeOptions, subPathInited bool) {
	for i := len(subPath.ops) - 1; i >= 0; i-- {
		op := subPath.ops[i]
		var nextP point
		if i > 0 {
			op := subPath.ops[i-1]
			switch op.typ {
			case opTypeLineTo:
				nextP = op.p1
			case opTypeQuadTo:
				nextP = op.p2
			}
		} else {
			nextP = subPath.start
		}
		switch op.typ {
		case opTypeLineTo:
			if nextP == op.p1 {
				continue
			}
			appendParalleledLine(strokePath, op.p1, nextP, options.Width/2, !subPathInited)
			subPathInited = true
		case opTypeQuadTo:
			if nextP == op.p1 && nextP == op.p2 {
				continue
			}
			appendParalleledQuad(strokePath, op.p2, op.p1, nextP, options.Width/2, !subPathInited)
			subPathInited = true
		}
	}
}

func appendParalleledLine(path *Path, p0, p1 point, dist float32, init bool) {
	if p0 == p1 {
		panic("not reached")
	}

	tan := vec2{x: p1.x - p0.x, y: p1.y - p0.y}
	v := tan.perp().norm().mul(dist)
	pp0 := p0.add(v)
	if init {
		path.MoveTo(pp0.x, pp0.y)
	} else {
		path.LineTo(pp0.x, pp0.y) // TODO: Add a curve based on the line join option.
	}
	pp1 := p1.add(v)
	path.LineTo(pp1.x, pp1.y)
}

func appendParalleledLineForQuadIfNeeded(path *Path, p0, p1, p2 point, dist float32, init bool) bool {
	if p0 == p1 && p0 == p2 {
		panic("not reached")
	}
	if p0 != p1 && p0 == p2 {
		appendParalleledLine(path, p0, p1, dist, init)
		return true
	}
	if p0 == p1 && p0 != p2 {
		appendParalleledLine(path, p0, p2, dist, init)
		return true
	}
	if (p1.x-p0.x)*(p2.y-p0.y)-(p2.x-p0.x)*(p1.y-p0.y) == 0 {
		appendParalleledLine(path, p0, p2, dist, init)
		return true
	}
	return false
}

func appendParalleledQuad(path *Path, p0, p1, p2 point, dist float32, init bool) {
	if appendParalleledLineForQuadIfNeeded(path, p0, p1, p2, dist, init) {
		return
	}

	// B(t) = (1-t)*(1-t)*p0 + 2*(1-t)*t*p1 + t*t*p2
	// B'(t) = 2*(1-t)*(p1-p0) + 2*t*(p2-p1)
	// B'(0) = 2*(p1-p0)
	// B'(0.5) = p2-p0
	// B'(1) = 2*(p2-p1)
	// B''(t) = 2*(p0 - 2*p1 + p2)

	// t = 0
	tan0 := vec2{x: p1.x - p0.x, y: p1.y - p0.y} // The size doesn't matter.
	v0 := tan0.perp().norm().mul(dist)
	pp0 := p0.add(v0)
	if init {
		path.MoveTo(pp0.x, pp0.y)
	} else {
		path.LineTo(pp0.x, pp0.y) // TODO: Add a curve based on the line join option.
	}

	appendParalleledQuad2(path, p0, p1, p2, dist, 0)
}

func appendParalleledQuad2(path *Path, p0, p1, p2 point, dist float32, level int) {
	if p0 == p1 && p0 == p2 {
		return
	}
	if appendParalleledLineForQuadIfNeeded(path, p0, p1, p2, dist, false) {
		return
	}

	// t = 0
	tan0 := vec2{x: p1.x - p0.x, y: p1.y - p0.y} // The size doesn't matter.
	v0 := tan0.perp().norm().mul(dist)
	pp0 := p0.add(v0)

	// t = 0.5
	tan1 := vec2{x: p2.x - p0.x, y: p2.y - p0.y}
	v1 := tan1.perp().norm().mul(dist)
	pp1 := p1.add(v1)

	// t = 1
	tan2 := vec2{x: p2.x - p1.x, y: p2.y - p1.y} // The size doesn't matter.
	v2 := tan2.perp().norm().mul(dist)
	pp2 := p2.add(v2)

	if level > 5 {
		path.QuadTo(pp1.x, pp1.y, pp2.x, pp2.y)
		return
	}

	// If any of the points is not a regular float32, do not call this function recursively.
	if !isRegularF32(pp0.x) || !isRegularF32(pp0.y) || !isRegularF32(pp1.x) || !isRegularF32(pp1.y) || !isRegularF32(pp2.x) || !isRegularF32(pp2.y) {
		path.QuadTo(pp1.x, pp1.y, pp2.x, pp2.y)
		return
	}

	var needSplit bool
	for _, t := range []float32{0.25, 0.75} {
		gotP := point{
			x: (1-t)*(1-t)*pp0.x + 2*(1-t)*t*pp1.x + t*t*pp2.x,
			y: (1-t)*(1-t)*pp0.y + 2*(1-t)*t*pp1.y + t*t*pp2.y,
		}

		// The size doesn't matter.
		tan := vec2{
			x: (1-t)*(p1.x-p0.x) + t*(p2.x-p1.x),
			y: (1-t)*(p1.y-p0.y) + t*(p2.y-p1.y),
		}
		v := tan.perp().norm().mul(dist)
		p := point{
			x: (1-t)*(1-t)*p0.x + 2*(1-t)*t*p1.x + t*t*p2.x + v.x,
			y: (1-t)*(1-t)*p0.y + 2*(1-t)*t*p1.y + t*t*p2.y + v.y,
		}
		expectedP := p.add(v)

		if !arePointsCloseEnough(gotP, expectedP, max(dist-1.0/16.0, 0), dist+1.0/16.0) {
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
	appendParalleledQuad2(path, p0, p01, p012, dist, level+1)
	appendParalleledQuad2(path, p012, p12, p2, dist, level+1)
}
