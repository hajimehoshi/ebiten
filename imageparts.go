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
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/internal/graphics"
)

// Deprecated (as of 1.1.0-alpha): Use ImageParts instead.
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

func u(x int, width int) int16 {
	return int16(math.MaxInt16 * x / int(graphics.NextPowerOf2Int32(int32(width))))
}

func v(y int, height int) int16 {
	return int16(math.MaxInt16 * y / int(graphics.NextPowerOf2Int32(int32(height))))
}

type textureQuads struct {
	parts  ImageParts
	width  int
	height int
}

func (t *textureQuads) Len() int {
	return t.parts.Len()
}

func (t *textureQuads) SetVertices(vertices []int16) int {
	l := t.Len()
	if len(vertices) < l*16 {
		panic(fmt.Sprintf("graphics: vertices size must be greater than %d but %d", l*16, len(vertices)))
	}
	p := t.parts
	w, h := t.width, t.height
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
		u0, v0, u1, v1 := u(sx0, w), v(sy0, h), u(sx1, w), v(sy1, h)
		vertices[16*n] = x0
		vertices[16*n+1] = y0
		vertices[16*n+2] = u0
		vertices[16*n+3] = v0
		vertices[16*n+4] = x1
		vertices[16*n+5] = y0
		vertices[16*n+6] = u1
		vertices[16*n+7] = v0
		vertices[16*n+8] = x0
		vertices[16*n+9] = y1
		vertices[16*n+10] = u0
		vertices[16*n+11] = v1
		vertices[16*n+12] = x1
		vertices[16*n+13] = y1
		vertices[16*n+14] = u1
		vertices[16*n+15] = v1
		n++
	}
	return n
}
