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
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// Shader represents a compiled shader program.
//
// For the details about the shader, see https://ebitengine.org/en/documents/shader.html.
type Shader struct {
	shader *ui.Shader
	unit   shaderir.Unit
}

// NewShader compiles a shader program in the shading language Kage, and returns the result.
//
// If the compilation fails, NewShader returns an error.
//
// For the details about the shader, see https://ebitengine.org/en/documents/shader.html.
func NewShader(src []byte) (*Shader, error) {
	ir, err := graphics.CompileShader(src)
	if err != nil {
		return nil, err
	}
	return &Shader{
		shader: ui.NewShader(ir),
		unit:   ir.Unit,
	}, nil
}

// Dispose disposes the shader program.
// After disposing, the shader is no longer available.
func (s *Shader) Dispose() {
	s.shader.MarkDisposed()
	s.shader = nil
}

func (s *Shader) appendUniforms(dst []uint32, uniforms map[string]any) []uint32 {
	return s.shader.AppendUniforms(dst, uniforms)
}

var (
	builtinShaders  [builtinshader.FilterCount][builtinshader.AddressCount][2]*Shader
	builtinShadersM sync.Mutex
)

func builtinShader(filter builtinshader.Filter, address builtinshader.Address, useColorM bool) *Shader {
	builtinShadersM.Lock()
	defer builtinShadersM.Unlock()

	var c int
	if useColorM {
		c = 1
	}
	if s := builtinShaders[filter][address][c]; s != nil {
		return s
	}

	var shader *Shader
	if address == builtinshader.AddressUnsafe && !useColorM {
		switch filter {
		case builtinshader.FilterNearest:
			shader = &Shader{shader: ui.NearestFilterShader}
		case builtinshader.FilterLinear:
			shader = &Shader{shader: ui.LinearFilterShader}
		}
	} else {
		src := builtinshader.Shader(filter, address, useColorM)
		s, err := NewShader(src)
		if err != nil {
			panic(fmt.Sprintf("ebiten: NewShader for a built-in shader failed: %v", err))
		}
		shader = s
	}

	builtinShaders[filter][address][c] = shader
	return shader
}
