// Copyright 2020 The Ebiten Authors
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

package ebiten

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"

	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/mipmap"
	"github.com/hajimehoshi/ebiten/internal/shader"
)

var shaderSuffix string

func init() {
	shaderSuffix = `
var __imageDstTextureSize vec2

func imageDstTextureSize() vec2 {
	return __imageDstTextureSize
}
`

	shaderSuffix += fmt.Sprintf(`
var __textureSizes [%d]vec2
`, graphics.ShaderImageNum)

	for i := 0; i < graphics.ShaderImageNum; i++ {
		shaderSuffix += fmt.Sprintf(`
func image%[1]dTextureSize() vec2 {
	return __textureSizes[%[1]d]
}
`, i)
	}

	shaderSuffix += fmt.Sprintf(`
// The unit is texture0's texels.
var __textureSourceOffsets [%[1]d]vec2

// The unit is texture0's texels.
var __textureSourceOrigin vec2

// The unit is texture0's texels.
var __textureSourceSize vec2
`, graphics.ShaderImageNum-1, graphics.ShaderImageNum)

	for i := 0; i < graphics.ShaderImageNum; i++ {
		pos := "pos"
		if i >= 1 {
			// Convert the position in texture0's texels to the target texture texels.
			pos = fmt.Sprintf("(pos + __textureSourceOffsets[%d]) * __textureSizes[0] / __textureSizes[%d]", i-1, i)
		}
		// __t%d is a special variable for a texture variable.
		shaderSuffix += fmt.Sprintf(`
func image%[1]dTextureAt(pos vec2) vec4 {
	// pos is the position in texture0's texels.
	return texture2D(__t%[1]d, %[2]s)
}

func image%[1]dTextureBoundsAt(pos vec2) vec4 {
	// pos is the position in texture0's texels.
	return texture2D(__t%[1]d, %[2]s) *
		step(__textureSourceOrigin.x, pos.x) *
		(1 - step(__textureSourceOrigin.x + __textureSourceSize.x, pos.x)) *
		step(__textureSourceOrigin.y, pos.y) *
		(1 - step(__textureSourceOrigin.y + __textureSourceSize.y, pos.y))
}
`, i, pos)
	}

	shaderSuffix += `
func __vertex(position vec2, texCoord vec2, color vec4) (vec4, vec2, vec4) {
	return mat4(
		2/imageDstTextureSize().x, 0, 0, 0,
		0, 2/imageDstTextureSize().y, 0, 0,
		0, 0, 1, 0,
		-1, -1, 0, 1,
	) * vec4(position, 0, 1), texCoord, color
}
`
}

type Shader struct {
	shader *mipmap.Shader
}

func NewShader(src []byte) (*Shader, error) {
	var buf bytes.Buffer
	buf.Write(src)
	buf.WriteString(shaderSuffix)

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", buf.Bytes(), parser.AllErrors)
	if err != nil {
		return nil, err
	}

	// TODO: Create a pseudo vertex entrypoint to treat the attribute values correctly.
	s, err := shader.Compile(fs, f, "__vertex", "Fragment", graphics.ShaderImageNum)
	if err != nil {
		return nil, err
	}

	return &Shader{
		shader: mipmap.NewShader(s),
	}, nil
}

func (s *Shader) Dispose() {
	s.shader.MarkDisposed()
	s.shader = nil
}
