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
	"mime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg/svgcss"
)

var (
	errStyle       = errors.New("opentypesvg: invalid style or paint")
	errUnsupported = errors.New("opentypesvg: unsupported style or paint feature")
)

const (
	// maxStyleValue bounds repeated parsing of individual SVG attribute and property values.
	maxStyleValue = 1 << 14

	maxCSSBytes       = svgcss.MaxBytes
	maxCSSRules       = svgcss.MaxRules
	maxDeclarations   = svgcss.MaxDeclarations
	maxSelectorParts  = svgcss.MaxSelectorParts
	maxColorDepth     = 16
	maxGradientStops  = 1 << 16
	maxPaletteEntries = 1 << 16
	maxDashEntries    = 1024
	// maxStyleWork bounds charged bytes and matching/traversal operations.
	maxStyleWork = 1 << 24
	// maxCachedStyleValues bounds records across specified and source styles.
	maxCachedStyleValues = 1 << 20
)

type property uint8

const (
	propColor property = iota
	propFill
	propStroke
	propFillRule
	propClipRule
	propStrokeWidth
	propStrokeCap
	propStrokeJoin
	propStrokeMiter
	propStrokeDash
	propStrokeOffset
	propFillOpacity
	propStrokeOpacity
	propOpacity
	propStopColor
	propStopOpacity
	propClipPath
	propDisplay
	propVisibility
	propInterpolation
	propFilter
	propMask
	propMarker
	propMarkerStart
	propMarkerMid
	propMarkerEnd
	propVectorEffect
	propPaintOrder
	propMixBlendMode
	propTransform
	propTransformOrigin
	propTransformBox
	propX
	propY
	propWidth
	propHeight
	propCx
	propCy
	propR
	propRx
	propRy
	propD
	propMaskImage
	propertyCount
)

type propertyInfo struct {
	name        string
	initial     string
	inherited   bool
	unsupported bool
	cssOnly     bool
}

