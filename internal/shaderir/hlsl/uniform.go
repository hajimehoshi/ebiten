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

package hlsl

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

const UniformVariableBoundaryInDwords = 4

// UniformVariableOffsetsInDwords returns the offsets of the uniform variables in DWROD units in the HLSL layout.
func UniformVariableOffsetsInDwords(program *shaderir.Program) []int {
	// https://docs.microsoft.com/en-us/windows/win32/direct3dhlsl/dx-graphics-hlsl-packing-rules
	// https://github.com/microsoft/DirectXShaderCompiler/wiki/Buffer-Packing

	align := func(x int) int {
		if x == 0 {
			return 0
		}
		return ((x-1)/UniformVariableBoundaryInDwords + 1) * UniformVariableBoundaryInDwords
	}

	var offsetsInDwords []int
	var headInDwords int

	// TODO: Reorder the variables with packoffset.
	// See https://docs.microsoft.com/en-us/windows/win32/direct3dhlsl/dx-graphics-hlsl-variable-packoffset
	for _, u := range program.Uniforms {
		switch u.Main {
		case shaderir.Bool:
			offsetsInDwords = append(offsetsInDwords, headInDwords)
			headInDwords += 1
		case shaderir.Float:
			offsetsInDwords = append(offsetsInDwords, headInDwords)
			headInDwords += 1
		case shaderir.Int:
			offsetsInDwords = append(offsetsInDwords, headInDwords)
			headInDwords += 1
		case shaderir.Vec2, shaderir.IVec2:
			if headInDwords%UniformVariableBoundaryInDwords >= 3 {
				headInDwords = align(headInDwords)
			}
			offsetsInDwords = append(offsetsInDwords, headInDwords)
			headInDwords += 2
		case shaderir.Vec3, shaderir.IVec3:
			if headInDwords%UniformVariableBoundaryInDwords >= 2 {
				headInDwords = align(headInDwords)
			}
			offsetsInDwords = append(offsetsInDwords, headInDwords)
			headInDwords += 3
		case shaderir.Vec4, shaderir.IVec4:
			if headInDwords%UniformVariableBoundaryInDwords >= 1 {
				headInDwords = align(headInDwords)
			}
			offsetsInDwords = append(offsetsInDwords, headInDwords)
			headInDwords += 4
		case shaderir.Mat2:
			// For matrices, each column is aligned to the boundary.
			headInDwords = align(headInDwords)
			offsetsInDwords = append(offsetsInDwords, headInDwords)
			headInDwords += 6
		case shaderir.Mat3:
			headInDwords = align(headInDwords)
			offsetsInDwords = append(offsetsInDwords, headInDwords)
			headInDwords += 11
		case shaderir.Mat4:
			headInDwords = align(headInDwords)
			offsetsInDwords = append(offsetsInDwords, headInDwords)
			headInDwords += 16
		case shaderir.Array:
			// Each array is 16-byte aligned.
			// TODO: What if the array has 2 or more dimensions?
			headInDwords = align(headInDwords)
			offsetsInDwords = append(offsetsInDwords, headInDwords)
			n := u.Sub[0].DwordCount()
			switch u.Sub[0].Main {
			case shaderir.Mat2:
				n = 6
			case shaderir.Mat3:
				n = 11
			case shaderir.Mat4:
				n = 16
			}
			headInDwords += (u.Length - 1) * align(n)
			// The last element is not with a padding.
			headInDwords += n
		case shaderir.Struct:
			// TODO: Implement this
			panic("hlsl: offset for a struct is not implemented yet")
		default:
			panic(fmt.Sprintf("hlsl: unexpected type: %s", u.String()))
		}
	}

	return offsetsInDwords
}
