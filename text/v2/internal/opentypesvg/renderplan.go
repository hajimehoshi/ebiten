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
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
)

// Render budgets bound submitted geometry and requested surface regions.
const (
	// One maximum-area draw and clip must fit in a single flush.
	// Metal probes reached 352.25 MiB of added GPU image storage: atlas page
	// growth and clip masks amplify region area. This is not a VRAM cap.
	maxStencilPixels = 8 << 20
	// Bound total rasterization work independently of reusable stencil storage.
	maxTotalStencilPixels = 64 << 20
	maxRenderSampleWork   = 1 << 28
	maxGroupDepth         = 16
	maxGroupPixels        = 16 << 20
	maxStrokeCommands     = 1 << 20
	maxRenderCommands     = 1 << 18
	maxRenderInstances    = 1 << 14
	maxRenderStops        = 64
	MaxSampleDimension    = 4096
	MaxSamplePixels       = 4 << 20
	maxRenderCoordinate   = 1 << 20
)

// RenderOptions describes a glyph's output in destination pixels.
type RenderOptions struct {
	GlyphID       uint16
	UnitsPerEm    float64
	Width, Height int
	// Samples is the number of point samples per axis (1, 2, 4, or 8).
	Samples int
	// AntiAlias enables vector antialiasing and requires Samples to be 1.
	AntiAlias bool
	// Transform is SVG matrix(a,b,c,d,e,f), mapping font coordinates to output pixels.
	// Zero means identity.
	Transform  [6]float64
	Foreground color.NRGBA
	Palette    []color.NRGBA
}

// RenderCommand is a move, line, quadratic, close, or directed point cap in stroke space.
type RenderCommand struct {
	Kind         byte
	X, Y, CX, CY float64
	// DX and DY give a point cap's unit direction; zero means the positive x axis.
	DX, DY float64
}

// RenderPaint describes a solid color or gradient in sample coordinates.
type RenderPaint struct {
	Color             [4]float32
	Stops             [][4]float32
	Offsets           []float32
	Inverse           [6]float32
	Linear            [3]float32
	Focus, Center     [2]float32
	Radius            float32
	Radial, LinearRGB bool
	Spread            int
}

// RenderDraw is a paint operation or an isolated group boundary.
type RenderDraw struct {
	// GroupStart and GroupEnd delimit an isolated sample surface.
	GroupStart, GroupEnd bool
	Opacity              float32
	Image                *image.RGBA
	ImageTransform       [6]float64
	// FlushBefore finishes the preceding sample batch before this draw.
	FlushBefore       bool
	Commands          []RenderCommand
	Transform         [6]float64
	StrokeWidth       float64
	LineCap, LineJoin string
	MiterLimit        float64
	EvenOdd           bool
	Paint             RenderPaint
	Clip              *RenderClip
}

// RenderClip is a path leaf or a union/intersection of point-sample regions.
type RenderClip struct {
	Commands     []RenderCommand
	Transform    [6]float64
	EvenOdd      bool
	Intersection bool
	Children     []*RenderClip
}

// RenderPlan is an owned, immutable painter-ordered glyph with a fixed output grid.
// Callers must not modify its fields or slice contents.
type RenderPlan struct {
	Width, Height, Samples int
	AntiAlias              bool
	Draws                  []RenderDraw
	GroupDepth             int
}

type renderNode struct {
	lengths      lengthContext
	element      *element
	style        style
	local        matrix
	geometry     path
	children     []*renderNode
	bounds       viewBox
	hasBounds    bool
	boundsOnly   bool
	image        *image.RGBA
	imageMapping matrix
}

type nodeOptions struct {
	clip           bool
	boundsRequired bool
	boundsOnly     bool
}

type renderBuilder struct {
	document                   *document
	styles                     *styleResolver
	instances, commands        int
	sampleWork, strokeCommands int
	totalStencilPixels         int
	batchStencilPixels         int
	batchEvenOdd               bool
	active                     map[*element]bool
	clipActive                 map[*element]bool
	groupDepth                 int
	images                     map[string]*image.RGBA
	imagePixels, imageBytes    int
	plan                       RenderPlan
}

