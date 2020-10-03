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

package restorable

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Shader struct {
	shader *graphicscommand.Shader
	ir     *shaderir.Program
}

func NewShader(program *shaderir.Program) *Shader {
	s := &Shader{
		shader: graphicscommand.NewShader(program),
		ir:     program,
	}
	theImages.addShader(s)
	return s
}

func (s *Shader) Dispose() {
	theImages.removeShader(s)
	s.shader.Dispose()
	s.shader = nil
	s.ir = nil
}

func (s *Shader) restore() {
	s.shader = graphicscommand.NewShader(s.ir)
}
