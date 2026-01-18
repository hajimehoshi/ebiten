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

// This file emulates the deprecated APIs (FillRule and AntiAlias) with the non-deprecated APIs.
//
// The pseudo stencil buffer implementation is based on the following article:
// https://medium.com/@evanwallace/easy-scalable-text-rendering-on-the-gpu-c3f4d782c5ac

package ebiten

import (
	"fmt"
	"image"
	"slices"
	"sync"
)

var (
	stencilBufferFillShader    *Shader
	stencilBufferNonZeroShader *Shader
	stencilBufferEvenOddShader *Shader
)

//ebitengine:shadersource
const stencilBufferFillShaderSrc = `//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	if frontfacing() {
		return vec4(0, 1.0 / 255.0, 0, 0)
	}
	return vec4(1.0 / 255.0, 0, 0, 0)
}
`

//ebitengine:shadersource
const stencilBufferNonZeroShaderSrc = `//kage:unit pixels

package main

func round(x float) float {
	return floor(x + 0.5)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0UnsafeAt(srcPos)
	w := abs(int(round(c.g*255)) - int(round(c.r*255)))
	v := min(float(w), 1)
	return v * imageSrc1UnsafeAt(srcPos) * color
}
`

//ebitengine:shadersource
const stencilBufferEvenOddShaderSrc = `//kage:unit pixels

package main

func round(x float) float {
	return floor(x + 0.5)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0UnsafeAt(srcPos)
	w := abs(int(round(c.g*255)) - int(round(c.r*255)))
	v := float(w % 2)
	return v * imageSrc1UnsafeAt(srcPos) * color
}
`

func ensureStencilBufferShaders() (*Shader, error) {
	if stencilBufferFillShader != nil {
		return stencilBufferFillShader, nil
	}
	s, err := NewShader([]byte(stencilBufferFillShaderSrc))
	if err != nil {
		return nil, err
	}
	stencilBufferFillShader = s
	return stencilBufferFillShader, err
}

func ensureStencilBufferNonZeroShader() (*Shader, error) {
	if stencilBufferNonZeroShader != nil {
		return stencilBufferNonZeroShader, nil
	}
	s, err := NewShader([]byte(stencilBufferNonZeroShaderSrc))
	if err != nil {
		return nil, err
	}
	stencilBufferNonZeroShader = s
	return stencilBufferNonZeroShader, nil
}

func ensureStencilBufferEvenOddShader() (*Shader, error) {
	if stencilBufferEvenOddShader != nil {
		return stencilBufferEvenOddShader, nil
	}
	s, err := NewShader([]byte(stencilBufferEvenOddShaderSrc))
	if err != nil {
		return nil, err
	}
	stencilBufferEvenOddShader = s
	return stencilBufferEvenOddShader, nil
}

var (
	stencilBufferM sync.Mutex

	stencilBufferImage *Image
	offscreenImage1    *Image
	offscreenImage2    *Image
)

func ensureStencilBufferImage(bounds image.Rectangle) *Image {
	var prevBounds image.Rectangle
	if stencilBufferImage != nil && !bounds.In(stencilBufferImage.Bounds()) {
		prevBounds = stencilBufferImage.Bounds()
		stencilBufferImage.Deallocate()
		stencilBufferImage = nil
	}
	if stencilBufferImage == nil {
		stencilBufferImage = NewImageWithOptions(bounds.Union(prevBounds), nil)
	} else {
		stencilBufferImage.Clear()
	}
	return stencilBufferImage
}

func ensureOffscreenImage1(bounds image.Rectangle) *Image {
	var prevBounds image.Rectangle
	if offscreenImage1 != nil && !bounds.In(offscreenImage1.Bounds()) {
		prevBounds = offscreenImage1.Bounds()
		offscreenImage1.Deallocate()
		offscreenImage1 = nil
	}
	if offscreenImage1 == nil {
		offscreenImage1 = NewImageWithOptions(bounds.Union(prevBounds), nil)
	} else {
		offscreenImage1.Clear()
	}
	return offscreenImage1
}

func ensureOffscreenImage2(bounds image.Rectangle) *Image {
	var prevBounds image.Rectangle
	if offscreenImage2 != nil && !bounds.In(offscreenImage2.Bounds()) {
		prevBounds = offscreenImage2.Bounds()
		offscreenImage2.Deallocate()
		offscreenImage2 = nil
	}
	if offscreenImage2 == nil {
		offscreenImage2 = NewImageWithOptions(bounds.Union(prevBounds), nil)
	} else {
		offscreenImage2.Clear()
	}
	return offscreenImage2
}

