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

const (
	gradientUnits = iota
	gradientTransform
	gradientSpread
	gradientX1
	gradientY1
	gradientX2
	gradientY2
	gradientCX
	gradientCY
	gradientR
	gradientFX
	gradientFY
	gradientFR
	gradientAttributeCount
)

var gradientAttributes = [gradientAttributeCount]string{
	"gradientUnits", "gradientTransform", "spreadMethod", "x1", "y1", "x2", "y2", "cx", "cy", "r", "fx", "fy", "fr",
}

type gradientStop struct {
	Offset float64
	Color  rgba
}

// gradientDefinition describes a reusable gradient.
type gradientDefinition struct {
	interpolation string
	radial        bool
	attributes    [gradientAttributeCount]string
	// Stops belong to the defining gradient and are shared immutably by href users.
	stops []gradientStop
	depth int
}

type paintContext struct {
	Lengths lengthContext
	Bounds  viewBox
}

// resolvedGradient expresses coordinates in gradient space and its map to user space.
// Stops include stop-opacity; fill/stroke and compositing opacity are separate.
type resolvedGradient struct {
	Radial        bool
	Start         point
	End           point
	Center        point
	Focus         point
	Radius        float64
	Transform     matrix
	Spread        string
	Interpolation string
	Disabled      bool
	Solid         bool
	stops         []gradientStop
}

func (g resolvedGradient) stopCount() int              { return len(g.stops) }
func (g resolvedGradient) stop(index int) gradientStop { return g.stops[index] }

func (r *styleResolver) gradient(e *element, active map[*element]bool, depth int) (*gradientDefinition, error) {
	if err := r.spend(1); err != nil {
		return nil, err
	}
	if depth >= maxReferenceDepth {
		return nil, errLimit
	}
	if active[e] {
		return nil, errReferenceCycle
	}
	if g := r.gradients[e]; g != nil {
		if depth+g.depth > maxReferenceDepth {
			return nil, errLimit
		}
		return g, nil
	}
	if e.name.Local != "linearGradient" && e.name.Local != "radialGradient" {
		if e.name.Local == "pattern" {
			return nil, errUnsupported
		}
		return nil, errReference
	}

	active[e] = true
	defer delete(active, e)
	g := &gradientDefinition{
		radial: e.name.Local == "radialGradient",
		depth:  1,
	}
	s, err := r.sourceStyle(e)
	if err != nil {
		return nil, err
	}
	g.interpolation = s.Interpolation
	ref, ok := e.attribute("", "href")
	if !ok {
		ref, ok = e.attribute(xlinkNamespace, "href")
	}
	if ok {
		if err := r.spend(len(ref)); err != nil {
			return nil, err
		}
		target, err := r.stylesheet.document.resolve(ref)
		if err != nil {
			return nil, err
		}
		base, err := r.gradient(target, active, depth+1)
		if err != nil {
			return nil, err
		}
		g.attributes = base.attributes
		g.stops = base.stops
		g.depth = base.depth + 1
	}
	for i, name := range gradientAttributes {
		if text, ok := e.attribute("", name); ok {
			if len(text) > maxStyleValue {
				return nil, errLimit
			}
			g.attributes[i] = strings.TrimSpace(text)
			if g.attributes[i] == "" {
				return nil, errStyle
			}
		}
	}
	// Stops inherit properties in their source tree, independently of href and paint users.
	var ownStops []gradientStop
	var previous float64
	for _, child := range e.children {
		if err := r.spend(1); err != nil {
			return nil, err
		}
		if child.name.Local != "stop" {
			continue
		}
		if len(ownStops) >= maxGradientStops {
			return nil, errLimit
		}
		s, err := r.sourceStyle(child)
		if err != nil {
			return nil, err
		}
		text, ok := child.attribute("", "offset")
		if !ok {
			text = "0"
		}
		if len(text) > maxStyleValue {
			return nil, errLimit
		}
		if err := r.spend(len(text) + len(s.value(propStopColor))); err != nil {
			return nil, err
		}
		offset, err := opacity(strings.TrimSpace(text))
		if err != nil {
			return nil, err
		}
		offset = max(previous, offset)
		previous = offset
		c, err := parseColor(s.value(propStopColor), s.Color, r.colors)
		if err != nil {
			return nil, err
		}
		// Apply palette alpha once at the stop without modifying s.StopOpacity.
		// Fill/stroke opacity and group compositing are applied by the paint user.
		c.A *= s.StopOpacity
		ownStops = append(ownStops, gradientStop{
			Offset: offset,
			Color:  c,
		})
	}
	if len(ownStops) > 0 {
		g.stops = ownStops
	}
	if err := r.reserveCache(len(r.gradients), 0); err != nil {
		return nil, err
	}
	r.gradients[e] = g
	return g, nil
}

