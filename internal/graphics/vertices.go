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

const (
	IndicesNum     = (1 << 16) / 3 * 3 // Adjust num for triangles.
	VertexFloatNum = 12
)

type VertexPutter interface {
	PutVertex(dst []float32, dx, dy, sx, sy float32, bx0, by0, bx1, by1 float32, cr, cg, cb, ca float32)
}

func PutQuadVertices(dst []float32, putter VertexPutter, sx0, sy0, sx1, sy1 int, a, b, c, d, tx, ty float32, cr, cg, cb, ca float32) {
	x := float32(sx1 - sx0)
	y := float32(sy1 - sy0)
	ax, by, cx, dy := a*x, b*y, c*x, d*y
	u0, v0, u1, v1 := float32(sx0), float32(sy0), float32(sx1), float32(sy1)
	putter.PutVertex(dst[:VertexFloatNum], tx, ty, u0, v0, u0, v0, u1, v1, cr, cg, cb, ca)
	putter.PutVertex(dst[VertexFloatNum:2*VertexFloatNum], ax+tx, cx+ty, u1, v0, u0, v0, u1, v1, cr, cg, cb, ca)
	putter.PutVertex(dst[2*VertexFloatNum:3*VertexFloatNum], by+tx, dy+ty, u0, v1, u0, v0, u1, v1, cr, cg, cb, ca)
	putter.PutVertex(dst[3*VertexFloatNum:4*VertexFloatNum], ax+by+tx, cx+dy+ty, u1, v1, u0, v0, u1, v1, cr, cg, cb, ca)
}

var (
	quadIndices = []uint16{0, 1, 2, 1, 2, 3}
)

func QuadIndices() []uint16 {
	return quadIndices
}