func shaderFromFillRule(fillRule FillRule) *Shader {
	switch fillRule {
	case FillRuleNonZero:
		s, err := ensureStencilBufferNonZeroShader()
		if err != nil {
			panic(fmt.Sprintf("ebiten: failed to ensure stencil buffer non-zero shader: %v", err))
		}
		return s
	case FillRuleEvenOdd:
		s, err := ensureStencilBufferEvenOddShader()
		if err != nil {
			panic(fmt.Sprintf("ebiten: failed to ensure stencil buffer even-odd shader: %v", err))
		}
		return s
	default:
		panic("ebiten: not reached")
	}
}

func drawTrianglesWithStencilBuffer(dst *Image, vertices []Vertex, indices []uint32, img *Image, options *DrawTrianglesOptions) {
	if options.FillRule == FillRuleFillAll {
		if !options.AntiAlias {
			panic("not reached")
		}
		doDrawTrianglesWithAntialias(dst, vertices, indices, img, options, nil, nil)
		return
	}
	doDrawTrianglesShaderWithStencilBuffer(dst, vertices, indices, img, options, nil, nil)
}

func drawTrianglesShaderWithStencilBuffer(dst *Image, vertices []Vertex, indices []uint32, shader *Shader, options *DrawTrianglesShaderOptions) {
	if options.FillRule == FillRuleFillAll {
		if !options.AntiAlias {
			panic("not reached")
		}
		doDrawTrianglesWithAntialias(dst, vertices, indices, nil, nil, shader, options)
		return
	}
	doDrawTrianglesShaderWithStencilBuffer(dst, vertices, indices, nil, nil, shader, options)
}

var (
	tmpVerticesForStencilBuffer []Vertex
)

// doDrawTrianglesWithAntialias draw triangles with antialiasing.
//
// doDrawTrianglesWithAntialias doesn't batch draw calls, so this might not be efficient.
// This is different from Ebitengine v2.9's behavior.
// However, this function is for the legacy API and its usage is expected to be minimal.
func doDrawTrianglesWithAntialias(dst *Image, vertices []Vertex, indices []uint32, img *Image, dtOptions *DrawTrianglesOptions, shader *Shader, dtsOptions *DrawTrianglesShaderOptions) {
	stencilBufferM.Lock()
	defer stencilBufferM.Unlock()

	bounds := dst.Bounds()
	bounds.Min.X *= 2
	bounds.Max.X *= 2
	bounds.Min.Y *= 2
	bounds.Max.Y *= 2

	tmpVerticesForStencilBuffer = slices.Grow(tmpVerticesForStencilBuffer, len(vertices))
	vs := tmpVerticesForStencilBuffer[:len(vertices)]
	copy(vs, vertices)
	for i := range vs {
		vs[i].DstX *= 2
		vs[i].DstY *= 2
	}

	var uncommonBlend bool
	if dtOptions != nil {
		if dtOptions.CompositeMode != CompositeModeCustom {
			uncommonBlend = dtOptions.CompositeMode != CompositeModeSourceOver
		} else {
			uncommonBlend = dtOptions.Blend != BlendSourceOver
		}
	} else if dtsOptions != nil {
		if dtsOptions.CompositeMode != CompositeModeCustom {
			uncommonBlend = dtsOptions.CompositeMode != CompositeModeSourceOver
		} else {
			uncommonBlend = dtsOptions.Blend != BlendSourceOver
		}
	}

	// Copy the current destination image for the blending, if the blend mode is not the regular alpha blending.
	os1 := ensureOffscreenImage1(bounds).SubImage(bounds).(*Image)
	if uncommonBlend {
		op := &DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.Blend = BlendCopy
		os1.DrawImage(dst, op)
	}

	if dtOptions != nil {
		op := &DrawTrianglesOptions{}
		op.ColorM = dtOptions.ColorM
		op.ColorScaleMode = dtOptions.ColorScaleMode
		op.CompositeMode = dtOptions.CompositeMode
		op.Blend = dtOptions.Blend
		op.Filter = dtOptions.Filter
		op.Address = dtOptions.Address
		op.DisableMipmaps = dtOptions.DisableMipmaps
		os1.DrawTriangles32(vs, indices, img, op)
	} else if dtsOptions != nil {
		op := &DrawTrianglesShaderOptions{}
		op.Uniforms = dtsOptions.Uniforms
		op.Images = dtsOptions.Images
		op.CompositeMode = dtsOptions.CompositeMode
		op.Blend = dtsOptions.Blend
		os1.DrawTrianglesShader32(vs, indices, shader, op)
	}

	op := &DrawImageOptions{}
	op.GeoM.Scale(0.5, 0.5)
	op.Filter = FilterLinear
	if uncommonBlend {
		op.Blend = BlendCopy
	}
	dst.DrawImage(os1, op)
}

