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
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"image"
	"math"
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

func (t *textureQuads) Vertex(i int) (x0, y0, x1, y1 int) {
	return t.parts.Dst(i)
}

func (t *textureQuads) Texture(i int) (u0, v0, u1, v1 int) {
	x0, y0, x1, y1 := t.parts.Src(i)
	w, h := t.width, t.height
	return u(x0, w), v(y0, h), u(x1, w), v(y1, h)
}
