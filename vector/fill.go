// Copyright 2025 The Ebitengine Authors
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
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

type offsetAndColor struct {
	offsetX    float32
	offsetY    float32
	colorR     float32
	colorG     float32
	colorB     float32
	colorA     float32
	imageIndex int
}

var (
	offsetAndColorsNonAA = []offsetAndColor{
		{
			offsetX: 0,
			offsetY: 0,
			colorR:  1,
			colorG:  0,
			colorB:  0,
			colorA:  0,
		},
	}

	// https://learn.microsoft.com/en-us/windows/win32/api/d3d11/ne-d3d11-d3d11_standard_multisample_quality_levels
	offsetAndColorsAA = []offsetAndColor{
		{
			offsetX:    1.0 / 16.0,
			offsetY:    -3.0 / 16.0,
			colorR:     1,
			colorG:     0,
			colorB:     0,
			colorA:     0,
			imageIndex: 0,
		},
		{
			offsetX:    -1.0 / 16.0,
			offsetY:    3.0 / 16.0,
			colorR:     0,
			colorG:     1,
			colorB:     0,
			colorA:     0,
			imageIndex: 0,
		},
		{
			offsetX:    5.0 / 16.0,
			offsetY:    1.0 / 16.0,
			colorR:     0,
			colorG:     0,
			colorB:     1,
			colorA:     0,
			imageIndex: 0,
		},
		{
			offsetX:    -3.0 / 16.0,
			offsetY:    -5.0 / 16.0,
			colorR:     0,
			colorG:     0,
			colorB:     0,
			colorA:     1,
			imageIndex: 0,
		},
		{
			offsetX:    -5.0 / 16.0,
			offsetY:    5.0 / 16.0,
			colorR:     1,
			colorG:     0,
			colorB:     0,
			colorA:     0,
			imageIndex: 1,
		},
		{
			offsetX:    -7.0 / 16.0,
			offsetY:    -1.0 / 16.0,
			colorR:     0,
			colorG:     1,
			colorB:     0,
			colorA:     0,
			imageIndex: 1,
		},
		{
			offsetX:    3.0 / 16.0,
			offsetY:    7.0 / 16.0,
			colorR:     0,
			colorG:     0,
			colorB:     1,
			colorA:     0,
			imageIndex: 1,
		},
		{
			offsetX:    7.0 / 16.0,
			offsetY:    -7.0 / 16.0,
			colorR:     0,
			colorG:     0,
			colorB:     0,
			colorA:     1,
			imageIndex: 1,
		},
	}
)

// theStencilBufferAtlasImage is an atlas image for stencil buffer images.
// Stencil buffer images are integrated into this image for batching.
var theStencilBufferAtlasImage *ebiten.Image

type fillPathsState struct {
	paths  []*Path
	colors []color.Color

	stencilBufferImages      []*ebiten.Image
	pathBounds               []image.Rectangle
	stencilBufferImageBounds []image.Rectangle

	vertices []ebiten.Vertex
	indices  []uint32

	antialias bool
	fillRule  FillRule
}

func roundUpAtlasSize(size int) int {
	if size < 16 {
		return 16
	}
	return int(math.Ceil(math.Pow(1.5, math.Ceil(math.Log(float64(size))/math.Log(1.5)))))
}

