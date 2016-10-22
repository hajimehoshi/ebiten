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

type textureQuads struct {
	parts  ImageParts
	width  int
	height int
}

func (t *textureQuads) vertices() []uint8 {
	l := t.parts.Len()
	vertices := make([]uint8, l*32)
	p := t.parts
	w, h := t.width, t.height
	width2p := int(graphics.NextPowerOf2Int32(int32(w)))
	height2p := int(graphics.NextPowerOf2Int32(int32(h)))
	n := 0
	for i := 0; i < l; i++ {
		dx0, dy0, dx1, dy1 := p.Dst(i)
		if dx0 == dx1 || dy0 == dy1 {
			continue
		}
		x0, y0, x1, y1 := int16(dx0), int16(dy0), int16(dx1), int16(dy1)
		sx0, sy0, sx1, sy1 := p.Src(i)
		if sx0 == sx1 || sy0 == sy1 {
			continue
		}
		u0, v0, u1, v1 := u(sx0, width2p), v(sy0, height2p), u(sx1, width2p), v(sy1, height2p)
		// Use direct assign here. `append` function might be slow on browsers.
		if endian.IsLittle() {
			vertices[32*n] = uint8(x0)
			vertices[32*n+1] = uint8(x0 >> 8)
			vertices[32*n+2] = uint8(y0)
			vertices[32*n+3] = uint8(y0 >> 8)
			vertices[32*n+4] = uint8(u0)
			vertices[32*n+5] = uint8(u0 >> 8)
			vertices[32*n+6] = uint8(v0)
			vertices[32*n+7] = uint8(v0 >> 8)
			vertices[32*n+8] = uint8(x1)
			vertices[32*n+9] = uint8(x1 >> 8)
			vertices[32*n+10] = uint8(y0)
			vertices[32*n+11] = uint8(y0 >> 8)
			vertices[32*n+12] = uint8(u1)
			vertices[32*n+13] = uint8(u1 >> 8)
			vertices[32*n+14] = uint8(v0)
			vertices[32*n+15] = uint8(v0 >> 8)
			vertices[32*n+16] = uint8(x0)
			vertices[32*n+17] = uint8(x0 >> 8)
			vertices[32*n+18] = uint8(y1)
			vertices[32*n+19] = uint8(y1 >> 8)
			vertices[32*n+20] = uint8(u0)
			vertices[32*n+21] = uint8(u0 >> 8)
			vertices[32*n+22] = uint8(v1)
			vertices[32*n+23] = uint8(v1 >> 8)
			vertices[32*n+24] = uint8(x1)
			vertices[32*n+25] = uint8(x1 >> 8)
			vertices[32*n+26] = uint8(y1)
			vertices[32*n+27] = uint8(y1 >> 8)
			vertices[32*n+28] = uint8(u1)
			vertices[32*n+29] = uint8(u1 >> 8)
			vertices[32*n+30] = uint8(v1)
			vertices[32*n+31] = uint8(v1 >> 8)
		} else {
			vertices[32*n] = uint8(x0 >> 8)
			vertices[32*n+1] = uint8(x0)
			vertices[32*n+2] = uint8(y0 >> 8)
			vertices[32*n+3] = uint8(y0)
			vertices[32*n+4] = uint8(u0 >> 8)
			vertices[32*n+5] = uint8(u0)
			vertices[32*n+6] = uint8(v0 >> 8)
			vertices[32*n+7] = uint8(v0)
			vertices[32*n+8] = uint8(x1 >> 8)
			vertices[32*n+9] = uint8(x1)
			vertices[32*n+10] = uint8(y0 >> 8)
			vertices[32*n+11] = uint8(y0)
			vertices[32*n+12] = uint8(u1 >> 8)
			vertices[32*n+13] = uint8(u1)
			vertices[32*n+14] = uint8(v0 >> 8)
			vertices[32*n+15] = uint8(v0)
			vertices[32*n+16] = uint8(x0 >> 8)
			vertices[32*n+17] = uint8(x0)
			vertices[32*n+18] = uint8(y1 >> 8)
			vertices[32*n+19] = uint8(y1)
			vertices[32*n+20] = uint8(u0 >> 8)
			vertices[32*n+21] = uint8(u0)
			vertices[32*n+22] = uint8(v1 >> 8)
			vertices[32*n+23] = uint8(v1)
			vertices[32*n+24] = uint8(x1 >> 8)
			vertices[32*n+25] = uint8(x1)
			vertices[32*n+26] = uint8(y1 >> 8)
			vertices[32*n+27] = uint8(y1)
			vertices[32*n+28] = uint8(u1 >> 8)
			vertices[32*n+29] = uint8(u1)
			vertices[32*n+30] = uint8(v1 >> 8)
			vertices[32*n+31] = uint8(v1)
		}
		n++
	}
	return vertices[:n*32]
}
