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

	"github.com/hajimehoshi/ebiten/internal/clock"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/shareable"
	"github.com/hajimehoshi/ebiten/internal/ui"
	"github.com/hajimehoshi/ebiten/internal/web"
)

func newGraphicsContext(f func(*Image) error) *graphicsContext {
	return &graphicsContext{
		f: f,
	}
}

type graphicsContext struct {
	f            func(*Image) error
	offscreen    *Image
	screen       *Image
	screenWidth  int
	screenHeight int
	initialized  bool
	invalidated  bool // browser only
	offsetX      float64
	offsetY      float64
}

func (c *graphicsContext) Invalidate() {
	// Note that this is called only on browsers so far.
	// TODO: On mobiles, this function is not called and instead IsTexture is called
	// to detect if the context is lost. This is simple but might not work on some platforms.
	// Should Invalidate be called explicitly?
	c.invalidated = true
}

func (c *graphicsContext) SetSize(screenWidth, screenHeight int, screenScale float64) {
	if c.screen != nil {
		_ = c.screen.Dispose()
	}
	if c.offscreen != nil {
		_ = c.offscreen.Dispose()
	}
	c.offscreen = newVolatileImage(screenWidth, screenHeight)

	w := int(float64(screenWidth) * screenScale)
	h := int(float64(screenHeight) * screenScale)
	px0, py0, px1, py1 := ui.ScreenPadding()
	c.screen = newImageWithScreenFramebuffer(w+int(math.Ceil(px0+px1)), h+int(math.Ceil(py0+py1)))
	c.screenWidth = w
	c.screenHeight = h

	c.offsetX = px0
	c.offsetY = py0
}

func (c *graphicsContext) initializeIfNeeded() error {
	if !c.initialized {
		if err := shareable.InitializeGLState(); err != nil {
			return err
		}
		c.initialized = true
	}
	if err := c.restoreIfNeeded(); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) Update(afterFrameUpdate func()) error {
	updateCount := clock.Update()

	if err := c.initializeIfNeeded(); err != nil {
		return err
	}
	for i := 0; i < updateCount; i++ {
		c.offscreen.fill(0, 0, 0, 0)

		setRunningSlowly(i < updateCount-1)
		if err := hooks.RunBeforeUpdateHooks(); err != nil {
			return err
		}
		if err := c.f(c.offscreen); err != nil {
			return err
		}
		afterFrameUpdate()
	}

	// Clear the screen framebuffer by DrawImage instad of Fill
	// to clear the whole region including fullscreen's padding.
	// TODO: This clear is needed only when the screen size is changed.
	if c.offsetX > 0 || c.offsetY > 0 {
		op := &DrawImageOptions{}
		w, h := emptyImage.Size()
		sw, sh := c.screen.Size()
		op.GeoM.Scale(float64(sw)/float64(w), float64(sh)/float64(h))
		op.CompositeMode = CompositeModeCopy
		c.screen.DrawImage(emptyImage, op)
	}

	sw, _ := c.offscreen.Size()
	scale := float64(c.screenWidth) / float64(sw)

	op := &DrawImageOptions{}
	// c.screen is special: its Y axis is down to up,
	// and the origin point is lower left.
	op.GeoM.Scale(scale, -scale)
	op.GeoM.Translate(0, float64(c.screenHeight))
	op.GeoM.Translate(c.offsetX, c.offsetY)

	op.CompositeMode = CompositeModeCopy
	op.Filter = filterScreen
	_ = c.screen.DrawImage(c.offscreen, op)

	if err := shareable.ResolveStaleImages(); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) needsRestoring() (bool, error) {
	if web.IsBrowser() {
		return c.invalidated, nil
	}
	return c.offscreen.shareableImage.IsInvalidated()
}

func (c *graphicsContext) restoreIfNeeded() error {
	if !shareable.IsRestoringEnabled() {
		return nil
	}
	r, err := c.needsRestoring()
	if err != nil {
		return err
	}
	if !r {
		return nil
	}
	if err := shareable.Restore(); err != nil {
		return err
	}
	c.invalidated = false
	return nil
}
