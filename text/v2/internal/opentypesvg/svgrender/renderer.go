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

// Package svgrender draws prepared OpenType SVG glyphs within the graphics lifecycle.
package svgrender

import (
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Renderer owns reusable point-sample surfaces and paint shaders.
// Its methods must be called from the Ebitengine graphics lifecycle.
type Renderer struct {
	// Sample, paint, and group surfaces use the same point-sample grid.
	// Vector applies binary path/clip coverage to the paint image;
	// AntiAlias must remain false.
	// Resolve only after all paint has been composed into samples.
	samples, paint    *ebiten.Image
	groups            []*ebiten.Image
	flushSource       *ebiten.Image
	gradient, resolve *ebiten.Shader
}

// Deallocate releases the renderer's GPU resources.
func (r *Renderer) Deallocate() {
	for _, group := range r.groups {
		if group != nil {
			group.Deallocate()
		}
	}
	r.groups = nil
	if r.samples != nil {
		r.samples.Deallocate()
		r.samples = nil
	}
	if r.paint != nil {
		r.paint.Deallocate()
		r.paint = nil
	}
	if r.flushSource != nil {
		r.flushSource.Deallocate()
		r.flushSource = nil
	}
	if r.gradient != nil {
		r.gradient.Deallocate()
		r.gradient = nil
	}
	if r.resolve != nil {
		r.resolve.Deallocate()
		r.resolve = nil
	}
}

// Draw composites a prepared glyph into dst at its bounds' origin.
func (r *Renderer) Draw(dst *ebiten.Image, plan *opentypesvg.RenderPlan) error {
	if plan == nil || dst == nil {
		return fmt.Errorf("svgrender: nil destination or plan")
	}
	w, h, s := plan.Width, plan.Height, plan.Samples
	if w <= 0 || h <= 0 || s != 1 && s != 2 && s != 4 && s != 8 || w > opentypesvg.MaxSampleDimension/s || h > opentypesvg.MaxSampleDimension/s || w*s > opentypesvg.MaxSamplePixels/(h*s) {
		return fmt.Errorf("svgrender: invalid sample surface dimensions")
	}
	if dst.Bounds().Dx() < w || dst.Bounds().Dy() < h {
		return fmt.Errorf("svgrender: destination is too small")
	}
	if len(plan.Draws) == 0 {
		return nil
	}
	// Compile before changing the destination or allocating scratch images.
	if r.gradient == nil {
		shader, err := ebiten.NewShader([]byte(gradientShader))
		if err != nil {
			return err
		}
		r.gradient = shader
	}
	if r.resolve == nil {
		shader, err := ebiten.NewShader([]byte(resolveShader))
		if err != nil {
			return err
		}
		r.resolve = shader
	}
	width, height := w*s, h*s
	if r.samples == nil || r.samples.Bounds().Dx() != width || r.samples.Bounds().Dy() != height {
		if r.samples != nil {
			r.samples.Deallocate()
		}
		if r.paint != nil {
			r.paint.Deallocate()
			r.paint = nil
		}
		bounds := image.Rect(0, 0, width, height)
		// The samples and paint scratch surfaces stay off vector's stencil atlas
		// to limit page growth and retained texture storage for large glyphs.
		// Their dependent render passes do not benefit from sharing an atlas
		// for draw batching.
		r.samples = ebiten.NewImageWithOptions(bounds, &ebiten.NewImageOptions{
			Unmanaged: true,
		})
	}
	// Release incompatible or excess surfaces before allocating this plan's pool.
	for i, group := range r.groups {
		if group != nil && (i >= plan.GroupDepth || group.Bounds() != r.samples.Bounds()) {
			group.Deallocate()
			r.groups[i] = nil
		}
	}
	if len(r.groups) > plan.GroupDepth {
		r.groups = r.groups[:plan.GroupDepth]
	}
	for len(r.groups) < plan.GroupDepth {
		r.groups = append(r.groups, nil)
	}
	for i := range r.groups {
		if r.groups[i] == nil {
			// Separate textures keep the full-size group surfaces from enlarging
			// the atlas shared with vector's stencil textures.
			r.groups[i] = ebiten.NewImageWithOptions(r.samples.Bounds(), &ebiten.NewImageOptions{
				Unmanaged: true,
			})
		}
	}
	uploads := map[*image.RGBA]*ebiten.Image{}
	defer func() {
		for _, upload := range uploads {
			upload.Deallocate()
		}
	}()
	target := r.samples
	var parents []*ebiten.Image
	r.samples.Clear()
	clips := map[*opentypesvg.RenderClip]vector.Clip{}
	for _, draw := range plan.Draws {
		if draw.GroupStart {
			r.flushSamples(target)
			parents = append(parents, target)
			target = r.groups[len(parents)-1]
			target.Clear()
			continue
		}
		if draw.GroupEnd {
			r.flushSamples(target)
			child := target
			target = parents[len(parents)-1]
			parents = parents[:len(parents)-1]
			var op ebiten.DrawImageOptions
			op.ColorScale.ScaleAlpha(draw.Opacity)
			target.DrawImage(child, &op)
			continue
		}
		if draw.FlushBefore {
			r.flushSamples(target)
		}
		path := makePath(draw.Commands)
		var output vector.Path
		if draw.StrokeWidth > 0 {
			op := vector.AddStrokeOptions{
				StrokeOptions: vector.StrokeOptions{
					Width:      float32(draw.StrokeWidth),
					MiterLimit: float32(draw.MiterLimit),
				},
				GeoM: geom(draw.Transform),
			}
			switch draw.LineCap {
			case "round":
				op.LineCap = vector.LineCapRound
			case "square":
				op.LineCap = vector.LineCapSquare
			}
			switch draw.LineJoin {
			case "round":
				op.LineJoin = vector.LineJoinRound
			case "bevel":
				op.LineJoin = vector.LineJoinBevel
			}
			output.AddStroke(&path, &op)
			addPointCaps(&output, draw)
		} else {
			output.AddPath(&path, &vector.AddPathOptions{
				GeoM: geom(draw.Transform),
			})
		}
		op := vector.DrawPathOptions{
			Clip: makeClip(draw.Clip, clips),
		}
		var fill vector.FillOptions
		if draw.EvenOdd {
			fill.FillRule = vector.FillRuleEvenOdd
		}
		if draw.Image != nil {
			source := uploads[draw.Image]
			if source == nil {
				source = ebiten.NewImageFromImage(draw.Image)
				uploads[draw.Image] = source
			}
			op.Paint = &vector.ImagePaint{
				Image:  source,
				GeoM:   geom(draw.ImageTransform),
				Filter: ebiten.FilterNearest,
			}
			vector.FillPath(target, &output, &fill, &op)
			continue
		}
		p := draw.Paint
		if len(p.Stops) == 0 {
			op.ColorScale.Scale(p.Color[0], p.Color[1], p.Color[2], p.Color[3])
			vector.FillPath(target, &output, &fill, &op)
			continue
		}
		r.fillGradient(target, &output, &fill, &op, p)
	}
	// This is the only resolve, averaging every sample in each output pixel.
	// Isolated groups use this same grid, origin, and dimensions. Their
	// clipping and opacity must be applied before compositing into the parent
	// sample surface. Resolving a nested group here would lose shared-edge
	// coverage needed by subsequent clips and composition.
	vertices := rectangleVertices(dst.Bounds().Min, w, h)
	op := ebiten.DrawTrianglesShaderOptions{
		Uniforms: map[string]any{"Samples": float32(s)},
	}
	op.Images[0] = r.samples
	dst.DrawTrianglesShader32(vertices, []uint32{0, 1, 2, 1, 3, 2}, r.resolve, &op)
	return nil
}

func geom(v [6]float64) ebiten.GeoM {
	var m ebiten.GeoM
	m.SetElement(0, 0, v[0])
	m.SetElement(1, 0, v[1])
	m.SetElement(0, 1, v[2])
	m.SetElement(1, 1, v[3])
	m.SetElement(0, 2, v[4])
	m.SetElement(1, 2, v[5])
	return m
}

func makePath(commands []opentypesvg.RenderCommand) vector.Path {
	var p vector.Path
	for _, c := range commands {
		switch c.Kind {
		case 'M':
			p.MoveTo(float32(c.X), float32(c.Y))
		case 'L':
			p.LineTo(float32(c.X), float32(c.Y))
		case 'Q':
			p.QuadTo(float32(c.CX), float32(c.CY), float32(c.X), float32(c.Y))
		case 'Z':
			p.Close()
		}
	}
	return p
}

func makeClip(c *opentypesvg.RenderClip, cache map[*opentypesvg.RenderClip]vector.Clip) vector.Clip {
	if c == nil {
		return nil
	}
	if clip, ok := cache[c]; ok {
		return clip
	}
	var result vector.Clip
	if len(c.Commands) > 0 {
		local := makePath(c.Commands)
		var path vector.Path
		path.AddPath(&local, &vector.AddPathOptions{
			GeoM: geom(c.Transform),
		})
		clip := &vector.PathClip{
			Path: &path,
		}
		if c.EvenOdd {
			clip.FillRule = vector.FillRuleEvenOdd
		}
		result = clip
	} else {
		set := &vector.ClipSet{}
		if c.Intersection {
			set.Operation = vector.ClipOperationIntersection
		}
		for _, child := range c.Children {
			set.Children = append(set.Children, makeClip(child, cache))
		}
		result = set
	}
	cache[c] = result
	return result
}

func paintUniforms(p opentypesvg.RenderPaint) map[string]any {
	stops := make([]float32, 64*4)
	offsets := make([]float32, 64)
	for i, c := range p.Stops {
		copy(stops[i*4:], c[:])
		offsets[i] = p.Offsets[i]
	}
	var radial, linear float32
	if p.Radial {
		radial = 1
	}
	if p.LinearRGB {
		linear = 1
	}
	return map[string]any{
		"Stops": stops, "Offsets": offsets, "Count": float32(len(p.Stops)),
		"Inverse": p.Inverse[:], "Linear": p.Linear[:], "Center": p.Center[:], "Focus": p.Focus[:], "Radius": p.Radius,
		"Radial": radial, "LinearRGB": linear, "Spread": float32(p.Spread),
	}
}

func rectangleVertices(origin image.Point, w, h int) []ebiten.Vertex {
	return []ebiten.Vertex{
		{
			DstX: float32(origin.X),
			DstY: float32(origin.Y),
			SrcX: 0,
			SrcY: 0,
		},
		{
			DstX: float32(origin.X + w),
			DstY: float32(origin.Y),
			SrcX: float32(w),
			SrcY: 0,
		},
		{
			DstX: float32(origin.X),
			DstY: float32(origin.Y + h),
			SrcX: 0,
			SrcY: float32(h),
		},
		{
			DstX: float32(origin.X + w),
			DstY: float32(origin.Y + h),
			SrcX: float32(w),
			SrcY: float32(h),
		},
	}
}

func addPointCaps(output *vector.Path, draw opentypesvg.RenderDraw) {
	if draw.LineCap == "butt" {
		return
	}
	var start opentypesvg.RenderCommand
	var present, drawn, nonzero bool
	var caps vector.Path
	finish := func() {
		if present && drawn && !nonzero {
			appendPointCap(&caps, start, draw.StrokeWidth, draw.LineCap)
		}
	}
	for _, c := range draw.Commands {
		if c.Kind == 'P' {
			appendPointCap(&caps, c, draw.StrokeWidth, draw.LineCap)
			continue
		}
		if c.Kind == 'M' {
			finish()
			start = c
			present = true
			drawn = false
			nonzero = false
			continue
		}
		present = true
		drawn = true
		nonzero = nonzero || c.X != start.X || c.Y != start.Y || c.Kind == 'Q' && (c.CX != start.X || c.CY != start.Y)
		if c.Kind == 'Z' {
			finish()
			start = opentypesvg.RenderCommand{
				Kind: 'M',
				X:    c.X,
				Y:    c.Y,
			}
			present = false
			drawn = false
			nonzero = false
		}
	}
	finish()
	output.AddPath(&caps, &vector.AddPathOptions{
		GeoM: geom(draw.Transform),
	})
}

func appendPointCap(path *vector.Path, position opentypesvg.RenderCommand, width float64, lineCap string) {
	x, y, radius := position.X, position.Y, width/2
	// Match AddStroke's winding before the shared affine transform.
	if lineCap == "round" {
		path.MoveTo(float32(x+radius), float32(y))
		path.Arc(float32(x), float32(y), float32(radius), 0, 2*math.Pi, vector.CounterClockwise)
	} else {
		dx, dy := position.DX, position.DY
		if dx == 0 && dy == 0 {
			dx = 1
		}
		ux, uy := dx*radius, dy*radius
		vx, vy := -uy, ux
		path.MoveTo(float32(x-ux-vx), float32(y-uy-vy))
		path.LineTo(float32(x-ux+vx), float32(y-uy+vy))
		path.LineTo(float32(x+ux+vx), float32(y+uy+vy))
		path.LineTo(float32(x+ux-vx), float32(y+uy-vy))
	}
	path.Close()
}

func flushSamples(samples, paint *ebiten.Image) {
	// DrawImage invokes usage callbacks before rejecting a singular transform.
	// The pending vector fills finish without drawing or resolving any samples.
	var options ebiten.DrawImageOptions
	options.GeoM.Scale(0, 0)
	samples.DrawImage(paint, &options)
}

func (r *Renderer) fillGradient(target *ebiten.Image, path *vector.Path, fill *vector.FillOptions, draw *vector.DrawPathOptions, paint opentypesvg.RenderPaint) {
	// Finish prior draws even if this gradient path is culled, matching the
	// preparation pass's separate stencil budget for each gradient draw.
	r.flushSamples(target)
	bounds := path.Bounds()
	if bounds.Empty() {
		return
	}
	// Snapping during FillPath may extend the integer bounds by one pixel.
	bounds = bounds.Inset(-1).Intersect(target.Bounds())
	if bounds.Empty() {
		return
	}
	if r.paint == nil {
		r.paint = ebiten.NewImageWithOptions(target.Bounds(), &ebiten.NewImageOptions{
			Unmanaged: true,
		})
	}
	source := r.paint.RecyclableSubImage(bounds)
	defer source.Recycle()
	var transform ebiten.GeoM
	transform.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	uniforms := paintUniforms(paint)
	uniforms["PaintOrigin"] = []float32{float32(bounds.Min.X), float32(bounds.Min.Y)}
	source.DrawRectShader(bounds.Dx(), bounds.Dy(), r.gradient, &ebiten.DrawRectShaderOptions{
		GeoM:     transform,
		Blend:    ebiten.BlendCopy,
		Uniforms: uniforms,
	})
	draw.Paint = &vector.ImagePaint{
		Image:  source,
		GeoM:   transform,
		Filter: ebiten.FilterNearest,
	}
	vector.FillPath(target, path, fill, draw)
	// Complete the gradient batch before reusing its paint pixels or view.
	r.flushSamples(target)
}

func (r *Renderer) flushSamples(target *ebiten.Image) {
	source := r.paint
	if source == nil {
		// Solid batches need only a dummy source for the zero-area draw.
		if r.flushSource == nil {
			r.flushSource = ebiten.NewImage(1, 1)
		}
		source = r.flushSource
	}
	flushSamples(target, source)
}
