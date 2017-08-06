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

	"github.com/hajimehoshi/ebiten/internal/restorable"
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
	if c.offscreen2 != nil {
		_ = c.offscreen2.Dispose()
	}
	offscreen := newVolatileImage(screenWidth, screenHeight, FilterNearest)

	intScreenScale := int(math.Ceil(screenScale))
	w := screenWidth * intScreenScale
	h := screenHeight * intScreenScale
	offscreen2 := newVolatileImage(w, h, FilterLinear)

	w = int(float64(screenWidth) * screenScale)
	h = int(float64(screenHeight) * screenScale)
	ox, oy := ui.ScreenOffset()
	c.screen = newImageWithScreenFramebuffer(w, h, ox, oy)
	_ = c.screen.Clear()

	c.offscreen = offscreen
	c.offscreen2 = offscreen2
	c.screenScale = screenScale
}

func (c *graphicsContext) initializeIfNeeded() error {
	if !c.initialized {
		if err := restorable.ResetGLState(); err != nil {
			return err
		}
		c.initialized = true
	}
	if err := c.restoreIfNeeded(); err != nil {
		return err
	}
	return nil
}

func drawWithFittingScale(dst *Image, src *Image) {
	wd, hd := dst.Size()
	ws, hs := src.Size()
	sw := float64(wd) / float64(ws)
	sh := float64(hd) / float64(hs)
	op := &DrawImageOptions{}
	op.GeoM.Scale(sw, sh)
	_ = dst.DrawImage(src, op)
}

func (c *graphicsContext) Update(updateCount int) error {
	if err := c.initializeIfNeeded(); err != nil {
		return err
	}
	for i := 0; i < updateCount; i++ {
		restorable.ClearVolatileImages()
		setRunningSlowly(i < updateCount-1)
		if err := c.f(c.offscreen); err != nil {
			return err
		}
	}
	if 0 < updateCount {
		drawWithFittingScale(c.offscreen2, c.offscreen)
	}
	_ = c.screen.Clear()
	drawWithFittingScale(c.screen, c.offscreen2)

	if err := restorable.FlushAndResolveStalePixels(); err != nil {
		return err
	}
	return nil
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
