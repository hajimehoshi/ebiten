// Copyright 2022 The Ebiten Authors
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

package graphics

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/shader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

// shaderSuffix returns the Kage source appended to a user's fragment shader.
//
// The engine's core always operates in the pixel unit: the region uniforms hold pixels, and __texelAt
// fetches a texel by its integer pixel coordinates. A texel-unit shader is supported by generating builtin
// functions that convert between texels and pixels at the boundary, so the unit never leaks into the core.
func shaderSuffix(unit shader.Unit) (string, error) {
	if unit != shader.Pixels && unit != shader.Texels {
		return "", fmt.Errorf("graphics: unexpected unit: %d", unit)
	}

	var shaderSuffix strings.Builder
	shaderSuffix.WriteString(fmt.Sprintf(`
var __imageDstTextureSize vec2

// imageDstTextureSize returns the destination image's texture size in pixels.
//
// Deprecated: as of v2.6. Use the pixel-unit mode.
func imageDstTextureSize() vec2 {
	return __imageDstTextureSize
}

var __imageSrcTextureSizes [%[1]d]vec2

// imageSrcTextureSize returns the 0th source image's texture size in pixels.
// As an image is a part of internal texture, the texture is usually bigger than the image.
// The texture's size is useful when you want to calculate pixels from texels in the texel mode.
//
// Deprecated: as of v2.6. Use the pixel-unit mode.
func imageSrcTextureSize() vec2 {
	return __imageSrcTextureSizes[0]
}
`, ShaderSrcImageCount))

	// The region uniforms always hold pixels. In the texel unit, the public functions convert them to
	// texels by dividing by the texture size.
	dstOrigin := "__imageDstRegionOrigin"
	dstSize := "__imageDstRegionSize"
	srcOrigin0 := "__imageSrcRegionOrigins[0]"
	srcSize0 := "__imageSrcRegionSizes[0]"
	if unit == shader.Texels {
		// A source texture size can be zero when no source image is given. Guard against a division by
		// zero with max so that a zero region stays zero, matching the case without a source image.
		dstOrigin = "__imageDstRegionOrigin / __imageDstTextureSize"
		dstSize = "__imageDstRegionSize / __imageDstTextureSize"
		srcOrigin0 = "__imageSrcRegionOrigins[0] / max(__imageSrcTextureSizes[0], vec2(1))"
		srcSize0 = "__imageSrcRegionSizes[0] / max(__imageSrcTextureSizes[0], vec2(1))"
	}

	shaderSuffix.WriteString(fmt.Sprintf(`
// The unit is the destination texture's pixel or texel.
var __imageDstRegionOrigin vec2

// The unit is the source texture's pixel or texel.
var __imageDstRegionSize vec2

// imageDstRegionOnTexture returns the destination image's region (the origin and the size) on its texture.
// The unit is the source texture's pixel or texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
//
// Deprecated: as of v2.6. Use imageDstOrigin or imageDstSize.
func imageDstRegionOnTexture() (vec2, vec2) {
	return %[1]s, %[2]s
}

// imageDstOrigin returns the destination image's origin on its texture.
// The unit is the source texture's pixel or texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
func imageDstOrigin() vec2 {
	return %[1]s
}

// imageDstSize returns the destination image's size.
// The unit is the source texture's pixel or texel.
func imageDstSize() vec2 {
	return %[2]s
}
`, dstOrigin, dstSize))

	shaderSuffix.WriteString(fmt.Sprintf(`
// The unit is the source texture's pixel or texel.
var __imageSrcRegionOrigins [%[3]d]vec2

// The unit is the source texture's pixel or texel.
var __imageSrcRegionSizes [%[3]d]vec2

// imageSrcRegionOnTexture returns the 0th source image's region (the origin and the size) on its texture.
// The unit is the source texture's pixel or texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
//
// Deprecated: as of v2.6. Use imageSrc0Origin or imageSrc0Size instead.
func imageSrcRegionOnTexture() (vec2, vec2) {
	return %[1]s, %[2]s
}
`, srcOrigin0, srcSize0, ShaderSrcImageCount))

	for i := range ShaderSrcImageCount {
		srcOrigin := fmt.Sprintf("__imageSrcRegionOrigins[%d]", i)
		srcSize := fmt.Sprintf("__imageSrcRegionSizes[%d]", i)
		if unit == shader.Texels {
			srcOrigin = fmt.Sprintf("__imageSrcRegionOrigins[%[1]d] / max(__imageSrcTextureSizes[%[1]d], vec2(1))", i)
			srcSize = fmt.Sprintf("__imageSrcRegionSizes[%[1]d] / max(__imageSrcTextureSizes[%[1]d], vec2(1))", i)
		}
		shaderSuffix.WriteString(fmt.Sprintf(`
// imageSrc%[1]dOrigin returns the source image's region origin on its texture.
// The unit is the source texture's pixel or texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
func imageSrc%[1]dOrigin() vec2 {
	return %[2]s
}

// imageSrc%[1]dSize returns the source image's size.
// The unit is the source texture's pixel or texel.
func imageSrc%[1]dSize() vec2 {
	return %[3]s
}
`, i, srcOrigin, srcSize))

		// In the texel unit, the argument is in texels of the 0th texture. Convert it to pixels of the
		// 0th texture so that the rest of the body matches the pixel unit.
		var convert string
		if unit == shader.Texels {
			convert = "\tpos = pos * __imageSrcTextureSizes[0]\n"
		}
		// pos is in pixels of the 0th texture. Convert it to the i-th texture's pixels.
		texPos := "pos"
		if i >= 1 {
			texPos = fmt.Sprintf("pos - __imageSrcRegionOrigins[0] + __imageSrcRegionOrigins[%d]", i)
		}
		// In the texel unit, all the source region sizes are the same (#1870), so the 0th source region
		// size is always used for the bounds check.
		sizeIdx := i
		if unit == shader.Texels {
			sizeIdx = 0
		}
		// __t%d is a special variable for a texture variable.
		shaderSuffix.WriteString(fmt.Sprintf(`
func imageSrc%[1]dUnsafeAt(pos vec2) vec4 {
	// pos is the position in positions of the source texture (= 0th image's texture).
%[2]s	return __texelAt(__t%[1]d, %[3]s)
}

func imageSrc%[1]dAt(pos vec2) vec4 {
	// pos is the position of the source texture (= 0th image's texture).
	// If pos is in the region, the result is (1, 1). Otherwise, either element is 0.
%[2]s	in := step(__imageSrcRegionOrigins[0], pos) - step(__imageSrcRegionOrigins[0] + __imageSrcRegionSizes[%[4]d], pos)
	return __texelAt(__t%[1]d, %[3]s) * in.x * in.y
}
`, i, convert, texPos, sizeIdx))
	}

	// The core feeds srcPos to __vertex in pixels. In the texel unit, convert it to texels for the
	// user's fragment shader. The texture size is guarded against zero (no source image) to avoid a
	// division by zero; the position is then left in pixels, matching the pixel unit.
	srcPos := "srcPos"
	if unit == shader.Texels {
		srcPos = "srcPos / max(__imageSrcTextureSizes[0], vec2(1))"
	}
	shaderSuffix.WriteString(fmt.Sprintf(`
var __projectionMatrix mat4

func __vertex(dstPos vec2, srcPos vec2, color vec4, custom vec4) (vec4, vec2, vec4, vec4) {
	return __projectionMatrix * vec4(dstPos, 0, 1), %s, color, custom
}
`, srcPos))
	return shaderSuffix.String(), nil
}

