// Copyright 2023 The Ebitengine Authors
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

package directx

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type stencilMode int

const (
	prepareStencil stencilMode = iota
	drawWithStencil
	noStencil
)

type pipelineStateKey struct {
	blend       graphicsdriver.Blend
	stencilMode stencilMode
	screen      bool
}

type Shader struct {
	graphics       *Graphics
	id             graphicsdriver.ShaderID
	uniformTypes   []shaderir.Type
	uniformOffsets []int
	vertexShader   *_ID3DBlob
	pixelShader    *_ID3DBlob
	pipelineStates map[pipelineStateKey]*_ID3D12PipelineState
}

func (s *Shader) ID() graphicsdriver.ShaderID {
	return s.id
}

func (s *Shader) Dispose() {
	s.graphics.removeShader(s)
}

func (s *Shader) disposeImpl() {
	for c, p := range s.pipelineStates {
		p.Release()
		delete(s.pipelineStates, c)
	}

	if s.pixelShader != nil {
		s.pixelShader.Release()
		s.pixelShader = nil
	}
	if s.vertexShader != nil {
		count := s.vertexShader.Release()
		if count == 0 {
			for k, v := range vertexShaderCache {
				if v == s.vertexShader {
					delete(vertexShaderCache, k)
				}
			}
		}
		s.vertexShader = nil
	}
}

func (s *Shader) pipelineState(blend graphicsdriver.Blend, stencilMode stencilMode, screen bool) (*_ID3D12PipelineState, error) {
	key := pipelineStateKey{
		blend:       blend,
		stencilMode: stencilMode,
		screen:      screen,
	}
	if state, ok := s.pipelineStates[key]; ok {
		return state, nil
	}

	state, err := s.graphics.pipelineStates.newPipelineState(s.graphics.device, s.vertexShader, s.pixelShader, blend, stencilMode, screen)
	if err != nil {
		return nil, err
	}
	if s.pipelineStates == nil {
		s.pipelineStates = map[pipelineStateKey]*_ID3D12PipelineState{}
	}
	s.pipelineStates[key] = state
	return state, nil
}

func (s *Shader) adjustUniforms(uniforms []uint32, shader *Shader) []uint32 {
	var fs []uint32
	var idx int
	for i, typ := range shader.uniformTypes {
		if len(fs) < s.uniformOffsets[i]/4 {
			fs = append(fs, make([]uint32, s.uniformOffsets[i]/4-len(fs))...)
		}

		n := typ.Uint32Count()
		switch typ.Main {
		case shaderir.Float:
			fs = append(fs, uniforms[idx:idx+1]...)
		case shaderir.Int:
			fs = append(fs, uniforms[idx:idx+1]...)
		case shaderir.Vec2, shaderir.IVec2:
			fs = append(fs, uniforms[idx:idx+2]...)
		case shaderir.Vec3, shaderir.IVec3:
			fs = append(fs, uniforms[idx:idx+3]...)
		case shaderir.Vec4, shaderir.IVec4:
			fs = append(fs, uniforms[idx:idx+4]...)
		case shaderir.Mat2:
			fs = append(fs,
				uniforms[idx+0], uniforms[idx+2], 0, 0,
				uniforms[idx+1], uniforms[idx+3],
			)
		case shaderir.Mat3:
			fs = append(fs,
				uniforms[idx+0], uniforms[idx+3], uniforms[idx+6], 0,
				uniforms[idx+1], uniforms[idx+4], uniforms[idx+7], 0,
				uniforms[idx+2], uniforms[idx+5], uniforms[idx+8],
			)
		case shaderir.Mat4:
			if i == graphics.ProjectionMatrixUniformVariableIndex {
				// In DirectX, the NDC's Y direction (upward) and the framebuffer's Y direction (downward) don't
				// match. Then, the Y direction must be inverted.
				// Invert the sign bits as float32 values.
				fs = append(fs,
					uniforms[idx+0], uniforms[idx+4], uniforms[idx+8], uniforms[idx+12],
					uniforms[idx+1]^(1<<31), uniforms[idx+5]^(1<<31), uniforms[idx+9]^(1<<31), uniforms[idx+13]^(1<<31),
					uniforms[idx+2], uniforms[idx+6], uniforms[idx+10], uniforms[idx+14],
					uniforms[idx+3], uniforms[idx+7], uniforms[idx+11], uniforms[idx+15],
				)
			} else {
				fs = append(fs,
					uniforms[idx+0], uniforms[idx+4], uniforms[idx+8], uniforms[idx+12],
					uniforms[idx+1], uniforms[idx+5], uniforms[idx+9], uniforms[idx+13],
					uniforms[idx+2], uniforms[idx+6], uniforms[idx+10], uniforms[idx+14],
					uniforms[idx+3], uniforms[idx+7], uniforms[idx+11], uniforms[idx+15],
				)
			}
		case shaderir.Array:
			// Each element is aligned to the boundary.
			switch typ.Sub[0].Main {
			case shaderir.Float:
				for j := 0; j < typ.Length; j++ {
					fs = append(fs, uniforms[idx+j])
					if j < typ.Length-1 {
						fs = append(fs, 0, 0, 0)
					}
				}
			case shaderir.Int:
				for j := 0; j < typ.Length; j++ {
					fs = append(fs, uniforms[idx+j])
					if j < typ.Length-1 {
						fs = append(fs, 0, 0, 0)
					}
				}
			case shaderir.Vec2, shaderir.IVec2:
				for j := 0; j < typ.Length; j++ {
					fs = append(fs, uniforms[idx+2*j:idx+2*(j+1)]...)
					if j < typ.Length-1 {
						fs = append(fs, 0, 0)
					}
				}
			case shaderir.Vec3, shaderir.IVec3:
				for j := 0; j < typ.Length; j++ {
					fs = append(fs, uniforms[idx+3*j:idx+3*(j+1)]...)
					if j < typ.Length-1 {
						fs = append(fs, 0)
					}
				}
			case shaderir.Vec4, shaderir.IVec4:
				fs = append(fs, uniforms[idx:idx+4*typ.Length]...)
			case shaderir.Mat2:
				for j := 0; j < typ.Length; j++ {
					u := uniforms[idx+4*j : idx+4*(j+1)]
					fs = append(fs,
						u[0], u[2], 0, 0,
						u[1], u[3], 0, 0,
					)
				}
				if typ.Length > 0 {
					fs = fs[:len(fs)-2]
				}
			case shaderir.Mat3:
				for j := 0; j < typ.Length; j++ {
					u := uniforms[idx+9*j : idx+9*(j+1)]
					fs = append(fs,
						u[0], u[3], u[6], 0,
						u[1], u[4], u[7], 0,
						u[2], u[5], u[8], 0,
					)
				}
				if typ.Length > 0 {
					fs = fs[:len(fs)-1]
				}
			case shaderir.Mat4:
				for j := 0; j < typ.Length; j++ {
					u := uniforms[idx+16*j : idx+16*(j+1)]
					fs = append(fs,
						u[0], u[4], u[8], u[12],
						u[1], u[5], u[9], u[13],
						u[2], u[6], u[10], u[14],
						u[3], u[7], u[11], u[15],
					)
				}
			default:
				panic(fmt.Sprintf("directx: not implemented type for uniform variables: %s", typ.String()))
			}
		default:
			panic(fmt.Sprintf("directx: not implemented type for uniform variables: %s", typ.String()))
		}

		idx += n
	}
	return fs
}
