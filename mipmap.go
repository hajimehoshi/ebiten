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
	"fmt"
	"image"
	"image/color"
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

func newMipmap(width, height int) *mipmap {
	return &mipmap{
		orig: shareable.NewImage(width, height),
		imgs: map[image.Rectangle]levelToImage{},
	}
}

func newScreenFramebufferMipmap(width, height int) *mipmap {
	return &mipmap{
		orig: shareable.NewScreenFramebufferImage(width, height),
		imgs: map[image.Rectangle]levelToImage{},
	}
}

func (m *mipmap) original() *shareable.Image {
	// TODO: Remove this
	return m.orig
}

func (m *mipmap) makeVolatile() {
	m.orig.MakeVolatile()
	m.disposeMipmaps()
}

func (m *mipmap) dump(name string) error {
	return m.orig.Dump(name)
}

func (m *mipmap) fill(clr color.Color) {
	m.orig.Fill(clr)
	m.disposeMipmaps()
}

func (m *mipmap) replacePixels(pix []byte) {
	m.orig.ReplacePixels(pix)
	m.disposeMipmaps()
}

func (m *mipmap) size() (int, int) {
	return m.orig.Size()
}

func (m *mipmap) at(x, y int) (r, g, b, a byte) {
	return m.orig.At(x, y)
}

func (m *mipmap) level(r image.Rectangle, level int) *shareable.Image {
	if level == 0 {
		panic("ebiten: level must be non-zero at level")
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

	var src *shareable.Image
	vs := vertexSlice(4)
	var filter driver.Filter
	switch {
	case level == 1:
		src = m.orig
		graphics.PutQuadVertices(vs, src, r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1)
		filter = driver.FilterLinear
	case level > 1:
		src = m.level(r, level-1)
		if src == nil {
			imgs[level] = nil
			return nil
		}
		w, h := src.Size()
		graphics.PutQuadVertices(vs, src, 0, 0, w, h, 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1)
		filter = driver.FilterLinear
	case level == -1:
		src = m.orig
		graphics.PutQuadVertices(vs, src, r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, 2, 0, 0, 2, 0, 0, 1, 1, 1, 1)
		filter = driver.FilterNearest
	case level < -1:
		src = m.level(r, level+1)
		if src == nil {
			imgs[level] = nil
			return nil
		}
		w, h := src.Size()
		graphics.PutQuadVertices(vs, src, 0, 0, w, h, 2, 0, 0, 2, 0, 0, 1, 1, 1, 1)
		filter = driver.FilterNearest
	default:
		panic(fmt.Sprintf("ebiten: invalid level: %d", level))
	}
	is := graphics.QuadIndices()

	size := r.Size()
	w2, h2 := size.X, size.Y
	if level > 0 {
		for i := 0; i < level; i++ {
			w2 /= 2
			h2 /= 2
			if w2 == 0 || h2 == 0 {
				imgs[level] = nil
				return nil
			}
		}
	} else {
		for i := 0; i < -level; i++ {
			w2 *= 2
			h2 *= 2
		}
	}
	s := shareable.NewImage(w2, h2)
	s.DrawTriangles(src, vs, is, nil, driver.CompositeModeCopy, filter, driver.AddressClampToZero)
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

// mipmapLevel returns an appropriate mipmap level for the given determinant of a geometry matrix.
//
// mipmapLevel panics if det is NaN or 0.
func (m *mipmap) mipmapLevel(geom *GeoM, width, height int, filter driver.Filter) int {
	det := geom.det()
	if math.IsNaN(float64(det)) {
		panic("ebiten: det must be finite at mipmapLevel")
	}
	if det == 0 {
		panic("ebiten: dst must be non zero at mipmapLevel")
	}

	// Use 'negative' mipmap to render edges correctly (#611, #907).
	// It looks like 128 is the enlargement factor that causes edge missings to pass the test TestImageStretch.
	const tooBigScale = 128
	if sx, sy := geomScaleSize(geom); sx >= tooBigScale || sy >= tooBigScale {
		// If the filter is not nearest, the target needs to be rendered with graduation. Don't use mipmaps.
		if filter != driver.FilterNearest {
			return 0
		}

		const mipmapMaxSize = 1024
		w, h := width, height
		if w >= mipmapMaxSize || h >= mipmapMaxSize {
			return 0
		}

		level := 0
		for sx >= tooBigScale || sy >= tooBigScale {
			level--
			sx /= 2
			sy /= 2
			w *= 2
			h *= 2
			if w >= mipmapMaxSize || h >= mipmapMaxSize {
				break
			}
		}
		return level
	}

	if filter != driver.FilterLinear {
		return 0
	}
	if m.original().IsVolatile() {
		return 0
	}

	// This is a separate function for testing.
	return mipmapLevelForDownscale(det)
}

func mipmapLevelForDownscale(det float32) int {
	if math.IsNaN(float64(det)) {
		panic("ebiten: det must be finite at mipmapLevelForDownscale")
	}
	if det == 0 {
		panic("ebiten: dst must be non zero at mipmapLevelForDownscale")
	}

	// TODO: Should this be determined by x/y scales instead of det?
	d := math.Abs(float64(det))
	level := 0
	for d < 0.25 {
		level++
		d *= 4
	}
	return level
}

func pow2(power int) float32 {
	if power >= 0 {
		x := 1
		return float32(x << uint(power))
	}

	x := float32(1)
	for i := 0; i < -power; i++ {
		x /= 2
	}
	return x
}

func maxf32(values ...float32) float32 {
	max := float32(math.Inf(-1))
	for _, v := range values {
		if max < v {
			max = v
		}
	}
	return max
}

func minf32(values ...float32) float32 {
	min := float32(math.Inf(1))
	for _, v := range values {
		if min > v {
			min = v
		}
	}
	return min
}

func geomScaleSize(geom *GeoM) (sx, sy float32) {
	a, b, c, d, _, _ := geom.elements()
	// (0, 1)
	x0 := 0*a + 1*b
	y0 := 0*c + 1*d

	// (1, 0)
	x1 := 1*a + 0*b
	y1 := 1*c + 0*d

	// (1, 1)
	x2 := 1*a + 1*b
	y2 := 1*c + 1*d

	maxx := maxf32(0, x0, x1, x2)
	maxy := maxf32(0, y0, y1, y2)
	minx := minf32(0, x0, x1, x2)
	miny := minf32(0, y0, y1, y2)

	return maxx - minx, maxy - miny
}
