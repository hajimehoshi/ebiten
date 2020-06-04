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
	"image/color"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/mipmap"
	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

type Image struct {
	img    *mipmap.Mipmap
	width  int
	height int

	hasFill   bool
	fillColor color.RGBA

	pixels               []byte
	needsToResolvePixels bool
}

func BeginFrame() error {
	if err := mipmap.BeginFrame(); err != nil {
		return err
	}
	return flushDelayedCommands()
}

func EndFrame() error {
	return mipmap.EndFrame()
}

func NewImage(width, height int, volatile bool) *Image {
	i := &Image{}
	i.initialize(width, height, volatile)
	return i
}

func (i *Image) initialize(width, height int, volatile bool) {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.initialize(width, height, volatile)
			return nil
		})
		return
	}
	i.img = mipmap.New(width, height, volatile)
	i.width = width
	i.height = height
}

func NewScreenFramebufferImage(width, height int) *Image {
	i := &Image{}
	i.initializeAsScreenFramebuffer(width, height)
	return i
}

func (i *Image) initializeAsScreenFramebuffer(width, height int) {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.initializeAsScreenFramebuffer(width, height)
			return nil
		})
		return
	}

	i.img = mipmap.NewScreenFramebufferMipmap(width, height)
	i.width = width
	i.height = height
}

func (i *Image) invalidatePendingPixels() {
	i.pixels = nil
	i.needsToResolvePixels = false
	i.hasFill = false
}

func (i *Image) resolvePendingPixels(keepPendingPixels bool) {
	if i.needsToResolvePixels && i.hasFill {
		panic("buffered: needsToResolvePixels and hasFill must not be true at the same time")
	}
	if i.needsToResolvePixels {
		i.img.ReplacePixels(i.pixels)
		if !keepPendingPixels {
			i.pixels = nil
		}
		i.needsToResolvePixels = false
	}
	i.resolvePendingFill()
}

func (i *Image) resolvePendingFill() {
	if !i.hasFill {
		return
	}
	i.img.Fill(i.fillColor)
	i.hasFill = false
}

func (i *Image) MarkDisposed() {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.MarkDisposed()
			return nil
		})
		return
	}
	i.invalidatePendingPixels()
	i.img.MarkDisposed()
}

func (i *Image) At(x, y int) (r, g, b, a byte, err error) {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()
	if needsToDelayCommands {
		panic("buffered: the command queue is not available yet at At")
	}

	if x < 0 || y < 0 || x >= i.width || y >= i.height {
		return 0, 0, 0, 0, nil
	}

	// If there are pixels or pending fillling that needs to be resolved, use this rather than resolving.
	// Resolving them needs to access GPU and is expensive (#1137).
	if i.hasFill {
		return i.fillColor.R, i.fillColor.G, i.fillColor.B, i.fillColor.A, nil
	}

	if i.pixels == nil {
		pix, err := i.img.Pixels(0, 0, i.width, i.height)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		i.pixels = pix
	}

	idx := i.width*y + x
	return i.pixels[4*idx], i.pixels[4*idx+1], i.pixels[4*idx+2], i.pixels[4*idx+3], nil
}

func (i *Image) Dump(name string, blackbg bool) error {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()
	if needsToDelayCommands {
		panic("buffered: the command queue is not available yet at Dump")
	}
	return i.img.Dump(name, blackbg)
}

func (i *Image) Fill(clr color.RGBA) {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.Fill(clr)
			return nil
		})
		return
	}

	// Defer filling the image so that successive fillings will be merged into one (#1134).
	i.invalidatePendingPixels()
	i.fillColor = clr
	i.hasFill = true
}

func (i *Image) ReplacePixels(pix []byte, x, y, width, height int) error {
	if l := 4 * width * height; len(pix) != l {
		panic(fmt.Sprintf("buffered: len(pix) was %d but must be %d", len(pix), l))
	}

	// This is an optimization to avoid mutex for the case when ReplacePixels is called very often (e.g., Set).
	// If i.pixels is not nil, delayed commands have already been flushed.
	// needsToDelayCommands should be false, but we don't check it because this is out of the mutex lock.
	// (#1137)
	if i.pixels != nil {
		// If the region is the whole image, don't use this optimization, or more memory is consumed by
		// keeping pixels.
		if !(x == 0 && y == 0 && width == i.width && height == i.height) {
			i.replacePendingPixels(pix, x, y, width, height)
			return nil
		}
	}

	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if needsToDelayCommands {
		copied := make([]byte, len(pix))
		copy(copied, pix)
		delayedCommands = append(delayedCommands, func() error {
			i.ReplacePixels(copied, x, y, width, height)
			return nil
		})
		return nil
	}

	i.resolvePendingFill()

	if x == 0 && y == 0 && width == i.width && height == i.height {
		i.invalidatePendingPixels()
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

func (i *Image) DrawImage(src *Image, bounds image.Rectangle, a, b, c, d, tx, ty float32, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter) {
	if i == src {
		panic("buffered: Image.DrawImage: src must be different from the receiver")
	}

	g := mipmap.GeoM{
		A:  a,
		B:  b,
		C:  c,
		D:  d,
		Tx: tx,
		Ty: ty,
	}

	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.drawImage(src, bounds, g, colorm, mode, filter)
			return nil
		})
		return
	}

	i.drawImage(src, bounds, g, colorm, mode, filter)
}

func (i *Image) drawImage(src *Image, bounds image.Rectangle, g mipmap.GeoM, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter) {
	src.resolvePendingPixels(true)
	i.resolvePendingPixels(false)
	i.img.DrawImage(src.img, bounds, g, colorm, mode, filter)
}

// DrawTriangles draws the src image with the given vertices.
//
// Copying vertices and indices is the caller's responsibility.
func (i *Image) DrawTriangles(src *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, shader *Shader, uniforms []interface{}) {
	var srcs []*Image
	if src != nil {
		srcs = append(srcs, src)
	}
	for _, u := range uniforms {
		if src, ok := u.(*Image); ok {
			srcs = append(srcs, src)
		}
	}

	for _, src := range srcs {
		if i == src {
			panic("buffered: Image.DrawTriangles: src must be different from the receiver")
		}
	}

	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			// Arguments are not copied. Copying is the caller's responsibility.
			i.DrawTriangles(src, vertices, indices, colorm, mode, filter, address, shader, uniforms)
			return nil
		})
		return
	}

	for _, src := range srcs {
		src.resolvePendingPixels(true)
	}
	i.resolvePendingPixels(false)

	var s *mipmap.Shader
	if shader != nil {
		s = shader.shader
	}
	us := make([]interface{}, len(uniforms))
	for k, v := range uniforms {
		switch v := v.(type) {
		case *Image:
			i.resolvePendingPixels(true)
			us[k] = v.img
		default:
			us[k] = v
		}
	}

	var srcImg *mipmap.Mipmap
	if src != nil {
		srcImg = src.img
	}
	i.img.DrawTriangles(srcImg, vertices, indices, colorm, mode, filter, address, s, us)
}

type Shader struct {
	shader *mipmap.Shader
}

func NewShader(program *shaderir.Program) *Shader {
	return &Shader{
		shader: mipmap.NewShader(program),
	}
}

func (s *Shader) MarkDisposed() {
	s.shader.MarkDisposed()
	s.shader = nil
}
