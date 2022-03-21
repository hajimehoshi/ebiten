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
// As of v2.3.0, the error value is always nil, and
// the actual complation happens lazily after the main loop starts.
//
// For the details about the shader, see https://ebiten.org/documents/shader.html.
func NewShader(src []byte) (*Shader, error) {
	return &Shader{
		shader: ui.NewShader(src),
	}, nil
}

// Dispose disposes the shader program.
// After disposing, the shader is no longer available.
func (s *Shader) Dispose() {
	s.shader.MarkDisposed()
	s.shader = nil
}
