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
	"go/parser"
	"go/token"
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/shader"
)

func TestDump(t *testing.T) {
	tests := []struct {
		Name string
		Src  string
		VS   string
		FS   string
	}{
		{
			Name: "uniforms",
			Src: `package main

var (
	Foo vec2
	Boo vec4
)`,
			VS: `uniform vec2 U0;
uniform vec4 U1;`,
		},
		{
			Name: "func",
			Src: `package main

func Foo(foo vec2) vec4 {
}`,
			VS: `void F0(in vec2 l0, out vec4 l1) {
}`,
		},
		{
			Name: "func body",
			Src: `package main

func Foo(foo vec2) vec4 {
	return vec4(foo, 0, 1)
}`,
			VS: `void F0(in vec2 l0, out vec4 l1) {
	l1 = vec4(l0, 0.0, 1.0);
	return;
}`,
		},
		{
			Name: "var init",
			Src: `package main

func Foo(foo vec2) vec4 {
	var ret vec4 = vec4(foo, 0, 1)
	return ret
}`,
			VS: `void F0(in vec2 l0, out vec4 l1) {
	vec4 l2 = vec4(0.0);
	l2 = vec4(l0, 0.0, 1.0);
	l1 = l2;
	return;
}`,
		},
		{
			Name: "var init 2",
			Src: `package main

func Foo(foo vec2) vec4 {
	var bar1 vec4 = vec4(foo, 0, 1)
	bar1.x = bar1.x
	var bar2 vec4 = bar1
	return bar2
}`,
			VS: `void F0(in vec2 l0, out vec4 l1) {
	vec4 l2 = vec4(0.0);
	vec4 l3 = vec4(0.0);
	l2 = vec4(l0, 0.0, 1.0);
	(l2).x = (l2).x;
	l3 = l2;
	l1 = l3;
	return;
}`,
		},
		{
			Name: "var multiple init",
			Src: `package main

func Foo(foo vec2) vec4 {
	var bar1, bar2 vec2 = foo, foo
	return vec4(bar1, bar2)
}`,
			VS: `void F0(in vec2 l0, out vec4 l1) {
	vec2 l2 = vec2(0.0);
	vec2 l3 = vec2(0.0);
	l2 = l0;
	l3 = l0;
	l1 = vec4(l2, l3);
	return;
}`,
		},
		{
			Name: "multiple out params",
			Src: `package main

func Foo(foo vec4) (float, float, float, float) {
	return foo.x, foo.y, foo.z, foo.w
}`,
			VS: `void F0(in vec4 l0, out float l1, out float l2, out float l3, out float l4) {
	l1 = (l0).x;
	l2 = (l0).y;
	l3 = (l0).z;
	l4 = (l0).w;
	return;
}`,
		},
		{
			Name: "blocks",
			Src: `package main

func Foo(foo vec2) vec4 {
	var r vec4
	{
		r.x = foo.x
		var foo vec3
		{
			r.y = foo.y
			var foo vec4
			r.z = foo.z
		}
		{
			r.y = foo.y
			var foo vec4
			r.z = foo.z
		}
	}
	return r
}`,
			VS: `void F0(in vec2 l0, out vec4 l1) {
	vec4 l2 = vec4(0.0);
	{
		vec3 l3 = vec3(0.0);
		(l2).x = (l0).x;
		{
			vec4 l4 = vec4(0.0);
			(l2).y = (l3).y;
			(l2).z = (l4).z;
		}
		{
			vec4 l4 = vec4(0.0);
			(l2).y = (l3).y;
			(l2).z = (l4).z;
		}
	}
	l1 = l2;
	return;
}`,
		},
		{
			Name: "define",
			Src: `package main

func Foo(foo vec2) vec4 {
	r := vec4(foo, 0, 1)
	return r
}`,
			VS: `void F0(in vec2 l0, out vec4 l1) {
	vec4 l2 = vec4(0.0);
	l2 = vec4(l0, 0.0, 1.0);
	l1 = l2;
	return;
}`,
		},
		{
			Name: "call",
			Src: `package main

func Foo(x vec2) vec2 {
	return Bar(x)
}

func Bar(x vec2) vec2 {
	return x
}`,
			VS: `void F0(in vec2 l0, out vec2 l1) {
	vec2 l2 = vec2(0.0);
	F1(l0, l2);
	l1 = l2;
	return;
}
void F1(in vec2 l0, out vec2 l1) {
	l1 = l0;
	return;
}`,
		},
		{
			Name: "vertex",
			Src: `package main

func Vertex(position vec2, texCoord vec2, color vec4) (position vec4, texCoord vec2, color vec4) {
	projectionMatrix := mat4(
		2 / ScreenSize.x, 0, 0, 0,
		0, 2 / ScreenSize.y, 0, 0,
		0, 0, 1, 0,
		-1, -1, 0, 1,
	)
	return projectionMatrix * vec4(position, 0, 1), texCoord, color
}

var ScreenSize vec2`,
			VS: `uniform vec2 U0;
attribute vec2 A0;
attribute vec2 A1;
attribute vec4 A2;
varying vec2 V0;
varying vec4 V1;

void main(void) {
	mat4 l0 = mat4(0.0);
	l0 = mat4((2.0) / ((U0).x), 0.0, 0.0, 0.0, 0.0, (2.0) / ((U0).y), 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, -(1.0), -(1.0), 0.0, 1.0);
	gl_Position = (l0) * (vec4(A0, 0.0, 1.0));
	V0 = A1;
	V1 = A2;
	return;
}`,
			FS: `uniform vec2 U0;
varying vec2 V0;
varying vec4 V1;`,
		},
		{
			Name: "vertex and fragment",
			Src: `package main

func Vertex(position vec2, texCoord vec2, color vec4) (position vec4, texCoord vec2, color vec4) {
	projectionMatrix := mat4(
		2 / ScreenSize.x, 0, 0, 0,
		0, 2 / ScreenSize.y, 0, 0,
		0, 0, 1, 0,
		-1, -1, 0, 1,
	)
	return projectionMatrix * vec4(position, 0, 1), texCoord, color
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}

var ScreenSize vec2`,
			VS: `uniform vec2 U0;
attribute vec2 A0;
attribute vec2 A1;
attribute vec4 A2;
varying vec2 V0;
varying vec4 V1;

void main(void) {
	mat4 l0 = mat4(0.0);
	l0 = mat4((2.0) / ((U0).x), 0.0, 0.0, 0.0, 0.0, (2.0) / ((U0).y), 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, -(1.0), -(1.0), 0.0, 1.0);
	gl_Position = (l0) * (vec4(A0, 0.0, 1.0));
	V0 = A1;
	V1 = A2;
	return;
}`,
			FS: `uniform vec2 U0;
varying vec2 V0;
varying vec4 V1;

void main(void) {
	gl_FragColor = vec4(1.0, 0.0, 0.0, 1.0);
	return;
}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "", []byte(tc.Src), parser.AllErrors)
			if err != nil {
				t.Fatal(err)
				return
			}

			s, err := Compile(fset, f, "Vertex", "Fragment")
			if err != nil {
				t.Error(err)
				return
			}
			vs, fs := s.Glsl()
			if got, want := vs, tc.VS+"\n"; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
			if tc.FS != "" {
				if got, want := fs, tc.FS+"\n"; got != want {
					t.Errorf("got: %v, want: %v", got, want)
				}
			}
		})
	}
}
