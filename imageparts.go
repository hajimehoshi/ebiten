// Copyright 2015 Hajime Hoshi
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
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/internal/endian"
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

// An ImagePart is deprecated (as of 1.1.0-alpha): Use ImageParts instead.
type ImagePart struct {
	Dst image.Rectangle
	Src image.Rectangle
}

// An ImageParts represents the parts of the destination image and the parts of the source image.
type ImageParts interface {
	Len() int
	Dst(i int) (x0, y0, x1, y1 int)
	Src(i int) (x0, y0, x1, y1 int)
}

// NOTE: Remove this in the future.
type imageParts []ImagePart

func (p imageParts) Len() int {
	return len(p)
}

func (p imageParts) Dst(i int) (x0, y0, x1, y1 int) {
	dst := &p[i].Dst
	return dst.Min.X, dst.Min.Y, dst.Max.X, dst.Max.Y
}

func (p imageParts) Src(i int) (x0, y0, x1, y1 int) {
	src := &p[i].Src
	return src.Min.X, src.Min.Y, src.Max.X, src.Max.Y
}

type wholeImage struct {
	width  int
	height int
}

func (w *wholeImage) Len() int {
	return 1
}

func (w *wholeImage) Dst(i int) (x0, y0, x1, y1 int) {
	return 0, 0, w.width, w.height
}

func (w *wholeImage) Src(i int) (x0, y0, x1, y1 int) {
	return 0, 0, w.width, w.height
}

func u(x, width2p int) int16 {
	return int16(math.MaxInt16 * x / width2p)
}

func v(y, height2p int) int16 {
	return int16(math.MaxInt16 * y / height2p)
}

func vertices(parts ImageParts, width, height int) []uint8 {
	totalSize := graphics.QuadVertexSizeInBytes()
	oneSize := totalSize / 4
	l := parts.Len()
	vertices := make([]uint8, l*totalSize)
	width2p := graphics.NextPowerOf2Int(width)
	height2p := graphics.NextPowerOf2Int(height)
	n := 0
	vs := make([]int16, 16)
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
		vs[0] = x0
		vs[1] = y0
		vs[2] = u0
		vs[3] = v0
		vs[4] = x1
		vs[5] = y0
		vs[6] = u1
		vs[7] = v0
		vs[8] = x0
		vs[9] = y1
		vs[10] = u0
		vs[11] = v1
		vs[12] = x1
		vs[13] = y1
		vs[14] = u1
		vs[15] = v1
		// Use direct assign here. `append` function might be slow on browsers.
		if endian.IsLittle() {
			for i, v := range vs {
				vertices[totalSize*n+oneSize*(i/4)+2*(i%4)] = uint8(v)
				vertices[totalSize*n+oneSize*(i/4)+2*(i%4)+1] = uint8(v >> 8)
			}
		} else {
			for i, v := range vs {
				vertices[totalSize*n+oneSize*(i/4)+2*(i%4)] = uint8(v >> 8)
				vertices[totalSize*n+oneSize*(i/4)+2*(i%4)+1] = uint8(v)
			}
		}
		n++
	}
	return vertices[:n*totalSize]
}