func (f *fillPathsState) appendStencilBufferImages(images []*ebiten.Image, dstBounds image.Rectangle) []*ebiten.Image {
	if len(f.pathBounds) == 0 {
		return images
	}

	boundsCount := len(f.pathBounds)
	if f.antialias {
		boundsCount *= 2
	}
	f.stencilBufferImageBounds = f.stencilBufferImageBounds[:0]
	f.stencilBufferImageBounds = slices.Grow(f.stencilBufferImageBounds, boundsCount)

	var atlasWidth int
	var atlasHeight int
	for _, pb := range f.pathBounds {
		pb = pb.Intersect(dstBounds)
		// Extend the bounds a little bit to avoid creating an image too often.
		w := roundUpAtlasSize(pb.Dx())
		h := roundUpAtlasSize(pb.Dy())
		imageSize := image.Pt(w, h)
		imageBounds := image.Rect(0, atlasHeight, imageSize.X, atlasHeight+imageSize.Y)
		f.stencilBufferImageBounds = append(f.stencilBufferImageBounds, imageBounds)
		if f.antialias {
			f.stencilBufferImageBounds = append(f.stencilBufferImageBounds, imageBounds.Add(image.Pt(imageSize.X, 0)))
		}

		atlasWidth = max(atlasWidth, imageSize.X)
		if f.antialias {
			atlasWidth = max(atlasWidth, imageSize.X*2)
		}
		atlasHeight += imageSize.Y
	}
	if theStencilBufferAtlasImage != nil {
		if theStencilBufferAtlasImage.Bounds().Dx() < atlasWidth || theStencilBufferAtlasImage.Bounds().Dy() < atlasHeight {
			atlasWidth = max(atlasWidth, theStencilBufferAtlasImage.Bounds().Dx())
			atlasHeight = max(atlasHeight, theStencilBufferAtlasImage.Bounds().Dy())
			theStencilBufferAtlasImage.Deallocate()
			theStencilBufferAtlasImage = nil
		}
	}
	if theStencilBufferAtlasImage == nil {
		theStencilBufferAtlasImage = ebiten.NewImage(atlasWidth, atlasHeight)
	} else {
		theStencilBufferAtlasImage.Clear()
	}

	for _, b := range f.stencilBufferImageBounds {
		images = append(images, theStencilBufferAtlasImage.SubImage(b).(*ebiten.Image))
	}
	return images
}

func floor(x float32) int {
	return int(math.Floor(float64(x)))
}

func ceil(x float32) int {
	return int(math.Ceil(float64(x)))
}

func pathBounds(path *Path) image.Rectangle {
	// Note that (image.Rectangle).Union doesn't work well with empty rectangles.
	totalMinX := float32(math.Inf(1))
	totalMinY := float32(math.Inf(1))
	totalMaxX := float32(math.Inf(-1))
	totalMaxY := float32(math.Inf(-1))

	for _, subPath := range path.subPaths {
		if !subPath.isValid() {
			continue
		}

		minX := float32(math.Inf(1))
		minY := float32(math.Inf(1))
		maxX := float32(math.Inf(-1))
		maxY := float32(math.Inf(-1))
		cur := subPath.start
		for _, op := range subPath.ops {
			switch op.typ {
			case opTypeLineTo:
				minX = min(minX, cur.x, op.p1.x)
				minY = min(minY, cur.y, op.p1.y)
				maxX = max(maxX, cur.x, op.p1.x)
				maxY = max(maxY, cur.y, op.p1.y)
				cur = op.p1
			case opTypeQuadTo:
				// The candidates are the two control points on the edges (cur and op.p2), and an extremum point.
				// B(t) = (1-t)*(1-t)*p0 + 2*(1-t)*t*p1 + t*t*p2
				// B'(t) = 2*(1-t)*(p1-p0) + 2*t*(p2-p1)
				// B'(t) = 0 <=> t = (p0-p1) / (p0-2*p1+p2)
				// Avoid an extreme denominator for precision.
				if denom := cur.x - 2*op.p1.x + op.p2.x; abs(denom) >= 1.0/16.0 {
					if t := (cur.x - op.p1.x) / denom; t > 0 && t < 1 {
						ex := (1-t)*(1-t)*cur.x + 2*t*(1-t)*op.p1.x + t*t*op.p2.x
						minX = min(minX, cur.x, ex, op.p2.x)
						maxX = max(maxX, cur.x, ex, op.p2.x)
					} else {
						minX = min(minX, cur.x, op.p2.x)
						maxX = max(maxX, cur.x, op.p2.x)
					}
				} else {
					// The curve is almost linear. Include all the points for safety.
					minX = min(minX, cur.x, op.p1.x, op.p2.x)
					maxX = max(maxX, cur.x, op.p1.x, op.p2.x)
				}
				if denom := cur.y - 2*op.p1.y + op.p2.y; abs(denom) >= 1.0/16.0 {
					if t := (cur.y - op.p1.y) / denom; t > 0 && t < 1 {
						ex := (1-t)*(1-t)*cur.y + 2*t*(1-t)*op.p1.y + t*t*op.p2.y
						minY = min(minY, cur.y, ex, op.p2.y)
						maxY = max(maxY, cur.y, ex, op.p2.y)
					} else {
						minY = min(minY, cur.y, op.p2.y)
						maxY = max(maxY, cur.y, op.p2.y)
					}
				} else {
					minY = min(minY, cur.y, op.p1.y, op.p2.y)
					maxY = max(maxY, cur.y, op.p1.y, op.p2.y)
				}
				cur = op.p2
			}
		}
		totalMinX = min(totalMinX, minX)
		totalMinY = min(totalMinY, minY)
		totalMaxX = max(totalMaxX, maxX)
		totalMaxY = max(totalMaxY, maxY)
	}
	if totalMinX >= totalMaxX || totalMinY >= totalMaxY {
		return image.Rectangle{}
	}
	return image.Rect(floor(totalMinX), floor(totalMinY), ceil(totalMaxX), ceil(totalMaxY))
}

