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
	"math"

	"github.com/gopherjs/gopherjs/js"
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

func vertices(parts ImageParts, width, height int, geo *GeoM) []int16 {
	// TODO: This function should be in graphics package?
	totalSize := graphics.QuadVertexSizeInBytes() / 2
	oneSize := totalSize / 4
	l := parts.Len()
	a := js.Global.Get("ArrayBuffer").New(l * totalSize * 2)
	af32 := js.Global.Get("Float32Array").New(a)
	a16 := js.Global.Get("Int16Array").New(a)
	w := uint(0)
	h := uint(0)
	for (1 << w) < width {
		w++
	}
	for (1 << h) < height {
		h++
	}
	gs := []float64{geo.Element(0, 0),
		geo.Element(0, 1),
		geo.Element(1, 0),
		geo.Element(1, 1),
		geo.Element(0, 2),
		geo.Element(1, 2)}
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
		u0 := (math.MaxInt16 * sx0) >> w
		v0 := (math.MaxInt16 * sy0) >> h
		u1 := (math.MaxInt16 * sx1) >> w
		v1 := (math.MaxInt16 * sy1) >> h
		offset := n * totalSize
		a16.SetIndex(offset, dx0)
		a16.SetIndex(offset+1, dy0)
		a16.SetIndex(offset+2, u0)
		a16.SetIndex(offset+3, v0)
		for j, g := range gs {
			af32.SetIndex((offset+4)/2+j, g)
		}
		offset += oneSize
		a16.SetIndex(offset, dx1)
		a16.SetIndex(offset+1, dy0)
		a16.SetIndex(offset+2, u1)
		a16.SetIndex(offset+3, v0)
		for j, g := range gs {
			af32.SetIndex((offset+4)/2+j, g)
		}
		offset += oneSize
		a16.SetIndex(offset, dx0)
		a16.SetIndex(offset+1, dy1)
		a16.SetIndex(offset+2, u0)
		a16.SetIndex(offset+3, v1)
		for j, g := range gs {
			af32.SetIndex((offset+4)/2+j, g)
		}
		offset += oneSize
		a16.SetIndex(offset, dx1)
		a16.SetIndex(offset+1, dy1)
		a16.SetIndex(offset+2, u1)
		a16.SetIndex(offset+3, v1)
		for j, g := range gs {
			af32.SetIndex((offset+4)/2+j, g)
		}
		n++
	}
	// TODO: Need to slice
	return a16.Interface().([]int16)
}
