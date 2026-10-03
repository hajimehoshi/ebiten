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

func init() {
	b := whiteImage.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	for i := range pix {
		pix[i] = 0xff
	}
	// This is hacky, but WritePixels is better than Fill in terms of automatic texture packing.
	whiteImage.WritePixels(pix)
}

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

func circleVertexCount(r float64) int {
	const maxCircleVertexCount = 8192

	// A full circle with a larger radius cannot have finite float32 vertices.
	if !(r > 0) || r > math.MaxFloat32 {
		return 0
	}

	// At this count, the error from approximating a circle is comparable to
	// float32 precision, so additional vertices cannot meaningfully improve it.
	if r >= maxCircleVertexCount/math.Pi {
		return maxCircleVertexCount
	}
	return int(math.Ceil(math.Pi * r))
}

var (
	thePathPool = sync.Pool{
		New: func() any {
			return &Path{}
		},
	}
)

// StrokeLine strokes a line (x0, y0)-(x1, y1) with the specified width and color.
func StrokeLine(dst *ebiten.Image, x0, y0, x1, y1 float32, strokeWidth float32, clr color.Color, antialias bool) {
	if antialias {
		path := thePathPool.Get().(*Path)
		defer func() {
			path.Reset()
			thePathPool.Put(path)
		}()
		path.MoveTo(x0, y0)
		path.LineTo(x1, y1)
		strokeOp := &StrokeOptions{}
		strokeOp.Width = strokeWidth
		drawOp := &DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(clr)
		StrokePath(dst, path, strokeOp, drawOp)
		return
	}

	// Use a regular DrawImage for batching.
	op := &ebiten.DrawImageOptions{}
	op.GeoM = strokeLineGeoM(x0, y0, x1, y1, strokeWidth)
	op.ColorScale.ScaleWithColor(clr)
	dst.DrawImage(whiteSubImage, op)
}

func strokeLineGeoM(x0, y0, x1, y1, strokeWidth float32) ebiten.GeoM {
	dx := float64(x1) - float64(x0)
	dy := float64(y1) - float64(y0)
	var geoM ebiten.GeoM
	geoM.Scale(math.Hypot(dx, dy), float64(strokeWidth))
	geoM.Translate(0, -float64(strokeWidth)/2)
	geoM.Rotate(math.Atan2(dy, dx))
	geoM.Translate(float64(x0), float64(y0))
	return geoM
}

// FillRect fills a rectangle with the specified position (x, y), size (width, height) and color.
func FillRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color, antialias bool) {
	fillRect(dst, float64(x), float64(y), float64(width), float64(height), clr, antialias)
}

func fillRect(dst *ebiten.Image, x, y, width, height float64, clr color.Color, antialias bool) {
	if antialias {
		path := thePathPool.Get().(*Path)
		defer func() {
			path.Reset()
			thePathPool.Put(path)
		}()
		path.moveTo(x, y)
		path.lineTo(x, y+height)
		path.lineTo(x+width, y+height)
		path.lineTo(x+width, y)
		drawOp := &DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(clr)
		FillPath(dst, path, nil, drawOp)
		return
	}

	// Use a regular DrawImage for batching.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(width, height)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	dst.DrawImage(whiteSubImage, op)
}

// DrawFilledRect fills a rectangle with the specified position (x, y), size (width, height) and color.
//
// Deprecated: as of v2.9. Use [FillRect] instead.
func DrawFilledRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color, antialias bool) {
	FillRect(dst, x, y, width, height, clr, antialias)
}

// StrokeRect strokes a rectangle with the specified position (x, y), size (width, height), stroke width and color.
func StrokeRect(dst *ebiten.Image, x, y, width, height float32, strokeWidth float32, clr color.Color, antialias bool) {
	strokeRect(dst, float64(x), float64(y), float64(width), float64(height), float64(strokeWidth), clr, antialias)
}

func strokeRect(dst *ebiten.Image, x, y, width, height, strokeWidth float64, clr color.Color, antialias bool) {
	if antialias {
		path := thePathPool.Get().(*Path)
		defer func() {
			path.Reset()
			thePathPool.Put(path)
		}()
		path.moveTo(x, y)
		path.lineTo(x, y+height)
		path.lineTo(x+width, y+height)
		path.lineTo(x+width, y)
		path.Close()
		strokeOp := &StrokeOptions{}
		strokeOp.Width = float32(strokeWidth)
		strokeOp.MiterLimit = 10
		drawOp := &DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(clr)
		StrokePath(dst, path, strokeOp, drawOp)
		return
	}

	if strokeWidth <= 0 {
		return
	}

	if strokeWidth >= width || strokeWidth >= height {
		fillRect(dst, x-strokeWidth/2, y-strokeWidth/2, width+strokeWidth, height+strokeWidth, clr, false)
		return
	}

	// Use a regular DrawImage for batching.
	{
		// Render the top side.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(width+strokeWidth, strokeWidth)
		op.GeoM.Translate(x-strokeWidth/2, y-strokeWidth/2)
		op.ColorScale.ScaleWithColor(clr)
		dst.DrawImage(whiteSubImage, op)
	}
	{
		// Render the left side.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(strokeWidth, height-strokeWidth)
		op.GeoM.Translate(x-strokeWidth/2, y+strokeWidth/2)
		op.ColorScale.ScaleWithColor(clr)
		dst.DrawImage(whiteSubImage, op)
	}
	{
		// Render the right side.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(strokeWidth, height-strokeWidth)
		op.GeoM.Translate(x+width-strokeWidth/2, y+strokeWidth/2)
		op.ColorScale.ScaleWithColor(clr)
		dst.DrawImage(whiteSubImage, op)
	}
	{
		// Render the bottom side.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(width+strokeWidth, strokeWidth)
		op.GeoM.Translate(x-strokeWidth/2, y+height-strokeWidth/2)
		op.ColorScale.ScaleWithColor(clr)
		dst.DrawImage(whiteSubImage, op)
	}
}

