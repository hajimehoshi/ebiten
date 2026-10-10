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

import (
	"math"
	"strings"
)

// matrix represents SVG's matrix(a,b,c,d,e,f), mapping
// (x,y) to (A*x+C*y+E, B*x+D*y+F).
type matrix struct {
	A, B, C, D, E, F float64
}

func identity() matrix {
	return matrix{
		A: 1,
		D: 1,
	}
}

// apply maps a point, rejecting nonfinite inputs and results.
func (m matrix) apply(p point) (point, error) {
	if !m.isValid() || !areValidNumbers(p.X, p.Y) {
		return point{}, errGeometry
	}
	q := point{X: m.A*p.X + m.C*p.Y + m.E, Y: m.B*p.X + m.D*p.Y + m.F}
	if !areValidNumbers(q.X, q.Y) {
		return point{}, errGeometry
	}
	return q, nil
}

func (m matrix) isValid() bool {
	return areValidNumbers(m.A, m.B, m.C, m.D, m.E, m.F)
}

// mul composes m with n, applying n to points first.
func (m matrix) mul(n matrix) (matrix, error) {
	r := matrix{
		A: m.A*n.A + m.C*n.B,
		B: m.B*n.A + m.D*n.B,
		C: m.A*n.C + m.C*n.D,
		D: m.B*n.C + m.D*n.D,
		E: m.A*n.E + m.C*n.F + m.E,
		F: m.B*n.E + m.D*n.F + m.F,
	}
	if !m.isValid() || !n.isValid() || !r.isValid() {
		return matrix{}, errGeometry
	}
	return r, nil
}

// parseTransform parses an SVG transform attribute.
func parseTransform(text string) (matrix, error) {
	if len(text) > maxBytes {
		return matrix{}, errLimit
	}
	s := numberScanner{
		text: text,
	}
	s.space()
	result := identity()
	for count := 0; s.pos < len(text); count++ {
		if count >= maxSegments {
			return matrix{}, errLimit
		}
		start := s.pos
		for s.pos < len(text) && isAlpha(text[s.pos]) {
			s.pos++
		}
		name := text[start:s.pos]
		s.space()
		if s.pos == len(text) || text[s.pos] != '(' {
			return matrix{}, errGeometry
		}
		s.pos++
		s.space()
		var args [6]float64
		var n int
		for {
			if n == len(args) {
				return matrix{}, errGeometry
			}
			v, err := s.number()
			if err != nil {
				return matrix{}, err
			}
			args[n] = v
			n++
			s.space()
			if s.pos < len(text) && text[s.pos] == ')' {
				s.pos++
				break
			}
			s.separator(false)
		}
		m := identity()
		switch name {
		case "matrix":
			if n != 6 {
				return matrix{}, errGeometry
			}
			m = matrix{
				A: args[0],
				B: args[1],
				C: args[2],
				D: args[3],
				E: args[4],
				F: args[5],
			}
		case "translate":
			if n < 1 || n > 2 {
				return matrix{}, errGeometry
			}
			m.E = args[0]
			m.F = args[1]
		case "scale":
			if n < 1 || n > 2 {
				return matrix{}, errGeometry
			}
			m.A = args[0]
			m.D = args[0]
			if n == 2 {
				m.D = args[1]
			}
		case "rotate":
			if n != 1 && n != 3 {
				return matrix{}, errGeometry
			}
			sin, cos := math.Sincos(math.Mod(args[0], 360) * math.Pi / 180)
			m.A = cos
			m.B = sin
			m.C = -sin
			m.D = cos
			if n == 3 {
				m.E = args[1] - cos*args[1] + sin*args[2]
				m.F = args[2] - sin*args[1] - cos*args[2]
			}
		case "skewX", "skewY":
			if n != 1 || math.Mod(math.Abs(args[0]), 180) == 90 {
				return matrix{}, errGeometry
			}
			v := math.Tan(math.Mod(args[0], 180) * math.Pi / 180)
			if name == "skewX" {
				m.C = v
			} else {
				m.B = v
			}
		default:
			return matrix{}, errGeometry
		}
		var err error
		result, err = result.mul(m)
		if err != nil {
			return matrix{}, err
		}
		if s.pos == len(text) {
			break
		}
		// Adjacent functions are accepted for compatibility. Numeric argument
		// boundaries follow CSS Transforms' SVG transform-list syntax.
		// https://www.w3.org/TR/css-transforms-1/#svg-syntax
		s.separator(false)
		// Transform definitions allow one or more comma-wsp productions.
		for s.pos < len(text) && text[s.pos] == ',' {
			s.pos++
			s.space()
		}
		if s.pos == len(text) && strings.HasSuffix(strings.TrimRight(text, xmlWhitespace), ",") {
			return matrix{}, errGeometry
		}
	}
	return result, nil
}

// localTransform returns only this element's transform and use-position translation.
func (e *element) localTransform(context lengthContext) (matrix, error) {
	text, _ := e.attribute("", "transform")
	m, err := parseTransform(text)
	if err != nil {
		return matrix{}, err
	}
	if e.name.Local != "use" {
		return m, nil
	}
	x, err := e.length("x", context, horizontal)
	if err != nil {
		return matrix{}, err
	}
	y, err := e.length("y", context, vertical)
	if err != nil {
		return matrix{}, err
	}
	return m.mul(matrix{
		A: 1,
		D: 1,
		E: x,
		F: y,
	})
}
