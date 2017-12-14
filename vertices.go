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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/restorable"
)

var (
	quadFloat32Num     = restorable.QuadVertexSizeInBytes() / 4
	theVerticesBackend = &verticesBackend{}
)

type verticesBackend struct {
	backend []float32
	head    int
}

func (v *verticesBackend) get() []float32 {
	const num = 256
	if v.backend == nil {
		v.backend = make([]float32, quadFloat32Num*num)
	}
	s := v.backend[v.head : v.head+quadFloat32Num]
	v.head += quadFloat32Num
	if v.head+quadFloat32Num > len(v.backend) {
		v.backend = nil
		v.head = 0
	}
	return s
}

func vertices(sx0, sy0, sx1, sy1 int, width, height int, geo *affine.GeoM) []float32 {
	if sx0 >= sx1 || sy0 >= sy1 {
		return nil
	}
	// TODO: This function should be in graphics package?
	vs := theVerticesBackend.get()
	a, b, c, d, tx, ty := geo.Elements()
	g0 := float32(a)
	g1 := float32(b)
	g2 := float32(c)
	g3 := float32(d)
	g4 := float32(tx)
	g5 := float32(ty)
	w := 1
	h := 1
	for w < width {
		w *= 2
	}
	for h < height {
		h *= 2
	}
	wf := float32(w)
	hf := float32(h)
	x0, y0, x1, y1 := float32(0), float32(0), float32(sx1-sx0), float32(sy1-sy0)
	u0, v0, u1, v1 := float32(sx0)/wf, float32(sy0)/hf, float32(sx1)/wf, float32(sy1)/hf

	// Vertex coordinates
	vs[0] = x0
	vs[1] = y0

	// Texture coordinates: first 2 values indicates the actual coodinate, and
	// the second indicates diagonally opposite coodinates.
	// The second is needed to calculate source rectangle size in shader programs.
	vs[2] = u0
	vs[3] = v0
	vs[4] = u1
	vs[5] = v1

	// Geometry matrix
	vs[6] = g0
	vs[7] = g1
	vs[8] = g2
	vs[9] = g3
	vs[10] = g4
	vs[11] = g5

	vs[12] = x1
	vs[13] = y0
	vs[14] = u1
	vs[15] = v0
	vs[16] = u0
	vs[17] = v1
	vs[18] = g0
	vs[19] = g1
	vs[20] = g2
	vs[21] = g3
	vs[22] = g4
	vs[23] = g5

	vs[24] = x0
	vs[25] = y1
	vs[26] = u0
	vs[27] = v1
	vs[28] = u1
	vs[29] = v0
	vs[30] = g0
	vs[31] = g1
	vs[32] = g2
	vs[33] = g3
	vs[34] = g4
	vs[35] = g5

	vs[36] = x1
	vs[37] = y1
	vs[38] = u1
	vs[39] = v1
	vs[40] = u0
	vs[41] = v0
	vs[42] = g0
	vs[43] = g1
	vs[44] = g2
	vs[45] = g3
	vs[46] = g4
	vs[47] = g5

	return vs
}
