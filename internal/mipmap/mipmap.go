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
	"math"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/buffered"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
)

func canUseMipmap(imageType atlas.ImageType) bool {
	switch imageType {
	case atlas.ImageTypeRegular, atlas.ImageTypeUnmanaged:
		return true
	}
	return false
}

// Mipmap is a set of buffered.Image sorted by the order of mipmap level.
// The level 0 image is a regular image and higher-level images are used for mipmap.
type Mipmap struct {
	width     int
	height    int
	imageType atlas.ImageType
	orig      *buffered.Image
	imgs      map[int]imageWithDirtyFlag
}

type imageWithDirtyFlag struct {
	img   *buffered.Image
	dirty bool
}

func New(width, height int, imageType atlas.ImageType) *Mipmap {
	return &Mipmap{
		width:     width,
		height:    height,
		orig:      buffered.NewImage(width, height, imageType),
		imageType: imageType,
	}
}

func (m *Mipmap) DumpScreenshot(graphicsDriver graphicsdriver.Graphics, name string, blackbg bool) (string, error) {
	return m.orig.DumpScreenshot(graphicsDriver, name, blackbg)
}

func (m *Mipmap) WritePixels(pix []byte, region image.Rectangle) {
	m.orig.WritePixels(pix, region)
	m.markDirty()
}

func (m *Mipmap) markDirty() {
	for i, img := range m.imgs {
		img.dirty = true
		m.imgs[i] = img
	}
}

func (m *Mipmap) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, region image.Rectangle) (ok bool, err error) {
	return m.orig.ReadPixels(graphicsDriver, pixels, region)
}

func (m *Mipmap) DrawTriangles(srcs [graphics.ShaderSrcImageCount]*Mipmap, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *atlas.Shader, uniforms []uint32, fillRule graphicsdriver.FillRule, canSkipMipmap bool, hint restorable.Hint) {
	if len(indices) == 0 {
		return
	}

	// Use the fast path if mipmap is not used.
	if canSkipMipmap || srcs[0] == nil || !canUseMipmap(srcs[0].imageType) {
		var imgs [graphics.ShaderSrcImageCount]*buffered.Image
		for i, src := range srcs {
			if src == nil {
				continue
			}
			imgs[i] = src.orig
		}
		m.orig.DrawTriangles(imgs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, fillRule, hint)
		m.markDirty()
		return
	}

	level := math.MaxInt32
	for i := 0; i < len(indices); i += 3 {
		idx0 := indices[i]
		idx1 := indices[i+1]
		idx2 := indices[i+2]
		dx0 := vertices[graphics.VertexFloatCount*idx0]
		dy0 := vertices[graphics.VertexFloatCount*idx0+1]
		sx0 := vertices[graphics.VertexFloatCount*idx0+2]
		sy0 := vertices[graphics.VertexFloatCount*idx0+3]
		dx1 := vertices[graphics.VertexFloatCount*idx1]
		dy1 := vertices[graphics.VertexFloatCount*idx1+1]
		sx1 := vertices[graphics.VertexFloatCount*idx1+2]
		sy1 := vertices[graphics.VertexFloatCount*idx1+3]
		dx2 := vertices[graphics.VertexFloatCount*idx2]
		dy2 := vertices[graphics.VertexFloatCount*idx2+1]
		sx2 := vertices[graphics.VertexFloatCount*idx2+2]
		sy2 := vertices[graphics.VertexFloatCount*idx2+3]
		if l := mipmapLevelFromDistance(dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1); level > l {
			level = l
		}
		if l := mipmapLevelFromDistance(dx1, dy1, dx2, dy2, sx1, sy1, sx2, sy2); level > l {
			level = l
		}
		if l := mipmapLevelFromDistance(dx2, dy2, dx0, dy0, sx2, sy2, sx0, sy0); level > l {
			level = l
		}
	}
	if level == math.MaxInt32 {
		panic("mipmap: level must be calculated at least once but not")
	}

	var imgs [graphics.ShaderSrcImageCount]*buffered.Image
	for i, src := range srcs {
		if src == nil {
			continue
		}
		if level != 0 {
			if img := src.level(level); img != nil {
				s := float32(pow2(level))
				for i := 0; i < len(vertices); i += graphics.VertexFloatCount {
					vertices[i+2] /= s
					vertices[i+3] /= s
				}
				imgs[i] = img
				continue
			}
		}
		imgs[i] = src.orig
	}

	m.orig.DrawTriangles(imgs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, fillRule, hint)
	m.markDirty()
}

func (m *Mipmap) setImg(level int, img *buffered.Image) {
	if m.imgs == nil {
		m.imgs = map[int]imageWithDirtyFlag{}
	}
	m.imgs[level] = imageWithDirtyFlag{
		img:   img,
		dirty: false,
	}
}

func (m *Mipmap) level(level int) *buffered.Image {
	if level == 0 {
		panic("mipmap: level must be non-zero at level")
	}

	if !canUseMipmap(m.imageType) {
		panic("mipmap: mipmap images for a screen image is not implemented yet")
	}

	img, ok := m.imgs[level]
	if ok && !img.dirty {
		return img.img
	}

	var srcW, srcH int
	var src *buffered.Image
	vs := make([]float32, 4*graphics.VertexFloatCount)
	switch {
	case level == 1:
		src = m.orig
		srcW = m.width
		srcH = m.height
	case level > 1:
		src = m.level(level - 1)
		if src == nil {
			m.setImg(level, nil)
			return nil
		}
		srcW = sizeForLevel(m.width, level-1)
		srcH = sizeForLevel(m.height, level-1)
	default:
		panic(fmt.Sprintf("mipmap: invalid level: %d", level))
	}

	graphics.QuadVerticesFromSrcAndMatrix(vs, 0, 0, float32(srcW), float32(srcH), 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1)

	is := graphics.QuadIndices()

	dstW := sizeForLevel(m.width, level)
	dstH := sizeForLevel(m.height, level)
	if dstW == 0 || dstH == 0 {
		m.setImg(level, nil)
		return nil
	}
	// buffered.NewImage panics with a too big size when actual allocation happens.
	// 4096 should be a safe size in most environments (#1399).
	// Unfortunately a precise max image size cannot be obtained here since this requires GPU access.
	if dstW > 4096 || dstH > 4096 {
		m.setImg(level, nil)
		return nil
	}

	var s *buffered.Image
	if img.img != nil {
		// As s is overwritten, this doesn't have to be cleared.
		s = img.img
	} else {
		s = buffered.NewImage(dstW, dstH, m.imageType)
	}

	dstRegion := image.Rect(0, 0, dstW, dstH)
	srcRegion := image.Rect(0, 0, srcW, srcH)
	s.DrawTriangles([graphics.ShaderSrcImageCount]*buffered.Image{src}, vs, is, graphicsdriver.BlendCopy, dstRegion, [graphics.ShaderSrcImageCount]image.Rectangle{srcRegion}, atlas.LinearFilterShader, nil, graphicsdriver.FillRuleFillAll, restorable.HintOverwriteDstRegion)
	m.setImg(level, s)

	return m.imgs[level].img
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

func (m *Mipmap) Deallocate() {
	for _, img := range m.imgs {
		if img.img == nil {
			continue
		}
		img.img.Deallocate()
	}
	for k := range m.imgs {
		delete(m.imgs, k)
	}
	m.orig.Deallocate()
}

// mipmapLevel returns an appropriate mipmap level for the given distance.
func mipmapLevelFromDistance(dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1 float32) int {
	const maxLevel = 6

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
