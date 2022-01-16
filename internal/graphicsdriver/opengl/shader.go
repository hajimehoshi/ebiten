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

package opengl

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/glsl"
)

type Shader struct {
	id       driver.ShaderID
	graphics *Graphics

	ir *shaderir.Program
	p  program
}

func newShader(id driver.ShaderID, graphics *Graphics, program *shaderir.Program) (*Shader, error) {
	s := &Shader{
		id:       id,
		graphics: graphics,
		ir:       program,
	}
	if err := s.compile(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Shader) ID() driver.ShaderID {
	return s.id
}

func (s *Shader) Dispose() {
	s.graphics.context.deleteProgram(s.p)
	s.graphics.removeShader(s)
}

func (s *Shader) compile() error {
	vssrc, fssrc := glsl.Compile(s.ir, s.graphics.context.glslVersion())

	vs, err := s.graphics.context.newVertexShader(vssrc)
	if err != nil {
		return fmt.Errorf("opengl: vertex shader compile error: %v, source:\n%s", err, vssrc)
	}
	defer s.graphics.context.deleteShader(vs)

	fs, err := s.graphics.context.newFragmentShader(fssrc)
	if err != nil {
		return fmt.Errorf("opengl: fragment shader compile error: %v, source:\n%s", err, fssrc)
	}
	defer s.graphics.context.deleteShader(fs)

	p, err := s.graphics.context.newProgram([]shader{vs, fs}, theArrayBufferLayout.names())
	if err != nil {
		return err
	}

	s.p = p
	return nil
}