var properties = [propertyCount]propertyInfo{
	propColor: {
		name:      "color",
		initial:   "currentColor",
		inherited: true,
	},
	propFill: {
		name:      "fill",
		initial:   "black",
		inherited: true,
	},
	propStroke: {
		name:      "stroke",
		initial:   "none",
		inherited: true,
	},
	propFillRule: {
		name:      "fill-rule",
		initial:   "nonzero",
		inherited: true,
	},
	propClipRule: {
		name:      "clip-rule",
		initial:   "nonzero",
		inherited: true,
	},
	propStrokeWidth: {
		name:      "stroke-width",
		initial:   "1",
		inherited: true,
	},
	propStrokeCap: {
		name:      "stroke-linecap",
		initial:   "butt",
		inherited: true,
	},
	propStrokeJoin: {
		name:      "stroke-linejoin",
		initial:   "miter",
		inherited: true,
	},
	propStrokeMiter: {
		name:      "stroke-miterlimit",
		initial:   "4",
		inherited: true,
	},
	propStrokeDash: {
		name:      "stroke-dasharray",
		initial:   "none",
		inherited: true,
	},
	propStrokeOffset: {
		name:      "stroke-dashoffset",
		initial:   "0",
		inherited: true,
	},
	propFillOpacity: {
		name:      "fill-opacity",
		initial:   "1",
		inherited: true,
	},
	propStrokeOpacity: {
		name:      "stroke-opacity",
		initial:   "1",
		inherited: true,
	},
	propOpacity: {
		name:    "opacity",
		initial: "1",
	},
	propStopColor: {
		name:    "stop-color",
		initial: "black",
	},
	propStopOpacity: {
		name:    "stop-opacity",
		initial: "1",
	},
	propClipPath: {
		name:    "clip-path",
		initial: "none",
	},
	propDisplay: {
		name:    "display",
		initial: "inline",
	},
	propVisibility: {
		name:      "visibility",
		initial:   "visible",
		inherited: true,
	},
	propInterpolation: {
		name:      "color-interpolation",
		initial:   "sRGB",
		inherited: true,
	},
	propFilter: {
		name:        "filter",
		initial:     "none",
		inherited:   false,
		unsupported: true,
	},
	propMask: {
		name:        "mask",
		initial:     "none",
		inherited:   false,
		unsupported: true,
	},
	// The SVG 1.1 attribute index lists marker longhands, not the shorthand.
	// https://www.w3.org/TR/SVG11/attindex.html#PresentationAttributeIndex
	propMarker: {
		cssOnly:     true,
		name:        "marker",
		initial:     "none",
		inherited:   true,
		unsupported: true,
	},
	propMarkerStart: {
		name:        "marker-start",
		initial:     "none",
		inherited:   true,
		unsupported: true,
	},
	propMarkerMid: {
		name:        "marker-mid",
		initial:     "none",
		inherited:   true,
		unsupported: true,
	},
	propMarkerEnd: {
		name:        "marker-end",
		initial:     "none",
		inherited:   true,
		unsupported: true,
	},
	propVectorEffect: {
		name:        "vector-effect",
		initial:     "none",
		inherited:   false,
		unsupported: true,
	},
	propPaintOrder: {
		name:        "paint-order",
		initial:     "normal",
		inherited:   true,
		unsupported: true,
	},
	propMixBlendMode: {
		name:        "mix-blend-mode",
		initial:     "normal",
		inherited:   false,
		unsupported: true,
	},
	propTransform: {
		name:        "transform",
		initial:     "none",
		unsupported: true,
		cssOnly:     true,
	},
	propTransformOrigin: {
		name:        "transform-origin",
		initial:     "0 0",
		unsupported: true,
		cssOnly:     false,
	},
	propTransformBox: {
		name:        "transform-box",
		initial:     "view-box",
		unsupported: true,
		cssOnly:     false,
	},
	propX: {
		name:        "x",
		initial:     "0",
		unsupported: true,
		cssOnly:     true,
	},
	propY: {
		name:        "y",
		initial:     "0",
		unsupported: true,
		cssOnly:     true,
	},
	propWidth: {
		name:        "width",
		initial:     "auto",
		unsupported: true,
		cssOnly:     true,
	},
	propHeight: {
		name:        "height",
		initial:     "auto",
		unsupported: true,
		cssOnly:     true,
	},
	propCx: {
		name:        "cx",
		initial:     "0",
		unsupported: true,
		cssOnly:     true,
	},
	propCy: {
		name:        "cy",
		initial:     "0",
		unsupported: true,
		cssOnly:     true,
	},
	propR: {
		name:        "r",
		initial:     "0",
		unsupported: true,
		cssOnly:     true,
	},
	propRx: {
		name:        "rx",
		initial:     "auto",
		unsupported: true,
		cssOnly:     true,
	},
	propRy: {
		name:        "ry",
		initial:     "auto",
		unsupported: true,
		cssOnly:     true,
	},
	propD: {
		name:        "d",
		initial:     "none",
		unsupported: true,
		cssOnly:     true,
	},
	propMaskImage: {
		name:        "mask-image",
		initial:     "none",
		unsupported: true,
	},
}

type propertySource uint8

const (
	cssDeclaration propertySource = iota
	presentationAttribute
)

func classifyProperty(name string, source propertySource) (property, bool, error) {
	if strings.HasPrefix(name, "--") {
		return 0, false, errUnsupported
	}
	for p, info := range properties {
		if name == info.name {
			if source == presentationAttribute && info.cssOnly {
				return 0, false, nil
			}
			return property(p), true, nil
		}
	}
	return 0, false, nil
}

// stylesheet is a document's immutable set of parsed author rules.
type stylesheet struct {
	document *document
	rules    []cssRule
}

