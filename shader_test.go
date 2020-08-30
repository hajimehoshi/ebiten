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

package ebiten_test

import (
	"image/color"
	"testing"

	. "github.com/hajimehoshi/ebiten"
)

func TestShaderFill(t *testing.T) {
	const w, h = 16, 16

	dst, _ := NewImage(w, h, FilterDefault)
	s, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w/2, h/2, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if i < w/2 && j < h/2 {
				want = color.RGBA{0xff, 0, 0, 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderFillWithDrawImage(t *testing.T) {
	const w, h = 16, 16

	dst, _ := NewImage(w, h, FilterDefault)
	s, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	src, _ := NewImage(w/2, h/2, FilterDefault)
	op := &DrawImageOptions{}
	op.Shader = s
	dst.DrawImage(src, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if i < w/2 && j < h/2 {
				want = color.RGBA{0xff, 0, 0, 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderFunction(t *testing.T) {
	const w, h = 16, 16

	dst, _ := NewImage(w, h, FilterDefault)
	s, err := NewShader([]byte(`package main

func clr(red float) (float, float, float, float) {
	return red, 0, 0, 1
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(clr(1))
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{0xff, 0, 0, 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderShadowing(t *testing.T) {
	_, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var position vec4
	return position
}
`))
	if err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestShaderDuplicatedVariables(t *testing.T) {
	_, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var foo vec4
	var foo vec4
	return foo
}
`))
	if err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	_, err = NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var foo, foo vec4
	return foo
}
`))
	if err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	_, err = NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var foo vec4
	foo := vec4(0)
	return foo
}
`))
	if err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	_, err = NewShader([]byte(`package main

func Foo() (vec4, vec4) {
	return vec4(0), vec4(0)
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	foo, foo := Foo()
	return foo
}
`))
	if err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}
