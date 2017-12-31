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
	"image/color"

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
	return verticesColor(sx0, sy0, sx1, sy1, width, height, color.RGBA{255, 255, 255, 255}, color.RGBA{255, 255, 255, 255}, color.RGBA{255, 255, 255, 255}, color.RGBA{255, 255, 255, 255}, geo)
}

func verticesColor(sx0, sy0, sx1, sy1 int, width, height int, c0, c1, c2, c3 color.RGBA, geo *affine.GeoM) []float32 {
	if sx0 >= sx1 || sy0 >= sy1 {
		return nil
	}
	if sx1 <= 0 || sy1 <= 0 {
		return nil
	}

	// TODO: This function should be in graphics package?
	vs := theVerticesBackend.get()

	if sx0 < 0 || sy0 < 0 {
		dx := 0.0
		dy := 0.0
		if sx0 < 0 {
			dx = -float64(sx0)
			sx0 = 0
		}
		if sy0 < 0 {
			dy = -float64(sy0)
			sy0 = 0
		}
		g := affine.GeoM{}
		g.Translate(dx, dy)
		g.Concat(geo)
		geo = &g
	}

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
	// vertex color
	vs[12] = float32(c0.R) / 255
	vs[13] = float32(c0.G) / 255
	vs[14] = float32(c0.B) / 255
	vs[15] = float32(c0.A) / 255

	vs[16] = x1
	vs[17] = y0
	vs[18] = u1
	vs[19] = v0
	vs[20] = u0
	vs[21] = v1
	vs[22] = g0
	vs[23] = g1
	vs[24] = g2
	vs[25] = g3
	vs[26] = g4
	vs[27] = g5
	vs[28] = float32(c1.R) / 255
	vs[29] = float32(c1.G) / 255
	vs[30] = float32(c1.B) / 255
	vs[31] = float32(c1.A) / 255

	vs[32] = x0
	vs[33] = y1
	vs[34] = u0
	vs[35] = v1
	vs[36] = u1
	vs[37] = v0
	vs[38] = g0
	vs[39] = g1
	vs[40] = g2
	vs[41] = g3
	vs[42] = g4
	vs[43] = g5
	vs[44] = float32(c2.R) / 255
	vs[45] = float32(c2.G) / 255
	vs[46] = float32(c2.B) / 255
	vs[47] = float32(c2.A) / 255

	vs[48] = x1
	vs[49] = y1
	vs[50] = u1
	vs[51] = v1
	vs[52] = u0
	vs[53] = v0
	vs[54] = g0
	vs[55] = g1
	vs[56] = g2
	vs[57] = g3
	vs[58] = g4
	vs[59] = g5
	vs[60] = float32(c3.R) / 255
	vs[61] = float32(c3.G) / 255
	vs[62] = float32(c3.B) / 255
	vs[63] = float32(c3.A) / 255

	return vs
}
