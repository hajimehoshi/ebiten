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

func useCachedVerticesAndIndices(fn func([]ebiten.Vertex, []uint32) (vs []ebiten.Vertex, is []uint32)) {
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

func drawVerticesForUtil(dst *ebiten.Image, vs []ebiten.Vertex, is []uint32, clr color.Color, antialias bool, fillRule ebiten.FillRule) {
	r, g, b, a := clr.RGBA()
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = float32(r) / 0xffff
		vs[i].ColorG = float32(g) / 0xffff
		vs[i].ColorB = float32(b) / 0xffff
		vs[i].ColorA = float32(a) / 0xffff
	}

	op := &ebiten.DrawTrianglesOptions{}
	op.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
	op.AntiAlias = antialias
	op.FillRule = fillRule
	dst.DrawTriangles32(vs, is, whiteSubImage, op)
}

// StrokeLine strokes a line (x0, y0)-(x1, y1) with the specified width and color.
//
// clr has be to be a solid (non-transparent) color.
func StrokeLine(dst *ebiten.Image, x0, y0, x1, y1 float32, strokeWidth float32, clr color.Color, antialias bool) {
	if antialias {
		var path Path
		path.MoveTo(x0, y0)
		path.LineTo(x1, y1)
		op := &StrokeOptions{}
		op.Width = strokeWidth
		StrokePath(dst, &path, clr, true, op)
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

// DrawFilledRect fills a rectangle with the specified width and color.
func DrawFilledRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color, antialias bool) {
	if antialias {
		var path Path
		path.MoveTo(x, y)
		path.LineTo(x, y+height)
		path.LineTo(x+width, y+height)
		path.LineTo(x+width, y)
		DrawFilledPath(dst, &path, clr, true, FillRuleNonZero)
		return
	}

	// Use a regular DrawImage for batching.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(width), float64(height))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	dst.DrawImage(whiteSubImage, op)
}

// StrokeRect strokes a rectangle with the specified width and color.
//
// clr has be to be a solid (non-transparent) color.
func StrokeRect(dst *ebiten.Image, x, y, width, height float32, strokeWidth float32, clr color.Color, antialias bool) {
	if antialias {
		var path Path
		path.MoveTo(x, y)
		path.LineTo(x, y+height)
		path.LineTo(x+width, y+height)
		path.LineTo(x+width, y)
		path.Close()
		op := &StrokeOptions{}
		op.Width = strokeWidth
		op.MiterLimit = 10
		StrokePath(dst, &path, clr, true, op)
		return
	}

	if strokeWidth <= 0 {
		return
	}

	if strokeWidth >= width || strokeWidth >= height {
		DrawFilledRect(dst, x-strokeWidth/2, y-strokeWidth/2, width+strokeWidth, height+strokeWidth, clr, false)
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

// DrawFilledCircle fills a circle with the specified center position (cx, cy), the radius (r), width and color.
func DrawFilledCircle(dst *ebiten.Image, cx, cy, r float32, clr color.Color, antialias bool) {
	if antialias {
		var path Path
		path.Arc(cx, cy, r, 0, 2*math.Pi, Clockwise)
		DrawFilledPath(dst, &path, clr, true, FillRuleNonZero)
		return
	}

	// Use a regular DrawTriangles32 for batching.
	cr, cg, cb, ca := clr.RGBA()
	crf := float32(cr) / 0xffff
	cgf := float32(cg) / 0xffff
	cbf := float32(cb) / 0xffff
	caf := float32(ca) / 0xffff
	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
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

// StrokeCircle strokes a circle with the specified center position (cx, cy), the radius (r), width and color.
//
// clr has be to be a solid (non-transparent) color.
func StrokeCircle(dst *ebiten.Image, cx, cy, r float32, strokeWidth float32, clr color.Color, antialias bool) {
	if antialias {
		var path Path
		path.Arc(cx, cy, r, 0, 2*math.Pi, Clockwise)
		path.Close()
		op := &StrokeOptions{}
		op.Width = strokeWidth
		op.LineJoin = LineJoinRound
		StrokePath(dst, &path, clr, true, op)
		return
	}

	if strokeWidth <= 0 {
		return
	}

	if strokeWidth >= r {
		DrawFilledCircle(dst, cx, cy, r+strokeWidth/2, clr, false)
		return
	}

	// Use a regular DrawTriangles32 for batching.
	cr, cg, cb, ca := clr.RGBA()
	crf := float32(cr) / 0xffff
	cgf := float32(cg) / 0xffff
	cbf := float32(cb) / 0xffff
	caf := float32(ca) / 0xffff
	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
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
	FillRuleNonZero FillRule = FillRule(ebiten.FillRuleNonZero)

	// FillRuleEvenOdd means that triangles are rendered based on the even-odd rule.
	// If and only if the number of overlaps is odd, the region is rendered.
	FillRuleEvenOdd FillRule = FillRule(ebiten.FillRuleEvenOdd)
)

// DrawFilledRect fills the specified path with the specified color.
func DrawFilledPath(dst *ebiten.Image, path *Path, clr color.Color, antialias bool, fillRule FillRule) {
	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		vs, is = path.AppendVerticesAndIndicesForFilling32(vs, is)
		drawVerticesForUtil(dst, vs, is, clr, antialias, ebiten.FillRule(fillRule))
		return vs, is
	})
}

// StrokePath strokes the specified path with the specified color and stroke options.
//
// clr has be to be a solid (non-transparent) color.
func StrokePath(dst *ebiten.Image, path *Path, clr color.Color, antialias bool, options *StrokeOptions) {
	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		vs, is = path.AppendVerticesAndIndicesForStroke32(vs, is, options)
		drawVerticesForUtil(dst, vs, is, clr, antialias, ebiten.FillRuleFillAll)
		return vs, is
	})
}