// PrepareRender interprets a complete glyph without initializing graphics.
// On any error it returns no plan, including for a malformed geometry prefix.
func PrepareRender(source []byte, options RenderOptions) (*RenderPlan, error) {
	if options.AntiAlias && options.Samples != 1 || options.Width <= 0 || options.Height <= 0 || options.Samples != 1 && options.Samples != 2 && options.Samples != 4 && options.Samples != 8 {
		return nil, errGeometry
	}
	if options.Width > MaxSampleDimension/options.Samples || options.Height > MaxSampleDimension/options.Samples || options.Width*options.Samples > MaxSamplePixels/(options.Height*options.Samples) {
		return nil, errLimit
	}
	d, err := ParseRenderDocument(source)
	if err != nil {
		return nil, err
	}
	return d.Prepare(options)
}

// RenderDocument is an immutable SVG document shared by glyphs and color contexts.
type RenderDocument struct {
	document *document
	sheet    *stylesheet
}

// ParseRenderDocument parses an OpenType SVG document without initializing graphics.
func ParseRenderDocument(source []byte) (*RenderDocument, error) {
	d, err := parse(source)
	if err != nil {
		return nil, err
	}
	sheet, err := newStylesheet(d)
	if err != nil {
		return nil, err
	}
	return &RenderDocument{document: d, sheet: sheet}, nil
}

// Prepare interprets a glyph in the requested output coordinate system.
func (doc *RenderDocument) Prepare(options RenderOptions) (*RenderPlan, error) {
	return doc.prepare(options, nil)
}

func (doc *RenderDocument) prepare(options RenderOptions, images map[string]*image.RGBA) (*RenderPlan, error) {
	if options.AntiAlias && options.Samples != 1 || options.Width <= 0 || options.Height <= 0 || options.Samples != 1 && options.Samples != 2 && options.Samples != 4 && options.Samples != 8 {
		return nil, errGeometry
	}
	if options.Width > MaxSampleDimension/options.Samples || options.Height > MaxSampleDimension/options.Samples || options.Width*options.Samples > MaxSamplePixels/(options.Height*options.Samples) {
		return nil, errLimit
	}
	d, sheet := doc.document, doc.sheet
	if len(options.Palette) > maxPaletteEntries {
		return nil, errLimit
	}
	colors := colorContext{
		Foreground: renderColor(options.Foreground),
	}
	for _, c := range options.Palette {
		colors.Palette = append(colors.Palette, renderColor(c))
	}
	r, err := newStyleResolver(sheet, colors)
	if err != nil {
		return nil, err
	}
	viewport, err := d.viewport(options.UnitsPerEm, lengthContext{
		PixelsPerInch: 96,
	})
	if err != nil {
		return nil, err
	}
	b := renderBuilder{
		document:   d,
		images:     images,
		styles:     r,
		active:     map[*element]bool{},
		clipActive: map[*element]bool{},
		plan: RenderPlan{
			Width:     options.Width,
			Height:    options.Height,
			Samples:   options.Samples,
			AntiAlias: options.AntiAlias,
		},
	}
	// Images shared by the bounds and output passes remain charged to the plan.
	for href, img := range images {
		b.imageBytes += len(href)
		b.imagePixels += img.Bounds().Dx() * img.Bounds().Dy()
	}
	entry := d.ids["glyph"+strconv.FormatUint(uint64(options.GlyphID), 10)]
	if entry == nil {
		for id := range d.ids {
			if suffix, ok := strings.CutPrefix(id, "glyph"); ok {
				if _, err := strconv.ParseUint(suffix, 10, 16); err == nil {
					return nil, errGlyphNotFound
				}
			}
		}
		entry = d.root
	}
	if viewport.Disabled {
		return &b.plan, nil
	}
	output := renderMatrix(options.Transform)
	output, err = (matrix{
		A: float64(options.Samples),
		D: float64(options.Samples),
	}).mul(output)
	if err != nil {
		return nil, err
	}
	if entry != d.root {
		output, err = output.mul(viewport.Transform)
		if err != nil {
			return nil, err
		}
	}
	node, err := b.node(entry, nil, viewport.Lengths, 0, nodeOptions{})
	if err != nil {
		return nil, err
	}
	if node != nil {
		if entry == d.root {
			// The root transform surrounds the viewport mapping.
			node.local, err = node.local.mul(viewport.Transform)
			if err != nil {
				return nil, err
			}
		}
		if err := b.emit(node, output, nil, 0); err != nil {
			return nil, err
		}
	}
	return &b.plan, nil
}

