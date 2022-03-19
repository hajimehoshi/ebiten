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
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// Shader represents a compiled shader program.
//
// For the details about the shader, see https://ebiten.org/documents/shader.html.
type Shader struct {
	shader *ui.Shader
}

// NewShader compiles a shader program in the shading language Kage, and retruns the result.
//
// If the compilation fails, NewShader returns an error.
//
// For the details about the shader, see https://ebiten.org/documents/shader.html.
func NewShader(src []byte) (*Shader, error) {
	s, err := ui.NewShader(src)
	if err != nil {
		return nil, err
	}
	return &Shader{
		shader: s,
	}, nil
}

// Dispose disposes the shader program.
// After disposing, the shader is no longer available.
func (s *Shader) Dispose() {
	s.shader.MarkDisposed()
	s.shader = nil
}

func (s *Shader) convertUniforms(uniforms map[string]interface{}) [][]float32 {
	return s.shader.ConvertUniforms(uniforms)
}