// doDrawTrianglesShaderWithStencilBuffer draw triangles with stencil buffer.
//
// doDrawTrianglesShaderWithStencilBuffer doesn't batch draw calls, so this might not be efficient.
// This is different from Ebitengine v2.9's behavior.
// However, this function is for the legacy API and its usage is expected to be minimal.
func doDrawTrianglesShaderWithStencilBuffer(dst *Image, vertices []Vertex, indices []uint32, img *Image, dtOptions *DrawTrianglesOptions, shader *Shader, dtsOptions *DrawTrianglesShaderOptions) {
	stencilBufferM.Lock()
	defer stencilBufferM.Unlock()

	vs := vertices
	bounds := dst.Bounds()
	if dtOptions != nil && dtOptions.AntiAlias || dtsOptions != nil && dtsOptions.AntiAlias {
		bounds.Min.X *= 2
		bounds.Max.X *= 2
		bounds.Min.Y *= 2
		bounds.Max.Y *= 2

		tmpVerticesForStencilBuffer = slices.Grow(tmpVerticesForStencilBuffer, len(vertices))
		vs = tmpVerticesForStencilBuffer[:len(vertices)]
		copy(vs, vertices)
		for i := range vs {
			vs[i].DstX *= 2
			vs[i].DstY *= 2
		}
	}

	// Create an offscreen image to render the vertices as they are, without blendings.
	os1 := ensureOffscreenImage1(bounds).SubImage(bounds).(*Image)
	if dtOptions != nil {
		op := &DrawTrianglesOptions{}
		op.ColorM = dtOptions.ColorM
		op.ColorScaleMode = dtOptions.ColorScaleMode
		op.Filter = dtOptions.Filter
		op.Address = dtOptions.Address
		op.DisableMipmaps = dtOptions.DisableMipmaps
		os1.DrawTriangles32(vs, indices, img, op)
	} else if dtsOptions != nil {
		op := &DrawTrianglesShaderOptions{}
		op.Uniforms = dtsOptions.Uniforms
		op.Images = dtsOptions.Images
		os1.DrawTrianglesShader32(vs, indices, shader, op)
	}

	// Create an offscreen image with the stencil buffer.
	var finalOS *Image
	{
		// Create a stencil buffer image.
		stencilBufferShader, err := ensureStencilBufferShaders()
		if err != nil {
			panic(err)
		}
		stencilImg := ensureStencilBufferImage(bounds).SubImage(bounds).(*Image)
		stencilOp := &DrawTrianglesShaderOptions{}
		stencilOp.Blend = BlendLighter
		stencilImg.DrawTrianglesShader32(vs, indices, stencilBufferShader, stencilOp)

		op := &DrawRectShaderOptions{}
		op.Images[0] = stencilImg
		op.Images[1] = os1
		os2 := ensureOffscreenImage2(bounds).SubImage(bounds).(*Image)
		var fillRule FillRule
		if dtOptions != nil {
			fillRule = dtOptions.FillRule
		} else if dtsOptions != nil {
			fillRule = dtsOptions.FillRule
		}
		os2.DrawRectShader(bounds.Dx(), bounds.Dy(), shaderFromFillRule(fillRule), op)
		finalOS = os2
	}

	// Render the offscreen image onto dst.
	// Note that some blends like BlendXor might not work correctly, but this is expected,
	// as this logic is for the legacy API.
	op := &DrawImageOptions{}
	if dtOptions != nil {
		op.CompositeMode = dtOptions.CompositeMode
		op.Blend = dtOptions.Blend
		if dtOptions.AntiAlias {
			op.GeoM.Scale(0.5, 0.5)
			op.Filter = FilterLinear
		}
	} else if dtsOptions != nil {
		op.CompositeMode = dtsOptions.CompositeMode
		op.Blend = dtsOptions.Blend
		if dtsOptions.AntiAlias {
			op.GeoM.Scale(0.5, 0.5)
			op.Filter = FilterLinear
		}
	}
	dst.DrawImage(finalOS, op)
}
