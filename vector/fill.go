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

// theAtlas manages the atlas for stencil buffer images.
// theAtlas is a singleton to avoid unnecessary texture allocations.
var theAtlas atlas

type fillPathsState struct {
	paths  []*Path
	colors []ebiten.ColorScale

	vertices []ebiten.Vertex
	indices  []uint32

	antialias bool
	blend     ebiten.Blend
	fillRule  FillRule
}

func (f *fillPathsState) reset() {
	for _, p := range f.paths {
		p.Reset()
	}
	f.paths = f.paths[:0]
	f.colors = slices.Delete(f.colors, 0, len(f.colors))
}

func (f *fillPathsState) addPath(path *Path, clr ebiten.ColorScale) {
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

	theAtlas.setPaths(dst.Bounds(), f.paths, f.antialias)

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

			stencilBufferImage := theAtlas.stencilBufferImageAt(i, f.antialias, oac.imageIndex)
			if stencilBufferImage == nil {
				continue
			}
			pp := theAtlas.pathRenderingPositionAt(i)
			dstOffsetX := float32(-pp.X + stencilBufferImage.Bounds().Min.X - max(0, dst.Bounds().Min.X-pp.X))
			dstOffsetY := float32(-pp.Y + stencilBufferImage.Bounds().Min.Y - max(0, dst.Bounds().Min.Y-pp.Y))

			for i := range path.subPaths {
				subPath := &path.subPaths[i]
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

			stencilBufferImage := theAtlas.stencilBufferImageAt(i, f.antialias, oac.imageIndex)
			if stencilBufferImage == nil {
				continue
			}
			pp := theAtlas.pathRenderingPositionAt(i)
			dstOffsetX := float32(-pp.X + stencilBufferImage.Bounds().Min.X - max(0, dst.Bounds().Min.X-pp.X))
			dstOffsetY := float32(-pp.Y + stencilBufferImage.Bounds().Min.Y - max(0, dst.Bounds().Min.Y-pp.Y))
			for i := range path.subPaths {
				subPath := &path.subPaths[i]
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

		stencilImage := theAtlas.stencilBufferImageAt(i, f.antialias, 0)
		if stencilImage == nil {
			continue
		}
		srcRegion := stencilImage.Bounds()

		var offsetX, offsetY float32
		if f.antialias {
			stencilImage1 := theAtlas.stencilBufferImageAt(i, f.antialias, 1)
			offsetX = float32(stencilImage1.Bounds().Min.X - stencilImage.Bounds().Min.X)
			offsetY = float32(stencilImage1.Bounds().Min.Y - stencilImage.Bounds().Min.Y)
		}

		pp := theAtlas.pathRenderingPositionAt(i)

		vs = vs[:0]
		is = is[:0]
		dstOffsetX := max(0, dst.Bounds().Min.X-pp.X)
		dstOffsetY := max(0, dst.Bounds().Min.Y-pp.Y)
		var clrR, clrG, clrB, clrA float32
		clrR = f.colors[i].R()
		clrG = f.colors[i].G()
		clrB = f.colors[i].B()
		clrA = f.colors[i].A()
		vs = append(vs,
			ebiten.Vertex{
				DstX:    float32(pp.X + dstOffsetX),
				DstY:    float32(pp.Y + dstOffsetY),
				SrcX:    float32(srcRegion.Min.X),
				SrcY:    float32(srcRegion.Min.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: offsetX,
				Custom1: offsetY,
			},
			ebiten.Vertex{
				DstX:    float32(pp.X + srcRegion.Dx() + dstOffsetX),
				DstY:    float32(pp.Y + dstOffsetY),
				SrcX:    float32(srcRegion.Max.X),
				SrcY:    float32(srcRegion.Min.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: offsetX,
				Custom1: offsetY,
			},
			ebiten.Vertex{
				DstX:    float32(pp.X + dstOffsetX),
				DstY:    float32(pp.Y + srcRegion.Dy() + dstOffsetY),
				SrcX:    float32(srcRegion.Min.X),
				SrcY:    float32(srcRegion.Max.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: offsetX,
				Custom1: offsetY,
			},
			ebiten.Vertex{
				DstX:    float32(pp.X + srcRegion.Dx() + dstOffsetX),
				DstY:    float32(pp.Y + srcRegion.Dy() + dstOffsetY),
				SrcX:    float32(srcRegion.Max.X),
				SrcY:    float32(srcRegion.Max.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: offsetX,
				Custom1: offsetY,
			})
		is = append(is, 0, 1, 2, 1, 2, 3)

		op := &ebiten.DrawTrianglesShaderOptions{}
		op.Blend = f.blend
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
