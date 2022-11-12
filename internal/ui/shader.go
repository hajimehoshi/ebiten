// Copyright 2022 The Ebiten Authors
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

package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/mipmap"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Shader struct {
	shader *mipmap.Shader

	uniformNames       []string
	uniformTypes       []shaderir.Type
	uniformNameToIndex map[string]int
	uniformNameToType  map[string]shaderir.Type
}

func NewShader(ir *shaderir.Program) *Shader {
	return &Shader{
		shader:       mipmap.NewShader(ir),
		uniformNames: ir.UniformNames,
		uniformTypes: ir.Uniforms,
	}
}

func (s *Shader) MarkDisposed() {
	s.shader.MarkDisposed()
	s.shader = nil
}

func (s *Shader) ConvertUniforms(uniforms map[string]any) [][]uint32 {
	nameToU32s := map[string][]uint32{}
	for name, v := range uniforms {
		switch v := v.(type) {
		case float32:
			nameToU32s[name] = []uint32{math.Float32bits(v)}
		case []float32:
			u32s := make([]uint32, len(v))
			for i := range v {
				u32s[i] = math.Float32bits(v[i])
			}
			nameToU32s[name] = u32s
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

	us := make([][]uint32, len(s.uniformNameToIndex))
	for name, idx := range s.uniformNameToIndex {
		if v, ok := nameToU32s[name]; ok {
			us[idx] = v
			continue
		}
		t := s.uniformNameToType[name]
		us[idx] = make([]uint32, t.FloatCount())
	}

	// TODO: Panic if uniforms include an invalid name

	return us
}
