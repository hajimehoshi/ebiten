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
	"math"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/buffered"
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func BeginFrame() error {
	return buffered.BeginFrame()
}

func EndFrame() error {
	return buffered.EndFrame()
}

// Mipmap is a set of buffered.Image sorted by the order of mipmap level.
// The level 0 image is a regular image and higher-level images are used for mipmap.
type Mipmap struct {
	width    int
	height   int
	volatile bool
	orig     *buffered.Image
	imgs     map[int]*buffered.Image
}

func New(width, height int) *Mipmap {
	return &Mipmap{
		width:  width,
		height: height,
		orig:   buffered.NewImage(width, height),
	}
}

func NewScreenFramebufferMipmap(width, height int) *Mipmap {
	return &Mipmap{
		width:  width,
		height: height,
		orig:   buffered.NewScreenFramebufferImage(width, height),
	}
}

func (m *Mipmap) SetVolatile(volatile bool) {
	m.volatile = volatile
	if m.volatile {
		m.disposeMipmaps()
	}
	m.orig.SetVolatile(volatile)
}

func (m *Mipmap) DumpScreenshot(name string, blackbg bool) error {
	return m.orig.DumpScreenshot(name, blackbg)
}

func (m *Mipmap) ReplacePixels(pix []byte, x, y, width, height int) error {
	if err := m.orig.ReplacePixels(pix, x, y, width, height); err != nil {
		return err
	}
	m.disposeMipmaps()
	return nil
}

func (m *Mipmap) Pixels(x, y, width, height int) ([]byte, error) {
	return m.orig.Pixels(x, y, width, height)
}

func (m *Mipmap) DrawTriangles(srcs [graphics.ShaderImageNum]*Mipmap, vertices []float32, indices []uint16, colorm affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, dstRegion, srcRegion driver.Region, subimageOffsets [graphics.ShaderImageNum - 1][2]float32, shader *Shader, uniforms []interface{}, evenOdd bool, canSkipMipmap bool) {
	if len(indices) == 0 {
		return
	}

	level := 0
	// TODO: Do we need to check all the sources' states of being volatile?
	if !canSkipMipmap && srcs[0] != nil && !srcs[0].volatile && filter != driver.FilterScreen {
		level = math.MaxInt32
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
			if l := mipmapLevelFromDistance(dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1, filter); level > l {
				level = l
			}
			if l := mipmapLevelFromDistance(dx1, dy1, dx2, dy2, sx1, sy1, sx2, sy2, filter); level > l {
				level = l
			}
			if l := mipmapLevelFromDistance(dx2, dy2, dx0, dy0, sx2, sy2, sx0, sy0, filter); level > l {
				level = l
			}
		}
		if level == math.MaxInt32 {
			panic("mipmap: level must be calculated at least once but not")
		}
	}

	var s *buffered.Shader
	if shader != nil {
		s = shader.shader
	}

	var imgs [graphics.ShaderImageNum]*buffered.Image
	for i, src := range srcs {
		if src == nil {
			continue
		}
		if level != 0 {
			if img := src.level(level); img != nil {
				const n = graphics.VertexFloatNum
				s := float32(pow2(level))
				for i := 0; i < len(vertices)/n; i++ {
					vertices[i*n+2] /= s
					vertices[i*n+3] /= s
				}
				imgs[i] = img
				continue
			}
		}
		imgs[i] = src.orig
	}

	m.orig.DrawTriangles(imgs, vertices, indices, colorm, mode, filter, address, dstRegion, srcRegion, subimageOffsets, s, uniforms, evenOdd)
	m.disposeMipmaps()
}

func (m *Mipmap) setImg(level int, img *buffered.Image) {
	if m.imgs == nil {
		m.imgs = map[int]*buffered.Image{}
	}
	m.imgs[level] = img
}

func (m *Mipmap) level(level int) *buffered.Image {
	if level == 0 {
		panic("ebiten: level must be non-zero at level")
	}

	if m.volatile {
		panic("ebiten: mipmap images for a volatile image is not implemented yet")
	}

	if img, ok := m.imgs[level]; ok {
		return img
	}

	var src *buffered.Image
	var vs []float32
	var filter driver.Filter
	switch {
	case level == 1:
		src = m.orig
		vs = graphics.QuadVertices(0, 0, float32(m.width), float32(m.height), 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1)
		filter = driver.FilterLinear
	case level > 1:
		src = m.level(level - 1)
		if src == nil {
			m.setImg(level, nil)
			return nil
		}
		w := sizeForLevel(m.width, level-1)
		h := sizeForLevel(m.height, level-1)
		vs = graphics.QuadVertices(0, 0, float32(w), float32(h), 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1)
		filter = driver.FilterLinear
	default:
		panic(fmt.Sprintf("ebiten: invalid level: %d", level))
	}
	is := graphics.QuadIndices()

	w2 := sizeForLevel(m.width, level-1)
	h2 := sizeForLevel(m.height, level-1)
	if w2 == 0 || h2 == 0 {
		m.setImg(level, nil)
		return nil
	}
	// buffered.NewImage panics with a too big size when actual allocation happens.
	// 4096 should be a safe size in most environments (#1399).
	// Unfortunately a precise max image size cannot be obtained here since this requires GPU access.
	if w2 > 4096 || h2 > 4096 {
		m.setImg(level, nil)
		return nil
	}
	s := buffered.NewImage(w2, h2)
	s.SetVolatile(m.volatile)

	dstRegion := driver.Region{
		X:      0,
		Y:      0,
		Width:  float32(w2),
		Height: float32(h2),
	}
	s.DrawTriangles([graphics.ShaderImageNum]*buffered.Image{src}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, filter, driver.AddressUnsafe, dstRegion, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
	m.setImg(level, s)

	return m.imgs[level]
}

func sizeForLevel(x int, level int) int {
	for i := 0; i < level; i++ {
		x /= 2
		if x == 0 {
			return 0
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
		if img != nil {
			img.MarkDisposed()
		}
	}
	for k := range m.imgs {
		delete(m.imgs, k)
	}
}

// mipmapLevel returns an appropriate mipmap level for the given distance.
func mipmapLevelFromDistance(dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1 float32, filter driver.Filter) int {
	const maxLevel = 6

	if filter == driver.FilterScreen {
		return 0
	}

	d := (dx1-dx0)*(dx1-dx0) + (dy1-dy0)*(dy1-dy0)
	s := (sx1-sx0)*(sx1-sx0) + (sy1-sy0)*(sy1-sy0)
	if s == 0 {
		return 0
	}
	scale := d / s

	// Scale can be infinite when the specified scale is extremely big (#1398).
	if math.IsInf(float64(scale), 0) {
		return 0
	}

	// Scale can be zero when the specified scale is extremely small (#1398).
	if scale == 0 {
		return 0
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

	if level > maxLevel {
		level = maxLevel
	}

	return level
}

func pow2(power int) float32 {
	x := 1
	return float32(x << uint(power))
}

type Shader struct {
	shader *buffered.Shader
}

func NewShader(program *shaderir.Program) *Shader {
	return &Shader{
		shader: buffered.NewShader(program),
	}
}

func (s *Shader) MarkDisposed() {
	s.shader.MarkDisposed()
	s.shader = nil
}
