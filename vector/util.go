// Copyright 2022 The Ebitengine Authors
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

package vector

import (
	"image"
	"image/color"
	"math"
	"sync"
	_ "unsafe"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	whiteImage    = ebiten.NewImage(3, 3)
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

var (
	theCachedVerticesForUtil []ebiten.Vertex
	theCachedIndicesForUtil  []uint32
	theCacheForUtilM         sync.Mutex
)

func useCachedVerticesAndIndicesForUtil(fn func([]ebiten.Vertex, []uint32) (vs []ebiten.Vertex, is []uint32)) {
	theCacheForUtilM.Lock()
	defer theCacheForUtilM.Unlock()
	theCachedVerticesForUtil, theCachedIndicesForUtil = fn(theCachedVerticesForUtil[:0], theCachedIndicesForUtil[:0])
}

func init() {
	b := whiteImage.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	for i := range pix {
		pix[i] = 0xff
	}
	// This is hacky, but WritePixels is better than Fill in term of automatic texture packing.
	whiteImage.WritePixels(pix)
}

// StrokeLine strokes a line (x0, y0)-(x1, y1) with the specified width and color.
func StrokeLine(dst *ebiten.Image, x0, y0, x1, y1 float32, strokeWidth float32, clr color.Color, antialias bool) {
	if antialias {
		var path Path
		path.MoveTo(x0, y0)
		path.LineTo(x1, y1)
		strokeOp := &StrokeOptions{}
		strokeOp.Width = strokeWidth
		drawOp := &DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(clr)
		StrokePath(dst, &path, strokeOp, drawOp)
		return
	}

	// Use a regular DrawImage for batching.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(math.Hypot(float64(x1-x0), float64(y1-y0)), float64(strokeWidth))
	op.GeoM.Translate(0, -float64(strokeWidth)/2)
	op.GeoM.Rotate(math.Atan2(float64(y1-y0), float64(x1-x0)))
	op.GeoM.Translate(float64(x0), float64(y0))
	op.ColorScale.ScaleWithColor(clr)
	dst.DrawImage(whiteSubImage, op)
}

// FillRect fills a rectangle with the specified width and color.
func FillRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color, antialias bool) {
	if antialias {
		var path Path
		path.MoveTo(x, y)
		path.LineTo(x, y+height)
		path.LineTo(x+width, y+height)
		path.LineTo(x+width, y)
		drawOp := &DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(clr)
		FillPath(dst, &path, nil, drawOp)
		return
	}

	// Use a regular DrawImage for batching.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(width), float64(height))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	dst.DrawImage(whiteSubImage, op)
}

// DrawFilledRect fills a rectangle with the specified width and color.
//
// Deprecated: as of v2.9. Use [FillRect] instead.
func DrawFilledRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color, antialias bool) {
	FillRect(dst, x, y, width, height, clr, antialias)
}

// StrokeRect strokes a rectangle with the specified width and color.
func StrokeRect(dst *ebiten.Image, x, y, width, height float32, strokeWidth float32, clr color.Color, antialias bool) {
	if antialias {
		var path Path
		path.MoveTo(x, y)
		path.LineTo(x, y+height)
		path.LineTo(x+width, y+height)
		path.LineTo(x+width, y)
		path.Close()
		strokeOp := &StrokeOptions{}
		strokeOp.Width = strokeWidth
		strokeOp.MiterLimit = 10
		drawOp := &DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(clr)
		StrokePath(dst, &path, strokeOp, drawOp)
		return
	}

	if strokeWidth <= 0 {
		return
	}

	if strokeWidth >= width || strokeWidth >= height {
		FillRect(dst, x-strokeWidth/2, y-strokeWidth/2, width+strokeWidth, height+strokeWidth, clr, false)
		return
	}

	// Use a regular DrawImage for batching.
	{
		// Render the top side.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(width+strokeWidth), float64(strokeWidth))
		op.GeoM.Translate(float64(x-strokeWidth/2), float64(y-strokeWidth/2))
		op.ColorScale.ScaleWithColor(clr)
		dst.DrawImage(whiteSubImage, op)
	}
	{
		// Render the left side.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(strokeWidth), float64(height-strokeWidth))
		op.GeoM.Translate(float64(x-strokeWidth/2), float64(y+strokeWidth/2))
		op.ColorScale.ScaleWithColor(clr)
		dst.DrawImage(whiteSubImage, op)
	}
	{
		// Render the right side.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(strokeWidth), float64(height-strokeWidth))
		op.GeoM.Translate(float64(x+width-strokeWidth/2), float64(y+strokeWidth/2))
		op.ColorScale.ScaleWithColor(clr)
		dst.DrawImage(whiteSubImage, op)
	}
	{
		// Render the bottom side.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(width+strokeWidth), float64(strokeWidth))
		op.GeoM.Translate(float64(x-strokeWidth/2), float64(y+height-strokeWidth/2))
		op.ColorScale.ScaleWithColor(clr)
		dst.DrawImage(whiteSubImage, op)
	}
}

