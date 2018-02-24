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
	"github.com/hajimehoshi/ebiten/internal/clock"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/restorable"
	"github.com/hajimehoshi/ebiten/internal/ui"
	"github.com/hajimehoshi/ebiten/internal/web"
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
	initialized bool
	invalidated bool // browser only
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
	offscreen := newVolatileImage(screenWidth, screenHeight, FilterDefault)

	w := int(float64(screenWidth) * screenScale)
	h := int(float64(screenHeight) * screenScale)
	px0, py0, px1, py1 := ui.ScreenPadding()
	c.screen = newImageWithScreenFramebuffer(w, h, px0, py0, px1, py1)
	_ = c.screen.Clear()

	c.offscreen = offscreen
}

func (c *graphicsContext) initializeIfNeeded() error {
	if !c.initialized {
		if err := restorable.InitializeGLState(); err != nil {
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
		restorable.ClearVolatileImages()
		setRunningSlowly(i < updateCount-1)
		if err := hooks.Run(); err != nil {
			return err
		}
		if err := c.f(c.offscreen); err != nil {
			return err
		}
		afterFrameUpdate()
	}
	if 0 < updateCount {
		drawWithFittingScale(c.screen, c.offscreen, nil, filterScreen)
	}

	if err := restorable.ResolveStaleImages(); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) needsRestoring() (bool, error) {
	if web.IsBrowser() {
		return c.invalidated, nil
	}
	return c.offscreen.restorable.IsInvalidated()
}

func (c *graphicsContext) restoreIfNeeded() error {
	if !restorable.IsRestoringEnabled() {
		return nil
	}
	r, err := c.needsRestoring()
	if err != nil {
		return err
	}
	if !r {
		return nil
	}
	if err := restorable.Restore(); err != nil {
		return err
	}
	c.invalidated = false
	return nil
}
