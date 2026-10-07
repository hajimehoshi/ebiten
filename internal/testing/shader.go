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

package testing

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

// ShaderProgramFill returns a shader source to fill the framebuffer.
func ShaderProgramFill(r, g, b, a byte) *shaderir.Program {
	ir, err := graphics.CompileShader(fmt.Appendf(nil, `//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	return vec4(%0.9f, %0.9f, %0.9f, %0.9f)
}
`, float64(r)/0xff, float64(g)/0xff, float64(b)/0xff, float64(a)/0xff))
	if err != nil {
		panic(err)
	}
	return ir
}