func renderColor(c color.NRGBA) rgba {
	return rgba{R: float64(c.R) / 255, G: float64(c.G) / 255, B: float64(c.B) / 255, A: float64(c.A) / 255}
}

func renderMatrix(v [6]float64) matrix {
	if v == [6]float64{} {
		return identity()
	}
	return matrix{
		A: v[0],
		B: v[1],
		C: v[2],
		D: v[3],
		E: v[4],
		F: v[5],
	}
}

func (b *renderBuilder) charge(n int) error {
	if n > maxRenderCommands-b.commands {
		return errLimit
	}
	b.commands += n
	return nil
}

func (b *renderBuilder) node(e *element, parent *style, lengths lengthContext, depth int, options nodeOptions) (*renderNode, error) {
	clip, boundsRequired, boundsOnly := options.clip, options.boundsRequired, options.boundsOnly
	b.instances++
	if depth >= maxReferenceDepth || b.instances > maxRenderInstances {
		return nil, errLimit
	}
	if b.active[e] {
		return nil, errReferenceCycle
	}
	b.active[e] = true
	defer delete(b.active, e)
	switch e.name.Local {
	case "defs", "linearGradient", "radialGradient", "style", "stop", "clipPath", "mask", "pattern", "filter", "marker", "symbol":
		return nil, nil
	}
	if clip && !isShape(e.name.Local) && e.name.Local != "use" {
		return nil, nil
	}
	mode := stylePainting
	if clip {
		mode = styleAll
	} else if boundsOnly {
		mode = styleBounds
	}
	s, err := b.styles.computeMode(e, parent, mode)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", e.name.Local, err)
	}
	if !s.Display {
		return nil, nil
	}
	if !clip && s.Opacity == 0 {
		if !boundsRequired {
			return nil, nil
		}
		boundsOnly = true
		if err := b.styles.checkEffects(e, s, true); err != nil {
			return nil, err
		}
	}
	if !boundsOnly {
		target, err := b.styles.clip(s)
		if err != nil {
			return nil, err
		}
		if target != nil {
			units, _ := target.attribute("", "clipPathUnits")
			boundsRequired = boundsRequired || units == "objectBoundingBox"
		}
	}
	switch e.name.Local {
	case "g", "use", "image", "path", "rect", "circle", "ellipse", "line", "polygon", "polyline":
	case "svg":
		if e != b.document.root {
			return nil, fmt.Errorf("nested svg: %w", errUnsupported)
		}
	default:
		return nil, fmt.Errorf("element %s: %w", e.name.Local, errUnsupported)
	}
	local, err := e.localTransform(lengths)
	if err != nil {
		return nil, err
	}
	p, err := e.geometry(lengths)
	if err != nil {
		return nil, err
	}
	if err := b.charge(len(p.segments)); err != nil {
		return nil, err
	}
	n := &renderNode{
		boundsOnly: boundsOnly,
		lengths:    lengths,
		element:    e,
		style:      s,
		local:      local,
		geometry:   p,
	}
	if e.name.Local == "image" {
		if err := b.prepareImage(n); err != nil {
			return nil, err
		}
		p = n.geometry
	}
	n.bounds, n.hasBounds, err = geometryBounds(p, identity())
	if err != nil {
		return nil, err
	}
	var children []*element
	if e.name.Local == "g" || e.name.Local == "svg" {
		children = e.children
	}
	if e.name.Local == "use" {
		ref, ok := e.attribute("", "href")
		if !ok {
			ref, ok = e.attribute(xlinkNamespace, "href")
		}
		children = nil
		if ok {
			target, err := b.document.resolve(ref)
			if err != nil {
				return nil, err
			}
			if clip && !isShape(target.name.Local) {
				return nil, nil
			}
			switch target.name.Local {
			case "svg", "symbol", "defs", "clipPath", "linearGradient", "radialGradient", "mask", "pattern", "filter", "marker", "stop", "style":
				return nil, fmt.Errorf("use target %s: %w", target.name.Local, errUnsupported)
			}
			children = []*element{target}
		}
	}
	for _, child := range children {
		c, err := b.node(child, &s, lengths, depth+1, nodeOptions{
			clip:           clip,
			boundsRequired: boundsRequired,
			boundsOnly:     boundsOnly,
		})
		if err != nil {
			return nil, err
		}
		if c != nil {
			n.children = append(n.children, c)
			bounds, ok, err := b.subtreeBounds(c, c.local)
			if err != nil {
				return nil, err
			}
			if ok {
				n.bounds, n.hasBounds = unionBounds(n.bounds, n.hasBounds, bounds)
			}
		}
	}
	return n, nil
}

