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
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Image struct {
	img    *atlas.Image
	width  int
	height int

	pixels               []byte
	needsToResolvePixels bool
}

func BeginFrame() error {
	if err := atlas.BeginFrame(); err != nil {
		return err
	}
	return flushDelayedCommands()
}

func EndFrame() error {
	return atlas.EndFrame()
}

func NewImage(width, height int) *Image {
	i := &Image{}
	i.initialize(width, height)
	return i
}

func (i *Image) initialize(width, height int) {
	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() error {
			i.initialize(width, height)
			return nil
		}) {
			return
		}
	}
	i.img = atlas.NewImage(width, height)
	i.width = width
	i.height = height
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

func (i *Image) invalidatePendingPixels() {
	i.pixels = nil
	i.needsToResolvePixels = false
}

func (i *Image) resolvePendingPixels(keepPendingPixels bool) {
	if i.needsToResolvePixels {
		i.img.ReplacePixels(i.pixels)
		if !keepPendingPixels {
			i.pixels = nil
		}
		i.needsToResolvePixels = false
	}
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
	i.invalidatePendingPixels()
	i.img.MarkDisposed()
}

func (img *Image) Pixels(x, y, width, height int) (pix []byte, err error) {
	checkDelayedCommandsFlushed("Pixels")

	if !image.Rect(x, y, x+width, y+height).In(image.Rect(0, 0, img.width, img.height)) {
		return nil, fmt.Errorf("buffered: out of range")
	}

	pix = make([]byte, 4*width*height)

	if img.pixels == nil {
		pix, err := img.img.Pixels(0, 0, img.width, img.height)
		if err != nil {
			return nil, err
		}
		img.pixels = pix
	}

	for j := 0; j < height; j++ {
		copy(pix[4*j*width:4*(j+1)*width], img.pixels[4*((j+y)*img.width+x):])
	}
	return pix, nil
}

func (i *Image) DumpScreenshot(name string, blackbg bool) error {
	checkDelayedCommandsFlushed("Dump")
	return i.img.DumpScreenshot(name, blackbg)
}

func (i *Image) ReplacePixels(pix []byte, x, y, width, height int) error {
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
			return nil
		}
	}

	if x == 0 && y == 0 && width == i.width && height == i.height {
		i.invalidatePendingPixels()

		// Call ReplacePixels immediately. Do not buffer the command.
		// If a lot of new images are created but they are used at different timings,
		// pixels are sent to GPU at different timings, which is very inefficient.
		i.img.ReplacePixels(pix)
		return nil
	}

	// TODO: Can we use (*restorable.Image).ReplacePixels?
	if i.pixels == nil {
		pix, err := i.img.Pixels(0, 0, i.width, i.height)
		if err != nil {
			return err
		}
		i.pixels = pix
	}
	i.replacePendingPixels(pix, x, y, width, height)
	return nil
}

func (i *Image) replacePendingPixels(pix []byte, x, y, width, height int) {
	for j := 0; j < height; j++ {
		copy(i.pixels[4*((j+y)*i.width+x):], pix[4*j*width:4*(j+1)*width])
	}
	i.needsToResolvePixels = true
}

// DrawTriangles draws the src image with the given vertices.
//
// Copying vertices and indices is the caller's responsibility.
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageNum]*Image, vertices []float32, indices []uint16, colorm affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, dstRegion, srcRegion driver.Region, subimageOffsets [graphics.ShaderImageNum - 1][2]float32, shader *Shader, uniforms []interface{}, evenOdd bool) {
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
	i.invalidatePendingPixels()
}

type Shader struct {
	shader *atlas.Shader
}

func NewShader(program *shaderir.Program) *Shader {
	return &Shader{
		shader: atlas.NewShader(program),
	}
}

func (s *Shader) MarkDisposed() {
	s.shader.MarkDisposed()
	s.shader = nil
}
