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

// +build js

package ebiten

import (
	"github.com/gopherjs/gopherjs/js"
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

func vertices(parts ImageParts, width, height int, geo *GeoM) []float32 {
	// TODO: This function should be in graphics package?
	totalSize := graphics.QuadVertexSizeInBytes() / 4
	l := parts.Len()
	vs := js.Global.Get("Float32Array").New(l * totalSize)
	g0 := geo.Element(0, 0)
	g1 := geo.Element(0, 1)
	g2 := geo.Element(1, 0)
	g3 := geo.Element(1, 1)
	g4 := geo.Element(0, 2)
	g5 := geo.Element(1, 2)
	w := 1.0
	h := 1.0
	for w < float64(width) {
		w *= 2
	}
	for h < float64(height) {
		h *= 2
	}
	n := 0
	for i := 0; i < l; i++ {
		dx0, dy0, dx1, dy1 := parts.Dst(i)
		if dx0 == dx1 || dy0 == dy1 {
			continue
		}
		sx0, sy0, sx1, sy1 := parts.Src(i)
		if sx0 == sx1 || sy0 == sy1 {
			continue
		}
		u0, v0, u1, v1 := float64(sx0)/w, float64(sy0)/h, float64(sx1)/w, float64(sy1)/h
		vs.SetIndex(n, dx0)
		vs.SetIndex(n+1, dy0)
		vs.SetIndex(n+2, u0)
		vs.SetIndex(n+3, v0)
		vs.SetIndex(n+4, g0)
		vs.SetIndex(n+5, g1)
		vs.SetIndex(n+6, g2)
		vs.SetIndex(n+7, g3)
		vs.SetIndex(n+8, g4)
		vs.SetIndex(n+9, g5)

		vs.SetIndex(n+10, dx1)
		vs.SetIndex(n+11, dy0)
		vs.SetIndex(n+12, u1)
		vs.SetIndex(n+13, v0)
		vs.SetIndex(n+14, g0)
		vs.SetIndex(n+15, g1)
		vs.SetIndex(n+16, g2)
		vs.SetIndex(n+17, g3)
		vs.SetIndex(n+18, g4)
		vs.SetIndex(n+19, g5)

		vs.SetIndex(n+20, dx0)
		vs.SetIndex(n+21, dy1)
		vs.SetIndex(n+22, u0)
		vs.SetIndex(n+23, v1)
		vs.SetIndex(n+24, g0)
		vs.SetIndex(n+25, g1)
		vs.SetIndex(n+26, g2)
		vs.SetIndex(n+27, g3)
		vs.SetIndex(n+28, g4)
		vs.SetIndex(n+29, g5)

		vs.SetIndex(n+30, dx1)
		vs.SetIndex(n+31, dy1)
		vs.SetIndex(n+32, u1)
		vs.SetIndex(n+33, v1)
		vs.SetIndex(n+34, g0)
		vs.SetIndex(n+35, g1)
		vs.SetIndex(n+36, g2)
		vs.SetIndex(n+37, g3)
		vs.SetIndex(n+38, g4)
		vs.SetIndex(n+39, g5)

		n += totalSize
	}
	return vs.Interface().([]float32)
}
