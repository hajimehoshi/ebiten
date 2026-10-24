// Copyright 2026 The Ebitengine Authors
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

package opentypesvg

import "math"

type ellipseArc struct {
	center, u, v point
	start, sweep float64
}

func (a ellipseArc) at(t float64) point {
	s, c := math.Sincos(t)
	return point{X: a.center.X + a.u.X*c + a.v.X*s, Y: a.center.Y + a.u.Y*c + a.v.Y*s}
}

func endpointArc(from point, s segment) (ellipseArc, error) {
	radii, err := arcRadii(from, s)
	if err != nil {
		return ellipseArc{}, err
	}
	sn, cs := math.Sincos(s.Rotation * math.Pi / 180)
	dx, dy := (from.X-s.To.X)/2, (from.Y-s.To.Y)/2
	x, y := (cs*dx+sn*dy)/radii.X, (-sn*dx+cs*dy)/radii.Y
	length := x*x + y*y
	if length == 0 || !isFinite(length) {
		return ellipseArc{}, errGeometry
	}
	k := math.Sqrt(max(0, (1-length)/length))
	if s.LargeArc == s.Sweep {
		k = -k
	}
	cx, cy := k*y, -k*x
	a := ellipseArc{
		center: point{X: (from.X+s.To.X)/2 + cs*cx*radii.X - sn*cy*radii.Y, Y: (from.Y+s.To.Y)/2 + sn*cx*radii.X + cs*cy*radii.Y},
		u:      point{X: cs * radii.X, Y: sn * radii.X},
		v:      point{X: -sn * radii.Y, Y: cs * radii.Y},
		start:  math.Atan2(y-cy, x-cx),
	}
	end := math.Atan2(-y-cy, -x-cx)
	a.sweep = math.Mod(end-a.start+2*math.Pi, 2*math.Pi)
	if !s.Sweep {
		a.sweep -= 2 * math.Pi
	}
	if !areValidNumbers(a.center.X, a.center.Y, a.sweep) {
		return ellipseArc{}, errGeometry
	}
	return a, nil
}

func (a ellipseArc) contains(t float64) bool {
	if a.sweep >= 0 {
		return math.Mod(t-a.start+4*math.Pi, 2*math.Pi) <= a.sweep+1e-12
	}
	return math.Mod(a.start-t+4*math.Pi, 2*math.Pi) <= -a.sweep+1e-12
}

func mixPoint(a, b point, t float64) point {
	if t == 0 {
		return a
	}
	if t == 1 {
		return b
	}
	return point{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t}
}

