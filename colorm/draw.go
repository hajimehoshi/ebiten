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

package colorm

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
)

// DrawImageOptions represents options for DrawImage.
type DrawImageOptions struct {
	// GeoM is a geometry matrix to draw.
	// The default (zero) value is identity, which draws the image at (0, 0).
	GeoM ebiten.GeoM

	// ColorScale is a scale of color.
	//
	// ColorScale is slightly different from colorm.ColorM's Scale in terms of alphas.
	// ColorScale is applied to premultiplied-alpha colors, while colorm.ColorM is applied to straight-alpha colors.
	// Thus, ColorM.Scale(r, g, b, a) equals to ColorScale.Scale(r*a, g*a, b*a, a).
	//
	// The default (zero) value is identity, which is (1, 1, 1, 1).
	ColorScale ebiten.ColorScale

	// Blend is a blending way of the source color and the destination color.
	// The default (zero) value is the regular alpha blending.
	Blend ebiten.Blend

	// Filter is a type of texture filter.
	// The default (zero) value is ebiten.FilterNearest.
	Filter ebiten.Filter
}

// DrawImage draws src onto dst.
//
// DrawImage is basically the same as ebiten.DrawImage, but with a color matrix.
func DrawImage(dst, src *ebiten.Image, colorM ColorM, op *DrawImageOptions) {
	if op == nil {
		op = &DrawImageOptions{}
	}

	opShader := &ebiten.DrawRectShaderOptions{}
	opShader.GeoM = op.GeoM
	opShader.ColorScale = op.ColorScale
	opShader.CompositeMode = ebiten.CompositeModeCustom
	opShader.Blend = op.Blend
	opShader.Uniforms = uniforms(colorM)
	opShader.Images[0] = src
	s := builtinShader(builtinshader.Filter(op.Filter), builtinshader.AddressUnsafe)
	dst.DrawRectShader(src.Bounds().Dx(), src.Bounds().Dy(), s, opShader)
}

// DrawTrianglesOptions represents options for DrawTriangles.
type DrawTrianglesOptions struct {
	// ColorScaleMode is the mode of color scales in vertices.
	// The default (zero) value is ebiten.ColorScaleModeStraightAlpha.
	ColorScaleMode ebiten.ColorScaleMode

	// Blend is a blending way of the source color and the destination color.
	// The default (zero) value is the regular alpha blending.
	Blend ebiten.Blend

	// Filter is a type of texture filter.
	// The default (zero) value is ebiten.FilterNearest.
	Filter ebiten.Filter

	// Address is a sampler address mode.
	// The default (zero) value is ebiten.AddressUnsafe.
	Address ebiten.Address

	// FillRule indicates the rule how an overlapped region is rendered.
	//
	// The rules FileRuleNonZero and FillRuleEvenOdd are useful when you want to render a complex polygon.
	// A complex polygon is a non-convex polygon like a concave polygon, a polygon with holes, or a self-intersecting polygon.
	// See examples/vector for actual usages.
	//
	// The default (zero) value is ebiten.FillRuleFillAll.
	FillRule ebiten.FillRule

	// AntiAlias indicates whether the rendering uses anti-alias or not.
	// AntiAlias is useful especially when you pass vertices from the vector package.
	//
	// AntiAlias increases internal draw calls and might affect performance.
	// Use the build tag `ebitenginedebug` to check the number of draw calls if you care.
	//
	// The default (zero) value is false.
	AntiAlias bool
}

// DrawTriangles draws triangles onto dst.
//
// DrawTriangles is basically the same as ebiten.DrawTriangles, but with a color matrix.
func DrawTriangles(dst *ebiten.Image, vertices []ebiten.Vertex, indices []uint16, img *ebiten.Image, colorM ColorM, op *DrawTrianglesOptions) {
	if op == nil {
		op = &DrawTrianglesOptions{}
	}

	if op.ColorScaleMode == ebiten.ColorScaleModeStraightAlpha {
		vs := make([]ebiten.Vertex, len(vertices))
		copy(vs, vertices)
		for i := range vertices {
			vs[i].ColorR *= vs[i].ColorA
			vs[i].ColorG *= vs[i].ColorA
			vs[i].ColorB *= vs[i].ColorA
		}
		vertices = vs
	}

	opShader := &ebiten.DrawTrianglesShaderOptions{}
	opShader.CompositeMode = ebiten.CompositeModeCustom
	opShader.Blend = op.Blend
	opShader.FillRule = op.FillRule
	opShader.AntiAlias = op.AntiAlias
	opShader.Uniforms = uniforms(colorM)
	opShader.Images[0] = img
	s := builtinShader(builtinshader.Filter(op.Filter), builtinshader.Address(op.Address))
	dst.DrawTrianglesShader(vertices, indices, s, opShader)
}