func (b *renderBuilder) subtreeBounds(n *renderNode, transform matrix) (viewBox, bool, error) {
	if err := b.charge(len(n.geometry.segments)); err != nil {
		return viewBox{}, false, err
	}
	bounds, ok, err := geometryBounds(n.geometry, transform)
	if err != nil {
		return viewBox{}, false, err
	}
	for _, child := range n.children {
		m, err := transform.mul(child.local)
		if err != nil {
			return viewBox{}, false, err
		}
		cb, cok, err := b.subtreeBounds(child, m)
		if err != nil {
			return viewBox{}, false, err
		}
		if cok {
			bounds, ok = unionBounds(bounds, ok, cb)
		}
	}
	return bounds, ok, nil
}

func unionBounds(a viewBox, valid bool, c viewBox) (viewBox, bool) {
	if !valid {
		return c, true
	}
	x, y := min(a.X, c.X), min(a.Y, c.Y)
	return viewBox{
		X:      x,
		Y:      y,
		Width:  max(a.X+a.Width, c.X+c.Width) - x,
		Height: max(a.Y+a.Height, c.Y+c.Height) - y,
	}, true
}

func (b *renderBuilder) emit(n *renderNode, parent matrix, inherited *RenderClip, depth int) error {
	if n.boundsOnly {
		return nil
	}
	transform, err := parent.mul(n.local)
	if err != nil {
		return err
	}
	if transform.A*transform.D-transform.B*transform.C == 0 {
		return nil
	}
	clip, err := b.clip(n, transform, depth)
	if err != nil {
		return err
	}
	clip = intersectRenderClips(inherited, clip)
	isolated := n.style.Opacity != 1
	if isolated {
		area := b.plan.Width * b.plan.Height * b.plan.Samples * b.plan.Samples
		if b.groupDepth >= maxGroupDepth || area > maxGroupPixels/(b.groupDepth+1) || 2*area > maxRenderSampleWork-b.sampleWork {
			return errLimit
		}
		// Clearing and compositing each isolated surface both visit its samples.
		b.sampleWork += 2 * area
		b.groupDepth++
		b.plan.GroupDepth = max(b.plan.GroupDepth, b.groupDepth)
		b.plan.Draws = append(b.plan.Draws, RenderDraw{
			GroupStart: true,
		})
		b.batchStencilPixels = 0
	}
	if n.style.Visible && n.image != nil {
		if err := b.emitImage(n, transform, clip); err != nil {
			return err
		}
	}
	if n.style.Visible && n.element.name.Local != "image" && len(n.geometry.segments) > 0 {
		context := paintContext{
			Lengths: n.lengths,
			Bounds:  n.bounds,
		}
		fill, err := b.styles.resolvePaint(n.style, false, context)
		if err != nil {
			return err
		}
		stroke, err := b.styles.resolvePaint(n.style, true, context)
		if err != nil {
			return err
		}
		ss, err := b.styles.stroke(n.style, n.lengths)
		if err != nil {
			return err
		}
		if _, ok := n.element.attribute("", "pathLength"); ok && len(ss.Dash) > 0 {
			return fmt.Errorf("pathLength dash calibration: %w", errUnsupported)
		}
		for i, p := range []paint{fill, stroke} {
			if p.Kind == paintNone || i == 1 && ss.Width == 0 {
				continue
			}
			prepared, disabled, err := b.styles.preparePaint(n.style, i == 1, p, transform)
			if err != nil {
				return err
			}
			if disabled {
				continue
			}
			// The Frobenius norm bounds magnification in every direction. Curves
			// and dash lengths use this uniform stroke space; stroke expansion
			// precedes the remaining affine transform to preserve nonuniform widths.
			scale := math.Hypot(math.Hypot(transform.A, transform.B), math.Hypot(transform.C, transform.D))
			if !isFinite(scale) || scale == 0 {
				return errGeometry
			}
			commands, err := b.convertPath(n.geometry, scale)
			if err != nil {
				return err
			}
			draw := RenderDraw{
				Commands: commands,
				Transform: (matrix{
					A: transform.A / scale,
					B: transform.B / scale,
					C: transform.C / scale,
					D: transform.D / scale,
					E: transform.E,
					F: transform.F,
				}).values(),
				EvenOdd: n.style.FillRule == "evenodd",
				Paint:   prepared,
				Clip:    clip,
			}
			if i == 1 {
				draw.StrokeWidth = ss.Width * scale
				draw.LineCap = n.style.LineCap
				draw.LineJoin = n.style.LineJoin
				draw.MiterLimit = n.style.MiterLimit
				draw.EvenOdd = false
				if len(ss.Dash) > 0 {
					draw.Commands, err = b.dash(commands, ss, scale)
					if err != nil {
						return err
					}
				}
			}
			if i == 1 {
				// Offset quadratics split to at most 64 pieces per side; reserve
				// space for both sides, cubic round joins/caps, and normalization
				// before AddStroke. Its current recursive converters stop at level 6.
				cost := len(draw.Commands) * 1024
				if cost > maxStrokeCommands-b.strokeCommands {
					return errLimit
				}
				b.strokeCommands += cost
			}
			work := b.plan.Width * b.plan.Height * b.plan.Samples * b.plan.Samples
			if len(draw.Paint.Stops) > 0 {
				work *= 2
			}
			if work > maxRenderSampleWork-b.sampleWork {
				return errLimit
			}
			b.sampleWork += work
			if err := validateRenderDraw(draw); err != nil {
				return err
			}
			if err := b.prepareDrawBatch(&draw); err != nil {
				return err
			}
			b.plan.Draws = append(b.plan.Draws, draw)
		}
	}
	for _, child := range n.children {
		if err := b.emit(child, transform, clip, depth+1); err != nil {
			return err
		}
	}
	if isolated {
		b.plan.Draws = append(b.plan.Draws, RenderDraw{
			GroupEnd: true,
			Opacity:  float32(n.style.Opacity),
		})
		b.groupDepth--
		b.batchStencilPixels = 0
	}
	return nil
}

