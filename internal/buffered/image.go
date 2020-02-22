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
)

type Image struct {
	img    *mipmap.Mipmap
	width  int
	height int

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
}

func (i *Image) resolvePendingPixels(keepPendingPixels bool) {
	if !i.needsToResolvePixels {
		return
	}

	i.img.ReplacePixels(i.pixels)
	if !keepPendingPixels {
		i.pixels = nil
	}
	i.needsToResolvePixels = false
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
	// TODO: Use pending pixels
	i.resolvePendingPixels(true)
	return i.img.At(x, y)
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

	i.invalidatePendingPixels()
	i.img.Fill(clr)
}

func (i *Image) ReplacePixels(pix []byte, x, y, width, height int) error {
	if l := 4 * width * height; len(pix) != l {
		panic(fmt.Sprintf("buffered: len(pix) was %d but must be %d", len(pix), l))
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

	if x == 0 && y == 0 && width == i.width && height == i.height {
		i.invalidatePendingPixels()
		i.img.ReplacePixels(pix)
		return nil
	}

	// TODO: Can we use (*restorable.Image).ReplacePixels?
	if i.pixels == nil {
		pix := make([]byte, 4*i.width*i.height)
		idx := 0
		img := i.img
		sw, sh := i.width, i.height
		for j := 0; j < sh; j++ {
			for i := 0; i < sw; i++ {
				r, g, b, a, err := img.At(i, j)
				if err != nil {
					return err
				}
				pix[4*idx] = r
				pix[4*idx+1] = g
				pix[4*idx+2] = b
				pix[4*idx+3] = a
				idx++
			}
		}
		i.pixels = pix
	}
	for j := 0; j < height; j++ {
		copy(i.pixels[4*((j+y)*i.width+x):], pix[4*j*width:4*(j+1)*width])
	}
	i.needsToResolvePixels = true
	return nil
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
func (i *Image) DrawTriangles(src *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	if i == src {
		panic("buffered: Image.DrawTriangles: src must be different from the receiver")
	}

	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			// Arguments are not copied. Copying is the caller's responsibility.
			i.DrawTriangles(src, vertices, indices, colorm, mode, filter, address)
			return nil
		})
		return
	}

	src.resolvePendingPixels(true)
	i.resolvePendingPixels(false)
	i.img.DrawTriangles(src.img, vertices, indices, colorm, mode, filter, address)
}
