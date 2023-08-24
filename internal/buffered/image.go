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

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Image struct {
	img    *atlas.Image
	width  int
	height int

	// pixels is valid only when restorable.AlwaysReadPixelsFromGPU() returns true.
	pixels []byte
}

func BeginFrame(graphicsDriver graphicsdriver.Graphics) error {
	if err := atlas.BeginFrame(graphicsDriver); err != nil {
		return err
	}
	flushDelayedCommands()
	return nil
}

func EndFrame(graphicsDriver graphicsdriver.Graphics, swapBuffersForGL func()) error {
	return atlas.EndFrame(graphicsDriver, swapBuffersForGL)
}

func NewImage(width, height int, imageType atlas.ImageType) *Image {
	i := &Image{
		width:  width,
		height: height,
	}
	i.initialize(imageType)
	return i
}

func (i *Image) initialize(imageType atlas.ImageType) {
	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() {
			i.initializeImpl(imageType)
		}) {
			return
		}
	}
	i.initializeImpl(imageType)
}

func (i *Image) initializeImpl(imageType atlas.ImageType) {
	i.img = atlas.NewImage(i.width, i.height, imageType)
}

func (i *Image) invalidatePixels() {
	i.pixels = nil
}

func (i *Image) MarkDisposed() {
	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() {
			i.markDisposedImpl()
		}) {
			return
		}
	}
	i.markDisposedImpl()
}

func (i *Image) markDisposedImpl() {
	i.invalidatePixels()
	i.img.MarkDisposed()
}

func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, region image.Rectangle) error {
	checkDelayedCommandsFlushed("ReadPixels")

	// If restorable.AlwaysReadPixelsFromGPU() returns false, the pixel data is cached in the restorable package.
	if !restorable.AlwaysReadPixelsFromGPU() {
		if err := i.img.ReadPixels(graphicsDriver, pixels, region); err != nil {
			return err
		}
		return nil
	}

	if i.pixels == nil {
		pix := make([]byte, 4*i.width*i.height)
		if err := i.img.ReadPixels(graphicsDriver, pix, image.Rect(0, 0, i.width, i.height)); err != nil {
			return err
		}
		i.pixels = pix
	}

	lineWidth := 4 * region.Dx()
	for j := 0; j < region.Dy(); j++ {
		dstX := 4 * j * region.Dx()
		srcX := 4 * ((region.Min.Y+j)*i.width + region.Min.X)
		copy(pixels[dstX:dstX+lineWidth], i.pixels[srcX:srcX+lineWidth])
	}
	return nil
}

func (i *Image) DumpScreenshot(graphicsDriver graphicsdriver.Graphics, name string, blackbg bool) (string, error) {
	checkDelayedCommandsFlushed("Dump")
	return i.img.DumpScreenshot(graphicsDriver, name, blackbg)
}

// WritePixels replaces the pixels at the specified region.
func (i *Image) WritePixels(pix []byte, region image.Rectangle) {
	if l := 4 * region.Dx() * region.Dy(); len(pix) != l {
		panic(fmt.Sprintf("buffered: len(pix) was %d but must be %d", len(pix), l))
	}

	if maybeCanAddDelayedCommand() {
		copied := make([]byte, len(pix))
		copy(copied, pix)
		if tryAddDelayedCommand(func() {
			i.writePixelsImpl(copied, region)
		}) {
			return
		}
	}
	i.writePixelsImpl(pix, region)
}

func (i *Image) writePixelsImpl(pix []byte, region image.Rectangle) {
	i.invalidatePixels()
	i.img.WritePixels(pix, region)
}

// DrawTriangles draws the src image with the given vertices.
//
// Copying vertices and indices is the caller's responsibility.
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageCount]*Image, vertices []float32, indices []uint16, blend graphicsdriver.Blend, dstRegion graphicsdriver.Region, srcRegions [graphics.ShaderImageCount]graphicsdriver.Region, shader *Shader, uniforms []uint32, evenOdd bool) {
	for _, src := range srcs {
		if i == src {
			panic("buffered: Image.DrawTriangles: source images must be different from the receiver")
		}
	}

	if maybeCanAddDelayedCommand() {
		vs := make([]float32, len(vertices))
		copy(vs, vertices)
		is := make([]uint16, len(indices))
		copy(is, indices)
		us := make([]uint32, len(uniforms))
		copy(us, uniforms)
		if tryAddDelayedCommand(func() {
			i.drawTrianglesImpl(srcs, vs, is, blend, dstRegion, srcRegions, shader, us, evenOdd)
		}) {
			return
		}
	}
	i.drawTrianglesImpl(srcs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, evenOdd)
}

func (i *Image) drawTrianglesImpl(srcs [graphics.ShaderImageCount]*Image, vertices []float32, indices []uint16, blend graphicsdriver.Blend, dstRegion graphicsdriver.Region, srcRegions [graphics.ShaderImageCount]graphicsdriver.Region, shader *Shader, uniforms []uint32, evenOdd bool) {
	var imgs [graphics.ShaderImageCount]*atlas.Image
	for i, img := range srcs {
		if img == nil {
			continue
		}
		imgs[i] = img.img
	}

	i.invalidatePixels()
	i.img.DrawTriangles(imgs, vertices, indices, blend, dstRegion, srcRegions, shader.shader, uniforms, evenOdd)
}

type Shader struct {
	shader *atlas.Shader
}

func NewShader(ir *shaderir.Program) *Shader {
	s := &Shader{}
	s.initialize(ir)
	return s
}

func (s *Shader) initialize(ir *shaderir.Program) {
	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() {
			s.initializeImpl(ir)
		}) {
			return
		}
	}
	s.initializeImpl(ir)
}

func (s *Shader) initializeImpl(ir *shaderir.Program) {
	s.shader = atlas.NewShader(ir)
}

func (s *Shader) MarkDisposed() {
	if maybeCanAddDelayedCommand() {
		if tryAddDelayedCommand(func() {
			s.markDisposedImpl()
		}) {
			return
		}
	}
	s.markDisposedImpl()
}

func (s *Shader) markDisposedImpl() {
	s.shader.MarkDisposed()
	s.shader = nil
}

var (
	NearestFilterShader = &Shader{shader: atlas.NearestFilterShader}
	LinearFilterShader  = &Shader{shader: atlas.LinearFilterShader}
)