func validateRenderDraw(d RenderDraw) error {
	if !areValidNumbers(d.StrokeWidth, d.MiterLimit) || d.StrokeWidth*d.MiterLimit > maxRenderCoordinate {
		return errLimit
	}
	for _, v := range d.Transform {
		if !isFinite(v) || math.Abs(v) > maxRenderCoordinate {
			return errLimit
		}
	}
	for _, c := range d.Commands {
		if !areValidNumbers(c.X, c.Y, c.CX, c.CY, c.DX, c.DY) || max(math.Abs(c.X), math.Abs(c.Y), math.Abs(c.CX), math.Abs(c.CY), math.Abs(c.DX), math.Abs(c.DY)) > maxRenderCoordinate {
			return errLimit
		}
	}
	return nil
}

func intersectRenderClips(a, c *RenderClip) *RenderClip {
	if a == nil {
		return c
	}
	if c == nil {
		return a
	}
	return &RenderClip{
		Intersection: true,
		Children:     []*RenderClip{a, c},
	}
}

func (b *renderBuilder) clip(n *renderNode, transform matrix, depth int) (*RenderClip, error) {
	e, err := b.styles.clip(n.style)
	if err != nil || e == nil {
		return nil, err
	}
	if depth >= maxReferenceDepth {
		return nil, errLimit
	}
	if b.clipActive[e] {
		return nil, errReferenceCycle
	}
	b.clipActive[e] = true
	defer delete(b.clipActive, e)
	lengths := n.lengths
	units, _ := e.attribute("", "clipPathUnits")
	switch units {
	case "", "userSpaceOnUse":
	case "objectBoundingBox":
		lengths.Width, lengths.Height = 1, 1
		if !n.hasBounds || n.bounds.Width == 0 || n.bounds.Height == 0 {
			return &RenderClip{}, nil
		}
		bounds := n.bounds
		transform, err = transform.mul(matrix{
			A: bounds.Width,
			D: bounds.Height,
			E: bounds.X,
			F: bounds.Y,
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, errGeometry
	}
	s, err := b.styles.sourceStyle(e)
	if err != nil {
		return nil, err
	}
	local, err := e.localTransform(lengths)
	if err != nil {
		return nil, err
	}
	transform, err = transform.mul(local)
	if err != nil {
		return nil, err
	}
	union := &RenderClip{}

	root := &renderNode{
		element: e,
		style:   s,
		lengths: lengths,
	}
	for _, child := range e.children {
		node, err := b.node(child, &s, lengths, depth+1, nodeOptions{
			clip: true,
		})
		if err != nil {
			return nil, err
		}
		if node == nil {
			continue
		}
		c, err := b.clipNode(node, transform, depth+1)
		if err != nil {
			return nil, err
		}
		union.Children = append(union.Children, c)
		bounds, ok, err := b.subtreeBounds(node, node.local)
		if err != nil {
			return nil, err
		}
		if ok {
			root.bounds, root.hasBounds = unionBounds(root.bounds, root.hasBounds, bounds)
		}
	}
	own, err := b.clip(root, transform, depth+1)
	if err != nil {
		return nil, err
	}
	return intersectRenderClips(union, own), nil
}

func (b *renderBuilder) clipNode(n *renderNode, parent matrix, depth int) (*RenderClip, error) {
	if depth >= maxReferenceDepth {
		return nil, errLimit
	}
	m, err := parent.mul(n.local)
	if err != nil {
		return nil, err
	}
	union := &RenderClip{}
	if n.style.Visible && len(n.geometry.segments) > 0 {
		scale := math.Hypot(math.Hypot(m.A, m.B), math.Hypot(m.C, m.D))
		if scale > 0 {
			commands, err := b.convertPath(n.geometry, scale)
			if err != nil {
				return nil, err
			}
			transform := (matrix{
				A: m.A / scale,
				B: m.B / scale,
				C: m.C / scale,
				D: m.D / scale,
				E: m.E,
				F: m.F,
			}).values()
			if err := validateRenderDraw(RenderDraw{
				Commands:  commands,
				Transform: transform,
			}); err != nil {
				return nil, err
			}
			union.Children = append(union.Children, &RenderClip{
				Commands:  commands,
				Transform: transform,
				EvenOdd:   n.style.ClipRule == "evenodd",
			})
		}
	}
	for _, child := range n.children {
		c, err := b.clipNode(child, m, depth+1)
		if err != nil {
			return nil, err
		}
		union.Children = append(union.Children, c)
	}
	own, err := b.clip(n, m, depth+1)
	if err != nil {
		return nil, err
	}
	return intersectRenderClips(union, own), nil
}

func isShape(name string) bool {
	switch name {
	case "path", "rect", "circle", "ellipse", "line", "polygon", "polyline":
		return true
	}
	return false
}

func (b *renderBuilder) chargeClip(root *RenderClip, drawBounds image.Rectangle, pixels *int) error {
	seen := map[*RenderClip]bool{}
	var visit func(*RenderClip) error
	visit = func(c *RenderClip) error {
		if c == nil || seen[c] {
			return nil
		}
		if err := b.charge(1 + len(c.Commands)); err != nil {
			return err
		}
		seen[c] = true
		if len(c.Commands) > 0 {
			bounds := stencilBounds(c.Commands, c.Transform, 0, 0).Intersect(drawBounds)
			if err := addStencilPixels(pixels, bounds); err != nil {
				return err
			}
		}
		for _, child := range c.Children {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(root)
}

func addStencilPixels(pixels *int, bounds image.Rectangle) error {
	if bounds.Empty() {
		return nil
	}
	width := (bounds.Dx() + 15) &^ 15
	height := (bounds.Dy() + 15) &^ 15
	area := width * height
	if area > maxStencilPixels-*pixels {
		return errLimit
	}
	*pixels += area
	return nil
}

func (b *renderBuilder) prepareDrawBatch(draw *RenderDraw) error {
	surface := image.Rect(0, 0, b.plan.Width*b.plan.Samples, b.plan.Height*b.plan.Samples)
	bounds := stencilBounds(draw.Commands, draw.Transform, draw.StrokeWidth, draw.MiterLimit).Intersect(surface)
	var pixels int
	if err := addStencilPixels(&pixels, bounds); err != nil {
		return err
	}
	if err := b.chargeClip(draw.Clip, bounds, &pixels); err != nil {
		return err
	}
	if pixels > maxTotalStencilPixels-b.totalStencilPixels {
		return errLimit
	}
	b.totalStencilPixels += pixels
	gradient := len(draw.Paint.Stops) > 0
	// Each gradient paint finishes its own batch on the color target.
	if gradient {
		b.batchStencilPixels = 0
	}
	// A culled draw can skip vector's fill-rule handling. Explicitly finish
	// the old batch so its accounting never depends on that optimization.
	if draw.EvenOdd != b.batchEvenOdd && b.batchStencilPixels > 0 {
		draw.FlushBefore = true
		b.batchStencilPixels = 0
	}
	if pixels > maxStencilPixels-b.batchStencilPixels {
		draw.FlushBefore = true
		b.batchStencilPixels = 0
	}
	b.batchStencilPixels += pixels
	b.batchEvenOdd = draw.EvenOdd
	if gradient {
		b.batchStencilPixels = 0
	}
	return nil
}

func stencilBounds(commands []RenderCommand, transform [6]float64, width, miter float64) image.Rectangle {
	if len(commands) == 0 {
		return image.Rectangle{}
	}
	loX, loY := math.Inf(1), math.Inf(1)
	hiX, hiY := math.Inf(-1), math.Inf(-1)
	add := func(x, y float64) {
		x, y = float64(float32(x)), float64(float32(y))
		tx := transform[0]*x + transform[2]*y + transform[4]
		ty := transform[1]*x + transform[3]*y + transform[5]
		loX, loY = min(loX, tx), min(loY, ty)
		hiX, hiY = max(hiX, tx), max(hiY, ty)
	}
	for _, c := range commands {
		add(c.X, c.Y)
		if c.Kind == 'Q' {
			add(c.CX, c.CY)
		}
	}
	// The control hull includes the curve. Offset quadratics extend their
	// controls by at most three radii; joins use the miter limit. One extra
	// pixel covers vertex conversion and vector's subpixel snapping.
	radius := float64(float32(width)) / 2 * max(4, float64(float32(miter)))
	if !isFinite(radius) {
		return image.Rect(-8*maxRenderCoordinate, -8*maxRenderCoordinate, 8*maxRenderCoordinate, 8*maxRenderCoordinate)
	}
	padX := radius*(math.Abs(transform[0])+math.Abs(transform[2])) + 1
	padY := radius*(math.Abs(transform[1])+math.Abs(transform[3])) + 1
	return image.Rect(int(math.Floor(loX-padX)), int(math.Floor(loY-padY)), int(math.Ceil(hiX+padX)), int(math.Ceil(hiY+padY)))
}