func geometryBounds(p path, m matrix) (viewBox, bool, error) {
	// Bounding-box paints and clips require geometric extrema in user space.
	// Control hulls, stroke expansion, and clipped ink extents are not these bounds.
	var lo, hi, current point
	var valid bool
	add := func(p point) error {
		if !areValidNumbers(p.X, p.Y) {
			return errGeometry
		}
		if !valid {
			lo, hi, valid = p, p, true
		} else {
			lo.X = min(lo.X, p.X)
			lo.Y = min(lo.Y, p.Y)
			hi.X = max(hi.X, p.X)
			hi.Y = max(hi.Y, p.Y)
		}
		return nil
	}
	for _, s := range p.segments {
		if s.Kind == moveTo {
			current = s.To
			continue
		}
		start, err := m.apply(current)
		if err != nil {
			return viewBox{}, false, err
		}
		end, err := m.apply(s.To)
		if err != nil {
			return viewBox{}, false, err
		}
		if err := add(start); err != nil {
			return viewBox{}, false, err
		}
		if err := add(end); err != nil {
			return viewBox{}, false, err
		}
		switch s.Kind {
		case quadTo, cubicTo:
			c1, err := m.apply(s.Control1)
			if err != nil {
				return viewBox{}, false, err
			}
			c2 := c1
			if s.Kind == cubicTo {
				c2, err = m.apply(s.Control2)
				if err != nil {
					return viewBox{}, false, err
				}
			}
			for axis := range 2 {
				a, b, c, d := start.X, c1.X, c2.X, end.X
				if axis == 1 {
					a, b, c, d = start.Y, c1.Y, c2.Y, end.Y
				}
				var roots []float64
				if s.Kind == quadTo {
					if den := a - 2*b + d; den != 0 {
						roots = append(roots, (a-b)/den)
					}
				} else {
					aa, bb, cc := -a+3*b-3*c+d, 2*(a-2*b+c), b-a
					if aa == 0 {
						if bb != 0 {
							roots = append(roots, -cc/bb)
						}
					} else if disc := bb*bb - 4*aa*cc; disc >= 0 {
						q := -0.5 * (bb + math.Copysign(math.Sqrt(disc), bb))
						roots = append(roots, q/aa)
						if q != 0 {
							roots = append(roots, cc/q)
						}
					}
				}
				for _, t := range roots {
					if t <= 0 || t >= 1 {
						continue
					}
					q := mixPoint(mixPoint(start, c1, t), mixPoint(c1, end, t), t)
					if s.Kind == cubicTo {
						q = mixPoint(mixPoint(mixPoint(start, c1, t), mixPoint(c1, c2, t), t), mixPoint(mixPoint(c1, c2, t), mixPoint(c2, end, t), t), t)
					}
					if err := add(q); err != nil {
						return viewBox{}, false, err
					}
				}
			}
		case arcTo:
			if current != s.To && s.Radius.X > 0 && s.Radius.Y > 0 {
				a, err := endpointArc(current, s)
				if err != nil {
					return viewBox{}, false, err
				}
				a.center, err = m.apply(a.center)
				if err != nil {
					return viewBox{}, false, err
				}
				a.u = point{X: m.A*a.u.X + m.C*a.u.Y, Y: m.B*a.u.X + m.D*a.u.Y}
				a.v = point{X: m.A*a.v.X + m.C*a.v.Y, Y: m.B*a.v.X + m.D*a.v.Y}
				for _, t := range []float64{math.Atan2(a.v.X, a.u.X), math.Atan2(a.v.X, a.u.X) + math.Pi, math.Atan2(a.v.Y, a.u.Y), math.Atan2(a.v.Y, a.u.Y) + math.Pi} {
					if a.contains(t) {
						if err := add(a.at(t)); err != nil {
							return viewBox{}, false, err
						}
					}
				}
			}
		}
		current = s.To
	}
	return viewBox{
		X:      lo.X,
		Y:      lo.Y,
		Width:  hi.X - lo.X,
		Height: hi.Y - lo.Y,
	}, valid, nil
}

