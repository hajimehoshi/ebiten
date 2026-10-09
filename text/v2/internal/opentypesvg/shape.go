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

func (e *element) length(name string, context lengthContext, axis lengthAxis) (float64, error) {
	text, ok := e.attribute("", name)
	if !ok {
		return 0, nil
	}
	return resolveLength(text, context, axis)
}

// geometry returns local shape geometry; non-shape elements return an empty path.
func (e *element) geometry(context lengthContext) (path, error) {
	if e.name.Local == "path" {
		text, _ := e.attribute("", "d")
		return parsePath(text)
	}
	if e.name.Local == "polyline" || e.name.Local == "polygon" {
		text, _ := e.attribute("", "points")
		return parsePoints(text, e.name.Local == "polygon")
	}
	var names []string
	var axes []lengthAxis
	switch e.name.Local {
	case "rect":
		names = []string{"x", "y", "width", "height", "rx", "ry"}
		axes = []lengthAxis{horizontal, vertical, horizontal, vertical, horizontal, vertical}
	case "circle":
		names = []string{"cx", "cy", "r"}
		axes = []lengthAxis{horizontal, vertical, diagonal}
	case "ellipse":
		names = []string{"cx", "cy", "rx", "ry"}
		axes = []lengthAxis{horizontal, vertical, horizontal, vertical}
	case "line":
		names = []string{"x1", "y1", "x2", "y2"}
		axes = []lengthAxis{horizontal, vertical, horizontal, vertical}
	default:
		return path{}, nil
	}
	var v [6]float64
	for i, name := range names {
		value, err := e.length(name, context, axes[i])
		if err != nil {
			return path{}, err
		}
		v[i] = value
	}
	var p path
	move := func(x, y float64) {
		p.segments = append(p.segments, segment{
			Kind: moveTo,
			To:   point{X: x, Y: y},
		})
	}
	line := func(x, y float64) {
		p.segments = append(p.segments, segment{
			Kind: lineTo,
			To:   point{X: x, Y: y},
		})
	}
	arc := func(x, y, rx, ry float64) {
		p.segments = append(p.segments, segment{
			Kind:   arcTo,
			To:     point{X: x, Y: y},
			Radius: point{X: rx, Y: ry},
			Sweep:  true,
		})
	}
	finishSubpath := func() {
		p.segments = append(p.segments, segment{
			Kind: closePath,
			To:   p.segments[0].To,
		})
	}
	switch e.name.Local {
	case "line":
		move(v[0], v[1])
		line(v[2], v[3])
	case "circle", "ellipse":
		x, y, rx, ry := v[0], v[1], v[2], v[3]
		if e.name.Local == "circle" {
			ry = rx
		}
		if rx < 0 || ry < 0 {
			return path{}, errGeometry
		}
		if rx == 0 || ry == 0 {
			return path{}, nil
		}
		move(x+rx, y)
		arc(x, y+ry, rx, ry)
		arc(x-rx, y, rx, ry)
		arc(x, y-ry, rx, ry)
		arc(x+rx, y, rx, ry)
		finishSubpath()
	case "rect":
		x, y, w, h, rx, ry := v[0], v[1], v[2], v[3], v[4], v[5]
		if w < 0 || h < 0 || rx < 0 || ry < 0 {
			return path{}, errGeometry
		}
		if w == 0 || h == 0 {
			return path{}, nil
		}
		_, hasRX := e.attribute("", "rx")
		_, hasRY := e.attribute("", "ry")
		if !hasRX {
			rx = ry
		}
		if !hasRY {
			ry = rx
		}
		rx = math.Min(rx, w/2)
		ry = math.Min(ry, h/2)
		if rx == 0 || ry == 0 {
			move(x, y)
			line(x+w, y)
			line(x+w, y+h)
			line(x, y+h)
		} else {
			move(x+rx, y)
			line(x+w-rx, y)
			arc(x+w, y+ry, rx, ry)
			line(x+w, y+h-ry)
			arc(x+w-rx, y+h, rx, ry)
			line(x+rx, y+h)
			arc(x, y+h-ry, rx, ry)
			line(x, y+ry)
			arc(x+rx, y, rx, ry)
		}
		finishSubpath()
	}
	for _, seg := range p.segments {
		if !areValidNumbers(seg.To.X, seg.To.Y) {
			return path{}, errGeometry
		}
	}
	return p, nil
}

func parsePoints(text string, closed bool) (path, error) {
	if len(text) > maxBytes {
		return path{}, errLimit
	}
	s := numberScanner{
		text: text,
	}
	s.space()
	var p path
	for s.pos < len(text) {
		if len(p.segments) >= maxSegments-1 {
			return p, errLimit
		}
		x, err := s.number()
		if err != nil {
			return p, err
		}
		s.separator(false)
		y, err := s.number()
		if err != nil {
			return p, err
		}
		kind := lineTo
		if len(p.segments) == 0 {
			kind = moveTo
		}
		p.segments = append(p.segments, segment{
			Kind: kind,
			To:   point{X: x, Y: y},
		})
		if s.pos == len(text) {
			break
		}
		s.separator(false)
		if s.pos == len(text) && strings.HasSuffix(strings.TrimRight(text, xmlWhitespace), ",") {
			return p, errGeometry
		}
	}
	if closed && len(p.segments) > 0 {
		p.segments = append(p.segments, segment{
			Kind: closePath,
			To:   p.segments[0].To,
		})
	}
	return p, nil
}