// FillCircle fills a circle with the specified center position (cx, cy), the radius (r), width and color.
func FillCircle(dst *ebiten.Image, cx, cy, r float32, clr color.Color, antialias bool) {
	if antialias {
		var path Path
		path.Arc(cx, cy, r, 0, 2*math.Pi, Clockwise)
		drawOp := &DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(clr)
		FillPath(dst, &path, nil, drawOp)
		return
	}

	// Use a regular DrawTriangles32 for batching.
	cr, cg, cb, ca := clr.RGBA()
	crf := float32(cr) / 0xffff
	cgf := float32(cg) / 0xffff
	cbf := float32(cb) / 0xffff
	caf := float32(ca) / 0xffff
	useCachedVerticesAndIndicesForUtil(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		count := int(math.Ceil(math.Pi * float64(r)))
		for i := range count {
			angle := float64(i) * (2 * math.Pi / float64(count))
			sin, cos := math.Sincos(angle)
			x := cx + r*float32(cos)
			y := cy + r*float32(sin)
			vs = append(vs, ebiten.Vertex{
				DstX:   x,
				DstY:   y,
				SrcX:   1,
				SrcY:   1,
				ColorR: crf,
				ColorG: cgf,
				ColorB: cbf,
				ColorA: caf,
			})
			if i > 1 {
				idx := uint32(len(vs))
				is = append(is, 0, idx-1, idx-2)
			}
		}
		op := &ebiten.DrawTrianglesOptions{}
		op.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
		dst.DrawTriangles32(vs, is, whiteSubImage, op)
		return vs, is
	})
}

// DrawFilledCircle fills a circle with the specified center position (cx, cy), the radius (r), width and color.
//
// Deprecated: as of v2.9. Use [FillCircle] instead.
func DrawFilledCircle(dst *ebiten.Image, cx, cy, r float32, clr color.Color, antialias bool) {
	FillCircle(dst, cx, cy, r, clr, antialias)
}

// StrokeCircle strokes a circle with the specified center position (cx, cy), the radius (r), width and color.
func StrokeCircle(dst *ebiten.Image, cx, cy, r float32, strokeWidth float32, clr color.Color, antialias bool) {
	if antialias {
		var path Path
		path.Arc(cx, cy, r, 0, 2*math.Pi, Clockwise)
		path.Close()
		strokeOp := &StrokeOptions{}
		strokeOp.Width = strokeWidth
		strokeOp.LineJoin = LineJoinRound
		drawOp := &DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(clr)
		StrokePath(dst, &path, strokeOp, drawOp)
		return
	}

	if strokeWidth <= 0 {
		return
	}

	if strokeWidth >= r {
		FillCircle(dst, cx, cy, r+strokeWidth/2, clr, false)
		return
	}

	// Use a regular DrawTriangles32 for batching.
	cr, cg, cb, ca := clr.RGBA()
	crf := float32(cr) / 0xffff
	cgf := float32(cg) / 0xffff
	cbf := float32(cb) / 0xffff
	caf := float32(ca) / 0xffff
	useCachedVerticesAndIndicesForUtil(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		count := int(math.Ceil(math.Pi * float64(r+strokeWidth/2)))
		for i := range count {
			angle := float64(i) * (2 * math.Pi / float64(count))
			sin, cos := math.Sincos(angle)
			x0 := cx + (r+strokeWidth/2)*float32(cos)
			y0 := cy + (r+strokeWidth/2)*float32(sin)
			vs = append(vs, ebiten.Vertex{
				DstX:   x0,
				DstY:   y0,
				SrcX:   1,
				SrcY:   1,
				ColorR: crf,
				ColorG: cgf,
				ColorB: cbf,
				ColorA: caf,
			})
			x1 := cx + (r-strokeWidth/2)*float32(cos)
			y1 := cy + (r-strokeWidth/2)*float32(sin)
			vs = append(vs, ebiten.Vertex{
				DstX:   x1,
				DstY:   y1,
				SrcX:   1,
				SrcY:   1,
				ColorR: crf,
				ColorG: cgf,
				ColorB: cbf,
				ColorA: caf,
			})
			idx := uint32(2 * i)
			total := uint32(2 * count)
			is = append(is, idx, idx+1, (idx+2)%total, idx+1, (idx+2)%total, (idx+3)%total)
		}
		op := &ebiten.DrawTrianglesOptions{}
		op.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
		dst.DrawTriangles32(vs, is, whiteSubImage, op)
		return vs, is
	})
}

