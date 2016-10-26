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
	"math"
	"unsafe"

	"github.com/hajimehoshi/ebiten/internal/endian"
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

func floatsToInt16s(xs ...float64) []int16 {
	r := make([]int16, 0, len(xs)*2)
	for _, x := range xs {
		x32 := float32(x)
		n := *(*uint32)(unsafe.Pointer(&x32))
		if endian.IsLittle() {
			r = append(r, int16(n), int16(n>>16))
		} else {
			r = append(r, int16(n>>16), int16(n))
		}
	}
	return r
}

func u(x, width2p int) int16 {
	return int16(math.MaxInt16 * x / width2p)
}

func v(y, height2p int) int16 {
	return int16(math.MaxInt16 * y / height2p)
}

func vertices(parts ImageParts, width, height int, geo *GeoM) []int16 {
	// TODO: This function should be in graphics package?
	totalSize := graphics.QuadVertexSizeInBytes() / 2
	oneSize := totalSize / 4
	l := parts.Len()
	vs := make([]int16, l*totalSize)
	width2p := graphics.NextPowerOf2Int(width)
	height2p := graphics.NextPowerOf2Int(height)
	geo16 := floatsToInt16s(geo.Element(0, 0),
		geo.Element(0, 1),
		geo.Element(1, 0),
		geo.Element(1, 1),
		geo.Element(0, 2),
		geo.Element(1, 2))
	n := 0
	for i := 0; i < l; i++ {
		dx0, dy0, dx1, dy1 := parts.Dst(i)
		if dx0 == dx1 || dy0 == dy1 {
			continue
		}
		x0, y0, x1, y1 := int16(dx0), int16(dy0), int16(dx1), int16(dy1)
		sx0, sy0, sx1, sy1 := parts.Src(i)
		if sx0 == sx1 || sy0 == sy1 {
			continue
		}
		u0, v0, u1, v1 := u(sx0, width2p), v(sy0, height2p), u(sx1, width2p), v(sy1, height2p)
		offset := n * totalSize
		vs[offset] = x0
		vs[offset+1] = y0
		vs[offset+2] = u0
		vs[offset+3] = v0
		for j, g := range geo16 {
			vs[offset+4+j] = g
		}
		vs[offset+oneSize] = x1
		vs[offset+oneSize+1] = y0
		vs[offset+oneSize+2] = u1
		vs[offset+oneSize+3] = v0
		for j, g := range geo16 {
			vs[offset+oneSize+4+j] = g
		}
		vs[offset+2*oneSize] = x0
		vs[offset+2*oneSize+1] = y1
		vs[offset+2*oneSize+2] = u0
		vs[offset+2*oneSize+3] = v1
		for j, g := range geo16 {
			vs[offset+2*oneSize+4+j] = g
		}
		vs[offset+3*oneSize] = x1
		vs[offset+3*oneSize+1] = y1
		vs[offset+3*oneSize+2] = u1
		vs[offset+3*oneSize+3] = v1
		for j, g := range geo16 {
			vs[offset+3*oneSize+4+j] = g
		}
		n++
	}
	return vs[:n*totalSize]
}
