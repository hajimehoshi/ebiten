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

package shaderir_test

import (
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/shaderir"
)

func TestOutput(t *testing.T) {
	tests := []struct {
		Name    string
		Program Program
		Glsl    string
	}{
		{
			Name:    "Empty",
			Program: Program{},
			Glsl:    ``,
		},
		{
			Name: "Uniform",
			Program: Program{
				Uniforms: []Variable{
					{
						Name: "U0",
						Type: Type{Main: Float},
					},
				},
			},
			Glsl: `uniform float U0;`,
		},
		{
			Name: "UniformStruct",
			Program: Program{
				Uniforms: []Variable{
					{
						Name: "U0",
						Type: Type{
							Main: Struct,
							Sub: []Type{
								{Main: Float},
							},
						},
					},
				},
			},
			Glsl: `struct S0 {
	float M0;
};
uniform S0 U0;`,
		},
		{
			Name: "Vars",
			Program: Program{
				Uniforms: []Variable{
					{
						Name: "U0",
						Type: Type{Main: Float},
					},
				},
				Attributes: []Variable{
					{
						Name: "A0",
						Type: Type{Main: Vec2},
					},
				},
				Varyings: []Variable{
					{
						Name: "V0",
						Type: Type{Main: Vec3},
					},
				},
			},
			Glsl: `uniform float U0;
attribute vec2 A0;
varying vec3 V0;`,
		},
		{
			Name: "Func",
			Program: Program{
				Funcs: []Func{
					{
						Name: "F0",
					},
				},
			},
			Glsl: `void F0(void) {
}`,
		},
		{
			Name: "FuncParams",
			Program: Program{
				Funcs: []Func{
					{
						Name: "F0",
						InParams: []Type{
							{Main: Float},
							{Main: Vec2},
							{Main: Vec4},
						},
						InOutParams: []Type{
							{Main: Mat2},
						},
						OutParams: []Type{
							{Main: Mat4},
						},
					},
				},
			},
			Glsl: `void F0(in float l0, in vec2 l1, in vec4 l2, inout mat2 l3, out mat4 l4) {
}`,
		},
	}
	for _, tc := range tests {
		got := tc.Program.Glsl()
		want := tc.Glsl + "\n"
		if got != want {
			t.Errorf("%s: got: %s, want: %s", tc.Name, got, want)
		}
	}
}
