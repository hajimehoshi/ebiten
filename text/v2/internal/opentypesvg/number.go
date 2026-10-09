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
	"slices"
	"strconv"
	"strings"
)

var errGeometry = errors.New("opentypesvg: invalid geometry")

// A per-geometry budget complements the document byte and instance budgets.
const maxSegments = 1 << 18

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func areValidNumbers(v ...float64) bool {
	return !slices.ContainsFunc(v, func(x float64) bool {
		return !isFinite(x)
	})
}

type numberScanner struct {
	text string
	pos  int
}

func (s *numberScanner) space() bool {
	start := s.pos
	for s.pos < len(s.text) && strings.ContainsRune(xmlWhitespace, rune(s.text[s.pos])) {
		s.pos++
	}
	return start != s.pos
}

func (s *numberScanner) separator(required bool) bool {
	found := s.space()
	if s.pos < len(s.text) && s.text[s.pos] == ',' {
		s.pos++
		s.space()
		found = true
	}
	return found || !required
}

func isAlpha(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func (s *numberScanner) number() (float64, error) {
	start := s.pos
	if s.pos < len(s.text) && (s.text[s.pos] == '+' || s.text[s.pos] == '-') {
		s.pos++
	}
	var n int
	for s.pos < len(s.text) && isDigit(s.text[s.pos]) {
		s.pos++
		n++
	}
	if s.pos < len(s.text) && s.text[s.pos] == '.' {
		s.pos++
		for s.pos < len(s.text) && isDigit(s.text[s.pos]) {
			s.pos++
			n++
		}
	}
	if n == 0 {
		return 0, errGeometry
	}
	// An 'em' or 'ex' suffix belongs to a length, not an exponent.
	if s.pos < len(s.text) && (s.text[s.pos] == 'e' || s.text[s.pos] == 'E') && !(s.pos+1 < len(s.text) && (s.text[s.pos+1] == 'm' || s.text[s.pos+1] == 'x')) {
		s.pos++
		if s.pos < len(s.text) && (s.text[s.pos] == '+' || s.text[s.pos] == '-') {
			s.pos++
		}
		startDigits := s.pos
		for s.pos < len(s.text) && isDigit(s.text[s.pos]) {
			s.pos++
		}
		if s.pos == startDigits {
			return 0, errGeometry
		}
	}
	v, err := strconv.ParseFloat(s.text[start:s.pos], 64)
	if err != nil || !isFinite(v) {
		return 0, errGeometry
	}
	if v == 0 {
		// Preserve zero spellings while rejecting nonzero magnitudes lost to underflow.
		for i := start; i < s.pos && s.text[i] != 'e' && s.text[i] != 'E'; i++ {
			if s.text[i] >= '1' && s.text[i] <= '9' {
				return 0, errGeometry
			}
		}
	}
	return v, nil
}

// parseNumber parses one SVG attribute number.
func parseNumber(text string) (float64, error) {
	if len(text) > maxBytes {
		return 0, errLimit
	}
	s := numberScanner{
		text: text,
	}
	s.space()
	v, err := s.number()
	s.space()
	if err != nil || s.pos != len(text) {
		return 0, errGeometry
	}
	return v, nil
}

// lengthContext supplies lengths in the current user coordinate system.
// PixelsPerInch must be positive when resolving physical units.
type lengthContext struct {
	Width         float64
	Height        float64
	FontSize      float64
	XHeight       float64
	PixelsPerInch float64
}

type lengthAxis uint8

const (
	horizontal lengthAxis = iota
	vertical
	diagonal
)

// resolveLength resolves an SVG attribute length to user units using an explicit percentage axis.
func resolveLength(text string, context lengthContext, axis lengthAxis) (float64, error) {
	if len(text) > maxBytes {
		return 0, errLimit
	}
	s := numberScanner{
		text: strings.Trim(text, xmlWhitespace),
	}
	v, err := s.number()
	if err != nil {
		return 0, err
	}
	unit := s.text[s.pos:]
	factor := 1.0
	switch unit {
	case "", "px":
	case "em":
		factor = context.FontSize
	case "ex":
		factor = context.XHeight
	case "%":
		switch axis {
		case horizontal:
			factor = context.Width / 100
		case vertical:
			factor = context.Height / 100
		case diagonal:
			if context.Width < 0 || context.Height < 0 {
				return 0, errGeometry
			}
			factor = math.Hypot(context.Width/math.Sqrt2, context.Height/math.Sqrt2) / 100
		default:
			return 0, errGeometry
		}
	case "in", "cm", "mm", "pt", "pc":
		if !isFinite(context.PixelsPerInch) || context.PixelsPerInch <= 0 {
			return 0, errGeometry
		}
		factor = context.PixelsPerInch
		switch unit {
		case "cm":
			factor /= 2.54
		case "mm":
			factor /= 25.4
		case "pt":
			factor /= 72
		case "pc":
			factor /= 6
		}
	default:
		return 0, errGeometry
	}
	if !isFinite(factor) || factor < 0 || !isFinite(v*factor) {
		return 0, errGeometry
	}
	return v * factor, nil
}
