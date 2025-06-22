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
	cachedVertices []ebiten.Vertex
	cachedIndices  []uint32
	cacheM         sync.Mutex
)

func useCachedVerticesAndIndices(fn func([]ebiten.Vertex, []uint32) (vs []ebiten.Vertex, is []uint32)) {
	cacheM.Lock()
	defer cacheM.Unlock()
	cachedVertices, cachedIndices = fn(cachedVertices[:0], cachedIndices[:0])
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
	var path Path
	path.MoveTo(x0, y0)
	path.LineTo(x1, y1)
	strokeOp := &StrokeOptions{}
	strokeOp.Width = strokeWidth

	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		vs, is = path.AppendVerticesAndIndicesForStroke32(vs, is, strokeOp)
		drawVerticesForUtil(dst, vs, is, clr, antialias, ebiten.FillRuleFillAll)
		return vs, is
	})
}

// DrawFilledRect fills a rectangle with the specified width and color.
func DrawFilledRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color, antialias bool) {
	var path Path
	path.MoveTo(x, y)
	path.LineTo(x, y+height)
	path.LineTo(x+width, y+height)
	path.LineTo(x+width, y)

	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		vs, is = path.AppendVerticesAndIndicesForFilling32(vs, is)
		drawVerticesForUtil(dst, vs, is, clr, antialias, ebiten.FillRuleFillAll)
		return vs, is
	})
}

// StrokeRect strokes a rectangle with the specified width and color.
//
// clr has be to be a solid (non-transparent) color.
func StrokeRect(dst *ebiten.Image, x, y, width, height float32, strokeWidth float32, clr color.Color, antialias bool) {
	var path Path
	path.MoveTo(x, y)
	path.LineTo(x, y+height)
	path.LineTo(x+width, y+height)
	path.LineTo(x+width, y)
	path.Close()

	strokeOp := &StrokeOptions{}
	strokeOp.Width = strokeWidth
	strokeOp.MiterLimit = 10

	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		vs, is = path.AppendVerticesAndIndicesForStroke32(vs, is, strokeOp)
		drawVerticesForUtil(dst, vs, is, clr, antialias, ebiten.FillRuleFillAll)
		return vs, is
	})
}

// DrawFilledCircle fills a circle with the specified center position (cx, cy), the radius (r), width and color.
func DrawFilledCircle(dst *ebiten.Image, cx, cy, r float32, clr color.Color, antialias bool) {
	var path Path
	path.Arc(cx, cy, r, 0, 2*math.Pi, Clockwise)

	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		vs, is = path.AppendVerticesAndIndicesForFilling32(vs, is)
		drawVerticesForUtil(dst, vs, is, clr, antialias, ebiten.FillRuleFillAll)
		return vs, is
	})
}

// StrokeCircle strokes a circle with the specified center position (cx, cy), the radius (r), width and color.
//
// clr has be to be a solid (non-transparent) color.
func StrokeCircle(dst *ebiten.Image, cx, cy, r float32, strokeWidth float32, clr color.Color, antialias bool) {
	var path Path
	path.Arc(cx, cy, r, 0, 2*math.Pi, Clockwise)
	path.Close()

	strokeOp := &StrokeOptions{}
	strokeOp.Width = strokeWidth

	useCachedVerticesAndIndices(func(vs []ebiten.Vertex, is []uint32) ([]ebiten.Vertex, []uint32) {
		vs, is = path.AppendVerticesAndIndicesForStroke32(vs, is, strokeOp)
		drawVerticesForUtil(dst, vs, is, clr, antialias, ebiten.FillRuleFillAll)
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

func DrawFilledPath(dst *ebiten.Image, path *Path, clr color.Color, antialias bool, fillRule FillRule) {
	// Protect this function for compatibility.
	// TODO: Remove this in v3?
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
	if s.antialias != antialias || s.fillRule != fillRule {
		s.fillPaths(dst)
		s.reset()
	}
	s.antialias = antialias
	s.fillRule = fillRule
	s.addPath(path, clr)

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

func StrokePath(dst *ebiten.Image, path *Path, clr color.Color, antialias bool, options *StrokeOptions) {
	var stroke Path
	op := &AddPathStrokeOptions{}
	op.StrokeOptions = *options
	stroke.AddPathStroke(path, op)
	DrawFilledPath(dst, &stroke, clr, antialias, FillRuleNonZero)
}

//go:linkname addUsageCallback github.com/hajimehoshi/ebiten/v2.addUsageCallback
func addUsageCallback(img *ebiten.Image, fn func()) int64

//go:linkname removeUsageCallback github.com/hajimehoshi/ebiten/v2.removeUsageCallback
func removeUsageCallback(img *ebiten.Image, token int64)
