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

// viewBox is a rectangle in the source user coordinate system.
type viewBox struct {
	X, Y, Width, Height float64
}

// parseViewBox parses a nonnegative-size SVG viewBox attribute.
func parseViewBox(text string) (viewBox, error) {
	if len(text) > maxBytes {
		return viewBox{}, errLimit
	}
	s := numberScanner{
		text: text,
	}
	s.space()
	var v [4]float64
	for i := range v {
		if i > 0 && !s.separator(true) {
			return viewBox{}, errGeometry
		}
		n, err := s.number()
		if err != nil {
			return viewBox{}, err
		}
		v[i] = n
	}
	s.space()
	if s.pos != len(text) || v[2] < 0 || v[3] < 0 {
		return viewBox{}, errGeometry
	}
	return viewBox{
		X:      v[0],
		Y:      v[1],
		Width:  v[2],
		Height: v[3],
	}, nil
}

// viewportMapping maps a viewBox to a viewport; disabled is true for zero dimensions.
func viewportMapping(box viewBox, width, height float64, aspect string) (transform matrix, disabled bool, err error) {
	if len(aspect) > maxBytes {
		return matrix{}, false, errLimit
	}
	if !areValidNumbers(box.X, box.Y, box.Width, box.Height, width, height) || box.Width < 0 || box.Height < 0 || width < 0 || height < 0 {
		return matrix{}, false, errGeometry
	}
	align := "xMidYMid"
	var slice bool
	var fields []string
	rest := strings.Trim(aspect, xmlWhitespace)
	for rest != "" {
		if len(fields) == 3 {
			return matrix{}, false, errGeometry
		}
		end := strings.IndexAny(rest, xmlWhitespace)
		if end < 0 {
			fields = append(fields, rest)
			break
		}
		fields = append(fields, rest[:end])
		rest = strings.TrimLeft(rest[end:], xmlWhitespace)
	}
	if len(fields) > 0 && fields[0] == "defer" {
		fields = fields[1:]
		if len(fields) == 0 {
			return matrix{}, false, errGeometry
		}
	}
	if len(fields) > 0 {
		align = fields[0]
	}
	if len(fields) > 2 {
		return matrix{}, false, errGeometry
	}
	if len(fields) == 2 {
		switch fields[1] {
		case "slice":
			slice = true
		case "meet":
		default:
			return matrix{}, false, errGeometry
		}
	}
	var ax, ay float64
	if align != "none" {
		if len(align) != 8 || align[0] != 'x' || align[4] != 'Y' {
			return matrix{}, false, errGeometry
		}
		for i, part := range []string{align[1:4], align[5:8]} {
			var a float64
			switch part {
			case "Min":
			case "Mid":
				a = .5
			case "Max":
				a = 1
			default:
				return matrix{}, false, errGeometry
			}
			if i == 0 {
				ax = a
			} else {
				ay = a
			}
		}
	}
	if width == 0 || height == 0 || box.Width == 0 || box.Height == 0 {
		return identity(), true, nil
	}
	sx, sy := width/box.Width, height/box.Height
	if align != "none" {
		scale := math.Min(sx, sy)
		if slice {
			scale = math.Max(sx, sy)
		}
		sx = scale
		sy = scale
	}
	m := matrix{
		A: sx,
		D: sy,
		E: ax*(width-box.Width*sx) - box.X*sx,
		F: ay*(height-box.Height*sy) - box.Y*sy,
	}
	if !m.isValid() || sx == 0 || sy == 0 {
		return matrix{}, false, errGeometry
	}
	return m, false, nil
}

// glyphViewport describes the root mapping into font design units without a clip.
type glyphViewport struct {
	Transform matrix
	Lengths   lengthContext
	Disabled  bool
}

// viewport resolves the OpenType root viewport using the em square and supplied font metrics.
func (d *document) viewport(unitsPerEm float64, context lengthContext) (glyphViewport, error) {
	// Root dimensions define the viewBox destination, matching Skia's SVG root
	// mapping. Output pixel scaling remains the caller's job. Root x/y and
	// overflow do not shift or clip the glyph's y-down font coordinates.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/svg#coordinate-systems-and-glyph-metrics
	// https://github.com/google/skia/blob/main/modules/svg/src/SkSVGSVG.cpp
	if !isFinite(unitsPerEm) || unitsPerEm <= 0 {
		return glyphViewport{}, errGeometry
	}
	context.Width = unitsPerEm
	context.Height = unitsPerEm
	width, height := unitsPerEm, unitsPerEm
	if text, ok := d.root.attribute("", "width"); ok {
		v, err := resolveLength(text, context, horizontal)
		if err != nil {
			return glyphViewport{}, err
		}
		width = v
	}
	if text, ok := d.root.attribute("", "height"); ok {
		v, err := resolveLength(text, context, vertical)
		if err != nil {
			return glyphViewport{}, err
		}
		height = v
	}
	if width < 0 || height < 0 {
		return glyphViewport{}, errGeometry
	}
	context.Width = width
	context.Height = height
	result := glyphViewport{
		Transform: identity(),
		Lengths:   context,
		Disabled:  width == 0 || height == 0,
	}
	if text, ok := d.root.attribute("", "viewBox"); ok {
		box, err := parseViewBox(text)
		if err != nil {
			return glyphViewport{}, err
		}
		aspect, _ := d.root.attribute("", "preserveAspectRatio")
		m, disabled, err := viewportMapping(box, width, height, aspect)
		if err != nil {
			return glyphViewport{}, err
		}
		result.Transform = m
		result.Disabled = disabled
		result.Lengths.Width = box.Width
		result.Lengths.Height = box.Height
	}
	return result, nil
}
