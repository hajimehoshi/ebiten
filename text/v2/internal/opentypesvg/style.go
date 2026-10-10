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
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg/svgcss"
)

// style is an instance's computed properties, before geometry-dependent lengths.
type style struct {
	Color         rgba
	FillOpacity   float64
	StrokeOpacity float64
	Opacity       float64
	StopOpacity   float64
	FillRule      string
	ClipRule      string
	LineCap       string
	LineJoin      string
	MiterLimit    float64
	Display       bool
	Visible       bool
	Interpolation string
	values        propertyValues
}

type propertyValue struct {
	property property
	value    string
}

type propertyValues []propertyValue

func (values propertyValues) lookup(p property) (string, bool) {
	for _, value := range values {
		if value.property == p {
			return value.value, true
		}
		if value.property > p {
			break
		}
	}
	return "", false
}

func (s style) value(p property) string {
	if value, ok := s.values.lookup(p); ok {
		return value
	}
	return properties[p].initial
}

type strokeStyle struct {
	Width  float64
	Dash   []float64
	Offset float64
}

// styleResolver owns one document's style evaluation and its resource budget.
// Create a new resolver for a different palette or foreground snapshot.
type styleResolver struct {
	stylesheet   *stylesheet
	colors       colorContext
	specified    map[*element]propertyValues
	sourceStyles map[*element]style
	gradients    map[*element]*gradientDefinition
	work         int
	instances    int
	cachedValues int
}

func newStyleResolver(sheet *stylesheet, colors colorContext) (*styleResolver, error) {
	// Share the immutable stylesheet across interpretations, but keep host colors,
	// work budgets and caches local to this resolver. Palette storage is copied
	// below so host mutations cannot change already-computed styles.
	if len(colors.Palette) > maxPaletteEntries {
		return nil, errLimit
	}
	if !colors.Foreground.isValid() {
		return nil, errStyle
	}
	for _, c := range colors.Palette {
		if !c.isValid() {
			return nil, errStyle
		}
	}
	colors.Palette = slices.Clone(colors.Palette)
	r := &styleResolver{
		stylesheet:   sheet,
		colors:       colors,
		specified:    map[*element]propertyValues{},
		sourceStyles: map[*element]style{},
		gradients:    map[*element]*gradientDefinition{},
	}
	return r, nil
}

func (r *styleResolver) reserveCache(entryCount, values int) error {
	// Each cache holds at most one entry per source element. The document limit
	// already bounds entry counts; this check is a backstop if that limit changes.
	// The shared value-record budget can be reached by an otherwise valid document.
	if entryCount >= maxElements || values > maxCachedStyleValues-r.cachedValues {
		return errLimit
	}
	r.cachedValues += values
	return nil
}

func (r *styleResolver) spend(n int) error {
	if n > maxStyleWork-r.work {
		r.work = maxStyleWork
		return errLimit
	}
	r.work += n
	return nil
}

