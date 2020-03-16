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

package mipmap

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/shareable"
)

var graphicsDriver driver.Graphics

func SetGraphicsDriver(graphics driver.Graphics) {
	graphicsDriver = graphics
}

func BeginFrame() error {
	return shareable.BeginFrame()
}

func EndFrame() error {
	return shareable.EndFrame()
}

type GeoM struct {
	A  float32
	B  float32
	C  float32
	D  float32
	Tx float32
	Ty float32
}

func (g *GeoM) det() float32 {
	return g.A*g.D - g.B*g.C
}

type levelToImage map[int]*shareable.Image

// Mipmap is a set of shareable.Image sorted by the order of mipmap level.
// The level 0 image is a regular image and higher-level images are used for mipmap.
type Mipmap struct {
	volatile bool
	orig     *shareable.Image
	imgs     map[image.Rectangle]levelToImage
}

func New(width, height int, volatile bool) *Mipmap {
	return &Mipmap{
		volatile: volatile,
		orig:     shareable.NewImage(width, height, volatile),
		imgs:     map[image.Rectangle]levelToImage{},
	}
}

func NewScreenFramebufferMipmap(width, height int) *Mipmap {
	return &Mipmap{
		orig: shareable.NewScreenFramebufferImage(width, height),
		imgs: map[image.Rectangle]levelToImage{},
	}
}

func (m *Mipmap) Dump(name string, blackbg bool) error {
	return m.orig.Dump(name, blackbg)
}

func (m *Mipmap) Fill(clr color.RGBA) {
	m.orig.Fill(clr)
	m.disposeMipmaps()
}

func (m *Mipmap) ReplacePixels(pix []byte) {
	m.orig.ReplacePixels(pix)
	m.disposeMipmaps()
}

func (m *Mipmap) At(x, y int) (r, g, b, a byte, err error) {
	return m.orig.At(x, y)
}

func (m *Mipmap) DrawImage(src *Mipmap, bounds image.Rectangle, geom GeoM, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter) {
	if det := geom.det(); det == 0 {
		return
	} else if math.IsNaN(float64(det)) {
		return
	}

	level := src.mipmapLevel(geom, bounds.Dx(), bounds.Dy(), filter)

	if level > 0 {
		// If the image can be scaled into 0 size, adjust the level. (#839)
		w, h := bounds.Dx(), bounds.Dy()
		for level >= 0 {
			s := 1 << uint(level)
			if w/s == 0 || h/s == 0 {
				level--
				continue
			}
			break
		}

		if level < 0 {
			// As the render source is too small, nothing is rendered.
			return
		}
	}

	if level > 6 {
		level = 6
	}
	// If tooBigScale is 4, level -10 means that the maximum scale is 4 * 2^10 = 4096. This should be enough.
	if level < -10 {
		level = -10
	}

	cr, cg, cb, ca := float32(1), float32(1), float32(1), float32(1)
	if colorm != nil && colorm.ScaleOnly() {
		body, _ := colorm.UnsafeElements()
		cr = body[0]
		cg = body[5]
		cb = body[10]
		ca = body[15]
		colorm = nil
	}

	screen := filter == driver.FilterScreen
	if screen && level != 0 {
		panic("ebiten: Mipmap must not be used when the filter is FilterScreen")
	}

	a, b, c, d, tx, ty := geom.A, geom.B, geom.C, geom.D, geom.Tx, geom.Ty
	if level == 0 {
		vs := quadVertices(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, a, b, c, d, tx, ty, cr, cg, cb, ca, screen)
		is := graphics.QuadIndices()
		m.orig.DrawTriangles(src.orig, vs, is, colorm, mode, filter, driver.AddressClampToZero)
	} else if buf := src.level(bounds, level); buf != nil {
		w, h := sizeForLevel(bounds.Dx(), bounds.Dy(), level)
		s := pow2(level)
		a *= s
		b *= s
		c *= s
		d *= s
		vs := quadVertices(0, 0, w, h, a, b, c, d, tx, ty, cr, cg, cb, ca, false)
		is := graphics.QuadIndices()
		m.orig.DrawTriangles(buf, vs, is, colorm, mode, filter, driver.AddressClampToZero)
	}
	m.disposeMipmaps()
}

func (m *Mipmap) DrawTriangles(src *Mipmap, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	// TODO: Use a mipmap? (#909)

	if colorm != nil && colorm.ScaleOnly() {
		body, _ := colorm.UnsafeElements()
		cr := body[0]
		cg := body[5]
		cb := body[10]
		ca := body[15]
		colorm = nil
		const n = graphics.VertexFloatNum
		for i := 0; i < len(vertices)/n; i++ {
			vertices[i*n+8] *= cr
			vertices[i*n+9] *= cg
			vertices[i*n+10] *= cb
			vertices[i*n+11] *= ca
		}
	}
	m.orig.DrawTriangles(src.orig, vertices, indices, colorm, mode, filter, address)
	m.disposeMipmaps()
}

