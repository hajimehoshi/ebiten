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
	"math"
	"sync/atomic"

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
	offscreen2  *Image // TODO: better name
	screen      *Image
	screenScale float64
	initialized int32
}

func (c *graphicsContext) GLContext() *opengl.Context {
	if atomic.LoadInt32(&c.initialized) == 0 {
		return nil
	}
	return ui.GLContext()
}

func (c *graphicsContext) SetSize(screenWidth, screenHeight int, screenScale float64) error {
	if c.screen != nil {
		c.screen.Dispose()
	}
	if c.offscreen != nil {
		c.offscreen.Dispose()
	}
	if c.offscreen2 != nil {
		c.offscreen2.Dispose()
	}
	offscreen, err := newVolatileImage(screenWidth, screenHeight, FilterNearest)
	if err != nil {
		return err
	}

	intScreenScale := int(math.Ceil(screenScale))
	w := screenWidth * intScreenScale
	h := screenHeight * intScreenScale
	offscreen2, err := newVolatileImage(w, h, FilterLinear)
	if err != nil {
		return err
	}

	w = int(float64(screenWidth) * screenScale)
	h = int(float64(screenHeight) * screenScale)
	c.screen, err = newImageWithScreenFramebuffer(w, h)
	if err != nil {
		return err
	}
	c.screen.Clear()

	c.offscreen = offscreen
	c.offscreen2 = offscreen2
	c.screenScale = screenScale
	return nil
}

func (c *graphicsContext) needsRestoring(context *opengl.Context) (bool, error) {
	// FlushCommands is required because c.offscreen.impl might not have an actual texture.
	if err := graphics.FlushCommands(context); err != nil {
		return false, err
	}
	return c.offscreen.impl.isInvalidated(context), nil
}

func (c *graphicsContext) initializeIfNeeded(context *opengl.Context) error {
	if atomic.LoadInt32(&c.initialized) == 0 {
		if err := graphics.Reset(context); err != nil {
			return err
		}
		if err := theImagesForRestoring.flushPixels(context); err != nil {
			return err
		}
		atomic.StoreInt32(&c.initialized, 1)
	}
	r, err := c.needsRestoring(context)
	if err != nil {
		return err
	}
	if !r {
		return nil
	}
	if err := c.restore(context); err != nil {
		return err
	}

	return nil
}

func drawWithFittingScale(dst *Image, src *Image) error {
	wd, hd := dst.Size()
	ws, hs := src.Size()
	sw := float64(wd) / float64(ws)
	sh := float64(hd) / float64(hs)
	op := &DrawImageOptions{}
	op.GeoM.Scale(sw, sh)
	if err := dst.DrawImage(src, op); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) drawToDefaultRenderTarget(context *opengl.Context) error {
	if err := c.screen.Clear(); err != nil {
		return err
	}
	if err := drawWithFittingScale(c.screen, c.offscreen2); err != nil {
		return err
	}
	if err := graphics.FlushCommands(context); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) UpdateAndDraw(context *opengl.Context, updateCount int) error {
	if err := c.initializeIfNeeded(context); err != nil {
		return err
	}
	for i := 0; i < updateCount; i++ {
		if err := theImagesForRestoring.clearVolatileImages(); err != nil {
			return err
		}
		setRunningSlowly(i < updateCount-1)
		if err := c.f(c.offscreen); err != nil {
			return err
		}
	}
	if 0 < updateCount {
		if err := drawWithFittingScale(c.offscreen2, c.offscreen); err != nil {
			return err
		}
	}
	if err := c.drawToDefaultRenderTarget(context); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) restore(context *opengl.Context) error {
	if err := graphics.Reset(context); err != nil {
		return err
	}
	if err := theImagesForRestoring.restore(context); err != nil {
		return err
	}
	return nil
}
