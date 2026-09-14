// Copyright 2026 The Ebitengine Authors
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

package metal_test

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func arrayType(main shaderir.BasicType, length int) shaderir.Type {
	return shaderir.Type{Main: shaderir.Array, Sub: []shaderir.Type{{Main: main}}, Length: length}
}

func sequence(n int) []uint32 {
	v := make([]uint32, n)
	for i := range v {
		v[i] = uint32(i + 1)
	}
	return v
}

func TestAppendUniformVariablesShaderSwitch(t *testing.T) {
	layouts := []struct {
		types       []shaderir.Type
		input, want []uint32
	}{
		{
			types: []shaderir.Type{arrayType(shaderir.Mat4, 16)},
			input: sequence(256),
			want:  sequence(256),
		},
		{
			types: []shaderir.Type{{Main: shaderir.Bool}, arrayType(shaderir.Bool, 4)},
			input: []uint32{0, 0, 0, 0, 0},
			want:  []uint32{0, 0},
		},
		{
			types: nil,
			input: nil,
			want:  nil,
		},
		{
			types: []shaderir.Type{{Main: shaderir.Float}, {Main: shaderir.Mat3}},
			input: []uint32{9, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			want:  []uint32{9, 0, 0, 0, 1, 2, 3, 0, 4, 5, 6, 0, 7, 8, 9, 0},
		},
	}
	var dst []uint32
	for range 3 {
		for _, layout := range layouts {
			dst = metal.AppendUniformVariables(dst[:0], layout.types, layout.input)
			if !slices.Equal(dst, layout.want) {
				t.Errorf("packed = %#x, want %#x", dst, layout.want)
			}
		}
	}
}
