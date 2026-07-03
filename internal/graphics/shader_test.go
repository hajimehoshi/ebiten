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

package graphics_test

import (
	"bufio"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
)

func TestCompileShaderUnitDirective(t *testing.T) {
	const fragment = `package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`
	cases := []struct {
		name string
		src  string
		err  bool
	}{
		{
			name: "no directive",
			src:  fragment,
			err:  true,
		},
		{
			name: "texels",
			src:  "//kage:unit texels\n\n" + fragment,
			err:  true,
		},
		{
			name: "invalid value",
			src:  "//kage:unit foo\n\n" + fragment,
			err:  true,
		},
		{
			name: "pixels",
			src:  "//kage:unit pixels\n\n" + fragment,
			err:  false,
		},
		{
			name: "duplicated",
			src:  "//kage:unit pixels\n//kage:unit pixels\n\n" + fragment,
			err:  true,
		},
		{
			name: "pixels after a long line",
			src:  "// " + strings.Repeat("a", bufio.MaxScanTokenSize) + "\n//kage:unit pixels\n\n" + fragment,
			err:  false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := graphics.CompileShader([]byte(c.src))
			if err == nil && c.err {
				t.Errorf("CompileShader must return an error but does not")
			} else if err != nil && !c.err {
				t.Errorf("CompileShader must not return an error but returned %v", err)
			}
		})
	}
}
