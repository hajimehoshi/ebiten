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
	"io"
	"regexp"

	"github.com/hajimehoshi/ebiten/v2/internal/shader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

// Go's whitespace is U+0020 (SP), U+0009 (\t), U+000d (\r), and U+000A (\n).
// See https://go.dev/ref/spec#Tokens
var reUnit = regexp.MustCompile(`^[ \t\r\n]*//kage:unit\s+([^ \t\r\n]+)[ \t\r\n]*$`)

// ParseKageUnitDirective returns the value of the //kage:unit directive in src, or an empty string if
// the directive is absent. A duplicated directive is an error.
func ParseKageUnitDirective(src []byte) (string, error) {
	var value string
	var parsed bool

	for line := range bytes.Lines(src) {
		m := reUnit.FindSubmatch(line)
		if m == nil {
			continue
		}
		if parsed {
			return "", fmt.Errorf("graphics: at most one //kage:unit can exist in a shader")
		}
		value = string(m[1])
		parsed = true
	}

	return value, nil
}

// writeShaderBridge writes to w the Kage source appended to a user's fragment shader, bridging the user's
// builtin functions to the engine's __-prefixed uniforms and textures. The bridge operates in the pixel
// unit: the region uniforms hold pixels, and __texelAt fetches a texel by its integer pixel coordinates.
func writeShaderBridge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, `
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
`, ShaderSrcImageCount); err != nil {
		return err
	}

	if _, err := io.WriteString(w, `
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
	return __imageDstRegionOrigin, __imageDstRegionSize
}

// imageDstOrigin returns the destination image's origin on its texture.
// The unit is the source texture's pixel or texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
func imageDstOrigin() vec2 {
	return __imageDstRegionOrigin
}

// imageDstSize returns the destination image's size.
// The unit is the source texture's pixel or texel.
func imageDstSize() vec2 {
	return __imageDstRegionSize
}
`); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, `
// The unit is the source texture's pixel or texel.
var __imageSrcRegionOrigins [%[1]d]vec2

// The unit is the source texture's pixel or texel.
var __imageSrcRegionSizes [%[1]d]vec2

// imageSrcRegionOnTexture returns the 0th source image's region (the origin and the size) on its texture.
// The unit is the source texture's pixel or texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
//
// Deprecated: as of v2.6. Use imageSrc0Origin or imageSrc0Size instead.
func imageSrcRegionOnTexture() (vec2, vec2) {
	return __imageSrcRegionOrigins[0], __imageSrcRegionSizes[0]
}
`, ShaderSrcImageCount); err != nil {
		return err
	}

	for i := range ShaderSrcImageCount {
		if _, err := fmt.Fprintf(w, `
// imageSrc%[1]dOrigin returns the source image's region origin on its texture.
// The unit is the source texture's pixel or texel.
//
// As an image is a part of internal texture, the image can be located at an arbitrary position on the texture.
func imageSrc%[1]dOrigin() vec2 {
	return __imageSrcRegionOrigins[%[1]d]
}

// imageSrc%[1]dSize returns the source image's size.
// The unit is the source texture's pixel or texel.
func imageSrc%[1]dSize() vec2 {
	return __imageSrcRegionSizes[%[1]d]
}
`, i); err != nil {
			return err
		}

		// pos is in pixels of the 0th texture. Convert it to the i-th texture's pixels.
		texPos := "pos"
		if i >= 1 {
			texPos = fmt.Sprintf("pos - __imageSrcRegionOrigins[0] + __imageSrcRegionOrigins[%d]", i)
		}
		// __t%d is a special variable for a texture variable.
		if _, err := fmt.Fprintf(w, `
func imageSrc%[1]dUnsafeAt(pos vec2) vec4 {
	// pos is the position in positions of the source texture (= 0th image's texture).
	return __texelAt(__t%[1]d, %[2]s)
}

func imageSrc%[1]dAt(pos vec2) vec4 {
	// pos is the position of the source texture (= 0th image's texture).
	// If pos is in the region, the result is (1, 1). Otherwise, either element is 0.
	in := step(__imageSrcRegionOrigins[0], pos) - step(__imageSrcRegionOrigins[0] + __imageSrcRegionSizes[%[1]d], pos)
	return __texelAt(__t%[1]d, %[2]s) * in.x * in.y
}
`, i, texPos); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(w, `
var __projectionMatrix mat4

func __vertex(dstPos vec2, srcPos vec2, color vec4, custom vec4) (vec4, vec2, vec4, vec4) {
	return __projectionMatrix * vec4(dstPos, 0, 1), srcPos, color, custom
}
`); err != nil {
		return err
	}

	return nil
}

// completeShaderSource returns a complete shader source: the fragment shader source followed by the bridge.
func completeShaderSource(fragmentSrc []byte) []byte {
	var buf bytes.Buffer
	// Writing to a bytes.Buffer never fails.
	_, _ = buf.Write(fragmentSrc)
	_ = writeShaderBridge(&buf)
	return buf.Bytes()
}

// CompileShader compiles a pixel-unit fragment shader source into an intermediate representation.
// The source must select the pixel unit with the `//kage:unit pixels` directive.
// The returned program's FragmentSource holds the given source.
func CompileShader(fragmentSrc []byte) (*shaderir.Program, error) {
	value, err := ParseKageUnitDirective(fragmentSrc)
	if err != nil {
		return nil, err
	}
	if value != "pixels" {
		return nil, fmt.Errorf("graphics: the `//kage:unit pixels` directive is required")
	}

	const (
		vert = "__vertex"
		frag = "Fragment"
	)
	ir, err := shader.Compile(completeShaderSource(fragmentSrc), vert, frag, ShaderSrcImageCount)
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

// CalcSourceID returns the source ID of a pixel-unit fragment shader source.
func CalcSourceID(fragmentSrc []byte) shaderir.SourceID {
	return shaderir.CalcSourceID(completeShaderSource(fragmentSrc))
}