func newStylesheet(d *document) (*stylesheet, error) {
	// All source style elements participate, including shared defs and styles
	// outside a selected glyph. The resulting rules belong to the document and
	// can be reused with independent resolver/color contexts.
	sheet := &stylesheet{
		document: d,
	}
	var bytes, declarations int
	for _, e := range d.styles {
		if value, ok := e.attribute("", "type"); ok && strings.TrimSpace(value) != "" {
			mediaType, _, err := mime.ParseMediaType(value)
			if err != nil || mediaType != "text/css" {
				return nil, errUnsupported
			}
		}
		if value, ok := e.attribute("", "media"); ok {
			switch strings.ToLower(strings.TrimSpace(value)) {
			case "", "all", "screen":
			default:
				return nil, errUnsupported
			}
		}
		bytes += len(e.text)
		if bytes > maxCSSBytes {
			return nil, errLimit
		}
		parsed, err := svgcss.ParseStylesheet(e.text)
		if err != nil {
			return nil, cssError(err)
		}
		if len(sheet.rules)+len(parsed) > maxCSSRules {
			return nil, errLimit
		}
		for _, rule := range parsed {
			compiled, err := compileDeclarations(rule.Declarations)
			if err != nil {
				return nil, err
			}
			declarations += len(compiled)
			if declarations > maxDeclarations {
				return nil, errLimit
			}
			sheet.rules = append(sheet.rules, cssRule{
				selectors:    rule.Selectors,
				declarations: compiled,
			})
		}
	}
	return sheet, nil
}

type declaration struct {
	property  property
	value     string
	important bool
}

type cssRule struct {
	selectors    []svgcss.Selector
	declarations []declaration
}

func cssError(err error) error {
	switch {
	case errors.Is(err, svgcss.ErrSyntax):
		return errStyle
	case errors.Is(err, svgcss.ErrUnsupported):
		return errUnsupported
	case errors.Is(err, svgcss.ErrLimit):
		return errLimit
	default:
		return err
	}
}

func compileDeclarations(source []svgcss.Declaration) ([]declaration, error) {
	// Classify before parsing priority or values so unknown properties remain
	// ignored. Invalid known declarations are dropped before the cascade;
	// variable substitution is deferred until compute.
	var result []declaration
	for _, d := range source {
		p, ok, err := classifyProperty(d.Name, cssDeclaration)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		parsed, err := svgcss.ParseValue(d.Value)
		if err != nil {
			if errors.Is(err, svgcss.ErrLimit) || errors.Is(err, svgcss.ErrUnsupported) {
				return nil, cssError(err)
			}
			continue
		}
		value := normalizeProperty(p, parsed.Text)
		if len(value) > maxStyleValue {
			return nil, errLimit
		}
		if err := validateProperty(p, value); err != nil {
			if errors.Is(err, errLimit) || errors.Is(err, errUnsupported) {
				return nil, err
			}
			continue
		}
		result = append(result, declaration{
			property:  p,
			value:     value,
			important: parsed.Important,
		})
	}
	return result, nil
}

func normalizeProperty(p property, value string) string {
	if properties[p].unsupported && strings.EqualFold(value, properties[p].initial) {
		return properties[p].initial
	}
	if p == propPaintOrder {
		if order, err := paintOrder(value); err == nil {
			return order
		}
	}
	if p == propTransformOrigin {
		fields := strings.Fields(value)
		if len(fields) == 2 || len(fields) == 3 {
			zero := true
			for _, field := range fields {
				v, err := resolveLength(field, lengthContext{
					Width:         1,
					Height:        1,
					FontSize:      1,
					XHeight:       1,
					PixelsPerInch: 96,
				}, horizontal)
				if err != nil || v != 0 {
					zero = false
				}
			}
			if zero {
				return properties[p].initial
			}
		}
	}
	for _, keyword := range []string{"inherit", "initial", "unset", "none", "normal", "currentColor"} {
		if strings.EqualFold(value, keyword) {
			return keyword
		}
	}
	switch p {
	case propFillRule, propClipRule, propStrokeCap, propStrokeJoin, propDisplay, propVisibility:
		return strings.ToLower(value)
	case propInterpolation:
		for _, keyword := range []string{"sRGB", "linearRGB", "auto"} {
			if strings.EqualFold(value, keyword) {
				return keyword
			}
		}
	}
	return value
}
