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
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg/svgcss"
)

type paintKind uint8

const (
	paintNone paintKind = iota
	paintSolid
	paintGradient
)

// paint contains straight colors and a separate paint opacity.
// Opacity applies once to the sampled paint, before element compositing opacity.
type paint struct {
	Kind     paintKind
	Color    rgba
	Opacity  float64
	Gradient resolvedGradient
}

type paintValue struct {
	none      bool
	color     rgba
	reference string
	fallback  string
}

func paintURL(text string) (string, string, error) {
	if len(text) > maxStyleValue {
		return "", "", errLimit
	}
	if !svgcss.HasFunction(text, "url") {
		return "", "", errStyle
	}
	end := strings.IndexByte(text, ')')
	if end < 0 {
		return "", "", errStyle
	}
	ref := strings.TrimSpace(text[4:end])
	if len(ref) >= 2 && (ref[0] == '\'' || ref[0] == '"') && ref[len(ref)-1] == ref[0] {
		ref = ref[1 : len(ref)-1]
	}
	if !strings.HasPrefix(ref, "#") {
		return "", "", errUnsupported
	}
	if len(ref) < 2 || strings.ContainsAny(ref, " \t\r\n\f()\"'") {
		return "", "", errReference
	}
	return ref, strings.TrimSpace(text[end+1:]), nil
}

func parsePaint(text string, current rgba, colors colorContext, allowURL bool) (paintValue, error) {
	if allowURL && text == "none" {
		return paintValue{
			none: true,
		}, nil
	}
	if allowURL && svgcss.HasFunction(text, "url") {
		ref, fallback, err := paintURL(text)
		if err != nil {
			return paintValue{}, err
		}
		fallback = normalizeProperty(propFill, fallback)
		if fallback != "" && fallback != "none" {
			if _, err := parseColor(fallback, current, colors); err != nil {
				return paintValue{}, err
			}
		}
		return paintValue{
			reference: ref,
			fallback:  fallback,
		}, nil
	}
	c, err := parseColor(text, current, colors)
	if err != nil {
		return paintValue{}, err
	}
	return paintValue{
		color: c,
	}, nil
}

// resolvePaint resolves fill or stroke against the referencing element's geometry context.
func (r *styleResolver) resolvePaint(s style, stroke bool, context paintContext) (paint, error) {
	p := propFill
	alpha := s.FillOpacity
	if stroke {
		p = propStroke
		alpha = s.StrokeOpacity
	}
	if err := r.spend(1 + len(s.value(p))); err != nil {
		return paint{}, err
	}
	value, err := parsePaint(s.value(p), s.Color, r.colors, true)
	if err != nil {
		return paint{}, err
	}
	if value.none {
		return paint{
			Kind: paintNone,
		}, nil
	}
	if value.reference == "" {
		return paint{
			Kind:    paintSolid,
			Color:   value.color,
			Opacity: alpha,
		}, nil
	}
	target, err := r.stylesheet.document.resolve(value.reference)
	if err == nil {
		var definition *gradientDefinition
		definition, err = r.gradient(target, make(map[*element]bool), 0)
		if err == nil {
			var gradient resolvedGradient
			for _, value := range definition.attributes {
				if err := r.spend(1 + len(value)); err != nil {
					return paint{}, err
				}
			}
			gradient, err = definition.resolve(context)
			if err == nil {
				return paint{
					Kind:     paintGradient,
					Gradient: gradient,
					Opacity:  alpha,
				}, nil
			}
		}
	}
	// Unsupported rendering and exhausted budgets must remain visible to the caller.
	if errors.Is(err, errLimit) || errors.Is(err, errUnsupported) || value.fallback == "" {
		return paint{}, err
	}
	if err := r.spend(len(value.fallback)); err != nil {
		return paint{}, err
	}
	fallback, fallbackErr := parsePaint(value.fallback, s.Color, r.colors, false)
	if value.fallback == "none" {
		return paint{
			Kind: paintNone,
		}, nil
	}
	if fallbackErr != nil {
		return paint{}, fallbackErr
	}
	return paint{
		Kind:    paintSolid,
		Color:   fallback.color,
		Opacity: alpha,
	}, nil
}

func (r *styleResolver) clip(s style) (*element, error) {
	text := s.value(propClipPath)
	if text == "none" {
		return nil, nil
	}
	if err := r.spend(1 + len(text)); err != nil {
		return nil, err
	}
	if !svgcss.HasFunction(text, "url") {
		return nil, errUnsupported
	}
	ref, _, err := paintURL(text)
	if err != nil {
		return nil, err
	}
	e, err := r.stylesheet.document.resolve(ref)
	if err != nil {
		return nil, err
	}
	if e.name.Local != "clipPath" {
		return nil, errReference
	}
	return e, nil
}
