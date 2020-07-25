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

	"github.com/hajimehoshi/ebiten/internal/buffered"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/shader"
)

var shaderSuffix string

func init() {
	shaderSuffix = `
var __textureDstSize vec2

func textureDstSize() vec2 {
	return __textureDstSize
}

func __vertex(position vec2, texCoord vec2, color vec4) (vec4, vec2, vec4) {
	return mat4(
		2/textureDstSize().x, 0, 0, 0,
		0, 2/textureDstSize().y, 0, 0,
		0, 0, 1, 0,
		-1, -1, 0, 1,
	) * vec4(position, 0, 1), texCoord, color
}
`

	for i := 1; i < graphics.ShaderImageNum; i++ {
		shaderSuffix += fmt.Sprintf(`
var __offset%d vec2
`, i)
	}

	for i := 0; i < graphics.ShaderImageNum; i++ {
		var offset string
		if i >= 1 {
			offset = fmt.Sprintf(" + __offset%d", i)
		}
		// __t%d is a special variable for a texture variable.
		shaderSuffix += fmt.Sprintf(`
func texture%[1]dAt(pos vec2) vec4 {
	return texture2D(__t%[1]d, pos%[2]s)
}
`, i, offset)
	}
}

type Shader struct {
	shader *buffered.Shader
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
		shader: buffered.NewShader(s),
	}, nil
}

func (s *Shader) Dispose() {
	s.shader.MarkDisposed()
	s.shader = nil
}
