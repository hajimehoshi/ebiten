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
	img *shareable.Image
}

func BeginFrame() error {
	if err := shareable.BeginFrame(); err != nil {
		return err
	}
	makeDelayedCommandFlushable()

	return nil
}

func EndFrame() error {
	if !flushDelayedCommands() {
		panic("buffered: the command queue must be available at EndFrame")
	}
	return shareable.EndFrame()
}

func NewImage(width, height int, volatile bool) *Image {
	i := &Image{}
	enqueueDelayedCommand(func() {
		i.img = shareable.NewImage(width, height, volatile)
	})
	return i
}

func NewScreenFramebufferImage(width, height int) *Image {
	i := &Image{}
	enqueueDelayedCommand(func() {
		i.img = shareable.NewScreenFramebufferImage(width, height)
	})
	return i
}

func (i *Image) MarkDisposed() {
	enqueueDelayedCommand(func() {
		i.img.MarkDisposed()
	})
}

func (i *Image) At(x, y int) (r, g, b, a byte) {
	if !flushDelayedCommands() {
		panic("buffered: the command queue is not available yet at At")
	}
	return i.img.At(x, y)
}

func (i *Image) Dump(name string) error {
	if !flushDelayedCommands() {
		panic("buffered: the command queue is not available yet at Dump")
	}
	return i.img.Dump(name)
}

func (i *Image) Fill(clr color.RGBA) {
	enqueueDelayedCommand(func() {
		i.img.Fill(clr)
	})
}

func (i *Image) ClearFramebuffer() {
	enqueueDelayedCommand(func() {
		i.img.ClearFramebuffer()
	})
}

func (i *Image) ReplacePixels(pix []byte) {
	enqueueDelayedCommand(func() {
		i.img.ReplacePixels(pix)
	})
}

func (i *Image) DrawTriangles(src *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	if i == src {
		panic("buffered: Image.DrawTriangles: src must be different from the receiver")
	}
	enqueueDelayedCommand(func() {
		i.img.DrawTriangles(src.img, vertices, indices, colorm, mode, filter, address)
	})
}
