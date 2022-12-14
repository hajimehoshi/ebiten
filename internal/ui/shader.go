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

	uniformNames       []string
	uniformTypes       []shaderir.Type
	uniformUint32Count int
}

func NewShader(ir *shaderir.Program) *Shader {
	return &Shader{
		shader:       mipmap.NewShader(ir),
		uniformNames: ir.UniformNames[graphics.PreservedUniformVariablesCount:],
		uniformTypes: ir.Uniforms[graphics.PreservedUniformVariablesCount:],
	}
}

func (s *Shader) MarkDisposed() {
	s.shader.MarkDisposed()
	s.shader = nil
}

func (s *Shader) AppendUniforms(dst []uint32, uniforms map[string]any) []uint32 {
	if s.uniformUint32Count == 0 {
		for _, typ := range s.uniformTypes {
			s.uniformUint32Count += typ.Uint32Count()
		}
	}

	origLen := len(dst)
	if cap(dst)-len(dst) >= s.uniformUint32Count {
		dst = dst[:len(dst)+s.uniformUint32Count]
	} else {
		dst = append(dst, make([]uint32, s.uniformUint32Count)...)
	}

	idx := origLen
	for i, name := range s.uniformNames {
		typ := s.uniformTypes[i]

		if uv, ok := uniforms[name]; ok {
			// TODO: Panic if uniforms include an invalid name
			v := reflect.ValueOf(uv)
			t := v.Type()
			switch t.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				dst[idx] = uint32(v.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				dst[idx] = uint32(v.Uint())
			case reflect.Float32, reflect.Float64:
				dst[idx] = math.Float32bits(float32(v.Float()))
			case reflect.Slice, reflect.Array:
				l := v.Len()
				switch t.Elem().Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					for i := 0; i < l; i++ {
						dst[idx+i] = uint32(v.Index(i).Int())
					}
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					for i := 0; i < l; i++ {
						dst[idx+i] = uint32(v.Index(i).Uint())
					}
				case reflect.Float32, reflect.Float64:
					for i := 0; i < l; i++ {
						dst[idx+i] = math.Float32bits(float32(v.Index(i).Float()))
					}
				default:
					panic(fmt.Sprintf("ui: unexpected uniform value type: %s (%s)", name, v.Kind().String()))
				}
			default:
				panic(fmt.Sprintf("ui: unexpected uniform value type: %s (%s)", name, v.Kind().String()))
			}
		}

		idx += typ.Uint32Count()
	}

	return dst
}