func (b *renderBuilder) convertPath(p path, scale float64) ([]RenderCommand, error) {
	// Keep original curves until the sample-space scale is known. Conversion
	// tolerances below are in sample pixels; only dashing flattens the resulting
	// centerline. Vector's subsequent stroke expansion has its own approximation.
	var result []RenderCommand
	add := func(c RenderCommand) error {
		if err := b.charge(1); err != nil {
			return err
		}
		if !areValidNumbers(c.X, c.Y, c.CX, c.CY) || max(math.Abs(c.X), math.Abs(c.Y), math.Abs(c.CX), math.Abs(c.CY)) > maxRenderCoordinate {
			return errLimit
		}
		result = append(result, c)
		return nil
	}
	scaled := func(p point) point { return point{X: p.X * scale, Y: p.Y * scale} }
	var cubic func(point, point, point, point, int) error
	cubic = func(a, c, d, e point, depth int) error {
		// Degree reduction's cubic residual is bounded by its control hull.
		q := point{X: (3*c.X + 3*d.X - a.X - e.X) / 4, Y: (3*c.Y + 3*d.Y - a.Y - e.Y) / 4}
		c1, c2 := mixPoint(a, q, 2.0/3), mixPoint(e, q, 2.0/3)
		if max(math.Hypot(c1.X-c.X, c1.Y-c.Y), math.Hypot(c2.X-d.X, c2.Y-d.Y)) <= 0.025 {
			return add(RenderCommand{
				Kind: 'Q',
				X:    e.X,
				Y:    e.Y,
				CX:   q.X,
				CY:   q.Y,
			})
		}
		if depth >= 24 {
			return errLimit
		}
		ac, cd, de := mixPoint(a, c, .5), mixPoint(c, d, .5), mixPoint(d, e, .5)
		left, right := mixPoint(ac, cd, .5), mixPoint(cd, de, .5)
		mid := mixPoint(left, right, .5)
		if err := cubic(a, ac, left, mid, depth+1); err != nil {
			return err
		}
		return cubic(mid, right, de, e, depth+1)
	}
	var current point
	for _, s := range p.segments {
		to := scaled(s.To)
		var err error
		switch s.Kind {
		case moveTo:
			err = add(RenderCommand{
				Kind: 'M',
				X:    to.X,
				Y:    to.Y,
			})
		case lineTo:
			err = add(RenderCommand{
				Kind: 'L',
				X:    to.X,
				Y:    to.Y,
			})
		case closePath:
			err = add(RenderCommand{
				Kind: 'Z',
				X:    to.X,
				Y:    to.Y,
			})
		case quadTo:
			c := scaled(s.Control1)
			err = add(RenderCommand{
				Kind: 'Q',
				X:    to.X,
				Y:    to.Y,
				CX:   c.X,
				CY:   c.Y,
			})
		case cubicTo:
			err = cubic(scaled(current), scaled(s.Control1), scaled(s.Control2), to, 0)
		case arcTo:
			if current == s.To {
				break
			}
			if s.Radius.X == 0 || s.Radius.Y == 0 {
				err = add(RenderCommand{
					Kind: 'L',
					X:    to.X,
					Y:    to.Y,
				})
				break
			}
			var a ellipseArc
			a, err = endpointArc(current, s)
			if err != nil {
				break
			}
			// Cubic ellipse approximation error decreases with the sixth power of the angle.
			radius := max(math.Hypot(a.u.X, a.u.Y), math.Hypot(a.v.X, a.v.Y)) * scale
			angle := min(math.Pi/2, math.Pow(0.01/max(radius, 0.01), 1.0/6))
			count := math.Ceil(math.Abs(a.sweep) / angle)
			if !isFinite(count) || count > float64(maxRenderCommands-b.commands) {
				return nil, errLimit
			}
			step := a.sweep / count
			for i := 0; i < int(count); i++ {
				t0, t1 := a.start+float64(i)*step, a.start+float64(i+1)*step
				p0, p1 := a.at(t0), a.at(t1)
				k := 4.0 / 3 * math.Tan(step/4)
				c0 := point{X: p0.X + k*(-a.u.X*math.Sin(t0)+a.v.X*math.Cos(t0)), Y: p0.Y + k*(-a.u.Y*math.Sin(t0)+a.v.Y*math.Cos(t0))}
				c1 := point{X: p1.X - k*(-a.u.X*math.Sin(t1)+a.v.X*math.Cos(t1)), Y: p1.Y - k*(-a.u.Y*math.Sin(t1)+a.v.Y*math.Cos(t1))}
				if i == 0 {
					p0 = current
				}
				if i == int(count)-1 {
					p1 = s.To
				}
				if err = cubic(scaled(p0), scaled(c0), scaled(c1), scaled(p1), 0); err != nil {
					break
				}
			}
		}
		if err != nil {
			return nil, err
		}
		current = s.To
	}
	return result, nil
}