// FillRule is the rule whether an overlapped region is rendered or not.
type FillRule int

const (
	// FillRuleNonZero means that triangles are rendered based on the non-zero rule.
	// If and only if the number of overlaps is not 0, the region is rendered.
	FillRuleNonZero FillRule = iota

	// FillRuleEvenOdd means that triangles are rendered based on the even-odd rule.
	// If and only if the number of overlaps is odd, the region is rendered.
	FillRuleEvenOdd
)

var (
	theCallbackTokens      = map[*ebiten.Image]int64{}
	theFillPathsStates     = map[*ebiten.Image]*fillPathsState{}
	theFillPathsStatesPool = sync.Pool{
		New: func() any {
			return &fillPathsState{}
		},
	}
	theFillPathM sync.Mutex
)

// FillOptions is options to fill a path.
type FillOptions struct {
	// FillRule is the rule whether an overlapped region is rendered or not.
	// The default (zero) value is FillRuleNonZero.
	FillRule FillRule
}

// DrawPathOptions is options to draw a path.
type DrawPathOptions struct {
	// AntiAlias is whether the path is drawn with anti-aliasing.
	// The default (zero) value is false.
	AntiAlias bool

	// ColorScale is the color scale to apply to the path.
	// The default (zero) value is identity, which is (1, 1, 1, 1) (white).
	ColorScale ebiten.ColorScale

	// Blend is the blend mode to apply to the path.
	// The default (zero) value is ebiten.BlendSourceOver.
	Blend ebiten.Blend
}

// FillPath fills the specified path with the specified options.
func FillPath(dst *ebiten.Image, path *Path, fillOptions *FillOptions, drawPathOptions *DrawPathOptions) {
	if drawPathOptions == nil {
		drawPathOptions = &DrawPathOptions{}
	}
	if fillOptions == nil {
		fillOptions = &FillOptions{}
	}

	theFillPathM.Lock()
	defer theFillPathM.Unlock()

	// Remove the previous registered callbacks.
	if token, ok := theCallbackTokens[dst]; ok {
		removeUsageCallback(dst, token)
	}
	delete(theCallbackTokens, dst)

	if _, ok := theFillPathsStates[dst]; !ok {
		theFillPathsStates[dst] = theFillPathsStatesPool.Get().(*fillPathsState)
	}
	s := theFillPathsStates[dst]
	if s.antialias != drawPathOptions.AntiAlias || s.blend != drawPathOptions.Blend || s.fillRule != fillOptions.FillRule {
		s.fillPaths(dst)
		s.reset()
	}
	s.antialias = drawPathOptions.AntiAlias
	s.blend = drawPathOptions.Blend
	s.fillRule = fillOptions.FillRule
	s.addPath(path, drawPathOptions.ColorScale)

	token := addUsageCallback(dst, func() {
		// Remove the callback not to call this twice.
		if token, ok := theCallbackTokens[dst]; ok {
			removeUsageCallback(dst, token)
		}
		delete(theCallbackTokens, dst)

		s := theFillPathsStates[dst]
		s.fillPaths(dst)

		delete(theFillPathsStates, dst)
		s.reset()
		theFillPathsStatesPool.Put(s)
	})
	theCallbackTokens[dst] = token

}

// StrokePath strokes the specified path with the specified options.
func StrokePath(dst *ebiten.Image, path *Path, strokeOptions *StrokeOptions, drawPathOptions *DrawPathOptions) {
	var stroke Path
	op := &AddStrokeOptions{}
	op.StrokeOptions = *strokeOptions
	stroke.AddStroke(path, op)
	FillPath(dst, &stroke, nil, drawPathOptions)
}

//go:linkname addUsageCallback github.com/hajimehoshi/ebiten/v2.addUsageCallback
func addUsageCallback(img *ebiten.Image, fn func()) int64

//go:linkname removeUsageCallback github.com/hajimehoshi/ebiten/v2.removeUsageCallback
func removeUsageCallback(img *ebiten.Image, token int64)
