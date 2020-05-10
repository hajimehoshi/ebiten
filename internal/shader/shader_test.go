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

package shader_test

import (
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/shader"
)

func TestDump(t *testing.T) {
	tests := []struct {
		Name string
		Src  string
		Dump string
	}{
		{
			Name: "general",
			Src: `package main

type VertexOut struct {
	Position vec4 ` + "`kage:\"position\"`" + `
	TexCoord vec2
	Color    vec4
}

var Foo float
var (
	Bar       vec2
	Baz, Quux vec3
)

const C1 float = 1
const C2, C3 float = 2, 3

func F1(a, b vec2) vec4 {
	var c0 vec2 = a
	var c1, c2 = c0, 1.0
	c1.x = c2.x
	c3 := vec4{c0, c1}
	return c2
}
`,
			Dump: `var Position varying vec4 // position
var Color varying vec4
var TexCoord varying vec2
var Bar uniform vec2
var Baz uniform vec3
var Foo uniform float
var Quux uniform vec3
const C1 float = 1
const C2 float = 2
const C3 float = 3
func F1(a vec2, b vec2) (_ vec4) {
	var c0 vec2 = a
	var c1 vec2 = c0
	var c2 vec2 = 1.0
	var c3 vec4
	c1.x = c2.x
	c3 = vec4{c0, c1}
	return c2
}
`,
		},
		{
			Name: "AutoType",
			Src: `package main

var V0 = 0.0
func F() {
	v1 := V0
}
`,
			Dump: `var V0 uniform float
func F() {
	var v1 float
	v1 = V0
}
`,
		},
		{
			Name: "AutoType2",
			Src: `package main

var V0 = 0.0
func F() {
	v1 := V0
	{
		v2 := v1
	}
}
`,
			Dump: `var V0 uniform float
func F() {
	var v1 float
	v1 = V0
	{
		var v2 float
		v2 = v1
	}
}
`,
		},
	}
	for _, tc := range tests {
		s, err := NewShader([]byte(tc.Src))
		if err != nil {
			t.Error(err)
			continue
		}
		if got, want := s.Dump(), tc.Dump; got != want {
			t.Errorf("%s: got: %v, want: %v", tc.Name, got, want)
		}
	}
}
