// Copyright 2022 The Ebitengine Authors
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
	"slices"
	"sort"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/shader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func compileToIR(src []byte) (*shaderir.Program, error) {
	return shader.Compile(src, "Vertex", "Fragment", 0)
}

func TestReachableUniformVariablesFromBlock(t *testing.T) {
	src0 := `package main

var U0 float
var U1 float

func F0() float {
	return U0
}

func F1() {
	a := U0
	_ = a
}

func F2() {
}

func F3() float {
	return F0()
}

func F4() float {
	return F0() + U1
}

func neverCalled() float {
	return U0 + U1
}
`

	cases := []struct {
		source   string
		index    int
		expected []int
	}{
		{
			source:   src0,
			index:    0,
			expected: []int{0},
		},
		{
			source:   src0,
			index:    1,
			expected: []int{0},
		},
		{
			source:   src0,
			index:    2,
			expected: []int{},
		},
		{
			source:   src0,
			index:    3,
			expected: []int{0},
		},
		{
			source:   src0,
			index:    4,
			expected: []int{0, 1},
		},
	}

	for _, c := range cases {
		ir, err := compileToIR([]byte(c.source))
		if err != nil {
			t.Fatal(err)
		}
		got := ir.AppendReachableUniformVariablesFromBlock(nil, ir.Funcs[c.index].Block)
		sort.Ints(got)
		want := c.expected
		if !slices.Equal(got, want) {
			t.Errorf("test: %v, got: %v, want: %v", c, got, want)
		}
	}
}
