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

//go:build !playstation5

package opengl

import (
	"fmt"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/glsl"
)

type Shader struct {
	id       graphicsdriver.ShaderID
	graphics *Graphics

	ir *shaderir.Program
	p  program
}

func newShader(id graphicsdriver.ShaderID, graphics *Graphics, program *shaderir.Program) (*Shader, error) {
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

func (s *Shader) ID() graphicsdriver.ShaderID {
	return s.id
}

func (s *Shader) Dispose() {
	s.graphics.context.deleteProgram(s.p)
	s.graphics.removeShader(s)
}

func (s *Shader) compile() error {
	vssrc, fssrc := glsl.Compile(s.ir, s.graphics.context.glslVersion())

	vs, err := s.graphics.context.newShader(gl.VERTEX_SHADER, vssrc)
	if err != nil {
		return err
	}
	defer s.graphics.context.ctx.DeleteShader(uint32(vs))

	fs, err := s.graphics.context.newShader(gl.FRAGMENT_SHADER, fssrc)
	if err != nil {
		return err
	}
	defer s.graphics.context.ctx.DeleteShader(uint32(fs))

	p, err := s.graphics.context.newProgram([]shader{vs, fs}, theArrayBufferLayout.names())
	if err != nil {
		return err
	}

	// Check the shader compile status asynchronously if possible.
	// The function 'compile' itself is still blocking, but at least this gives a chance to other goroutines to run
	// while waiting for the shader compilation.
	if s.graphics.context.hasParallelShaderCompile() {
		for s.graphics.context.ctx.GetShaderi(uint32(vs), gl.COMPLETION_STATUS_KHR) != gl.TRUE ||
			s.graphics.context.ctx.GetShaderi(uint32(fs), gl.COMPLETION_STATUS_KHR) != gl.TRUE {
			runtime.Gosched()
		}
	}

	// Check errors only after linking fails.
	// See https://developer.mozilla.org/en-US/docs/Web/API/WebGL_API/WebGL_best_practices#dont_check_shader_compile_status_unless_linking_fails
	if s.graphics.context.ctx.GetProgrami(uint32(p), gl.LINK_STATUS) == gl.FALSE {
		programInfo := s.graphics.context.ctx.GetProgramInfoLog(uint32(p))
		vertexShaderInfo := s.graphics.context.ctx.GetShaderInfoLog(uint32(vs))
		fragmentShaderInfo := s.graphics.context.ctx.GetShaderInfoLog(uint32(fs))
		return fmt.Errorf("opengl: program error: %s\nvertex shader error: %s\nvertex shader source: %s\nfragment shader error: %s\nfragment shader source: %s",
			programInfo, vertexShaderInfo, vssrc, fragmentShaderInfo, fssrc)
	}

	s.p = p
	return nil
}
