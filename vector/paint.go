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

package vector

import (
	"image"
	"math"
	"weak"

	"github.com/hajimehoshi/ebiten/v2"
)

// Paint supplies the color of a filled or stroked path.
// Pointers to [ImagePaint] implement Paint.
type Paint interface {
	isPaint()
}

// ImagePaint paints a path with an image, sampling outside the image as transparent.
type ImagePaint struct {
	// Image is the source image. It must not be nil, disposed, or share its original image with the destination.
	// Changes to its pixels after drawing returns do not affect the drawing.
	Image *ebiten.Image

	// GeoM maps image-local coordinates to destination coordinates.
	// The source image's top-left is (0, 0), including for a sub-image.
	// The zero value is the identity matrix. A non-finite matrix, or one that cannot be
	// inverted in both directions within GeoM's precision, causes a panic.
	GeoM ebiten.GeoM

	// Filter is the sampling filter: FilterNearest (the default) or FilterLinear.
	// Other values cause a panic. Linear sampling is bilinear without mipmaps.
	Filter ebiten.Filter
}

func (*ImagePaint) isPaint() {}

type imagePaint struct {
	source  *ebiten.Image
	bounds  image.Rectangle
	inverse ebiten.GeoM
	filter  ebiten.Filter
}

func snapshotPaint(paint Paint, dst *ebiten.Image) imagePaint {
	if paint == nil {
		return imagePaint{}
	}
	p, ok := paint.(*ImagePaint)
	if !ok || p == nil || p.Image == nil {
		panic("vector: invalid paint")
	}
	if p.Filter != ebiten.FilterNearest && p.Filter != ebiten.FilterLinear {
		panic("vector: invalid image paint filter")
	}
	source := theImageBridge.OriginalImage(p.Image)
	if source == theImageBridge.OriginalImage(dst) {
		panic("vector: paint source and destination must differ")
	}
	inverse := p.GeoM
	if !inverse.IsInvertible() {
		panic("vector: image paint matrix must be invertible")
	}
	inverse.Invert()
	if !inverse.IsInvertible() {
		panic("vector: image paint inverse must be invertible")
	}
	for row := range 2 {
		for column := range 3 {
			v := inverse.Element(row, column)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				panic("vector: image paint matrix must be finite")
			}
			v = p.GeoM.Element(row, column)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				panic("vector: image paint matrix must be finite")
			}
		}
	}
	return imagePaint{
		source:  source,
		bounds:  p.Image.Bounds(),
		inverse: inverse,
		filter:  p.Filter,
	}
}

// A source has no queued vector writes while it has paint consumers. Before
// queuing another write, FillPath flushes those consumers. Image operations
// flush them through the source's usage callback. Protected by theFillPathM.
type paintSource struct {
	token        int64
	destinations map[weak.Pointer[ebiten.Image]]struct{}
}

var thePaintSources = map[weak.Pointer[ebiten.Image]]*paintSource{}

func registerPaintSource(src, dst *ebiten.Image) {
	key := weak.Make(src)
	s := thePaintSources[key]
	if s == nil {
		s = &paintSource{
			token:        theImageBridge.AddUsage(src, paintSourceCallback),
			destinations: map[weak.Pointer[ebiten.Image]]struct{}{},
		}
		thePaintSources[key] = s
	}
	s.destinations[weak.Make(dst)] = struct{}{}
}

func unregisterPaintSource(src *ebiten.Image, dst weak.Pointer[ebiten.Image]) {
	if src == nil {
		return
	}
	key := weak.Make(src)
	s := thePaintSources[key]
	if s == nil {
		return
	}
	delete(s.destinations, dst)
	if len(s.destinations) == 0 {
		theImageBridge.RemoveUsage(src, s.token)
		delete(thePaintSources, key)
	}
}

func paintSourceCallback(src *ebiten.Image) {
	theFillPathM.Lock()
	defer theFillPathM.Unlock()
	flushPaintSource(src)
}

func flushPaintSource(src *ebiten.Image) {
	key := weak.Make(src)
	s := thePaintSources[key]
	if s == nil {
		return
	}
	// Remove the shared source callback before any draw samples this image.
	// Each destination has only this source, so rendering cannot invoke another
	// source callback while the vector mutex is held.
	theImageBridge.RemoveUsage(src, s.token)
	delete(thePaintSources, key)
	for destination := range s.destinations {
		if dst := destination.Value(); dst != nil {
			flushFillPaths(dst)
		}
	}
}

func (p *imagePaint) draw(dst *ebiten.Image, vs []ebiten.Vertex, is []uint32, coverage *ebiten.Image, op *ebiten.DrawTrianglesShaderOptions, rule FillRule, aa, clipped bool) {
	// Snapshot the region independently of a caller-owned recyclable sub-image.
	src := p.source.RecyclableSubImage(p.bounds)
	defer src.Recycle()
	stencil := theImageBridge.OriginalImage(coverage)
	stencilBounds := stencil.Bounds()
	for i := range vs {
		v := &vs[i]
		x, y := p.inverse.Apply(float64(v.DstX), float64(v.DstY))
		offsetX, offsetY := v.Custom0, v.Custom1
		v.Custom0 = v.SrcX - float32(stencilBounds.Min.X)
		v.Custom1 = v.SrcY - float32(stencilBounds.Min.Y)
		v.Custom2, v.Custom3 = offsetX, offsetY
		v.SrcX = float32(x) + float32(p.bounds.Min.X)
		v.SrcY = float32(y) + float32(p.bounds.Min.Y)
	}
	// The entire stencil image keeps source-region uniforms stable between paths.
	// Vertex data addresses only the stencil regions assigned to this path.
	op.Images[0] = src
	op.Images[1] = stencil
	dst.DrawTrianglesShader32(vs, is, ensureImagePaintShader(rule, aa, clipped, p.filter), op)
}