func completeShaderSource(fragmentSrc []byte) ([]byte, error) {
	unit, err := shader.ParseCompilerDirectives(fragmentSrc)
	if err != nil {
		return nil, err
	}
	suffix, err := shaderSuffix(unit)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.Write(fragmentSrc)
	buf.WriteString(suffix)

	return buf.Bytes(), nil
}

func CompileShader(fragmentSrc []byte) (*shaderir.Program, error) {
	src, err := completeShaderSource(fragmentSrc)
	if err != nil {
		return nil, err
	}

	const (
		vert = "__vertex"
		frag = "Fragment"
	)
	ir, err := shader.Compile(src, vert, frag, ShaderSrcImageCount)
	if err != nil {
		return nil, err
	}

	if ir.VertexFunc.Block == nil {
		return nil, fmt.Errorf("graphics: vertex shader entry point '%s' is missing", vert)
	}
	if ir.FragmentFunc.Block == nil {
		return nil, fmt.Errorf("graphics: fragment shader entry point '%s' is missing", frag)
	}

	ir.FragmentSource = bytes.Clone(fragmentSrc)

	return ir, nil
}

func CalcSourceID(fragmentSrc []byte) (shaderir.SourceID, error) {
	src, err := completeShaderSource(fragmentSrc)
	if err != nil {
		return shaderir.SourceID{}, err
	}
	return shaderir.CalcSourceID(src), nil
}