// dash splits a final-scale centerline before stroke expansion.
func (b *renderBuilder) dash(commands []RenderCommand, s strokeStyle, scale float64) ([]RenderCommand, error) {
	var contours [][]point
	var closed []bool
	var current point
	var flatten func(point, point, point, int) error
	appendPoint := func(p point) error {
		if err := b.charge(1); err != nil {
			return err
		}
		contours[len(contours)-1] = append(contours[len(contours)-1], p)
		return nil
	}
	flatten = func(a, c, d point, depth int) error {
		mid := mixPoint(a, d, .5)
		if math.Hypot(c.X-mid.X, c.Y-mid.Y) <= .025 {
			return appendPoint(d)
		}
		if depth >= 24 {
			return errLimit
		}
		ac, cd := mixPoint(a, c, .5), mixPoint(c, d, .5)
		m := mixPoint(ac, cd, .5)
		if err := flatten(a, ac, m, depth+1); err != nil {
			return err
		}
		return flatten(m, cd, d, depth+1)
	}
	for _, c := range commands {
		if c.Kind != 'M' && len(contours) > 0 && closed[len(closed)-1] {
			contours = append(contours, []point{current})
			closed = append(closed, false)
		}
		p := point{X: c.X, Y: c.Y}
		switch c.Kind {
		case 'M':
			contours = append(contours, []point{p})
			closed = append(closed, false)
		case 'L', 'Z':
			if err := appendPoint(p); err != nil {
				return nil, err
			}
			if c.Kind == 'Z' {
				closed[len(closed)-1] = true
			}
		case 'Q':
			if err := flatten(current, point{X: c.CX, Y: c.CY}, p, 0); err != nil {
				return nil, err
			}
		}
		current = p
	}
	var total float64
	for _, v := range s.Dash {
		total += v * scale
	}
	if !isFinite(total) || total <= 0 {
		return nil, errGeometry
	}
	offset := math.Mod(s.Offset*scale, total)
	if offset < 0 {
		offset += total
	}
	var result []RenderCommand
	dot := func(p, direction point) error {
		if err := b.charge(1); err != nil {
			return err
		}
		result = append(result, RenderCommand{
			Kind: 'P',
			X:    p.X,
			Y:    p.Y,
			DX:   direction.X,
			DY:   direction.Y,
		})
		return nil
	}
	for ci, points := range contours {
		var index int
		remaining := s.Dash[0] * scale
		for offsetLeft := offset; offsetLeft > 0; {
			if remaining > offsetLeft {
				remaining -= offsetLeft
				break
			}
			offsetLeft -= remaining
			index = (index + 1) % len(s.Dash)
			remaining = s.Dash[index] * scale
		}
		var runs [][]point
		var run []point
		finish := func() {
			if len(run) > 0 {
				runs = append(runs, run)
				run = nil
			}
		}
		var direction point
		for i := 1; i < len(points); i++ {
			from, to := points[i-1], points[i]
			length := math.Hypot(to.X-from.X, to.Y-from.Y)
			if length == 0 {
				if index%2 == 0 {
					run = append(run, from, to)
				}
				continue
			}
			direction = point{X: (to.X - from.X) / length, Y: (to.Y - from.Y) / length}
			var walked float64
			for walked < length {
				if err := b.charge(1); err != nil {
					return nil, err
				}
				if remaining == 0 {
					if index%2 == 0 {
						p := mixPoint(from, to, walked/length)
						if err := dot(p, direction); err != nil {
							return nil, err
						}
					}
					index = (index + 1) % len(s.Dash)
					remaining = s.Dash[index] * scale
					continue
				}
				take := min(length-walked, remaining)
				if walked+take == walked {
					return nil, errLimit
				}
				if index%2 == 0 {
					if len(run) == 0 {
						run = append(run, mixPoint(from, to, walked/length))
					}
					run = append(run, mixPoint(from, to, (walked+take)/length))
				}
				walked += take
				remaining -= take
				if remaining <= 0 {
					if index%2 == 0 {
						finish()
					}
					index = (index + 1) % len(s.Dash)
					remaining = s.Dash[index] * scale
				}
			}
		}
		if remaining == 0 && index%2 == 0 && len(points) > 0 {
			p := points[len(points)-1]
			if err := dot(p, direction); err != nil {
				return nil, err
			}
		}
		finish()
		if closed[ci] && len(runs) > 1 && runs[0][0] == points[0] && runs[len(runs)-1][len(runs[len(runs)-1])-1] == points[0] {
			last := runs[len(runs)-1]
			runs[0] = append(last, runs[0][1:]...)
			runs = runs[:len(runs)-1]
		}
		for _, run := range runs {
			for i, p := range run {
				if err := b.charge(1); err != nil {
					return nil, err
				}
				kind := byte('L')
				if i == 0 {
					kind = 'M'
				}
				if closed[ci] && len(run) > 2 && i == len(run)-1 && p == run[0] {
					kind = 'Z'
				}
				result = append(result, RenderCommand{
					Kind: kind,
					X:    p.X,
					Y:    p.Y,
				})
			}
		}
	}
	return result, nil
}
