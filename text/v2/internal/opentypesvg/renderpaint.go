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
	"errors"
	"math"
)

func prepareRenderPaint(p paint, transform matrix) (RenderPaint, bool, error) {
	var result RenderPaint
	if p.Kind == paintNone {
		return result, true, nil
	}
	if p.Kind == paintGradient && p.Gradient.Disabled {
		return result, true, nil
	}
	c := p.Color
	if p.Kind == paintGradient && p.Gradient.Solid {
		c = p.Gradient.stops[len(p.Gradient.stops)-1].Color
	}
	if p.Kind == paintSolid || p.Gradient.Solid {
		a := c.A * p.Opacity
		result.Color = [4]float32{float32(c.R * a), float32(c.G * a), float32(c.B * a), float32(a)}
		return result, a == 0, nil
	}
	g := p.Gradient
	if len(g.stops) > maxRenderStops {
		return result, false, errLimit
	}
	m, err := transform.mul(g.Transform)
	if err != nil {
		return result, false, err
	}
	det := m.A*m.D - m.B*m.C
	if det == 0 || !isFinite(det) {
		return result, false, errGeometry
	}

	if g.Radial {
		inv := matrix{
			A: m.D / det,
			B: -m.B / det,
			C: -m.C / det,
			D: m.A / det,
			E: (m.C*m.F - m.D*m.E) / det,
			F: (m.B*m.E - m.A*m.F) / det,
		}
		normalization := matrix{
			A: 1 / g.Radius,
			D: 1 / g.Radius,
			E: -g.Center.X / g.Radius,
			F: -g.Center.Y / g.Radius,
		}
		inv, err = normalization.mul(inv)
		if err != nil {
			return result, false, errLimit
		}
		for i, v := range inv.values() {
			if !isFinite(v) || math.Abs(v) > maxRenderCoordinate {
				return result, false, errLimit
			}
			result.Inverse[i] = float32(v)
		}
		result.Radius = 1
		result.Focus = [2]float32{float32((g.Focus.X - g.Center.X) / g.Radius), float32((g.Focus.Y - g.Center.Y) / g.Radius)}
	} else {
		// Only the inverse projected onto the gradient axis contributes to t.
		dx, dy := g.End.X-g.Start.X, g.End.Y-g.Start.Y
		length := math.Hypot(dx, dy)
		ux, uy := dx/length, dy/length
		a := (ux*m.D - uy*m.B) / det / length
		b := (uy*m.A - ux*m.C) / det / length
		c := -(a*m.E + b*m.F) - (ux*g.Start.X+uy*g.Start.Y)/length
		for i, v := range [3]float64{a, b, c} {
			if !isFinite(v) || math.Abs(v) > maxRenderCoordinate {
				return result, false, errLimit
			}
			result.Linear[i] = float32(v)
		}
	}
	result.Radial = g.Radial
	result.LinearRGB = g.Interpolation == "linearRGB"
	switch g.Spread {
	case "repeat":
		result.Spread = 1
	case "reflect":
		result.Spread = 2
	}
	for _, stop := range g.stops {
		c := stop.Color
		if result.LinearRGB {
			// SVG linearRGB gradients interpolate decoded sRGB stops. The shader
			// encodes the result back to sRGB because Ebitengine does not convert it.
			c.R = srgbToLinear(c.R)
			c.G = srgbToLinear(c.G)
			c.B = srgbToLinear(c.B)
		}
		result.Offsets = append(result.Offsets, float32(stop.Offset))
		result.Stops = append(result.Stops, [4]float32{float32(c.R), float32(c.G), float32(c.B), float32(c.A * p.Opacity)})
	}
	return result, false, nil
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func (r *styleResolver) preparePaint(s style, stroke bool, p paint, transform matrix) (RenderPaint, bool, error) {
	prepared, disabled, err := prepareRenderPaint(p, transform)
	if !errors.Is(err, errGeometry) || p.Kind != paintGradient {
		return prepared, disabled, err
	}
	property := propFill
	if stroke {
		property = propStroke
	}
	value, parseErr := parsePaint(s.value(property), s.Color, r.colors, true)
	if parseErr != nil {
		return RenderPaint{}, false, parseErr
	}
	if value.fallback == "" {
		return RenderPaint{}, false, err
	}
	fallback, parseErr := parsePaint(value.fallback, s.Color, r.colors, true)
	if parseErr != nil {
		return RenderPaint{}, false, parseErr
	}
	replacement := paint{
		Kind:    paintSolid,
		Color:   fallback.color,
		Opacity: p.Opacity,
	}
	if fallback.none {
		replacement.Kind = paintNone
	}
	return prepareRenderPaint(replacement, transform)
}
