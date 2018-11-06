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

var (
	theVerticesBackend = &verticesBackend{}
)

type verticesBackend struct {
	backend []float32
	head    int
}

const (
	IndicesNum     = (1 << 16) / 3 * 3 // Adjust num for triangles.
	VertexFloatNum = 10
)

func (v *verticesBackend) slice(n int) []float32 {
	const num = 1024
	if n > num {
		panic("not reached")
	}

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
	return s
}

func isPowerOf2(x int) bool {
	return (x & (x - 1)) == 0
}

func QuadVertices(width, height int, sx0, sy0, sx1, sy1 int, a, b, c, d, tx, ty float32, cr, cg, cb, ca float32) []float32 {
	if !isPowerOf2(width) {
		panic("not reached")
	}
	if !isPowerOf2(height) {
		panic("not reached")
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
	return quadVerticesImpl(float32(sx1-sx0), float32(sy1-sy0), u0, v0, u1, v1, a, b, c, d, tx, ty, cr, cg, cb, ca)
}

func quadVerticesImpl(x, y, u0, v0, u1, v1, a, b, c, d, tx, ty, cr, cg, cb, ca float32) []float32 {
	// Specifying a range explicitly here is redundant but this helps optimization
	// to eliminate boundry checks.
	//
	// 4*VertexFloatNum is better than 40, but in GopherJS, optimization might not work.
	vs := theVerticesBackend.slice(4)[0:40]

	ax, by, cx, dy := a*x, b*y, c*x, d*y

	// Vertex coordinates
	vs[0] = tx
	vs[1] = ty

	// Texture coordinates: first 2 values indicates the actual coodinate, and
	// the second indicates diagonally opposite coodinates.
	// The second is needed to calculate source rectangle size in shader programs.
	vs[2] = u0
	vs[3] = v0
	vs[4] = u1
	vs[5] = v1
	vs[6] = cr
	vs[7] = cg
	vs[8] = cb
	vs[9] = ca

	// and the same for the other three coordinates
	vs[10] = ax + tx
	vs[11] = cx + ty
	vs[12] = u1
	vs[13] = v0
	vs[14] = u0
	vs[15] = v1
	vs[16] = cr
	vs[17] = cg
	vs[18] = cb
	vs[19] = ca

	vs[20] = by + tx
	vs[21] = dy + ty
	vs[22] = u0
	vs[23] = v1
	vs[24] = u1
	vs[25] = v0
	vs[26] = cr
	vs[27] = cg
	vs[28] = cb
	vs[29] = ca

	vs[30] = ax + by + tx
	vs[31] = cx + dy + ty
	vs[32] = u1
	vs[33] = v1
	vs[34] = u0
	vs[35] = v0
	vs[36] = cr
	vs[37] = cg
	vs[38] = cb
	vs[39] = ca

	return vs
}

var (
	quadIndices = []uint16{0, 1, 2, 1, 2, 3}
)

func QuadIndices() []uint16 {
	return quadIndices
}

func PutVertex(vs []float32, width, height int, dx, dy, sx, sy float32, cr, cg, cb, ca float32) {
	if !isPowerOf2(width) {
		panic("not reached")
	}
	if !isPowerOf2(height) {
		panic("not reached")
	}

	wf := float32(width)
	hf := float32(height)

	// Specify -1 for the source region, which means the source region is ignored.
	//
	// NaN would make more sense to represent an invalid state, but vertices including NaN values doesn't work on
	// some machines (#696). Let's use negative numbers to represent such state.
	vs[0] = dx
	vs[1] = dy
	vs[2] = sx / wf
	vs[3] = sy / hf
	vs[4] = -1
	vs[5] = -1
	vs[6] = cr
	vs[7] = cg
	vs[8] = cb
	vs[9] = ca
}
