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
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// Shader represents a compiled shader program.
//
// For the details about the shader, see https://ebiten.org/documents/shader.html.
type Shader struct {
	shader *ui.Shader

	uniformNames       []string
	uniformTypes       []shaderir.Type
	uniformNameToIndex map[string]int
	uniformNameToType  map[string]shaderir.Type
}

// NewShader compiles a shader program in the shading language Kage, and retruns the result.
//
// If the compilation fails, NewShader returns an error.
//
// For the details about the shader, see https://ebiten.org/documents/shader.html.
func NewShader(src []byte) (*Shader, error) {
	ir, err := graphics.CompileShader(src)
	if err != nil {
		return nil, err
	}
	return &Shader{
		shader:       ui.NewShader(ir),
		uniformNames: ir.UniformNames,
		uniformTypes: ir.Uniforms,
	}, nil
}

// Dispose disposes the shader program.
// After disposing, the shader is no longer available.
func (s *Shader) Dispose() {
	s.shader.MarkDisposed()
	s.shader = nil
}

func (s *Shader) convertUniforms(uniforms map[string]interface{}) [][]float32 {
	nameToF32s := map[string][]float32{}
	for name, v := range uniforms {
		switch v := v.(type) {
		case float32:
			nameToF32s[name] = []float32{v}
		case []float32:
			nameToF32s[name] = v
		default:
			panic(fmt.Sprintf("ebiten: unexpected uniform value type: %s, %T", name, v))
		}
	}

	if s.uniformNameToIndex == nil {
		s.uniformNameToIndex = map[string]int{}
		s.uniformNameToType = map[string]shaderir.Type{}

		var idx int
		for i, n := range s.uniformNames {
			if strings.HasPrefix(n, "__") {
				continue
			}
			s.uniformNameToIndex[n] = idx
			s.uniformNameToType[n] = s.uniformTypes[i]
			idx++
		}
	}

	us := make([][]float32, len(s.uniformNameToIndex))
	for name, idx := range s.uniformNameToIndex {
		if v, ok := nameToF32s[name]; ok {
			us[idx] = v
			continue
		}
		t := s.uniformNameToType[name]
		us[idx] = make([]float32, t.FloatCount())
	}

	// TODO: Panic if uniforms include an invalid name

	return us
}
