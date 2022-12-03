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

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/mipmap"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Shader struct {
	shader *mipmap.Shader

	uniformNames []string
	uniformTypes []shaderir.Type
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
	idxToU32s := make([][]uint32, len(s.uniformNames))
	for idx, name := range s.uniformNames[graphics.PreservedUniformVariablesCount:] {
		uv, ok := uniforms[name]
		if !ok {
			// TODO: Panic if uniforms include an invalid name
			continue
		}

		v := reflect.ValueOf(uv)
		t := v.Type()
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			idxToU32s[idx] = []uint32{uint32(v.Int())}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			idxToU32s[idx] = []uint32{uint32(v.Uint())}
		case reflect.Float32, reflect.Float64:
			idxToU32s[idx] = []uint32{math.Float32bits(float32(v.Float()))}
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
			idxToU32s[idx] = u32s
		default:
			panic(fmt.Sprintf("ebiten: unexpected uniform value type: %s (%s)", name, v.Kind().String()))
		}
	}

	us := make([][]uint32, len(s.uniformTypes)-graphics.PreservedUniformVariablesCount)
	for idx, typ := range s.uniformTypes[graphics.PreservedUniformVariablesCount:] {
		v := idxToU32s[idx]
		if v == nil {
			v = make([]uint32, typ.Uint32Count())
		}
		us[idx] = v
	}

	return us
}