// FillCircle fills a circle with the specified center position (cx, cy), the radius (r) and color.
func FillCircle(dst *ebiten.Image, cx, cy, r float32, clr color.Color, antialias bool) {
	fillCircle(dst, float64(cx), float64(cy), float64(r), clr, antialias)
}

func fillCircle(dst *ebiten.Image, cx, cy, r float64, clr color.Color, antialias bool) {
	if antialias {
		path := thePathPool.Get().(*Path)
		defer func() {
			path.Reset()
			thePathPool.Put(path)
		}()
		path.addArc(cx, cy, r, 0, 2*math.Pi, Clockwise)
		drawOp := &DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(clr)
		FillPath(dst, path, nil, drawOp)
		return
	}

	count := circleVertexCount(r)
	if count == 0 {
		return
	}

	// Use a regular DrawTriangles32 for batching.
	cr, cg, cb, ca := clr.RGBA()
	crf := float32(cr) / 0xffff
	cgf := float32(cg) / 0xffff
	cbf := float32(cb) / 0xffff
	caf := float32(ca) / 0xffff
	useCachedVerticesAndIndicesForUtil(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		for i := range count {
			angle := float64(i) * (2 * math.Pi / float64(count))
			sin, cos := math.Sincos(angle)
			x := float32(cx + r*cos)
			y := float32(cy + r*sin)
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

// DrawFilledCircle fills a circle with the specified center position (cx, cy), the radius (r) and color.
//
// Deprecated: as of v2.9. Use [FillCircle] instead.
func DrawFilledCircle(dst *ebiten.Image, cx, cy, r float32, clr color.Color, antialias bool) {
	FillCircle(dst, cx, cy, r, clr, antialias)
}

// StrokeCircle strokes a circle with the specified center position (cx, cy), the radius (r), width and color.
func StrokeCircle(dst *ebiten.Image, cx, cy, r float32, strokeWidth float32, clr color.Color, antialias bool) {
	strokeCircle(dst, float64(cx), float64(cy), float64(r), float64(strokeWidth), clr, antialias)
}

func strokeCircle(dst *ebiten.Image, cx, cy, r, strokeWidth float64, clr color.Color, antialias bool) {
	if antialias {
		path := thePathPool.Get().(*Path)
		defer func() {
			path.Reset()
			thePathPool.Put(path)
		}()
		path.addArc(cx, cy, r, 0, 2*math.Pi, Clockwise)
		path.Close()
		strokeOp := &StrokeOptions{}
		strokeOp.Width = float32(strokeWidth)
		strokeOp.LineJoin = LineJoinRound
		drawOp := &DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(clr)
		StrokePath(dst, path, strokeOp, drawOp)
		return
	}

	if strokeWidth <= 0 {
		return
	}

	if strokeWidth >= 2*r {
		fillCircle(dst, cx, cy, r+strokeWidth/2, clr, false)
		return
	}

	outerRadius := r + strokeWidth/2
	innerRadius := r - strokeWidth/2
	count := circleVertexCount(outerRadius)
	if count == 0 {
		return
	}

	// Use a regular DrawTriangles32 for batching.
	cr, cg, cb, ca := clr.RGBA()
	crf := float32(cr) / 0xffff
	cgf := float32(cg) / 0xffff
	cbf := float32(cb) / 0xffff
	caf := float32(ca) / 0xffff
	useCachedVerticesAndIndicesForUtil(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		for i := range count {
			angle := float64(i) * (2 * math.Pi / float64(count))
			sin, cos := math.Sincos(angle)
			x0 := float32(cx + outerRadius*cos)
			y0 := float32(cy + outerRadius*sin)
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
			x1 := float32(cx + innerRadius*cos)
			y1 := float32(cy + innerRadius*sin)
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

// StrokePath strokes the specified path with the specified options.
func StrokePath(dst *ebiten.Image, path *Path, strokeOptions *StrokeOptions, drawPathOptions *DrawPathOptions) {
	if path == nil || len(path.subPaths) == 0 {
		return
	}

	if strokeOptions == nil {
		strokeOptions = &StrokeOptions{}
	}
	stroke := thePathPool.Get().(*Path)
	defer func() {
		stroke.Reset()
		thePathPool.Put(stroke)
	}()
	op := &AddStrokeOptions{}
	op.StrokeOptions = *strokeOptions
	stroke.AddStroke(path, op)
	FillPath(dst, stroke, nil, drawPathOptions)
}