func gradientLength(text string, context lengthContext, axis lengthAxis) (float64, error) {
	if strings.HasSuffix(text, "em") || strings.HasSuffix(text, "ex") {
		return 0, errUnsupported
	}
	return resolveLength(text, context, axis)
}

func (g *gradientDefinition) resolve(context paintContext) (resolvedGradient, error) {
	// Apply defaults after href inheritance. Omitted focal coordinates follow
	// the final center, while explicitly inherited coordinates keep their values.
	// https://www.w3.org/TR/SVG11/pservers.html
	attributes := g.attributes
	defaults := [gradientAttributeCount]string{
		"objectBoundingBox", "", "pad", "0%", "0%", "100%", "0%", "50%", "50%", "50%", "", "", "0",
	}
	for i := range attributes {
		if attributes[i] == "" {
			attributes[i] = defaults[i]
		}
	}
	if attributes[gradientFX] == "" {
		attributes[gradientFX] = attributes[gradientCX]
	}
	if attributes[gradientFY] == "" {
		attributes[gradientFY] = attributes[gradientCY]
	}
	if g.radial {
		// Positive reference scales test for zero in any supported length unit.
		// This compatibility allowance is checked after href overrides and does
		// not enable nonzero SVG 2 focal radii or other relative font lengths.
		radius, err := resolveLength(attributes[gradientFR], lengthContext{
			Width:         1,
			Height:        1,
			FontSize:      1,
			XHeight:       1,
			PixelsPerInch: 96,
		}, diagonal)
		if err != nil || radius != 0 {
			return resolvedGradient{}, errUnsupported
		}
	}
	transform, err := parseTransform(attributes[gradientTransform])
	if err != nil {
		return resolvedGradient{}, err
	}
	result := resolvedGradient{
		Interpolation: g.interpolation,
		Radial:        g.radial,
		Transform:     transform,
		Spread:        attributes[gradientSpread],
		stops:         g.stops,
		Disabled:      len(g.stops) == 0,
		Solid:         len(g.stops) == 1,
	}
	switch result.Spread {
	case "pad", "reflect", "repeat":
	default:
		return resolvedGradient{}, errStyle
	}
	lengths := context.Lengths
	switch attributes[gradientUnits] {
	case "objectBoundingBox":
		// Keep coordinates normalized; compose the bounds map with gradientTransform
		// here. The interpreter supplies the painted element's outer transform.
		b := context.Bounds
		if !areValidNumbers(b.X, b.Y, b.Width, b.Height) || b.Width < 0 || b.Height < 0 {
			return resolvedGradient{}, errGeometry
		}
		lengths.Width = 1
		lengths.Height = 1
		result.Disabled = result.Disabled || b.Width == 0 || b.Height == 0
		result.Transform, err = (matrix{
			A: b.Width,
			D: b.Height,
			E: b.X,
			F: b.Y,
		}).mul(transform)
		if err != nil {
			return resolvedGradient{}, err
		}
	case "userSpaceOnUse":
	default:
		return resolvedGradient{}, errStyle
	}
	var coordinates [9]float64
	for i := gradientX1; i < gradientFR; i++ {
		if (!g.radial && i >= gradientCX) || (g.radial && i < gradientCX) {
			continue
		}
		axis := horizontal
		if i == gradientY1 || i == gradientY2 || i == gradientCY || i == gradientFY {
			axis = vertical
		}
		if i == gradientR {
			axis = diagonal
		}
		coordinates[i-gradientX1], err = gradientLength(attributes[i], lengths, axis)
		if err != nil {
			return resolvedGradient{}, err
		}
	}
	result.Start = point{X: coordinates[0], Y: coordinates[1]}
	result.End = point{X: coordinates[2], Y: coordinates[3]}
	result.Center = point{X: coordinates[4], Y: coordinates[5]}
	result.Radius = coordinates[6]
	result.Focus = point{X: coordinates[7], Y: coordinates[8]}
	if g.radial {
		if result.Radius < 0 {
			return resolvedGradient{}, errStyle
		}
		result.Solid = result.Solid || result.Radius == 0
		dx, dy := result.Focus.X-result.Center.X, result.Focus.Y-result.Center.Y
		distance := math.Hypot(dx, dy)
		if !isFinite(distance) {
			return resolvedGradient{}, errGeometry
		}
		if distance > result.Radius {
			scale := result.Radius / distance
			result.Focus = point{X: result.Center.X + dx*scale, Y: result.Center.Y + dy*scale}
		}
	} else {
		result.Solid = result.Solid || result.Start == result.End
	}
	if !areValidNumbers(result.Focus.X, result.Focus.Y) {
		return resolvedGradient{}, errGeometry
	}
	return result, nil
}
