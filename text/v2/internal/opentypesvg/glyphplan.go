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
	"image"
	"math"
)

// PrepareGlyph prepares a glyph with an output rectangle enclosing its ink.
// An empty rectangle with no error denotes valid empty artwork.
func (doc *RenderDocument) PrepareGlyph(options RenderOptions) (*RenderPlan, image.Rectangle, error) {
	// A one-pixel probe retains the complete geometry without budgeting a
	// speculative em-square surface. Geometry, image, expansion and coordinate
	// budgets still apply. The final pass charges the actual output surface.
	options.Width, options.Height = 1, 1
	images := make(map[string]*image.RGBA)
	probe, err := doc.prepare(options, images)
	if err != nil {
		return nil, image.Rectangle{}, err
	}
	var bounds image.Rectangle
	clipBounds := make(map[*RenderClip]image.Rectangle)
	for _, draw := range probe.Draws {
		if draw.GroupStart || draw.GroupEnd {
			continue
		}
		b, err := drawInkBounds(draw)
		if err != nil {
			return nil, image.Rectangle{}, err
		}
		if draw.Clip != nil {
			cb, err := clipInkBounds(draw.Clip, clipBounds)
			if err != nil {
				return nil, image.Rectangle{}, err
			}
			b = b.Intersect(cb)
		}
		bounds = bounds.Union(b)
	}
	if bounds.Empty() {
		return &RenderPlan{}, image.Rectangle{}, nil
	}
	s := float64(options.Samples)
	bounds = image.Rect(int(math.Floor(float64(bounds.Min.X)/s)), int(math.Floor(float64(bounds.Min.Y)/s)), int(math.Ceil(float64(bounds.Max.X)/s)), int(math.Ceil(float64(bounds.Max.Y)/s)))
	options.Width, options.Height = bounds.Dx(), bounds.Dy()
	transform := renderMatrix(options.Transform)
	transform.E -= float64(bounds.Min.X)
	transform.F -= float64(bounds.Min.Y)
	options.Transform = transform.values()
	plan, err := doc.prepare(options, images)
	if err != nil {
		return nil, image.Rectangle{}, err
	}
	return plan, bounds, nil
}

func drawInkBounds(draw RenderDraw) (image.Rectangle, error) {
	if draw.StrokeWidth != 0 {
		// Include the renderer's offset-curve and miter envelope under the full
		// affine transform. This is conservative for dashes and curved joins.
		return stencilBounds(draw.Commands, draw.Transform, draw.StrokeWidth, draw.MiterLimit), nil
	}
	var p path
	var start point
	for _, c := range draw.Commands {
		seg := segment{
			To: point{
				X: c.X,
				Y: c.Y,
			},
		}
		switch c.Kind {
		case 'M':
			seg.Kind = moveTo
			start = seg.To
		case 'L':
			seg.Kind = lineTo
		case 'Q':
			seg.Kind = quadTo
			seg.Control1 = point{
				X: c.CX,
				Y: c.CY,
			}
		case 'Z':
			seg.Kind = closePath
			seg.To = start
		default:
			continue
		}
		p.segments = append(p.segments, seg)
	}
	b, ok, err := geometryBounds(p, renderMatrix(draw.Transform))
	if err != nil || !ok {
		return image.Rectangle{}, err
	}
	return image.Rect(int(math.Floor(b.X)), int(math.Floor(b.Y)), int(math.Ceil(b.X+b.Width)), int(math.Ceil(b.Y+b.Height))), nil
}

func clipInkBounds(clip *RenderClip, memo map[*RenderClip]image.Rectangle) (image.Rectangle, error) {
	if bounds, ok := memo[clip]; ok {
		return bounds, nil
	}
	if clip.Children == nil {
		bounds, err := drawInkBounds(RenderDraw{
			Commands:  clip.Commands,
			Transform: clip.Transform,
		})
		if err == nil {
			memo[clip] = bounds
		}
		return bounds, err
	}
	var bounds image.Rectangle
	for i, child := range clip.Children {
		b, err := clipInkBounds(child, memo)
		if err != nil {
			return image.Rectangle{}, err
		}
		if i == 0 {
			bounds = b
		} else if clip.Intersection {
			bounds = bounds.Intersect(b)
		} else {
			bounds = bounds.Union(b)
		}
	}
	memo[clip] = bounds
	return bounds, nil
}
