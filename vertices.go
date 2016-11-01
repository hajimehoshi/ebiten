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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

func vertices(parts ImageParts, width, height int, geo *GeoM) []float32 {
	// TODO: This function should be in graphics package?
	totalSize := graphics.QuadVertexSizeInBytes() / 4
	oneSize := totalSize / 4
	l := parts.Len()
	vs := make([]float32, l*totalSize)
	geos := []float32{
		float32(geo.Element(0, 0)),
		float32(geo.Element(0, 1)),
		float32(geo.Element(1, 0)),
		float32(geo.Element(1, 1)),
		float32(geo.Element(0, 2)),
		float32(geo.Element(1, 2))}
	n := 0
	w := float32(1)
	h := float32(1)
	for w < float32(width) {
		w *= 2
	}
	for h < float32(height) {
		h *= 2
	}
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
		u0, v0, u1, v1 := float32(sx0)/w, float32(sy0)/h, float32(sx1)/w, float32(sy1)/h
		offset := n * totalSize
		vs[offset] = x0
		vs[offset+1] = y0
		vs[offset+2] = u0
		vs[offset+3] = v0
		for j, g := range geos {
			vs[offset+4+j] = g
		}
		offset += oneSize
		vs[offset] = x1
		vs[offset+1] = y0
		vs[offset+2] = u1
		vs[offset+3] = v0
		for j, g := range geos {
			vs[offset+4+j] = g
		}
		offset += oneSize
		vs[offset] = x0
		vs[offset+1] = y1
		vs[offset+2] = u0
		vs[offset+3] = v1
		for j, g := range geos {
			vs[offset+4+j] = g
		}
		offset += oneSize
		vs[offset] = x1
		vs[offset+1] = y1
		vs[offset+2] = u1
		vs[offset+3] = v1
		for j, g := range geos {
			vs[offset+4+j] = g
		}
		n++
	}
	return vs[:n*totalSize]
}
