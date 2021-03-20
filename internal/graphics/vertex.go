// Copyright 2019 The Ebiten Authors
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

package graphics

import (
	"sync"

	"github.com/hajimehoshi/ebiten/internal/web"
)

const (
	ShaderImageNum = 4

	// PreservedUniformVariablesNum represents the number of preserved uniform variables.
	// Any shaders in Ebiten must have these uniform variables.
	PreservedUniformVariablesNum = 1 + // the destination texture size
		1 + // the texture sizes array
		1 + // the offsets array of the second and the following images
		1 + // the texture source region's origin
		1 // the texture source region's size

	DestinationTextureSizeUniformVariableIndex    = 0
	TextureSizesUniformVariableIndex              = 1
	TextureSourceOffsetsUniformVariableIndex      = 2
	TextureSourceRegionOriginUniformVariableIndex = 3
	TextureSourceRegionSizeUniformVariableIndex   = 4
)

const (
	IndicesNum     = (1 << 16) / 3 * 3 // Adjust num for triangles.
	VertexFloatNum = 8
)

var (
	quadIndices = []uint16{0, 1, 2, 1, 2, 3}
)

func QuadIndices() []uint16 {
	return quadIndices
}

var (
	theVerticesBackend = &verticesBackend{}
)

type verticesBackend struct {
	backend []float32
	head    int
	m       sync.Mutex
}

func (v *verticesBackend) slice(n int) []float32 {
	v.m.Lock()
	defer v.m.Unlock()

	need := n * VertexFloatNum
	if v.head+need > len(v.backend) {
		v.backend = nil
		v.head = 0
	}
	if v.backend == nil {
		l := 1024
		if n > l {
			l = n
		}
		v.backend = make([]float32, VertexFloatNum*l)
	}

	s := v.backend[v.head : v.head+need]
	v.head += need
	return s
}

func vertexSlice(n int) []float32 {
	if web.IsBrowser() {
		// In GopherJS and Wasm, allocating memory by make is expensive. Use the backend instead.
		return theVerticesBackend.slice(n)
	}
	return make([]float32, n*VertexFloatNum)
}

func QuadVertices(sx0, sy0, sx1, sy1 float32, a, b, c, d, tx, ty float32, cr, cg, cb, ca float32) []float32 {
	x := sx1 - sx0
	y := sy1 - sy0
	ax, by, cx, dy := a*x, b*y, c*x, d*y
	u0, v0, u1, v1 := float32(sx0), float32(sy0), float32(sx1), float32(sy1)

	// This function is very performance-sensitive and implement in a very dumb way.
	vs := vertexSlice(4)
	_ = vs[:32]

	vs[0] = tx
	vs[1] = ty
	vs[2] = u0
	vs[3] = v0
	vs[4] = cr
	vs[5] = cg
	vs[6] = cb
	vs[7] = ca

	vs[8] = ax + tx
	vs[9] = cx + ty
	vs[10] = u1
	vs[11] = v0
	vs[12] = cr
	vs[13] = cg
	vs[14] = cb
	vs[15] = ca

	vs[16] = by + tx
	vs[17] = dy + ty
	vs[18] = u0
	vs[19] = v1
	vs[20] = cr
	vs[21] = cg
	vs[22] = cb
	vs[23] = ca

	vs[24] = ax + by + tx
	vs[25] = cx + dy + ty
	vs[26] = u1
	vs[27] = v1
	vs[28] = cr
	vs[29] = cg
	vs[30] = cb
	vs[31] = ca

	return vs
}
