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

var (
	// cachedStencilBufferAtlasImage is an atlas image for stencil buffer images.
	// Stencil buffer images are integrated into this image for batching.
	cachedStencilBufferAtlasImage *ebiten.Image

	cachedStencilBufferImages []*ebiten.Image
	cachedPathBounds          []image.Rectangle
)

func appendStencilBufferImages(images []*ebiten.Image, pathBounds []image.Rectangle, dstBounds image.Rectangle, antialias bool) []*ebiten.Image {
	// This function must be protected by cacheM.

	if len(pathBounds) == 0 {
		return images
	}

	boundsCount := len(pathBounds)
	if antialias {
		boundsCount *= 2
	}
	bounds := make([]image.Rectangle, 0, boundsCount)

	var atlasWidth int
	var atlasHeight int
	for _, pb := range pathBounds {
		pb = pb.Intersect(dstBounds)
		// Extend the bounds a little bit to avoid creating an image too often.
		imageSize := image.Pt((pb.Dx()+15)/16*16, (pb.Dy()+15)/16*16)
		imageBounds := image.Rect(0, atlasHeight, imageSize.X, atlasHeight+imageSize.Y)
		bounds = append(bounds, imageBounds)
		if antialias {
			bounds = append(bounds, imageBounds.Add(image.Pt(imageSize.X, 0)))
		}

		atlasWidth = max(atlasWidth, imageSize.X)
		if antialias {
			atlasWidth = max(atlasWidth, imageSize.X*2)
		}
		atlasHeight += imageSize.Y
	}
	if cachedStencilBufferAtlasImage != nil {
		if cachedStencilBufferAtlasImage.Bounds().Dx() < atlasWidth || cachedStencilBufferAtlasImage.Bounds().Dy() < atlasHeight {
			atlasWidth = max(atlasWidth, cachedStencilBufferAtlasImage.Bounds().Dx())
			atlasHeight = max(atlasHeight, cachedStencilBufferAtlasImage.Bounds().Dy())
			cachedStencilBufferAtlasImage.Deallocate()
			cachedStencilBufferAtlasImage = nil
		}
	}
	if cachedStencilBufferAtlasImage == nil {
		cachedStencilBufferAtlasImage = ebiten.NewImage(atlasWidth, atlasHeight)
	} else {
		cachedStencilBufferAtlasImage.Clear()
	}

	for _, b := range bounds {
		images = append(images, cachedStencilBufferAtlasImage.SubImage(b).(*ebiten.Image))
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
				minX = min(minX, cur.x, op.p1.x, op.p2.x)
				minY = min(minY, cur.y, op.p1.y, op.p2.y)
				maxX = max(maxX, cur.x, op.p1.x, op.p2.x)
				maxY = max(maxY, cur.y, op.p1.y, op.p2.y)
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

// FillPaths fills the specified path with the specified color.
func FillPaths(dst *ebiten.Image, paths []*Path, colors []color.Color, antialias bool, fillRule FillRule) {
	if len(paths) != len(colors) {
		panic("vector: the number of paths and colors must be the same")
	}

	cacheM.Lock()
	defer cacheM.Unlock()

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
	if !antialias && fillRule == FillRuleNonZero {
		if stencilBufferNonZeroShader == nil {
			s, err := ebiten.NewShader([]byte(stencilBufferNonZeroShaderSrc))
			if err != nil {
				panic(err)
			}
			stencilBufferNonZeroShader = s
		}
	}
	if antialias && fillRule == FillRuleNonZero {
		if stencilBufferNonZeroAAShader == nil {
			s, err := ebiten.NewShader([]byte(stencilBufferNonZeroAAShaderSrc))
			if err != nil {
				panic(err)
			}
			stencilBufferNonZeroAAShader = s
		}
	}
	if !antialias && fillRule == FillRuleEvenOdd {
		if stencilBufferEvenOddShader == nil {
			s, err := ebiten.NewShader([]byte(stencilBufferEvenOddShaderSrc))
			if err != nil {
				panic(err)
			}
			stencilBufferEvenOddShader = s
		}
	}
	if antialias && fillRule == FillRuleEvenOdd {
		if stencilBufferEvenOddAAShader == nil {
			s, err := ebiten.NewShader([]byte(stencilBufferEvenOddAAShaderSrc))
			if err != nil {
				panic(err)
			}
			stencilBufferEvenOddAAShader = s
		}
	}

	vs := cachedVertices[:0]
	is := cachedIndices[:0]
	defer func() {
		cachedVertices = vs
		cachedIndices = is
	}()

	cachedPathBounds = cachedPathBounds[:0]
	for _, path := range paths {
		cachedPathBounds = append(cachedPathBounds, pathBounds(path))
	}
	cachedStencilBufferImages = appendStencilBufferImages(cachedStencilBufferImages[:0], cachedPathBounds, dst.Bounds(), antialias)

	offsetAndColors := offsetAndColorsNonAA
	if antialias {
		offsetAndColors = offsetAndColorsAA
	}

	// First, render the polygons roughly.
	for i, path := range paths {
		if path == nil {
			continue
		}

		for _, oac := range offsetAndColors {
			vs = vs[:0]
			is = is[:0]

			stencilBufferImageIndex := i
			if antialias {
				stencilBufferImageIndex *= 2
			}
			image := cachedStencilBufferImages[stencilBufferImageIndex+oac.imageIndex]
			// Add an origin point. Any position works.
			vs = append(vs, ebiten.Vertex{
				DstX:   float32(image.Bounds().Dx()) / 2,
				DstY:   float32(image.Bounds().Dx()) / 2,
				ColorR: oac.colorR,
				ColorG: oac.colorG,
				ColorB: oac.colorB,
				ColorA: oac.colorA,
			})
			stencilBufferImage := cachedStencilBufferImages[stencilBufferImageIndex+oac.imageIndex]
			dstOffsetX := float32(-cachedPathBounds[i].Min.X + stencilBufferImage.Bounds().Min.X - max(0, dst.Bounds().Min.X-cachedPathBounds[i].Min.X))
			dstOffsetY := float32(-cachedPathBounds[i].Min.Y + stencilBufferImage.Bounds().Min.Y - max(0, dst.Bounds().Min.Y-cachedPathBounds[i].Min.Y))
			for _, subPath := range path.subPaths {
				if !subPath.isValid() {
					continue
				}

				cur := subPath.start
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
						is = append(is, idx, 0, idx+1)
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
						is = append(is, idx, 0, idx+1)
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
					is = append(is, idx, 0, idx+1)
				}
			}
			op := &ebiten.DrawTrianglesShaderOptions{}
			op.Blend = ebiten.BlendLighter
			stencilBufferImage.DrawTrianglesShader32(vs, is, stencilBufferFillShader, op)
		}
	}

	// Second, render the bezier curves.
	for i, path := range paths {
		if path == nil {
			continue
		}

		for _, oac := range offsetAndColors {
			vs = vs[:0]
			is = is[:0]

			stencilBufferImageIndex := i
			if antialias {
				stencilBufferImageIndex *= 2
			}
			stencilBufferImage := cachedStencilBufferImages[stencilBufferImageIndex+oac.imageIndex]
			dstOffsetX := float32(-cachedPathBounds[i].Min.X + stencilBufferImage.Bounds().Min.X - max(0, dst.Bounds().Min.X-cachedPathBounds[i].Min.X))
			dstOffsetY := float32(-cachedPathBounds[i].Min.Y + stencilBufferImage.Bounds().Min.Y - max(0, dst.Bounds().Min.Y-cachedPathBounds[i].Min.Y))
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
	for i, path := range paths {
		if path == nil {
			continue
		}

		stencilBufferImageIndex := i
		if antialias {
			stencilBufferImageIndex *= 2
		}
		stencilImage := cachedStencilBufferImages[stencilBufferImageIndex]
		pathBounds := cachedPathBounds[i]

		vs = vs[:0]
		is = is[:0]
		dstOffsetX := max(0, dst.Bounds().Min.X-cachedPathBounds[i].Min.X)
		dstOffsetY := max(0, dst.Bounds().Min.Y-cachedPathBounds[i].Min.Y)
		var clrR, clrG, clrB, clrA float32
		r, g, b, a := colors[i].RGBA()
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
		switch fillRule {
		case FillRuleNonZero:
			if antialias {
				shader = stencilBufferNonZeroAAShader
			} else {
				shader = stencilBufferNonZeroShader
			}
		case FillRuleEvenOdd:
			if antialias {
				shader = stencilBufferEvenOddAAShader
			} else {
				shader = stencilBufferEvenOddShader
			}
		}
		dst.DrawTrianglesShader32(vs, is, shader, op)
	}
}
