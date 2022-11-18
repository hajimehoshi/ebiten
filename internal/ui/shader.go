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
	"reflect"
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
		v := reflect.ValueOf(v)
		t := v.Type()
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			nameToU32s[name] = []uint32{uint32(v.Int())}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			nameToU32s[name] = []uint32{uint32(v.Uint())}
		case reflect.Float32, reflect.Float64:
			nameToU32s[name] = []uint32{math.Float32bits(float32(v.Float()))}
		case reflect.Slice, reflect.Array:
			u32s := make([]uint32, v.Len())
			switch t.Elem().Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				for i := range u32s {
					u32s[i] = uint32(v.Index(i).Int())
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				for i := range u32s {
					u32s[i] = uint32(v.Index(i).Uint())
				}
			case reflect.Float32, reflect.Float64:
				for i := range u32s {
					u32s[i] = math.Float32bits(float32(v.Index(i).Float()))
				}
			default:
				panic(fmt.Sprintf("ebiten: unexpected uniform value type: %s (%s)", name, v.Kind().String()))
			}
			nameToU32s[name] = u32s
		default:
			panic(fmt.Sprintf("ebiten: unexpected uniform value type: %s (%s)", name, v.Kind().String()))
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
		us[idx] = make([]uint32, t.Uint32Count())
	}

	// TODO: Panic if uniforms include an invalid name

	return us
}
