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

type point struct {
	X, Y float64
}
type segmentKind uint8

const (
	moveTo segmentKind = iota
	lineTo
	quadTo
	cubicTo
	arcTo
	closePath
)

// segment is an absolute path command; controls are used by quadratic and cubic curves.
// Arcs retain endpoint parameters, with rotation in degrees and nonnegative radii.
type segment struct {
	Kind     segmentKind
	To       point
	Control1 point
	Control2 point
	Radius   point
	Rotation float64
	LargeArc bool
	Sweep    bool
}

// path is a sequence of local-coordinate curve segments.
type path struct {
	segments []segment
}

func (p path) segmentCount() int {
	return len(p.segments)
}

func (p path) segment(index int) segment {
	return p.segments[index]
}

// parsePath parses SVG path data, returning the complete command prefix on syntax errors.
func parsePath(text string) (path, error) {
	if len(text) > maxBytes {
		return path{}, errLimit
	}
	var result path
	s := numberScanner{
		text: text,
	}
	var current, start, control point
	var command, previous byte
	for commands := 0; ; commands++ {
		s.space()
		if s.pos == len(text) {
			return result, nil
		}
		var explicit bool
		if c := text[s.pos]; isAlpha(c) {
			command = c
			s.pos++
			explicit = true
		}
		upper := command &^ 32
		if len(result.segments) == 0 && upper != 'M' {
			return result, errGeometry
		}
		if commands >= maxSegments {
			return result, errLimit
		}
		if upper == 'Z' {
			if !explicit {
				return result, errGeometry
			}
			result.segments = append(result.segments, segment{
				Kind: closePath,
				To:   start,
			})
			current = start
			previous = upper
			command = 0
			continue
		}
		var count int
		switch upper {
		case 'M', 'L', 'T':
			count = 2
		case 'H', 'V':
			count = 1
		case 'Q', 'S':
			count = 4
		case 'C':
			count = 6
		case 'A':
			count = 7
		default:
			return result, errGeometry
		}
		var args [7]float64
		for i := range count {
			if i == 0 && explicit {
				s.space()
			} else if !s.separator(upper == 'A' && i == 3) {
				return result, errGeometry
			}
			if upper == 'A' && (i == 3 || i == 4) {
				if s.pos == len(text) || (text[s.pos] != '0' && text[s.pos] != '1') {
					return result, errGeometry
				}
				args[i] = float64(text[s.pos] - '0')
				s.pos++
			} else {
				v, err := s.number()
				if err != nil {
					return result, err
				}
				args[i] = v
			}
		}
		argumentPoint := func(i int) point {
			p := point{X: args[i], Y: args[i+1]}
			if command >= 'a' {
				p.X += current.X
				p.Y += current.Y
			}
			return p
		}
		seg := segment{
			Kind: lineTo,
		}
		switch upper {
		case 'M':
			seg.Kind = moveTo
			seg.To = argumentPoint(0)
		case 'L':
			seg.To = argumentPoint(0)
		case 'H':
			seg.To = current
			seg.To.X = args[0]
			if command >= 'a' {
				seg.To.X += current.X
			}
		case 'V':
			seg.To = current
			seg.To.Y = args[0]
			if command >= 'a' {
				seg.To.Y += current.Y
			}
		case 'C':
			seg.Kind = cubicTo
			seg.Control1 = argumentPoint(0)
			seg.Control2 = argumentPoint(2)
			seg.To = argumentPoint(4)
		case 'S':
			seg.Kind = cubicTo
			seg.Control1 = current
			if previous == 'C' || previous == 'S' {
				seg.Control1 = point{X: current.X + (current.X - control.X), Y: current.Y + (current.Y - control.Y)}
			}
			seg.Control2 = argumentPoint(0)
			seg.To = argumentPoint(2)
		case 'Q':
			seg.Kind = quadTo
			seg.Control1 = argumentPoint(0)
			seg.To = argumentPoint(2)
		case 'T':
			seg.Kind = quadTo
			seg.Control1 = current
			if previous == 'Q' || previous == 'T' {
				seg.Control1 = point{X: current.X + (current.X - control.X), Y: current.Y + (current.Y - control.Y)}
			}
			seg.To = argumentPoint(0)
		case 'A':
			seg.Kind = arcTo
			seg.To = argumentPoint(5)
			seg.Radius = point{X: math.Abs(args[0]), Y: math.Abs(args[1])}
			seg.Rotation = math.Mod(args[2], 360)
			seg.LargeArc = args[3] == 1
			seg.Sweep = args[4] == 1
			if seg.Radius.X == 0 || seg.Radius.Y == 0 {
				seg = segment{
					Kind: lineTo,
					To:   seg.To,
				}
			} else if seg.To != current {
				radii, err := arcRadii(current, seg)
				if err != nil {
					return result, err
				}
				seg.Radius = radii
			}
		}
		if !areValidNumbers(seg.To.X, seg.To.Y, seg.Control1.X, seg.Control1.Y, seg.Control2.X, seg.Control2.Y) {
			return result, errGeometry
		}
		previous = upper
		if seg.Kind == cubicTo {
			control = seg.Control2
		} else {
			control = seg.Control1
		}
		if upper == 'M' {
			start = seg.To
			if command == 'm' {
				command = 'l'
			} else {
				command = 'L'
			}
		}
		// Coincident arc endpoints contribute no segment, but still reset reflection.
		if upper != 'A' || seg.To != current {
			result.segments = append(result.segments, seg)
		}
		current = seg.To
	}
}

// arcRadii returns the corrected radii for segment starting at from.
// The segment's radii must be positive.
func arcRadii(from point, segment segment) (point, error) {
	// SVG 1.1 F.6.6 scales both radii until the ellipse reaches both endpoints.
	sin, cos := math.Sincos(segment.Rotation * math.Pi / 180)
	dx, dy := from.X/2-segment.To.X/2, from.Y/2-segment.To.Y/2
	x, y := cos*dx+sin*dy, -sin*dx+cos*dy
	scale := math.Hypot(x/segment.Radius.X, y/segment.Radius.Y)
	radii := segment.Radius
	if scale > 1 {
		radii.X *= scale
		radii.Y *= scale
	}
	if !areValidNumbers(scale, radii.X, radii.Y) {
		return point{}, errGeometry
	}
	return radii, nil
}
