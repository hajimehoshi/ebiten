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

const UniformVariableBoundaryInDWords = 4

// UniformVariableOffsetsInDWords returns the offsets of the uniform variables in DWROD units in the HLSL layout.
func UniformVariableOffsetsInDWords(program *shaderir.Program) []int {
	// https://docs.microsoft.com/en-us/windows/win32/direct3dhlsl/dx-graphics-hlsl-packing-rules
	// https://github.com/microsoft/DirectXShaderCompiler/wiki/Buffer-Packing

	align := func(x int) int {
		if x == 0 {
			return 0
		}
		return ((x-1)/UniformVariableBoundaryInDWords + 1) * UniformVariableBoundaryInDWords
	}

	var offsetsInDWords []int
	var headInDWords int

	// TODO: Reorder the variables with packoffset.
	// See https://docs.microsoft.com/en-us/windows/win32/direct3dhlsl/dx-graphics-hlsl-variable-packoffset
	for _, u := range program.Uniforms {
		switch u.Main {
		case shaderir.Float:
			offsetsInDWords = append(offsetsInDWords, headInDWords)
			headInDWords += 1
		case shaderir.Int:
			offsetsInDWords = append(offsetsInDWords, headInDWords)
			headInDWords += 1
		case shaderir.Vec2, shaderir.IVec2:
			if headInDWords%UniformVariableBoundaryInDWords >= 3 {
				headInDWords = align(headInDWords)
			}
			offsetsInDWords = append(offsetsInDWords, headInDWords)
			headInDWords += 2
		case shaderir.Vec3, shaderir.IVec3:
			if headInDWords%UniformVariableBoundaryInDWords >= 2 {
				headInDWords = align(headInDWords)
			}
			offsetsInDWords = append(offsetsInDWords, headInDWords)
			headInDWords += 3
		case shaderir.Vec4, shaderir.IVec4:
			if headInDWords%UniformVariableBoundaryInDWords >= 1 {
				headInDWords = align(headInDWords)
			}
			offsetsInDWords = append(offsetsInDWords, headInDWords)
			headInDWords += 4
		case shaderir.Mat2:
			// For matrices, each column is aligned to the boundary.
			headInDWords = align(headInDWords)
			offsetsInDWords = append(offsetsInDWords, headInDWords)
			headInDWords += 6
		case shaderir.Mat3:
			headInDWords = align(headInDWords)
			offsetsInDWords = append(offsetsInDWords, headInDWords)
			headInDWords += 11
		case shaderir.Mat4:
			headInDWords = align(headInDWords)
			offsetsInDWords = append(offsetsInDWords, headInDWords)
			headInDWords += 16
		case shaderir.Array:
			// Each array is 16-byte aligned.
			// TODO: What if the array has 2 or more dimensions?
			headInDWords = align(headInDWords)
			offsetsInDWords = append(offsetsInDWords, headInDWords)
			n := u.Sub[0].Uint32Count()
			switch u.Sub[0].Main {
			case shaderir.Mat2:
				n = 6
			case shaderir.Mat3:
				n = 11
			case shaderir.Mat4:
				n = 16
			}
			headInDWords += (u.Length - 1) * align(n)
			// The last element is not with a padding.
			headInDWords += n
		case shaderir.Struct:
			// TODO: Implement this
			panic("hlsl: offset for a struct is not implemented yet")
		default:
			panic(fmt.Sprintf("hlsl: unexpected type: %s", u.String()))
		}
	}

	return offsetsInDWords
}
