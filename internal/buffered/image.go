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
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.img = mipmap.New(width, height, volatile)
			i.width = width
			i.height = height
			return nil
		})
		delayedCommandsM.Unlock()
		return i
	}
	delayedCommandsM.Unlock()

	i.img = mipmap.New(width, height, volatile)
	i.width = width
	i.height = height
	return i
}

func NewScreenFramebufferImage(width, height int) *Image {
	i := &Image{}
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.img = mipmap.NewScreenFramebufferMipmap(width, height)
			i.width = width
			i.height = height
			return nil
		})
		delayedCommandsM.Unlock()
		return i
	}
	delayedCommandsM.Unlock()

	i.img = mipmap.NewScreenFramebufferMipmap(width, height)
	i.width = width
	i.height = height
	return i
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
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.img.MarkDisposed()
			return nil
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()

	i.invalidatePendingPixels()
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

func (i *Image) Set(x, y int, r, g, b, a byte) error {
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			return i.set(x, y, r, g, b, a)
		})
		delayedCommandsM.Unlock()
		return nil
	}
	delayedCommandsM.Unlock()

	return i.set(x, y, r, g, b, a)
}

func (img *Image) set(x, y int, r, g, b, a byte) error {
	w, h := img.width, img.height
	if img.pixels == nil {
		pix := make([]byte, 4*w*h)
		idx := 0
		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				r, g, b, a, err := img.img.At(i, j)
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
		img.pixels = pix
	}
	img.pixels[4*(x+y*w)] = r
	img.pixels[4*(x+y*w)+1] = g
	img.pixels[4*(x+y*w)+2] = b
	img.pixels[4*(x+y*w)+3] = a
	img.needsToResolvePixels = true
	return nil
}

func (i *Image) Dump(name string) error {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()
	if needsToDelayCommands {
		panic("buffered: the command queue is not available yet at Dump")
	}
	return i.img.Dump(name)
}

func (i *Image) Fill(clr color.RGBA) {
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.img.Fill(clr)
			return nil
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()

	i.invalidatePendingPixels()
	i.img.Fill(clr)
}

func (i *Image) ReplacePixels(pix []byte) {
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			copied := make([]byte, len(pix))
			copy(copied, pix)
			i.img.ReplacePixels(copied)
			return nil
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()

	i.invalidatePendingPixels()
	i.img.ReplacePixels(pix)
}

func (i *Image) DrawImage(src *Image, bounds image.Rectangle, a, b, c, d, tx, ty float32, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter) {
	if i == src {
		panic("buffered: Image.DrawImage: src must be different from the receiver")
	}

	g := &mipmap.GeoM{
		A:  a,
		B:  b,
		C:  c,
		D:  d,
		Tx: tx,
		Ty: ty,
	}

	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.drawImage(src, bounds, g, colorm, mode, filter)
			return nil
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()

	i.drawImage(src, bounds, g, colorm, mode, filter)
}

func (i *Image) drawImage(src *Image, bounds image.Rectangle, g *mipmap.GeoM, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter) {
	src.resolvePendingPixels(true)
	i.resolvePendingPixels(false)
	i.img.DrawImage(src.img, bounds, g, colorm, mode, filter)
}

func (i *Image) DrawTriangles(src *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	if i == src {
		panic("buffered: Image.DrawTriangles: src must be different from the receiver")
	}

	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() error {
			i.drawTriangles(src, vertices, indices, colorm, mode, filter, address)
			return nil
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()
	i.drawTriangles(src, vertices, indices, colorm, mode, filter, address)
}

func (i *Image) drawTriangles(src *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	src.resolvePendingPixels(true)
	i.resolvePendingPixels(false)
	i.img.DrawTriangles(src.img, vertices, indices, colorm, mode, filter, address)
}