func (m *Mipmap) level(r image.Rectangle, level int) *shareable.Image {
	if level == 0 {
		panic("ebiten: level must be non-zero at level")
	}

	if m.volatile {
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
	var vs []float32
	var filter driver.Filter
	switch {
	case level == 1:
		src = m.orig
		vs = quadVertices(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1, false)
		filter = driver.FilterLinear
	case level > 1:
		src = m.level(r, level-1)
		if src == nil {
			imgs[level] = nil
			return nil
		}
		w, h := sizeForLevel(r.Dx(), r.Dy(), level-1)
		vs = quadVertices(0, 0, w, h, 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1, false)
		filter = driver.FilterLinear
	case level == -1:
		src = m.orig
		vs = quadVertices(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, 2, 0, 0, 2, 0, 0, 1, 1, 1, 1, false)
		filter = driver.FilterNearest
	case level < -1:
		src = m.level(r, level+1)
		if src == nil {
			imgs[level] = nil
			return nil
		}
		w, h := sizeForLevel(r.Dx(), r.Dy(), level+1)
		vs = quadVertices(0, 0, w, h, 2, 0, 0, 2, 0, 0, 1, 1, 1, 1, false)
		filter = driver.FilterNearest
	default:
		panic(fmt.Sprintf("ebiten: invalid level: %d", level))
	}
	is := graphics.QuadIndices()

	w2, h2 := sizeForLevel(r.Dx(), r.Dy(), level)
	if w2 == 0 || h2 == 0 {
		imgs[level] = nil
		return nil
	}
	s := shareable.NewImage(w2, h2, m.volatile)
	s.DrawTriangles(src, vs, is, nil, driver.CompositeModeCopy, filter, driver.AddressClampToZero)
	imgs[level] = s

	return imgs[level]
}

func sizeForLevel(origWidth, origHeight int, level int) (width, height int) {
	width = origWidth
	height = origHeight
	if level > 0 {
		for i := 0; i < level; i++ {
			width /= 2
			height /= 2
			if width == 0 || height == 0 {
				return 0, 0
			}
		}
	} else {
		for i := 0; i < -level; i++ {
			width *= 2
			height *= 2
		}
	}
	return
}

func (m *Mipmap) MarkDisposed() {
	m.disposeMipmaps()
	m.orig.MarkDisposed()
	m.orig = nil
}

func (m *Mipmap) disposeMipmaps() {
	for _, a := range m.imgs {
		for _, img := range a {
			img.MarkDisposed()
		}
	}
	for k := range m.imgs {
		delete(m.imgs, k)
	}
}

// mipmapLevel returns an appropriate mipmap level for the given determinant of a geometry matrix.
//
// mipmapLevel panics if det is NaN or 0.
func (m *Mipmap) mipmapLevel(geom GeoM, width, height int, filter driver.Filter) int {
	det := geom.det()
	if math.IsNaN(float64(det)) {
		panic("ebiten: det must be finite at mipmapLevel")
	}
	if det == 0 {
		panic("ebiten: dst must be non zero at mipmapLevel")
	}

	if filter == driver.FilterScreen {
		return 0
	}

	// Use 'negative' mipmap to render edges correctly (#611, #907).
	// It looks like 128 is the enlargement factor that causes edge missings to pass the test TestImageStretch.
	var tooBigScale float32 = 128
	if !graphicsDriver.HasHighPrecisionFloat() {
		tooBigScale = 4
	}
	if sx, sy := geomScaleSize(&geom); sx >= tooBigScale || sy >= tooBigScale {
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
	if m.volatile {
		return 0
	}

	// This is a separate function for testing.
	return MipmapLevelForDownscale(det)
}

func MipmapLevelForDownscale(det float32) int {
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

func maxf32(a, b, c, d float32) float32 {
	max := a
	if max < b {
		max = b
	}
	if max < c {
		max = c
	}
	if max < d {
		max = d
	}
	return max
}

func minf32(a, b, c, d float32) float32 {
	min := a
	if min > b {
		min = b
	}
	if min > c {
		min = c
	}
	if min > d {
		min = d
	}
	return min
}

func geomScaleSize(geom *GeoM) (sx, sy float32) {
	a, b, c, d := geom.A, geom.B, geom.C, geom.D
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