func (f *fillPathsState) reset() {
	for _, p := range f.paths {
		p.Reset()
	}
	f.paths = f.paths[:0]
	f.colors = slices.Delete(f.colors, 0, len(f.colors))
}

func (f *fillPathsState) addPath(path *Path, clr color.Color) {
	if path == nil {
		return
	}
	f.paths = slices.Grow(f.paths, 1)[:len(f.paths)+1]
	if f.paths[len(f.paths)-1] == nil {
		f.paths[len(f.paths)-1] = &Path{}
	}
	dst := f.paths[len(f.paths)-1]
	dst.addSubPaths(len(path.subPaths))
	for i, subPath := range path.subPaths {
		dst.subPaths[i].start = subPath.start
		dst.subPaths[i].closed = subPath.closed
		dst.subPaths[i].ops = slices.Grow(dst.subPaths[i].ops, len(subPath.ops))[:len(subPath.ops)]
		copy(dst.subPaths[i].ops, subPath.ops)
	}
	f.colors = append(f.colors, clr)
}

// fillPaths fills the specified path with the specified color.
func (f *fillPathsState) fillPaths(dst *ebiten.Image) {
	if len(f.paths) != len(f.colors) {
		panic("vector: the number of paths and colors must be the same")
	}

	if stencilBufferFillShader == nil {
		s, err := ebiten.NewShader([]byte(stencilBufferFillShaderSrc))
		if err != nil {
			panic(err)
		}
		stencilBufferFillShader = s
	}
	if stencilBufferBezierShader == nil {
		s, err := ebiten.NewShader([]byte(stencilBufferBezierShaderSrc))
		if err != nil {
			panic(err)
		}
		stencilBufferBezierShader = s
	}
	if !f.antialias && f.fillRule == FillRuleNonZero {
		if stencilBufferNonZeroShader == nil {
			s, err := ebiten.NewShader([]byte(stencilBufferNonZeroShaderSrc))
			if err != nil {
				panic(err)
			}
			stencilBufferNonZeroShader = s
		}
	}
	if f.antialias && f.fillRule == FillRuleNonZero {
		if stencilBufferNonZeroAAShader == nil {
			s, err := ebiten.NewShader([]byte(stencilBufferNonZeroAAShaderSrc))
			if err != nil {
				panic(err)
			}
			stencilBufferNonZeroAAShader = s
		}
	}
	if !f.antialias && f.fillRule == FillRuleEvenOdd {
		if stencilBufferEvenOddShader == nil {
			s, err := ebiten.NewShader([]byte(stencilBufferEvenOddShaderSrc))
			if err != nil {
				panic(err)
			}
			stencilBufferEvenOddShader = s
		}
	}
	if f.antialias && f.fillRule == FillRuleEvenOdd {
		if stencilBufferEvenOddAAShader == nil {
			s, err := ebiten.NewShader([]byte(stencilBufferEvenOddAAShaderSrc))
			if err != nil {
				panic(err)
			}
			stencilBufferEvenOddAAShader = s
		}
	}

	vs := f.vertices[:0]
	is := f.indices[:0]
	defer func() {
		f.vertices = vs
		f.indices = is
	}()

	f.pathBounds = f.pathBounds[:0]
	for _, path := range f.paths {
		f.pathBounds = append(f.pathBounds, pathBounds(path))
	}
	f.stencilBufferImages = f.appendStencilBufferImages(f.stencilBufferImages[:0], dst.Bounds())

	offsetAndColors := offsetAndColorsNonAA
	if f.antialias {
		offsetAndColors = offsetAndColorsAA
	}

	// First, render the polygons roughly.
	for i, path := range f.paths {
		if path == nil {
			continue
		}

		for _, oac := range offsetAndColors {
			vs = vs[:0]
			is = is[:0]

			stencilBufferImageIndex := i
			if f.antialias {
				stencilBufferImageIndex *= 2
			}

			stencilBufferImage := f.stencilBufferImages[stencilBufferImageIndex+oac.imageIndex]
			dstOffsetX := float32(-f.pathBounds[i].Min.X + stencilBufferImage.Bounds().Min.X - max(0, dst.Bounds().Min.X-f.pathBounds[i].Min.X))
			dstOffsetY := float32(-f.pathBounds[i].Min.Y + stencilBufferImage.Bounds().Min.Y - max(0, dst.Bounds().Min.Y-f.pathBounds[i].Min.Y))

			for _, subPath := range path.subPaths {
				if !subPath.isValid() {
					continue
				}

				// Add an origin point. Any position works in theory.
				// Use the sub-path's start point. Using one of the sub-path's points can reduce triangles.
				// Also, this point should be close to the other points and then triangle overlaps are reduced.
				// TODO: Use a better position like the center of the sub-path.
				originIdx := uint32(len(vs))
				cur := subPath.start
				vs = append(vs, ebiten.Vertex{
					DstX:   cur.x + oac.offsetX + dstOffsetX,
					DstY:   cur.y + oac.offsetY + dstOffsetY,
					ColorR: oac.colorR,
					ColorG: oac.colorG,
					ColorB: oac.colorB,
					ColorA: oac.colorA,
				})

				for _, op := range subPath.ops {
					switch op.typ {
					case opTypeLineTo:
						idx := uint32(len(vs))
						vs = append(vs,
							ebiten.Vertex{
								DstX:   cur.x + oac.offsetX + dstOffsetX,
								DstY:   cur.y + oac.offsetY + dstOffsetY,
								ColorR: oac.colorR,
								ColorG: oac.colorG,
								ColorB: oac.colorB,
								ColorA: oac.colorA,
							},
							ebiten.Vertex{
								DstX:   op.p1.x + oac.offsetX + dstOffsetX,
								DstY:   op.p1.y + oac.offsetY + dstOffsetY,
								ColorR: oac.colorR,
								ColorG: oac.colorG,
								ColorB: oac.colorB,
								ColorA: oac.colorA,
							})
						is = append(is, idx, originIdx, idx+1)
						cur = op.p1
					case opTypeQuadTo:
						idx := uint32(len(vs))
						vs = append(vs,
							ebiten.Vertex{
								DstX:   cur.x + oac.offsetX + dstOffsetX,
								DstY:   cur.y + oac.offsetY + dstOffsetY,
								ColorR: oac.colorR,
								ColorG: oac.colorG,
								ColorB: oac.colorB,
								ColorA: oac.colorA,
							},
							ebiten.Vertex{
								DstX:   op.p2.x + oac.offsetX + dstOffsetX,
								DstY:   op.p2.y + oac.offsetY + dstOffsetY,
								ColorR: oac.colorR,
								ColorG: oac.colorG,
								ColorB: oac.colorB,
								ColorA: oac.colorA,
							})
						is = append(is, idx, originIdx, idx+1)
						cur = op.p2
					}
				}
				// If the sub-path is not closed, add a supplementary line.
				if !subPath.closed {
					idx := uint32(len(vs))
					vs = append(vs,
						ebiten.Vertex{
							DstX:   cur.x + oac.offsetX + dstOffsetX,
							DstY:   cur.y + oac.offsetY + dstOffsetY,
							ColorR: oac.colorR,
							ColorG: oac.colorG,
							ColorB: oac.colorB,
							ColorA: oac.colorA,
						},
						ebiten.Vertex{
							DstX:   subPath.start.x + oac.offsetX + dstOffsetX,
							DstY:   subPath.start.y + oac.offsetY + dstOffsetY,
							ColorR: oac.colorR,
							ColorG: oac.colorG,
							ColorB: oac.colorB,
							ColorA: oac.colorA,
						})
					is = append(is, idx, originIdx, idx+1)
				}
			}
			op := &ebiten.DrawTrianglesShaderOptions{}
			op.Blend = ebiten.BlendLighter
			stencilBufferImage.DrawTrianglesShader32(vs, is, stencilBufferFillShader, op)
		}
	}

	// Second, render the bezier curves.
	for i, path := range f.paths {
		if path == nil {
			continue
		}

		for _, oac := range offsetAndColors {
			vs = vs[:0]
			is = is[:0]

			stencilBufferImageIndex := i
			if f.antialias {
				stencilBufferImageIndex *= 2
			}
			stencilBufferImage := f.stencilBufferImages[stencilBufferImageIndex+oac.imageIndex]
			dstOffsetX := float32(-f.pathBounds[i].Min.X + stencilBufferImage.Bounds().Min.X - max(0, dst.Bounds().Min.X-f.pathBounds[i].Min.X))
			dstOffsetY := float32(-f.pathBounds[i].Min.Y + stencilBufferImage.Bounds().Min.Y - max(0, dst.Bounds().Min.Y-f.pathBounds[i].Min.Y))
			for _, subPath := range path.subPaths {
				if !subPath.isValid() {
					continue
				}

				cur := subPath.start
				for _, op := range subPath.ops {
					switch op.typ {
					case opTypeLineTo:
						cur = op.p1
					case opTypeQuadTo:
						idx := uint32(len(vs))
						vs = append(vs,
							ebiten.Vertex{
								DstX:    cur.x + oac.offsetX + dstOffsetX,
								DstY:    cur.y + oac.offsetY + dstOffsetY,
								ColorR:  oac.colorR,
								ColorG:  oac.colorG,
								ColorB:  oac.colorB,
								ColorA:  oac.colorA,
								Custom0: 0, // u for Loop-Blinn algorithm
								Custom1: 0, // v for Loop-Blinn algorithm
							},
							ebiten.Vertex{
								DstX:    op.p1.x + oac.offsetX + dstOffsetX,
								DstY:    op.p1.y + oac.offsetY + dstOffsetY,
								ColorR:  oac.colorR,
								ColorG:  oac.colorG,
								ColorB:  oac.colorB,
								ColorA:  oac.colorA,
								Custom0: 0.5,
								Custom1: 0,
							},
							ebiten.Vertex{
								DstX:    op.p2.x + oac.offsetX + dstOffsetX,
								DstY:    op.p2.y + oac.offsetY + dstOffsetY,
								ColorR:  oac.colorR,
								ColorG:  oac.colorG,
								ColorB:  oac.colorB,
								ColorA:  oac.colorA,
								Custom0: 1,
								Custom1: 1,
							})
						is = append(is, idx, idx+1, idx+2)
						cur = op.p2
					}
				}
			}
			op := &ebiten.DrawTrianglesShaderOptions{}
			op.Blend = ebiten.BlendLighter
			stencilBufferImage.DrawTrianglesShader32(vs, is, stencilBufferBezierShader, op)
		}
	}

	// Render the stencil buffer with the specified color.
	for i, path := range f.paths {
		if path == nil {
			continue
		}

		stencilBufferImageIndex := i
		if f.antialias {
			stencilBufferImageIndex *= 2
		}
		stencilImage := f.stencilBufferImages[stencilBufferImageIndex]
		pathBounds := f.pathBounds[i]

		vs = vs[:0]
		is = is[:0]
		dstOffsetX := max(0, dst.Bounds().Min.X-f.pathBounds[i].Min.X)
		dstOffsetY := max(0, dst.Bounds().Min.Y-f.pathBounds[i].Min.Y)
		var clrR, clrG, clrB, clrA float32
		r, g, b, a := f.colors[i].RGBA()
		clrR = float32(r) / 0xffff
		clrG = float32(g) / 0xffff
		clrB = float32(b) / 0xffff
		clrA = float32(a) / 0xffff
		vs = append(vs,
			ebiten.Vertex{
				DstX:    float32(pathBounds.Min.X + dstOffsetX),
				DstY:    float32(pathBounds.Min.Y + dstOffsetY),
				SrcX:    float32(stencilImage.Bounds().Min.X),
				SrcY:    float32(stencilImage.Bounds().Min.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: float32(stencilImage.Bounds().Dx()),
				Custom1: 0,
			},
			ebiten.Vertex{
				DstX:    float32(pathBounds.Min.X + stencilImage.Bounds().Dx() + dstOffsetX),
				DstY:    float32(pathBounds.Min.Y + dstOffsetY),
				SrcX:    float32(stencilImage.Bounds().Max.X),
				SrcY:    float32(stencilImage.Bounds().Min.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: float32(stencilImage.Bounds().Dx()),
				Custom1: 0,
			},
			ebiten.Vertex{
				DstX:    float32(pathBounds.Min.X + dstOffsetX),
				DstY:    float32(pathBounds.Min.Y + stencilImage.Bounds().Dy() + dstOffsetY),
				SrcX:    float32(stencilImage.Bounds().Min.X),
				SrcY:    float32(stencilImage.Bounds().Max.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: float32(stencilImage.Bounds().Dx()),
				Custom1: 0,
			},
			ebiten.Vertex{
				DstX:    float32(pathBounds.Min.X + stencilImage.Bounds().Dx() + dstOffsetX),
				DstY:    float32(pathBounds.Min.Y + stencilImage.Bounds().Dy() + dstOffsetY),
				SrcX:    float32(stencilImage.Bounds().Max.X),
				SrcY:    float32(stencilImage.Bounds().Max.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: float32(stencilImage.Bounds().Dx()),
				Custom1: 0,
			})
		is = append(is, 0, 1, 2, 1, 2, 3)

		op := &ebiten.DrawTrianglesShaderOptions{}
		op.Images[0] = stencilImage
		var shader *ebiten.Shader
		switch f.fillRule {
		case FillRuleNonZero:
			if f.antialias {
				shader = stencilBufferNonZeroAAShader
			} else {
				shader = stencilBufferNonZeroShader
			}
		case FillRuleEvenOdd:
			if f.antialias {
				shader = stencilBufferEvenOddAAShader
			} else {
				shader = stencilBufferEvenOddShader
			}
		}
		dst.DrawTrianglesShader32(vs, is, shader, op)
	}
}
