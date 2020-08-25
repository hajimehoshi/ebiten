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

	dst, _ := NewImage(8, 8, FilterDefault)
	s, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(16, 16, s, nil)

	if got, want := dst.At(0, 0).(color.RGBA), (color.RGBA{0xff, 0, 0, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	if got, want := dst.At(w-1, h-1).(color.RGBA), (color.RGBA{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
