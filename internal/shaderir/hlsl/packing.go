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

const boundaryInBytes = 16

func CalcUniformMemoryOffsets(program *shaderir.Program) []int {
	// https://docs.microsoft.com/en-us/windows/win32/direct3dhlsl/dx-graphics-hlsl-packing-rules
	// https://github.com/microsoft/DirectXShaderCompiler/wiki/Buffer-Packing

	var offsets []int
	var head int

	align := func(x int) int {
		if x == 0 {
			return 0
		}
		return ((x-1)/boundaryInBytes + 1) * boundaryInBytes
	}

	// TODO: Reorder the variables with packoffset.
	// See https://docs.microsoft.com/en-us/windows/win32/direct3dhlsl/dx-graphics-hlsl-variable-packoffset
	for _, u := range program.Uniforms {
		switch u.Main {
		case shaderir.Float:
			offsets = append(offsets, head)
			head += 4
		case shaderir.Int:
			offsets = append(offsets, head)
			head += 4
		case shaderir.Vec2, shaderir.IVec2:
			if head%boundaryInBytes >= 4*3 {
				head = align(head)
			}
			offsets = append(offsets, head)
			head += 4 * 2
		case shaderir.Vec3, shaderir.IVec3:
			if head%boundaryInBytes >= 4*2 {
				head = align(head)
			}
			offsets = append(offsets, head)
			head += 4 * 3
		case shaderir.Vec4, shaderir.IVec4:
			if head%boundaryInBytes >= 4*1 {
				head = align(head)
			}
			offsets = append(offsets, head)
			head += 4 * 4
		case shaderir.Mat2:
			// For matrices, each column is aligned to the boundary.
			head = align(head)
			offsets = append(offsets, head)
			head += 4 * 6
		case shaderir.Mat3:
			head = align(head)
			offsets = append(offsets, head)
			head += 4 * 11
		case shaderir.Mat4:
			head = align(head)
			offsets = append(offsets, head)
			head += 4 * 16
		case shaderir.Array:
			// Each array is 16-byte aligned.
			// TODO: What if the array has 2 or more dimensions?
			head = align(head)
			offsets = append(offsets, head)
			n := u.Sub[0].Uint32Count()
			switch u.Sub[0].Main {
			case shaderir.Mat2:
				n = 6
			case shaderir.Mat3:
				n = 11
			case shaderir.Mat4:
				n = 16
			}
			head += (u.Length - 1) * align(4*n)
			// The last element is not with a padding.
			head += 4 * n
		case shaderir.Struct:
			// TODO: Implement this
			panic("hlsl: offset for a struct is not implemented yet")
		default:
			panic(fmt.Sprintf("hlsl: unexpected type: %s", u.String()))
		}
	}

	return offsets
}
