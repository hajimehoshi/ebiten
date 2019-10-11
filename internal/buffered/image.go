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
	"image/color"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/shareable"
)

type Image struct {
	img    *shareable.Image
	width  int
	height int

	pendingPixels []byte
}

func BeginFrame() error {
	if err := shareable.BeginFrame(); err != nil {
		return err
	}
	flushDelayedCommands()
	return nil
}

func EndFrame() error {
	return shareable.EndFrame()
}

func NewImage(width, height int, volatile bool) *Image {
	i := &Image{}
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() {
			i.img = shareable.NewImage(width, height, volatile)
			i.width = width
			i.height = height
		})
		delayedCommandsM.Unlock()
		return i
	}
	delayedCommandsM.Unlock()

	i.img = shareable.NewImage(width, height, volatile)
	i.width = width
	i.height = height
	return i
}

func NewScreenFramebufferImage(width, height int) *Image {
	i := &Image{}
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() {
			i.img = shareable.NewScreenFramebufferImage(width, height)
			i.width = width
			i.height = height
		})
		delayedCommandsM.Unlock()
		return i
	}
	delayedCommandsM.Unlock()

	i.img = shareable.NewScreenFramebufferImage(width, height)
	i.width = width
	i.height = height
	return i
}

func (i *Image) invalidatePendingPixels() {
	i.pendingPixels = nil
}

func (i *Image) resolvePendingPixels(keepPendingPixels bool) {
	if i.pendingPixels == nil {
		return
	}

	i.img.ReplacePixels(i.pendingPixels)
	if !keepPendingPixels {
		i.pendingPixels = nil
	}
}

func (i *Image) MarkDisposed() {
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() {
			i.img.MarkDisposed()
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()

	i.invalidatePendingPixels()
}

func (i *Image) At(x, y int) (r, g, b, a byte) {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()
	if needsToDelayCommands {
		panic("buffered: the command queue is not available yet at At")
	}
	// TODO: Use pending pixels
	i.resolvePendingPixels(true)
	return i.img.At(x, y)
}

func (i *Image) Set(x, y int, r, g, b, a byte) {
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() {
			i.set(x, y, r, g, b, a)
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()

	i.set(x, y, r, g, b, a)
}

func (img *Image) set(x, y int, r, g, b, a byte) {
	w, h := img.width, img.height
	if img.pendingPixels == nil {
		pix := make([]byte, 4*w*h)
		idx := 0
		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				r, g, b, a := img.img.At(i, j)
				pix[4*idx] = r
				pix[4*idx+1] = g
				pix[4*idx+2] = b
				pix[4*idx+3] = a
				idx++
			}
		}
		img.pendingPixels = pix
	}
	img.pendingPixels[4*(x+y*w)] = r
	img.pendingPixels[4*(x+y*w)+1] = g
	img.pendingPixels[4*(x+y*w)+2] = b
	img.pendingPixels[4*(x+y*w)+3] = a
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
		delayedCommands = append(delayedCommands, func() {
			i.img.Fill(clr)
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()

	i.invalidatePendingPixels()
	i.img.Fill(clr)
}

func (i *Image) ClearFramebuffer() {
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() {
			i.img.ClearFramebuffer()
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()

	i.img.ClearFramebuffer()
}

func (i *Image) ReplacePixels(pix []byte) {
	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() {
			i.img.ReplacePixels(pix)
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()

	i.invalidatePendingPixels()
	i.img.ReplacePixels(pix)
}

func (i *Image) DrawTriangles(src *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	if i == src {
		panic("buffered: Image.DrawTriangles: src must be different from the receiver")
	}

	delayedCommandsM.Lock()
	if needsToDelayCommands {
		delayedCommands = append(delayedCommands, func() {
			i.img.DrawTriangles(src.img, vertices, indices, colorm, mode, filter, address)
		})
		delayedCommandsM.Unlock()
		return
	}
	delayedCommandsM.Unlock()

	src.resolvePendingPixels(true)
	i.resolvePendingPixels(false)
	i.img.DrawTriangles(src.img, vertices, indices, colorm, mode, filter, address)
}
