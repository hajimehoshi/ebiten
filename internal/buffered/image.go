// Copyright 2019 The Ebiten Authors
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

package buffered

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Image struct {
	img    *atlas.Image
	width  int
	height int

	pixels               []byte
	mask                 []byte
	needsToResolvePixels bool
}

func BeginFrame(graphicsDriver graphicsdriver.Graphics) error {
	if err := atlas.BeginFrame(graphicsDriver); err != nil {
		return err
	}
	return flushDelayedCommands()
}

func EndFrame(graphicsDriver graphicsdriver.Graphics) error {
	return atlas.EndFrame(graphicsDriver)
}

func NewImage(width, height int) *Image {
	i := &Image{
		width:  width,
		height: height,
	}
	i.initialize()
	return i
}

func (i *Image) initialize() {
	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() error {
			i.initialize()
			return nil
		}) {
			return
		}
	}
	i.img = atlas.NewImage(i.width, i.height)
}

func (i *Image) SetIndependent(independent bool) {
	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() error {
			i.SetIndependent(independent)
			return nil
		}) {
			return
		}
	}
	i.img.SetIndependent(independent)
}

func (i *Image) SetVolatile(volatile bool) {
	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() error {
			i.SetVolatile(volatile)
			return nil
		}) {
			return
		}
	}
	i.img.SetVolatile(volatile)
}

func NewScreenFramebufferImage(width, height int) *Image {
	i := &Image{}
	i.initializeAsScreenFramebuffer(width, height)
	return i
}

func (i *Image) initializeAsScreenFramebuffer(width, height int) {
	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() error {
			i.initializeAsScreenFramebuffer(width, height)
			return nil
		}) {
			return
		}
	}

	i.img = atlas.NewScreenFramebufferImage(width, height)
	i.width = width
	i.height = height
}

func (i *Image) invalidatePixels() {
	i.pixels = nil
	i.mask = nil
	i.needsToResolvePixels = false
}

func (i *Image) resolvePendingPixels(keepPendingPixels bool) {
	if !i.needsToResolvePixels {
		return
	}

	i.img.ReplacePixels(i.pixels, i.mask)
	if !keepPendingPixels {
		i.pixels = nil
		i.mask = nil
	}
	i.needsToResolvePixels = false
}

func (i *Image) MarkDisposed() {
	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() error {
			i.MarkDisposed()
			return nil
		}) {
			return
		}
	}
	i.invalidatePixels()
	i.img.MarkDisposed()
}

func (img *Image) At(graphicsDriver graphicsdriver.Graphics, x, y int) (r, g, b, a byte, err error) {
	checkDelayedCommandsFlushed("At")

	idx := (y*img.width + x)
	if img.pixels != nil {
		if img.mask == nil {
			return img.pixels[4*idx], img.pixels[4*idx+1], img.pixels[4*idx+2], img.pixels[4*idx+3], nil
		}
		if img.mask[idx/8]<<(idx%8)&1 != 0 {
			return img.pixels[4*idx], img.pixels[4*idx+1], img.pixels[4*idx+2], img.pixels[4*idx+3], nil
		}

		img.resolvePendingPixels(false)
	}

	pix, err := img.img.Pixels(graphicsDriver)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	img.pixels = pix
	// When pixels represents the whole pixels, the mask is not needed.
	img.mask = nil
	return img.pixels[4*idx], img.pixels[4*idx+1], img.pixels[4*idx+2], img.pixels[4*idx+3], nil
}

func (i *Image) DumpScreenshot(graphicsDriver graphicsdriver.Graphics, name string, blackbg bool) error {
	checkDelayedCommandsFlushed("Dump")
	return i.img.DumpScreenshot(graphicsDriver, name, blackbg)
}

// ReplacePixels replaces the pixels at the specified region.
func (i *Image) ReplacePixels(pix []byte, x, y, width, height int) {
	if l := 4 * width * height; len(pix) != l {
		panic(fmt.Sprintf("buffered: len(pix) was %d but must be %d", len(pix), l))
	}

	if maybeCanAddDelayedCommand() {
		copied := make([]byte, len(pix))
		copy(copied, pix)
		if tryAddDelayedCommand(func() error {
			i.ReplacePixels(copied, x, y, width, height)
			return nil
		}) {
			return
		}
	}

	if x == 0 && y == 0 && width == i.width && height == i.height {
		i.invalidatePixels()
		i.img.ReplacePixels(pix, nil)
		return
	}

	// TODO: If width/height is big enough, ReplacePixels can be called instead of replacePendingPixels.
	// Check if this is efficient.

	i.replacePendingPixels(pix, x, y, width, height)
}

func (img *Image) replacePendingPixels(pix []byte, x, y, width, height int) {
	if img.pixels == nil {
		img.pixels = make([]byte, 4*img.width*img.height)
		if img.mask == nil {
			img.mask = make([]byte, (img.width*img.height-1)/8+1)
		}
	}
	for j := 0; j < height; j++ {
		copy(img.pixels[4*((j+y)*img.width+x):], pix[4*j*width:4*(j+1)*width])
	}

	// A mask is created only when partial regions are replaced by replacePendingPixels.
	if img.mask != nil {
		for j := 0; j < height; j++ {
			for i := 0; i < width; i++ {
				idx := (y+j)*img.width + x + i
				img.mask[idx/8] |= 1 << (idx % 8)
			}
		}
	}

	img.needsToResolvePixels = true
}

// DrawTriangles draws the src image with the given vertices.
//
// Copying vertices and indices is the caller's responsibility.
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageNum]*Image, vertices []float32, indices []uint16, colorm affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, subimageOffsets [graphics.ShaderImageNum - 1][2]float32, shader *Shader, uniforms [][]float32, evenOdd bool) {
	for _, src := range srcs {
		if i == src {
			panic("buffered: Image.DrawTriangles: source images must be different from the receiver")
		}
	}

	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() error {
			// Arguments are not copied. Copying is the caller's responsibility.
			i.DrawTriangles(srcs, vertices, indices, colorm, mode, filter, address, dstRegion, srcRegion, subimageOffsets, shader, uniforms, evenOdd)
			return nil
		}) {
			return
		}
	}

	var s *atlas.Shader
	var imgs [graphics.ShaderImageNum]*atlas.Image
	if shader == nil {
		// Fast path for rendering without a shader (#1355).
		img := srcs[0]
		img.resolvePendingPixels(true)
		imgs[0] = img.img
	} else {
		for i, img := range srcs {
			if img == nil {
				continue
			}
			img.resolvePendingPixels(true)
			imgs[i] = img.img
		}
		s = shader.shader
	}
	i.resolvePendingPixels(false)

	i.img.DrawTriangles(imgs, vertices, indices, colorm, mode, filter, address, dstRegion, srcRegion, subimageOffsets, s, uniforms, evenOdd)
	i.invalidatePixels()
}

type Shader struct {
	shader *atlas.Shader
}

func NewShader(ir *shaderir.Program) *Shader {
	return &Shader{
		shader: atlas.NewShader(ir),
	}
}

func (s *Shader) MarkDisposed() {
	s.shader.MarkDisposed()
	s.shader = nil
}
