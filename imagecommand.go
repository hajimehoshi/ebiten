// Copyright 2016 Hajime Hoshi
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

package ebiten

import (
	"errors"
	"image"
	"image/color"
	"runtime"

	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

// Commands are used only before the GL Context is created.

type imageCommand interface {
	Exec() error
}

var (
	imageCommandQueue = []imageCommand{}
)

func execBufferedImageCommands() error {
	imageM.Lock()
	defer imageM.Unlock()
	for _, c := range imageCommandQueue {
		if err := c.Exec(); err != nil {
			return err
		}
	}
	imageCommandQueue = nil
	return nil
}

type fillCommand struct {
	dst   *Image
	color color.Color
}

func (c *fillCommand) Exec() error {
	if c.dst.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	c.dst.pixels = nil
	return c.dst.framebuffer.Fill(glContext, c.color)
}

type drawImageCommand struct {
	dst           *Image
	src           *Image
	vertices      []int16
	geoM          GeoM
	colorM        ColorM
	compositeMode CompositeMode
}

func (c *drawImageCommand) Exec() error {
	if c.dst.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	c.dst.pixels = nil
	m := opengl.CompositeMode(c.compositeMode)
	return c.dst.framebuffer.DrawTexture(glContext, c.src.texture, c.vertices, &c.geoM, &c.colorM, m)
}

type replacePixelsCommand struct {
	dst    *Image
	pixels []uint8
}

func (c *replacePixelsCommand) Exec() error {
	if c.dst.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	return c.dst.framebuffer.ReplacePixels(glContext, c.dst.texture, c.pixels)
}

type disposeCommand struct {
	image *Image
}

func (c *disposeCommand) Exec() error {
	if c.image.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	if c.image.framebuffer != nil {
		if err := c.image.framebuffer.Dispose(glContext); err != nil {
			return err
		}
		c.image.framebuffer = nil
	}
	if c.image.texture != nil {
		if err := c.image.texture.Dispose(glContext); err != nil {
			return err
		}
		c.image.texture = nil
	}
	c.image.disposed = true
	c.image.pixels = nil
	runtime.SetFinalizer(c.image, nil)
	return nil
}

type newImageCommand struct {
	result *Image
	width  int
	height int
	filter Filter
}

func (c *newImageCommand) Exec() error {
	texture, err := graphics.NewTexture(glContext, c.width, c.height, glFilter(glContext, c.filter))
	if err != nil {
		return err
	}
	framebuffer, err := graphics.NewFramebufferFromTexture(glContext, texture)
	if err != nil {
		// TODO: texture should be removed here?
		return err
	}
	c.result.framebuffer = framebuffer
	c.result.texture = texture
	runtime.SetFinalizer(c.result, (*Image).Dispose)
	if err := c.result.framebuffer.Fill(glContext, color.Transparent); err != nil {
		return err
	}
	return nil
}

type newImageFromImageCommand struct {
	image  image.Image
	filter Filter
	result *Image
}

func (c *newImageFromImageCommand) Exec() error {
	texture, err := graphics.NewTextureFromImage(glContext, c.image, glFilter(glContext, c.filter))
	if err != nil {
		return err
	}
	framebuffer, err := graphics.NewFramebufferFromTexture(glContext, texture)
	if err != nil {
		// TODO: texture should be removed here?
		return err
	}
	c.result.framebuffer = framebuffer
	c.result.texture = texture
	runtime.SetFinalizer(c.result, (*Image).Dispose)
	return nil
}