func (r *styleResolver) matchesPart(e *element, p svgcss.SelectorPart) (bool, error) {
	if err := r.spend(1 + len(e.attributes)); err != nil {
		return false, err
	}
	if p.Tag != "" && p.Tag != "*" && p.Tag != e.name.Local {
		return false, nil
	}
	if p.ID != "" {
		id, _ := e.attribute("", "id")
		if err := r.spend(len(id)); err != nil {
			return false, err
		}
		if id != p.ID {
			return false, nil
		}
	}
	classes, _ := e.attribute("", "class")
	for _, class := range p.Classes {
		if err := r.spend(len(classes)); err != nil {
			return false, err
		}
		found := false
		for _, token := range strings.Fields(classes) {
			if token == class {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}

func (r *styleResolver) matches(e *element, s svgcss.Selector) (bool, error) {
	last := len(s.Parts) - 1
	ok, err := r.matchesPart(e, s.Parts[last])
	if err != nil || !ok {
		return false, err
	}
	if last == 0 {
		return true, nil
	}
	// Each state names the next selector part to match on an ancestor.
	// Keeping all viable states avoids both greedy mismatches and backtracking.
	var states [maxSelectorParts]bool
	states[last-1] = true
	for e = e.parent; e != nil; e = e.parent {
		var next [maxSelectorParts]bool
		for i := 0; i < last; i++ {
			if !states[i] {
				continue
			}
			ok, err := r.matchesPart(e, s.Parts[i])
			if err != nil {
				return false, err
			}
			if ok {
				if i == 0 {
					return true, nil
				}
				next[i-1] = true
			}
			if !s.Parts[i+1].Child {
				next[i] = true
			}
		}
		if !slices.Contains(next[:], true) {
			return false, nil
		}
		states = next
	}
	return false, nil
}

type cascadeValue struct {
	value       string
	specificity int
	important   bool
}

func (r *styleResolver) specifiedStyle(e *element) (propertyValues, error) {
	// SVG 1.1 use instances clone cascaded declarations matched in the source tree.
	// Instance inheritance happens separately in compute. Source ancestor styles
	// are not inherited into a selected glyph through this cache.
	// https://www.w3.org/TR/SVG11/struct.html#UseElement
	if values, ok := r.specified[e]; ok {
		return values, nil
	}
	// Attributes precede stylesheet rules, which precede inline declarations.
	// Equal-priority/equal-specificity declarations replace earlier values once,
	// independently of the ordering or repetition of an element's class tokens.
	var winners [propertyCount]cascadeValue
	apply := func(d declaration, specificity int) {
		targets := []property{d.property}
		if d.property == propMarker {
			targets = []property{propMarkerStart, propMarkerMid, propMarkerEnd}
		}
		// Only mask-image can make the unsupported mask observable. Expanding the
		// shorthand here preserves declaration order when mask:none resets it.
		if d.property == propMask {
			targets = []property{propMaskImage}
		}
		for _, p := range targets {
			old := winners[p]
			if old.value != "" && ((old.important && !d.important) || (old.important == d.important && old.specificity > specificity)) {
				continue
			}
			winners[p] = cascadeValue{
				value:       d.value,
				specificity: specificity,
				important:   d.important,
			}
		}
	}
	for _, a := range e.attributes {
		if a.Name.Space != "" {
			continue
		}
		if err := r.spend(1 + len(a.Value)); err != nil {
			return nil, err
		}
		p, ok, err := classifyProperty(a.Name.Local, presentationAttribute)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		v := normalizeProperty(p, strings.TrimSpace(a.Value))
		if err := validateProperty(p, v); err != nil {
			if errors.Is(err, errLimit) || errors.Is(err, errUnsupported) {
				return nil, err
			}
			continue
		}
		apply(declaration{
			property: p,
			value:    v,
		}, 0)
	}
	for _, rule := range r.stylesheet.rules {
		if err := r.spend(1); err != nil {
			return nil, err
		}
		specificity := -1
		for _, s := range rule.selectors {
			ok, err := r.matches(e, s)
			if err != nil {
				return nil, err
			}
			if ok {
				specificity = max(specificity, s.Specificity)
			}
		}
		if specificity < 0 {
			continue
		}
		if err := r.spend(len(rule.declarations)); err != nil {
			return nil, err
		}
		for _, d := range rule.declarations {
			apply(d, specificity)
		}
	}
	if text, ok := e.attribute("", "style"); ok {
		parsed, err := svgcss.ParseDeclarations(text)
		if err != nil {
			return nil, cssError(err)
		}
		declarations, err := compileDeclarations(parsed)
		if err != nil {
			return nil, err
		}
		for _, d := range declarations {
			apply(d, 1<<30)
		}
	}
	var count int
	for _, w := range winners {
		if w.value != "" {
			count++
		}
	}
	if err := r.reserveCache(len(r.specified), count); err != nil {
		return nil, err
	}
	var values propertyValues
	if count > 0 {
		values = make(propertyValues, 0, count)
	}
	for p, w := range winners {
		if w.value != "" {
			values = append(values, propertyValue{
				property: property(p),
				value:    w.value,
			})
		}
	}
	r.specified[e] = values
	return values, nil
}

func (r *styleResolver) initialStyle() style {
	var s style
	s.Color = r.colors.Foreground
	s.FillOpacity = 1
	s.StrokeOpacity = 1
	s.Opacity = 1
	s.StopOpacity = 1
	s.FillRule = "nonzero"
	s.ClipRule = "nonzero"
	s.LineCap = "butt"
	s.LineJoin = "miter"
	s.MiterLimit = 4
	s.Display = true
	s.Visible = true
	s.Interpolation = "sRGB"
	return s
}

// compute resolves an element with an explicit instance parent, or initial values if nil.
func (r *styleResolver) compute(e *element, parent *style) (style, error) {
	if err := r.spend(1); err != nil {
		return style{}, err
	}
	r.instances++
	if r.instances > maxExpandedElements {
		return style{}, errLimit
	}
	specified, err := r.specifiedStyle(e)
	if err != nil {
		return style{}, err
	}
	initial := r.initialStyle()
	if parent == nil {
		parent = &initial
	}
	var declarations [propertyCount]string
	for _, value := range specified {
		declarations[value.property] = value.value
	}
	var values [propertyCount]string
	s := initial
	for p, info := range properties {
		values[p] = info.initial
		value := declarations[p]
		switch {
		case value == "inherit" || (info.inherited && (value == "" || value == "unset")):
			values[p] = parent.value(property(p))
		case value == "" || value == "initial" || value == "unset":
		default:
			values[p] = value
		}
	}
	s.Display = values[propDisplay] != "none"
	if s.Display {
		for p, info := range properties {
			if !info.unsupported || !effectApplies(property(p), e) {
				continue
			}
			value := values[p]
			overridesAttribute := false
			// Even an initial CSS value can disable a non-initial XML geometry
			// attribute, so it must not silently fall through to the geometry parser.
			if info.cssOnly && declarations[p] != "" {
				if err := r.spend(len(e.attributes)); err != nil {
					return style{}, err
				}
				_, overridesAttribute = e.attribute("", info.name)
			}
			if value == info.initial && !overridesAttribute {
				continue
			}
			if err := r.spend(len(value)); err != nil {
				return style{}, err
			}
			return style{}, errUnsupported
		}
		if value := values[propClipPath]; value != "none" && !svgcss.HasFunction(value, "url") {
			if err := r.spend(len(value)); err != nil {
				return style{}, err
			}
			return style{}, errUnsupported
		}
	}
	// color:currentColor has the same computed value as inherited color.
	colorValue := declarations[propColor]
	switch colorValue {
	case "", "inherit", "unset", "currentColor", "currentcolor":
		s.Color = parent.Color
	case "initial":
		s.Color = r.colors.Foreground
	default:
		if err := r.spend(len(values[propColor])); err != nil {
			return style{}, err
		}
		s.Color, err = parseColor(values[propColor], parent.Color, r.colors)
		if err != nil {
			if errors.Is(err, errLimit) || errors.Is(err, errUnsupported) {
				return style{}, err
			}
			s.Color = parent.Color
		}
	}
	// Missing variables invalidate at computed-value time, after the cascade.
	for _, p := range []property{propFill, propStroke, propStopColor} {
		if err := r.spend(len(values[p])); err != nil {
			return style{}, err
		}
		if _, err := parsePaint(values[p], s.Color, r.colors, p != propStopColor); err != nil {
			if errors.Is(err, errLimit) || errors.Is(err, errUnsupported) {
				return style{}, err
			}
			if properties[p].inherited {
				values[p] = parent.value(property(p))
			} else {
				values[p] = properties[p].initial
			}
		}
	}
	for _, p := range []property{propFillOpacity, propStrokeOpacity, propOpacity, propStopOpacity, propStrokeMiter} {
		if err := r.spend(len(values[p])); err != nil {
			return style{}, err
		}
	}
	s.FillOpacity, _ = opacity(values[propFillOpacity])
	s.StrokeOpacity, _ = opacity(values[propStrokeOpacity])
	s.Opacity, _ = opacity(values[propOpacity])
	s.StopOpacity, _ = opacity(values[propStopOpacity])
	s.FillRule = values[propFillRule]
	s.ClipRule = values[propClipRule]
	s.LineCap = values[propStrokeCap]
	s.LineJoin = values[propStrokeJoin]
	s.MiterLimit, _ = parseNumber(values[propStrokeMiter])
	s.Display = values[propDisplay] != "none"
	s.Visible = values[propVisibility] == "visible"
	s.Interpolation = values[propInterpolation]
	// Retain non-initial values sparsely, including non-inherited properties:
	// a child can explicitly inherit them, even from a display:none parent.
	for p, value := range values {
		if value == properties[p].initial {
			continue
		}
		entry := propertyValue{
			property: property(p),
			value:    value,
		}
		s.values = append(s.values, entry)
	}
	return s, nil
}

func (r *styleResolver) sourceStyle(e *element) (style, error) {
	if e == nil {
		return r.initialStyle(), nil
	}
	if s, ok := r.sourceStyles[e]; ok {
		return s, nil
	}
	parent, err := r.sourceStyle(e.parent)
	if err != nil {
		return style{}, err
	}
	s, err := r.compute(e, &parent)
	if err != nil {
		return style{}, err
	}
	if err := r.reserveCache(len(r.sourceStyles), len(s.values)); err != nil {
		return style{}, err
	}
	r.sourceStyles[e] = s
	return s, nil
}

func opacity(text string) (float64, error) {
	value, err := parseNumber(strings.TrimSuffix(text, "%"))
	if err != nil {
		return 0, errStyle
	}
	if strings.HasSuffix(text, "%") {
		value /= 100
	}
	return min(1, max(0, value)), nil
}

func styleLength(text string, context lengthContext) (float64, error) {
	// The generic geometry helper accepts supplied font metrics; this style
	// layer enforces OpenType's exclusion of relative font units.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/svg#svg-capability-requirements-and-restrictions
	if strings.HasSuffix(text, "em") || strings.HasSuffix(text, "ex") {
		return 0, errUnsupported
	}
	return resolveLength(text, context, diagonal)
}

func dashArray(text string, context lengthContext) ([]float64, error) {
	if text == "none" {
		return nil, nil
	}
	// A comma requires a value on both sides; whitespace alone is a separator.
	groups := strings.Split(text, ",")
	var values []float64
	var total float64
	for _, group := range groups {
		fields := strings.Fields(group)
		if len(fields) == 0 {
			return nil, errStyle
		}
		for _, field := range fields {
			if len(values) >= maxDashEntries {
				return nil, errLimit
			}
			v, err := styleLength(field, context)
			if err != nil {
				return nil, err
			}
			if v < 0 || !isFinite(total+v) {
				return nil, errStyle
			}
			values = append(values, v)
			total += v
		}
	}
	if total == 0 {
		return nil, nil
	}
	if len(values)%2 != 0 {
		values = append(values, values...)
	}
	return values, nil
}

func (r *styleResolver) stroke(s style, context lengthContext) (strokeStyle, error) {
	// Inherited strings share storage, but resolving them repeats parsing work.
	// Charge each use, including when compute reused cached declarations.
	if err := r.spend(1 + len(s.value(propStrokeWidth)) + len(s.value(propStrokeOffset)) + len(s.value(propStrokeDash))); err != nil {
		return strokeStyle{}, err
	}
	width, err := styleLength(s.value(propStrokeWidth), context)
	if err != nil {
		return strokeStyle{}, err
	}
	offset, err := styleLength(s.value(propStrokeOffset), context)
	if err != nil {
		return strokeStyle{}, err
	}
	dash, err := dashArray(s.value(propStrokeDash), context)
	if err != nil {
		return strokeStyle{}, err
	}
	return strokeStyle{
		Width:  width,
		Offset: offset,
		Dash:   dash,
	}, nil
}

func validateProperty(p property, text string) error {
	if len(text) > maxStyleValue {
		return errLimit
	}
	if text == "inherit" || text == "initial" || text == "unset" {
		return nil
	}
	if properties[p].unsupported {
		return validateDeferred(p, text)
	}
	context := lengthContext{
		Width:         100,
		Height:        100,
		PixelsPerInch: 96,
	}
	var allowed string
	switch p {
	case propColor, propStopColor:
		return validateColor(text)
	case propFill, propStroke:
		if text == "none" {
			return nil
		}
		if svgcss.HasFunction(text, "url") {
			_, fallback, err := paintURL(text)
			if err != nil {
				return err
			}
			fallback = normalizeProperty(propFill, fallback)
			if fallback == "" || fallback == "none" {
				return nil
			}
			return validateColor(fallback)
		}
		return validateColor(text)
	case propFillRule, propClipRule:
		allowed = "nonzero evenodd"
	case propStrokeCap:
		allowed = "butt round square"
	case propStrokeJoin:
		allowed = "miter round bevel"
	case propStrokeWidth, propStrokeOffset:
		n, err := styleLength(text, context)
		if err != nil {
			return err
		}
		if p == propStrokeWidth && n < 0 {
			return errStyle
		}
		return nil
	case propStrokeMiter:
		n, err := parseNumber(text)
		if err != nil || n < 1 {
			return errStyle
		}
		return nil
	case propStrokeDash:
		_, err := dashArray(text, context)
		return err
	case propFillOpacity, propStrokeOpacity, propOpacity, propStopOpacity:
		_, err := opacity(text)
		return err
	case propClipPath:
		if text == "none" {
			return nil
		}
		if svgcss.HasFunction(text, "url") {
			_, fallback, err := paintURL(text)
			if err == nil && fallback == "" {
				return nil
			}
			if errors.Is(err, errUnsupported) {
				return err
			}
		}
		return validateClipValue(text)
	case propDisplay:
		allowed = "inline block list-item run-in compact marker table inline-table table-row-group table-header-group table-footer-group table-row table-column-group table-column table-cell table-caption none"
	case propVisibility:
		allowed = "visible hidden collapse"
	case propInterpolation:
		allowed = "sRGB linearRGB auto"
	}
	if slices.Contains(strings.Fields(allowed), text) {
		return nil
	}
	return errStyle
}

func validateColor(text string) error {
	if svgcss.HasFunction(text, "var") {
		_, err := svgcss.ParseVariable(text)
		return cssError(err)
	}
	_, err := parseColor(text, rgba{A: 1}, colorContext{})
	return err
}

func effectApplies(p property, e *element) bool {
	switch p {
	case propMarkerStart, propMarkerMid, propMarkerEnd:
		switch e.name.Local {
		case "path", "line", "polyline", "polygon":
			return true
		}
		return false
	}
	return true
}

func paintOrder(text string) (string, error) {
	// Markers are rejected on marker-capable elements independently of order.
	// Only the relative order of fill and stroke can affect supported artwork.
	fields := strings.Fields(strings.ToLower(text))
	if len(fields) == 1 && fields[0] == "normal" {
		return "normal", nil
	}
	if len(fields) == 0 || len(fields) > 3 {
		return "", errStyle
	}
	seen := make(map[string]bool)
	var order []string
	for _, field := range fields {
		if (field != "fill" && field != "stroke" && field != "markers") || seen[field] {
			return "", errStyle
		}
		seen[field] = true
		if field != "markers" {
			order = append(order, field)
		}
	}
	for _, field := range []string{"fill", "stroke"} {
		if !seen[field] {
			order = append(order, field)
		}
	}
	if order[0] == "fill" {
		return "normal", nil
	}
	return "stroke fill", nil
}

func validateDeferred(p property, text string) error {
	if text == properties[p].initial {
		return nil
	}
	context := lengthContext{
		Width:         1,
		Height:        1,
		FontSize:      1,
		XHeight:       1,
		PixelsPerInch: 96,
	}
	var allowed string
	switch p {
	case propFilter:
		return deferredFunctions(text, "url blur brightness contrast drop-shadow grayscale hue-rotate invert opacity saturate sepia")
	case propMask, propMaskImage:
		_, err := svgcss.Components(text)
		return cssError(err)
	case propMarker, propMarkerStart, propMarkerMid, propMarkerEnd:
		if svgcss.HasFunction(text, "url") {
			_, fallback, err := paintURL(text)
			if errors.Is(err, errUnsupported) {
				return nil
			}
			if err != nil || fallback != "" {
				return errStyle
			}
			return nil
		}
		return errStyle
	case propVectorEffect:
		allowed = "none non-scaling-stroke non-scaling-size non-rotation fixed-position"
	case propMixBlendMode:
		allowed = "normal multiply screen overlay darken lighten color-dodge color-burn hard-light soft-light difference exclusion hue saturation color luminosity plus-darker plus-lighter"
	case propPaintOrder:
		_, err := paintOrder(text)
		return err
	case propTransform:
		return deferredFunctions(text, "matrix matrix3d translate translateX translateY translateZ translate3d scale scaleX scaleY scaleZ scale3d rotate rotateX rotateY rotateZ rotate3d skew skewX skewY perspective")
	case propTransformBox:
		allowed = "content-box border-box fill-box stroke-box view-box"
	case propTransformOrigin:
		fields := strings.Fields(text)
		if len(fields) == 0 || len(fields) > 3 {
			return errStyle
		}
		for _, field := range fields {
			if slices.Contains([]string{"left", "right", "top", "bottom", "center"}, strings.ToLower(field)) {
				continue
			}
			if _, err := resolveLength(field, context, horizontal); err != nil {
				return errStyle
			}
		}
		return nil
	case propD:
		return deferredFunctions(text, "path")
	case propX, propY, propWidth, propHeight, propCx, propCy, propR, propRx, propRy:
		value, err := resolveLength(text, context, horizontal)
		if err != nil {
			return errStyle
		}
		if p != propX && p != propY && p != propCx && p != propCy && value < 0 {
			return errStyle
		}
		return nil
	}
	if slices.Contains(strings.Fields(allowed), strings.ToLower(text)) {
		return nil
	}
	return errStyle
}

func deferredFunctions(text, allowed string) error {
	components, err := svgcss.Components(text)
	if err != nil {
		return cssError(err)
	}
	for _, component := range components {
		if err := deferredFunction(component, allowed); err != nil {
			return err
		}
	}
	return nil
}

func deferredFunction(text, allowed string) error {
	name, body, ok := strings.Cut(text, "(")
	if !ok || !strings.HasSuffix(body, ")") || !slices.Contains(strings.Fields(strings.ToLower(allowed)), strings.ToLower(name)) {
		return errStyle
	}
	body = body[:len(body)-1]
	if err := svgcss.ValidateValue(body); err != nil {
		return cssError(err)
	}
	if strings.EqualFold(name, "url") && strings.TrimSpace(body) == "" {
		return errStyle
	}
	return nil
}

func validateClipValue(text string) error {
	components, err := svgcss.Components(text)
	if err != nil {
		return cssError(err)
	}
	var shape, box bool
	for _, component := range components {
		if slices.Contains(strings.Fields("margin-box border-box padding-box content-box fill-box stroke-box view-box"), strings.ToLower(component)) {
			if box {
				return errStyle
			}
			box = true
			continue
		}
		if shape {
			return errStyle
		}
		if err := deferredFunction(component, "inset circle ellipse polygon path"); err != nil {
			return err
		}
		shape = true
	}
	return nil
}
