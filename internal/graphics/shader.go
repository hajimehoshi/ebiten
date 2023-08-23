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

	"github.com/hajimehoshi/ebiten/v2/internal/shader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func shaderSuffix(unit shaderir.Unit) (string, error) {
	shaderSuffix := `
var __imageDstTextureSize vec2

// imageSrcTextureSize returns the destination image's texture size in pixels.
func imageDstTextureSize() vec2 {
	return __imageDstTextureSize
}
`

	shaderSuffix += fmt.Sprintf(`
var __srcTextureSizes [%[1]d]vec2

// imageSrcTextureSize returns the source image's texture size in pixels.
// As an image is a part of internal texture, the texture is usually bigger than the image.
// The texture's size is useful when you want to calculate pixels from texels in the texel mode.
func imageSrcTextureSize() vec2 {
	return __srcTextureSizes[0]
}

// The unit is the source texture's pixel or texel.
var __imageDstRegionOrigin vec2

// The unit is the source texture's pixel or texel.
var __imageDstRegionSize vec2

// imageDstRegionOnTexture returns the destination image's region (the origin and the size) on its texture.
// The unit is the source texture's pixel or texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
func imageDstRegionOnTexture() (vec2, vec2) {
	return __imageDstRegionOrigin, __imageDstRegionSize
}

// The unit is the source texture's pixel.
var __imageSrcOffsets [%[2]d]vec2

// The unit is the source texture's pixel or texel.
var __imageSrcRegionOrigin vec2

// The unit is the source texture's pixel or texel.
var __imageSrcRegionSize vec2

// imageSrcRegionOnTexture returns the source image's region (the origin and the size) on its texture.
// The unit is the source texture's pixel or texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
func imageSrcRegionOnTexture() (vec2, vec2) {
	return __imageSrcRegionOrigin, __imageSrcRegionSize
}
`, ShaderImageCount, ShaderImageCount-1)

	for i := 0; i < ShaderImageCount; i++ {
		pos := "pos"
		if i >= 1 {
			// Convert the position in texture0's positions to the target texture positions.
			switch unit {
			case shaderir.Pixels:
				pos = fmt.Sprintf("pos + __imageSrcOffsets[%d]", i-1)
			case shaderir.Texels:
				pos = fmt.Sprintf("(pos * __srcTextureSizes[0] + __imageSrcOffsets[%d]) / __srcTextureSizes[%d]", i-1, i)
			default:
				return "", fmt.Errorf("graphics: unexpected unit: %d", unit)
			}
		}
		// __t%d is a special variable for a texture variable.
		shaderSuffix += fmt.Sprintf(`
func imageSrc%[1]dUnsafeAt(pos vec2) vec4 {
	// pos is the position in positions of the source texture (= 0th image's texture).
	return __texelAt(__t%[1]d, %[2]s)
}

func imageSrc%[1]dAt(pos vec2) vec4 {
	// pos is the position of the source texture (= 0th image's texture).
	// If pos is in the region, the result is (1, 1). Otherwise, either element is 0.
	in := step(__imageSrcRegionOrigin, pos) - step(__imageSrcRegionOrigin + __imageSrcRegionSize, pos)
	return __texelAt(__t%[1]d, %[2]s) * in.x * in.y
}
`, i, pos)
	}

	shaderSuffix += `
var __projectionMatrix mat4

func __vertex(position vec2, texCoord vec2, color vec4) (vec4, vec2, vec4) {
	return __projectionMatrix * vec4(position, 0, 1), texCoord, color
}
`
	return shaderSuffix, nil
}

func CompileShader(src []byte) (*shaderir.Program, error) {
	unit, err := shader.ParseCompilerDirectives(src)
	if err != nil {
		return nil, err
	}
	suffix, err := shaderSuffix(unit)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.Write(src)
	buf.WriteString(suffix)

	const (
		vert = "__vertex"
		frag = "Fragment"
	)
	ir, err := shader.Compile(buf.Bytes(), vert, frag, ShaderImageCount)
	if err != nil {
		return nil, err
	}

	if ir.VertexFunc.Block == nil {
		return nil, fmt.Errorf("graphics: vertex shader entry point '%s' is missing", vert)
	}
	if ir.FragmentFunc.Block == nil {
		return nil, fmt.Errorf("graphics: fragment shader entry point '%s' is missing", frag)
	}

	return ir, nil
}
