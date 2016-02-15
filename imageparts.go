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

func u(x int, width int) int {
	return math.MaxInt16 * x / graphics.NextPowerOf2Int(width)
}

func v(y int, height int) int {
	return math.MaxInt16 * y / graphics.NextPowerOf2Int(height)
}

type textureQuads struct {
	parts  ImageParts
	width  int
	height int
}

func (t *textureQuads) Len() int {
	return t.parts.Len()
}

func (t *textureQuads) SetVertices(vertices []int16) (int, error) {
	l := t.Len()
	if len(vertices) < l*16 {
		return 0, fmt.Errorf("grphics: vertices size must be greater than %d but %d", l*16, len(vertices))
	}
	p := t.parts
	w, h := t.width, t.height
	n := 0
	for i := 0; i < l; i++ {
		x0, y0, x1, y1 := p.Dst(i)
		if x0 == x1 || y0 == y1 {
			continue
		}
		sx0, sy0, sx1, sy1 := p.Src(i)
		u0, v0, u1, v1 := u(sx0, w), v(sy0, h), u(sx1, w), v(sy1, h)
		if u0 == u1 || v0 == v1 {
			continue
		}
		vertices[16*n] = int16(x0)
		vertices[16*n+1] = int16(y0)
		vertices[16*n+2] = int16(u0)
		vertices[16*n+3] = int16(v0)
		vertices[16*n+4] = int16(x1)
		vertices[16*n+5] = int16(y0)
		vertices[16*n+6] = int16(u1)
		vertices[16*n+7] = int16(v0)
		vertices[16*n+8] = int16(x0)
		vertices[16*n+9] = int16(y1)
		vertices[16*n+10] = int16(u0)
		vertices[16*n+11] = int16(v1)
		vertices[16*n+12] = int16(x1)
		vertices[16*n+13] = int16(y1)
		vertices[16*n+14] = int16(u1)
		vertices[16*n+15] = int16(v1)
		n++
	}
	return n, nil
}
