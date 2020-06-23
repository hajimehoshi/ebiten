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
	"go/parser"
	"go/token"

	"github.com/hajimehoshi/ebiten/internal/buffered"
	"github.com/hajimehoshi/ebiten/internal/shader"
)

const shaderSuffix = `
var __viewportSize vec2

func viewportSize() vec2 {
	return __viewportSize
}
`

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
	s, err := shader.Compile(fs, f, "Vertex", "Fragment")
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
