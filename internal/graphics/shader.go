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
	"go/parser"
	"go/token"

	"github.com/hajimehoshi/ebiten/v2/internal/shader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

var shaderSuffix string

func init() {
	shaderSuffix = `
var __imageDstTextureSize vec2

// imageSrcTextureSize returns the destination image's texture size in pixels.
func imageDstTextureSize() vec2 {
	return __imageDstTextureSize
}
`

	shaderSuffix += fmt.Sprintf(`
var __textureSizes [%[1]d]vec2

// imageSrcTextureSize returns the source image's texture size in pixels.
// As an image is a part of internal texture, the texture is usually bigger than the image.
// The texture's size is useful when you want to calculate pixels from texels.
func imageSrcTextureSize() vec2 {
	return __textureSizes[0]
}

// The unit is the source texture's texel.
var __textureDestinationRegionOrigin vec2

// The unit is the source texture's texel.
var __textureDestinationRegionSize vec2

// imageDstRegionOnTexture returns the destination image's region (the origin and the size) on its texture.
// The unit is the source texture's texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
func imageDstRegionOnTexture() (vec2, vec2) {
	return __textureDestinationRegionOrigin, __textureDestinationRegionSize
}

// The unit is the source texture's texel.
var __textureSourceOffsets [%[2]d]vec2

// The unit is the source texture's texel.
var __textureSourceRegionOrigin vec2

// The unit is the source texture's texel.
var __textureSourceRegionSize vec2

// imageSrcRegionOnTexture returns the source image's region (the origin and the size) on its texture.
// The unit is the source texture's texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
func imageSrcRegionOnTexture() (vec2, vec2) {
	return __textureSourceRegionOrigin, __textureSourceRegionSize
}
`, ShaderImageCount, ShaderImageCount-1)

	for i := 0; i < ShaderImageCount; i++ {
		pos := "pos"
		if i >= 1 {
			// Convert the position in texture0's texels to the target texture texels.
			pos = fmt.Sprintf("(pos + __textureSourceOffsets[%d]) * __textureSizes[0] / __textureSizes[%d]", i-1, i)
		}
		// __t%d is a special variable for a texture variable.
		shaderSuffix += fmt.Sprintf(`
func imageSrc%[1]dUnsafeAt(pos vec2) vec4 {
	// pos is the position in texels of the source texture (= 0th image's texture).
	return texture2D(__t%[1]d, %[2]s)
}

func imageSrc%[1]dAt(pos vec2) vec4 {
	// pos is the position in texels of the source texture (= 0th image's texture).
	// If pos is in the region, the result is (1, 1). Otherwise, either element is 0.
	in := step(__textureSourceRegionOrigin, pos) - step(__textureSourceRegionOrigin + __textureSourceRegionSize, pos)
	return texture2D(__t%[1]d, %[2]s) * in.x * in.y
}
`, i, pos)
	}

	shaderSuffix += `
var __projectionMatrix mat4

func __vertex(position vec2, texCoord vec2, color vec4) (vec4, vec2, vec4) {
	return __projectionMatrix * vec4(position, 0, 1), texCoord, color
}
`
}

func CompileShader(src []byte) (*shaderir.Program, error) {
	var buf bytes.Buffer
	buf.Write(src)
	buf.WriteString(shaderSuffix)

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", buf.Bytes(), parser.AllErrors)
	if err != nil {
		return nil, err
	}

	const (
		vert = "__vertex"
		frag = "Fragment"
	)
	ir, err := shader.Compile(fs, f, vert, frag, ShaderImageCount)
	if err != nil {
		return nil, err
	}

	if ir.VertexFunc.Block == nil {
		return nil, fmt.Errorf("graphicscommand: vertex shader entry point '%s' is missing", vert)
	}
	if ir.FragmentFunc.Block == nil {
		return nil, fmt.Errorf("graphicscommand: fragment shader entry point '%s' is missing", frag)
	}

	return ir, nil
}
