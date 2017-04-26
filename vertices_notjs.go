// Copyright 2016 The Ebiten Authors
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

// +build !js

package ebiten

import (
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

func vertices(parts ImageParts, width, height int, geo *affine.GeoM) []float32 {
	// TODO: This function should be in graphics package?
	totalSize := graphics.QuadVertexSizeInBytes() / 4
	l := parts.Len()
	vs := make([]float32, l*totalSize)
	g := geo.UnsafeElements()
	g0 := float32(g[0])
	g1 := float32(g[1])
	g2 := float32(g[3])
	g3 := float32(g[4])
	g4 := float32(g[2])
	g5 := float32(g[5])
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
	n := 0
	for i := 0; i < l; i++ {
		dx0, dy0, dx1, dy1 := parts.Dst(i)
		if dx0 == dx1 || dy0 == dy1 {
			continue
		}
		x0, y0, x1, y1 := float32(dx0), float32(dy0), float32(dx1), float32(dy1)
		sx0, sy0, sx1, sy1 := parts.Src(i)
		if sx0 == sx1 || sy0 == sy1 {
			continue
		}
		u0, v0, u1, v1 := float32(sx0)/wf, float32(sy0)/hf, float32(sx1)/wf, float32(sy1)/hf
		// Adjust texels to fix a problem that outside texels are used (#317).
		u1 -= 1.0 / wf / texelAdjustment
		v1 -= 1.0 / hf / texelAdjustment
		vs[n] = x0
		vs[n+1] = y0
		vs[n+2] = u0
		vs[n+3] = v0
		vs[n+4] = g0
		vs[n+5] = g1
		vs[n+6] = g2
		vs[n+7] = g3
		vs[n+8] = g4
		vs[n+9] = g5

		vs[n+10] = x1
		vs[n+11] = y0
		vs[n+12] = u1
		vs[n+13] = v0
		vs[n+14] = g0
		vs[n+15] = g1
		vs[n+16] = g2
		vs[n+17] = g3
		vs[n+18] = g4
		vs[n+19] = g5

		vs[n+20] = x0
		vs[n+21] = y1
		vs[n+22] = u0
		vs[n+23] = v1
		vs[n+24] = g0
		vs[n+25] = g1
		vs[n+26] = g2
		vs[n+27] = g3
		vs[n+28] = g4
		vs[n+29] = g5

		vs[n+30] = x1
		vs[n+31] = y1
		vs[n+32] = u1
		vs[n+33] = v1
		vs[n+34] = g0
		vs[n+35] = g1
		vs[n+36] = g2
		vs[n+37] = g3
		vs[n+38] = g4
		vs[n+39] = g5

		n += totalSize
	}
	return vs
}
