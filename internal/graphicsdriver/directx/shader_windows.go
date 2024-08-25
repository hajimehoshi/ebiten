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
	"sync"
	"unsafe"

	"golang.org/x/sync/errgroup"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/hlsl"
)

const (
	VertexShaderProfile = "vs_4_0"
	PixelShaderProfile  = "ps_4_0"

	VertexShaderEntryPoint = "VSMain"
	PixelShaderEntryPoint  = "PSMain"
)

type fxcPair struct {
	vertex []byte
	pixel  []byte
}

type precompiledFXCs struct {
	binaries map[shaderir.SourceHash]fxcPair
	m        sync.Mutex
}

func (c *precompiledFXCs) put(hash shaderir.SourceHash, vertex, pixel []byte) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.binaries == nil {
		c.binaries = map[shaderir.SourceHash]fxcPair{}
	}
	if _, ok := c.binaries[hash]; ok {
		panic(fmt.Sprintf("directx: the precompiled library for the hash %s is already registered", hash.String()))
	}
	c.binaries[hash] = fxcPair{
		vertex: vertex,
		pixel:  pixel,
	}
}

func (c *precompiledFXCs) get(hash shaderir.SourceHash) ([]byte, []byte) {
	c.m.Lock()
	defer c.m.Unlock()

	f := c.binaries[hash]
	return f.vertex, f.pixel
}

var thePrecompiledFXCs precompiledFXCs

func RegisterPrecompiledFXCs(source []byte, vertex, pixel []byte) {
	thePrecompiledFXCs.put(shaderir.CalcSourceHash(source), vertex, pixel)
}

var vertexShaderCache = map[string]*_ID3DBlob{}

func compileShader(program *shaderir.Program) (vsh, psh *_ID3DBlob, ferr error) {
	defer func() {
		if ferr == nil {
			return
		}
		if vsh != nil {
			vsh.Release()
		}
		if psh != nil {
			psh.Release()
		}
	}()

	if vshBin, pshBin := thePrecompiledFXCs.get(program.SourceHash); vshBin != nil && pshBin != nil {
		var err error
		if vsh, err = _D3DCreateBlob(uint(len(vshBin))); err != nil {
			return nil, nil, err
		}
		if psh, err = _D3DCreateBlob(uint(len(pshBin))); err != nil {
			return nil, nil, err
		}
		copy(unsafe.Slice((*byte)(vsh.GetBufferPointer()), vsh.GetBufferSize()), vshBin)
		copy(unsafe.Slice((*byte)(psh.GetBufferPointer()), psh.GetBufferSize()), pshBin)
		return vsh, psh, nil
	}

	vs, ps, _ := hlsl.Compile(program)
	var flag uint32 = uint32(_D3DCOMPILE_OPTIMIZATION_LEVEL3)

	var wg errgroup.Group

	// Vertex shaders are likely the same. If so, reuse the same _ID3DBlob.
	if v, ok := vertexShaderCache[vs]; ok {
		// Increment the reference count not to release this object unexpectedly.
		// The value will be removed when the count reached 0.
		// See (*Shader).disposeImpl.
		v.AddRef()
		vsh = v
	} else {
		defer func() {
			if ferr == nil {
				vertexShaderCache[vs] = vsh
			}
		}()
		wg.Go(func() error {
			v, err := _D3DCompile([]byte(vs), "shader", nil, nil, VertexShaderEntryPoint, VertexShaderProfile, flag, 0)
			if err != nil {
				return fmt.Errorf("directx: D3DCompile for VSMain failed, original source: %s, %w", vs, err)
			}
			vsh = v
			return nil
		})
	}
	wg.Go(func() error {
		p, err := _D3DCompile([]byte(ps), "shader", nil, nil, PixelShaderEntryPoint, PixelShaderProfile, flag, 0)
		if err != nil {
			return fmt.Errorf("directx: D3DCompile for PSMain failed, original source: %s, %w", ps, err)
		}
		psh = p
		return nil
	})

	if err := wg.Wait(); err != nil {
		return nil, nil, err
	}

	return vsh, psh, nil
}

func constantBufferSize(uniformTypes []shaderir.Type, uniformOffsets []int) int {
	var size int
	for i, typ := range uniformTypes {
		if size < uniformOffsets[i]/4 {
			size = uniformOffsets[i] / 4
		}

		switch typ.Main {
		case shaderir.Float:
			size += 1
		case shaderir.Int:
			size += 1
		case shaderir.Vec2, shaderir.IVec2:
			size += 2
		case shaderir.Vec3, shaderir.IVec3:
			size += 3
		case shaderir.Vec4, shaderir.IVec4:
			size += 4
		case shaderir.Mat2:
			size += 6
		case shaderir.Mat3:
			size += 11
		case shaderir.Mat4:
			size += 16
		case shaderir.Array:
			// Each element is aligned to the boundary.
			switch typ.Sub[0].Main {
			case shaderir.Float:
				size += 4*(typ.Length-1) + 1
			case shaderir.Int:
				size += 4*(typ.Length-1) + 1
			case shaderir.Vec2, shaderir.IVec2:
				size += 4*(typ.Length-1) + 2
			case shaderir.Vec3, shaderir.IVec3:
				size += 4*(typ.Length-1) + 3
			case shaderir.Vec4, shaderir.IVec4:
				size += 4 * typ.Length
			case shaderir.Mat2:
				size += 8*(typ.Length-1) + 6
			case shaderir.Mat3:
				size += 12*(typ.Length-1) + 11
			case shaderir.Mat4:
				size += 16 * typ.Length
			default:
				panic(fmt.Sprintf("directx: not implemented type for uniform variables: %s", typ.String()))
			}
		default:
			panic(fmt.Sprintf("directx: not implemented type for uniform variables: %s", typ.String()))
		}
	}
	return size
}

func adjustUniforms(uniformTypes []shaderir.Type, uniformOffsets []int, uniforms []uint32) []uint32 {
	var fs []uint32
	var idx int
	for i, typ := range uniformTypes {
		if len(fs) < uniformOffsets[i]/4 {
			fs = append(fs, make([]uint32, uniformOffsets[i]/4-len(fs))...)
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
