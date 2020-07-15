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
	"github.com/hajimehoshi/ebiten/internal/shaderir"
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

// Mipmap is a set of shareable.Image sorted by the order of mipmap level.
// The level 0 image is a regular image and higher-level images are used for mipmap.
type Mipmap struct {
	width    int
	height   int
	volatile bool
	orig     *shareable.Image
	imgs     map[int]*shareable.Image
}

func New(width, height int, volatile bool) *Mipmap {
	return &Mipmap{
		width:    width,
		height:   height,
		volatile: volatile,
		orig:     shareable.NewImage(width, height, volatile),
		imgs:     map[int]*shareable.Image{},
	}
}

func NewScreenFramebufferMipmap(width, height int) *Mipmap {
	return &Mipmap{
		width:  width,
		height: height,
		orig:   shareable.NewScreenFramebufferImage(width, height),
		imgs:   map[int]*shareable.Image{},
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

func (m *Mipmap) Pixels(x, y, width, height int) ([]byte, error) {
	return m.orig.Pixels(x, y, width, height)
}

func (m *Mipmap) DrawImage(src *Mipmap, bounds image.Rectangle, geom GeoM, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter) {
	if det := geom.det(); det == 0 {
		return
	} else if math.IsNaN(float64(det)) {
		return
	}

	level := src.mipmapLevelFromGeoM(&geom, float32(bounds.Dx()), float32(bounds.Dy()), filter)

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
		sx0 := float32(bounds.Min.X)
		sy0 := float32(bounds.Min.Y)
		sx1 := float32(bounds.Max.X)
		sy1 := float32(bounds.Max.Y)
		vs := quadVertices(sx0, sy0, sx1, sy1, a, b, c, d, tx, ty, cr, cg, cb, ca, screen)
		is := graphics.QuadIndices()
		m.orig.DrawTriangles(src.orig, vs, is, colorm, mode, filter, driver.AddressUnsafe, driver.Region{}, nil, nil, nil)
	} else if buf := src.level(level); buf != nil {
		s := pow2(level)
		sx0 := float32(sizeForLevel(bounds.Min.X, level))
		sy0 := float32(sizeForLevel(bounds.Min.Y, level))
		sx1 := float32(sizeForLevel(bounds.Max.X, level))
		sy1 := float32(sizeForLevel(bounds.Max.Y, level))
		a *= s
		b *= s
		c *= s
		d *= s
		vs := quadVertices(sx0, sy0, sx1, sy1, a, b, c, d, tx, ty, cr, cg, cb, ca, false)
		is := graphics.QuadIndices()
		m.orig.DrawTriangles(buf, vs, is, colorm, mode, filter, driver.AddressUnsafe, driver.Region{}, nil, nil, nil)
	}
	m.disposeMipmaps()
}

func (m *Mipmap) DrawTriangles(src *Mipmap, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader, uniforms []interface{}, images []*Mipmap) {
	level := math.MaxInt32
	for i := 0; i < len(indices)/3; i++ {
		const n = graphics.VertexFloatNum
		dx0 := vertices[n*indices[3*i]+0]
		dy0 := vertices[n*indices[3*i]+1]
		sx0 := vertices[n*indices[3*i]+2]
		sy0 := vertices[n*indices[3*i]+3]
		dx1 := vertices[n*indices[3*i+1]+0]
		dy1 := vertices[n*indices[3*i+1]+1]
		sx1 := vertices[n*indices[3*i+1]+2]
		sy1 := vertices[n*indices[3*i+1]+3]
		dx2 := vertices[n*indices[3*i+2]+0]
		dy2 := vertices[n*indices[3*i+2]+1]
		sx2 := vertices[n*indices[3*i+2]+2]
		sy2 := vertices[n*indices[3*i+2]+3]
		if l := m.mipmapLevelFromDistance(dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1, filter); level > l {
			level = l
		}
		if l := m.mipmapLevelFromDistance(dx1, dy1, dx2, dy2, sx1, sy1, sx2, sy2, filter); level > l {
			level = l
		}
		if l := m.mipmapLevelFromDistance(dx2, dy2, dx0, dy0, sx2, sy2, sx0, sy0, filter); level > l {
			level = l
		}
	}

	if colorm != nil && colorm.ScaleOnly() {
		body, _ := colorm.UnsafeElements()
		cr := body[0]
		cg := body[5]
		cb := body[10]
		ca := body[15]
		colorm = nil
		const n = graphics.VertexFloatNum
		for i := 0; i < len(vertices)/n; i++ {
			vertices[i*n+4] *= cr
			vertices[i*n+5] *= cg
			vertices[i*n+6] *= cb
			vertices[i*n+7] *= ca
		}
	}

	var s *shareable.Shader
	if shader != nil {
		s = shader.shader
	}

	var srcimg *shareable.Image
	if src != nil {
		srcimg = src.orig
	}

	if level != 0 {
		if img := src.level(level); img != nil {
			srcimg = img
			const n = graphics.VertexFloatNum
			s := float32(pow2(level))
			for i := 0; i < len(vertices)/n; i++ {
				vertices[i*n+2] /= s
				vertices[i*n+3] /= s
			}
		}
	}

	// TODO: Do we need to consider mipmaps here?
	var imgs []*shareable.Image
	for _, img := range images {
		imgs = append(imgs, img.orig)
	}

	m.orig.DrawTriangles(srcimg, vertices, indices, colorm, mode, filter, address, sourceRegion, s, uniforms, imgs)
	m.disposeMipmaps()
}

func (m *Mipmap) level(level int) *shareable.Image {
	if level == 0 {
		panic("ebiten: level must be non-zero at level")
	}

	if m.volatile {
		panic("ebiten: mipmap images for a volatile image is not implemented yet")
	}

	if img, ok := m.imgs[level]; ok {
		return img
	}

	var src *shareable.Image
	var vs []float32
	var filter driver.Filter
	switch {
	case level == 1:
		src = m.orig
		vs = quadVertices(0, 0, float32(m.width), float32(m.height), 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1, false)
		filter = driver.FilterLinear
	case level > 1:
		src = m.level(level - 1)
		if src == nil {
			m.imgs[level] = nil
			return nil
		}
		w := sizeForLevel(m.width, level-1)
		h := sizeForLevel(m.height, level-1)
		vs = quadVertices(0, 0, float32(w), float32(h), 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1, false)
		filter = driver.FilterLinear
	case level == -1:
		src = m.orig
		vs = quadVertices(0, 0, float32(m.width), float32(m.height), 2, 0, 0, 2, 0, 0, 1, 1, 1, 1, false)
		filter = driver.FilterNearest
	case level < -1:
		src = m.level(level + 1)
		if src == nil {
			m.imgs[level] = nil
			return nil
		}
		w := sizeForLevel(m.width, level-1)
		h := sizeForLevel(m.height, level-1)
		vs = quadVertices(0, 0, float32(w), float32(h), 2, 0, 0, 2, 0, 0, 1, 1, 1, 1, false)
		filter = driver.FilterNearest
	default:
		panic(fmt.Sprintf("ebiten: invalid level: %d", level))
	}
	is := graphics.QuadIndices()

	w2 := sizeForLevel(m.width, level-1)
	h2 := sizeForLevel(m.height, level-1)
	if w2 == 0 || h2 == 0 {
		m.imgs[level] = nil
		return nil
	}
	s := shareable.NewImage(w2, h2, m.volatile)
	s.DrawTriangles(src, vs, is, nil, driver.CompositeModeCopy, filter, driver.AddressUnsafe, driver.Region{}, nil, nil, nil)
	m.imgs[level] = s

	return m.imgs[level]
}

func sizeForLevel(x int, level int) int {
	if level > 0 {
		for i := 0; i < level; i++ {
			x /= 2
			if x == 0 {
				return 0
			}
		}
	} else {
		for i := 0; i < -level; i++ {
			x *= 2
		}
	}
	return x
}

func (m *Mipmap) MarkDisposed() {
	m.disposeMipmaps()
	m.orig.MarkDisposed()
	m.orig = nil
}

func (m *Mipmap) disposeMipmaps() {
	for _, img := range m.imgs {
		img.MarkDisposed()
	}
	for k := range m.imgs {
		delete(m.imgs, k)
	}
}

func (m *Mipmap) mipmapLevelFromGeoM(geom *GeoM, sw, sh float32, filter driver.Filter) int {
	sx0 := float32(0)
	sy0 := float32(0)
	sx1 := sw
	sy1 := float32(0)
	sx2 := float32(0)
	sy2 := sh

	a, b, c, d := geom.A, geom.B, geom.C, geom.D
	dx0 := float32(0)
	dy0 := float32(0)
	dx1 := sx1*a + sy1*b
	dy1 := sx1*c + sy1*d
	dx2 := sx2*a + sy2*b
	dy2 := sx2*c + sy2*d

	l0 := m.mipmapLevelFromDistance(dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1, filter)
	l1 := m.mipmapLevelFromDistance(dx0, dy0, dx2, dy2, sx0, sy0, sx2, sy2, filter)

	if l0 < l1 {
		return l0
	}
	return l1
}

// mipmapLevel returns an appropriate mipmap level for the given distance.
func (m *Mipmap) mipmapLevelFromDistance(dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1 float32, filter driver.Filter) int {
	if filter == driver.FilterScreen {
		return 0
	}
	if m.volatile {
		return 0
	}

	d := (dx1-dx0)*(dx1-dx0) + (dy1-dy0)*(dy1-dy0)
	s := (sx1-sx0)*(sx1-sx0) + (sy1-sy0)*(sy1-sy0)
	scale := d / s

	// Use 'negative' mipmap to render edges correctly (#611, #907).
	// It looks like 128 is the enlargement factor that causes edge missings to pass the test TestImageStretch.
	var tooBigScale float32 = 128
	if !graphicsDriver.HasHighPrecisionFloat() {
		tooBigScale = 4
	}

	if scale >= tooBigScale*tooBigScale {
		// If the filter is not nearest, the target needs to be rendered with graduation. Don't use mipmaps.
		if filter != driver.FilterNearest {
			return 0
		}

		const mipmapMaxSize = 1024
		w, h := sx1-sx0, sy1-sy0
		if w >= mipmapMaxSize || h >= mipmapMaxSize {
			return 0
		}

		level := 0
		for scale >= tooBigScale*tooBigScale {
			level--
			scale /= 4
			w *= 2
			h *= 2
			if w >= mipmapMaxSize || h >= mipmapMaxSize {
				break
			}
		}

		// If tooBigScale is 4, level -10 means that the maximum scale is 4 * 2^10 = 4096. This should be
		// enough.
		if level < -10 {
			level = -10
		}
		return level
	}

	if filter != driver.FilterLinear {
		return 0
	}

	level := 0
	for scale < 0.25 {
		level++
		scale *= 4
	}

	if level > 0 {
		// If the image can be scaled into 0 size, adjust the level. (#839)
		w, h := int(sx1-sx0), int(sy1-sy0)
		for level >= 0 {
			s := 1 << uint(level)
			if (w > 0 && w/s == 0) || (h > 0 && h/s == 0) {
				level--
				continue
			}
			break
		}

		if level < 0 {
			// As the render source is too small, nothing is rendered.
			return 0
		}
	}

	if level > 6 {
		level = 6
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

type Shader struct {
	shader *shareable.Shader
}

func NewShader(program *shaderir.Program) *Shader {
	return &Shader{
		shader: shareable.NewShader(program),
	}
}

func (s *Shader) MarkDisposed() {
	s.shader.MarkDisposed()
	s.shader = nil
}
