// Copyright 2014 Hajime Hoshi
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
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

func newGraphicsContext(f func(*Image) error) *graphicsContext {
	return &graphicsContext{
		f: f,
	}
}

type graphicsContext struct {
	f           func(*Image) error
	offscreen   *Image
	screen      *Image
	screenScale int
	initialized bool
}

func (c *graphicsContext) SetSize(screenWidth, screenHeight, screenScale int) error {
	if c.screen != nil {
		c.screen.Dispose()
	}
	if c.offscreen != nil {
		c.offscreen.Dispose()
	}
	offscreen, err := NewImage(screenWidth, screenHeight, FilterNearest)
	if err != nil {
		return err
	}
	c.screen, err = newImageWithScreenFramebuffer(screenWidth*screenScale, screenHeight*screenScale)
	if err != nil {
		return err
	}
	c.screen.Clear()
	c.offscreen = offscreen
	c.screenScale = screenScale
	ui.GLContext().ResetViewportSize()
	return nil
}

func (c *graphicsContext) needsRestoring(context *opengl.Context) (bool, error) {
	imageM.Lock()
	defer imageM.Unlock()
	// FlushCommands is required because c.offscreen.impl might not have an actual texture.
	if err := graphics.FlushCommands(ui.GLContext()); err != nil {
		return false, err
	}
	return c.offscreen.impl.isInvalidated(context), nil
}

func (c *graphicsContext) initializeIfNeeded() error {
	if !c.initialized {
		if err := graphics.Initialize(ui.GLContext()); err != nil {
			return err
		}
		c.initialized = true
	}
	r, err := c.needsRestoring(ui.GLContext())
	if err != nil {
		return err
	}
	if r {
		if err := c.restore(); err != nil {
			return err
		}
	}
	return nil
}

func (c *graphicsContext) drawToDefaultRenderTarget() error {
	if err := c.screen.Clear(); err != nil {
		return err
	}
	scale := float64(c.screenScale)
	options := &DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	if err := c.screen.DrawImage(c.offscreen, options); err != nil {
		return err
	}
	if err := c.flush(); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) UpdateAndDraw() error {
	if err := c.initializeIfNeeded(); err != nil {
		return err
	}
	if err := c.offscreen.Clear(); err != nil {
		return err
	}
	if err := c.f(c.offscreen); err != nil {
		return err
	}
	if IsRunningSlowly() {
		return nil
	}
	if err := c.drawToDefaultRenderTarget(); err != nil {
		return err
	}
	exceptions := map[*imageImpl]struct{}{
		c.offscreen.impl: {},
		c.screen.impl:    {},
	}
	if err := theImages.savePixels(ui.GLContext(), exceptions); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) Draw() error {
	if err := c.initializeIfNeeded(); err != nil {
		return err
	}
	if err := c.drawToDefaultRenderTarget(); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) flush() error {
	// TODO: imageM is necessary to call graphics functions. Move this to graphics package.
	imageM.Lock()
	defer imageM.Unlock()
	if err := graphics.FlushCommands(ui.GLContext()); err != nil {
		return err
	}
	// Call glFlush to prevent black flicking (especially on Android (#226)).
	ui.GLContext().Flush()
	return nil
}

func (c *graphicsContext) restore() error {
	imageM.Lock()
	defer imageM.Unlock()
	ui.GLContext().Resume()
	if err := graphics.Initialize(ui.GLContext()); err != nil {
		return err
	}
	if err := theImages.restorePixels(ui.GLContext()); err != nil {
		return err
	}
	return nil
}
