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

package mipmap

import (
	"sync"

	"github.com/hajimehoshi/ebiten/internal/graphics"
)

var (
	theVerticesBackend = &verticesBackend{
		backend: make([]float32, graphics.VertexFloatNum*1024),
	}
)

type verticesBackend struct {
	backend []float32
	head    int
	m       sync.Mutex
}

func (v *verticesBackend) slice(n int, last bool) []float32 {
	v.m.Lock()

	need := n * graphics.VertexFloatNum
	if l := len(v.backend); v.head+need > l {
		for v.head+need > l {
			l *= 2
		}
		v.backend = make([]float32, l)
		v.head = 0
	}

	s := v.backend[v.head : v.head+need]
	if last {
		// If last is true, the vertices backend is sent to GPU and it is fine to reuse the slice.
		v.head = 0
	} else {
		v.head += need
	}

	v.m.Unlock()
	return s
}

func vertexSlice(n int, last bool) []float32 {
	return theVerticesBackend.slice(n, last)
}

func quadVertices(sx0, sy0, sx1, sy1 int, a, b, c, d, tx, ty float32, cr, cg, cb, ca float32, last bool) []float32 {
	x := float32(sx1 - sx0)
	y := float32(sy1 - sy0)
	ax, by, cx, dy := a*x, b*y, c*x, d*y
	u0, v0, u1, v1 := float32(sx0), float32(sy0), float32(sx1), float32(sy1)

	// This function is very performance-sensitive and implement in a very dumb way.
	vs := vertexSlice(4, last)
	_ = vs[:48]

	vs[0] = tx
	vs[1] = ty
	vs[2] = u0
	vs[3] = v0
	vs[4] = u0
	vs[5] = v0
	vs[6] = u1
	vs[7] = v1
	vs[8] = cr
	vs[9] = cg
	vs[10] = cb
	vs[11] = ca

	vs[12] = ax + tx
	vs[13] = cx + ty
	vs[14] = u1
	vs[15] = v0
	vs[16] = u0
	vs[17] = v0
	vs[18] = u1
	vs[19] = v1
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
	vs[30] = u1
	vs[31] = v1
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
	vs[42] = u1
	vs[43] = v1
	vs[44] = cr
	vs[45] = cg
	vs[46] = cb
	vs[47] = ca

	return vs
}
