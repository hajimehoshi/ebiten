// Copyright 2018 The Ebiten Authors
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

	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/shareable"
)

type levelToImage map[int]*shareable.Image

type mipmap struct {
	orig *shareable.Image
	imgs map[image.Rectangle]levelToImage
}

func newMipmap(s *shareable.Image) *mipmap {
	return &mipmap{
		orig: s,
		imgs: map[image.Rectangle]levelToImage{},
	}
}

func (m *mipmap) original() *shareable.Image {
	return m.orig
}

func (m *mipmap) level(r image.Rectangle, level int) *shareable.Image {
	if level <= 0 {
		panic("ebiten: level must be positive at level")
	}

	if m.orig.IsVolatile() {
		panic("ebiten: mipmap images for a volatile image is not implemented yet")
	}

	if _, ok := m.imgs[r]; !ok {
		m.imgs[r] = levelToImage{}
	}
	imgs := m.imgs[r]

	if img, ok := imgs[level]; ok {
		return img
	}

	size := r.Size()
	w, h := size.X, size.Y

	w2, h2 := w, h
	for i := 0; i < level; i++ {
		w2 /= 2
		h2 /= 2
		if w == 0 || h == 0 {
			imgs[level] = nil
			return nil
		}
	}
	s := shareable.NewImage(w2, h2)
	var src *shareable.Image
	vs := vertexSlice(4)
	if level == 1 {
		src = m.orig
		graphics.PutQuadVertices(vs, src, r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1)
	} else {
		src = m.level(r, level-1)
		if src == nil {
			imgs[level] = nil
			return nil
		}
		graphics.PutQuadVertices(vs, src, 0, 0, w, h, 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1)
	}
	is := graphics.QuadIndices()
	s.DrawTriangles(src, vs, is, nil, driver.CompositeModeCopy, driver.FilterLinear, driver.AddressClampToZero)
	imgs[level] = s

	return imgs[level]
}

func (m *mipmap) isDisposed() bool {
	return m.orig == nil
}

func (m *mipmap) dispose() {
	m.disposeMipmaps()
	m.orig.Dispose()
	m.orig = nil
}

func (m *mipmap) disposeMipmaps() {
	for _, a := range m.imgs {
		for _, img := range a {
			img.Dispose()
		}
	}
	for k := range m.imgs {
		delete(m.imgs, k)
	}
}

func (m *mipmap) clearFramebuffer() {
	m.orig.ClearFramebuffer()
}

func (m *mipmap) resetRestoringState() {
	m.orig.ResetRestoringState()
}

// mipmapLevel returns an appropriate mipmap level for the given determinant of a geometry matrix.
//
// mipmapLevel returns -1 if det is 0.
//
// mipmapLevel panics if det is NaN.
func mipmapLevel(det float32) int {
	if math.IsNaN(float64(det)) {
		panic("graphicsutil: det must be finite")
	}
	if det == 0 {
		return -1
	}

	d := math.Abs(float64(det))
	level := 0
	for d < 0.25 {
		level++
		d *= 4
	}
	return level
}
