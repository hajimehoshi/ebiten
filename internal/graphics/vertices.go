// Copyright 2017 The Ebiten Authors
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
	"fmt"
	"sync"
)

var (
	theVerticesBackend = &verticesBackend{}
)

type verticesBackend struct {
	backend []float32
	head    int
	m       sync.Mutex
}

const (
	IndicesNum     = (1 << 16) / 3 * 3 // Adjust num for triangles.
	VertexFloatNum = 12
)

func (v *verticesBackend) slice(n int) []float32 {
	const num = 1024
	if n > num {
		panic(fmt.Sprintf("graphics: n must be <= num but not: n: %d, num: %d", n, num))
	}

	v.m.Lock()

	need := n * VertexFloatNum
	if v.head+need > len(v.backend) {
		v.backend = nil
		v.head = 0
	}

	if v.backend == nil {
		v.backend = make([]float32, VertexFloatNum*num)
	}

	s := v.backend[v.head : v.head+need]
	v.head += need

	v.m.Unlock()
	return s
}

func QuadVertices(width, height int, sx0, sy0, sx1, sy1 int, a, b, c, d, tx, ty float32, cr, cg, cb, ca float32) []float32 {
	// width and height are the source image's size.

	// For performance reason, graphics.InternalImageSize is not applied to width/height here.

	if !isInternalImageSize(width) {
		panic(fmt.Sprintf("graphics: width must be an internal image size at QuadVertices: %d", width))
	}
	if !isInternalImageSize(height) {
		panic(fmt.Sprintf("graphics: height must be an internal image size at QuadVertices: %d", height))
	}

	if sx0 >= sx1 || sy0 >= sy1 {
		return nil
	}
	if sx1 <= 0 || sy1 <= 0 {
		return nil
	}

	wf := float32(width)
	hf := float32(height)
	u0, v0, u1, v1 := float32(sx0)/wf, float32(sy0)/hf, float32(sx1)/wf, float32(sy1)/hf
	return quadVerticesImpl(wf, hf, float32(sx1-sx0), float32(sy1-sy0), u0, v0, u1, v1, a, b, c, d, tx, ty, cr, cg, cb, ca)
}

const TexelAdjustmentFactor = 512.0

func quadVerticesImpl(sw, sh, x, y, u0, v0, u1, v1, a, b, c, d, tx, ty, cr, cg, cb, ca float32) []float32 {
	// Specifying a range explicitly here is redundant but this helps optimization
	// to eliminate boundary checks.
	//
	// 4*VertexFloatNum is better than 48, but in GopherJS, optimization might not work.
	vs := theVerticesBackend.slice(4)[0:48]

	ax, by, cx, dy := a*x, b*y, c*x, d*y

	du := 1.0 / sw / TexelAdjustmentFactor
	dv := 1.0 / sh / TexelAdjustmentFactor

	// Vertex coordinates
	vs[0] = tx
	vs[1] = ty

	// Texture coordinates: first 2 values indicates the actual coodinate, and
	// the second indicates diagonally opposite coodinates.
	// The second is needed to calculate source rectangle size in shader programs.
	vs[2] = u0
	vs[3] = v0
	vs[4] = u0
	vs[5] = v0
	vs[6] = u1 - du
	vs[7] = v1 - dv
	vs[8] = cr
	vs[9] = cg
	vs[10] = cb
	vs[11] = ca

	// and the same for the other three coordinates
	vs[12] = ax + tx
	vs[13] = cx + ty
	vs[14] = u1
	vs[15] = v0
	vs[16] = u0
	vs[17] = v0
	vs[18] = u1 - du
	vs[19] = v1 - dv
	vs[20] = cr
	vs[21] = cg
	vs[22] = cb
	vs[23] = ca

	vs[24] = by + tx
	vs[25] = dy + ty
	vs[26] = u0
	vs[27] = v1
	vs[28] = u0
	vs[29] = v0
	vs[30] = u1 - du
	vs[31] = v1 - dv
	vs[32] = cr
	vs[33] = cg
	vs[34] = cb
	vs[35] = ca

	vs[36] = ax + by + tx
	vs[37] = cx + dy + ty
	vs[38] = u1
	vs[39] = v1
	vs[40] = u0
	vs[41] = v0
	vs[42] = u1 - du
	vs[43] = v1 - dv
	vs[44] = cr
	vs[45] = cg
	vs[46] = cb
	vs[47] = ca

	return vs
}

var (
	quadIndices = []uint16{0, 1, 2, 1, 2, 3}
)

func QuadIndices() []uint16 {
	return quadIndices
}
