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
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/internal/clock"
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/shareable"
	"github.com/hajimehoshi/ebiten/internal/ui"
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
	screenScale  float64
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
	c.screenScale = screenScale

	if c.screen != nil {
		_ = c.screen.Dispose()
	}
	if c.offscreen != nil {
		_ = c.offscreen.Dispose()
	}
	c.offscreen, _ = NewImage(screenWidth, screenHeight, FilterDefault)
	c.offscreen.makeVolatile()

	// Round up the screensize not to cause glitches e.g. on Xperia (#622)
	w := int(math.Ceil(float64(screenWidth) * screenScale))
	h := int(math.Ceil(float64(screenHeight) * screenScale))
	px0, py0, px1, py1 := ui.ScreenPadding()
	c.screen = newImageWithScreenFramebuffer(w+int(math.Ceil(px0+px1)), h+int(math.Ceil(py0+py1)))
	c.screenWidth = w
	c.screenHeight = h

	c.offsetX = px0
	c.offsetY = py0
}

func (c *graphicsContext) initializeIfNeeded() error {
	if !c.initialized {
		if err := shareable.InitializeGraphicsDriverState(); err != nil {
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
	tps := int(MaxTPS())
	updateCount := clock.Update(tps)

	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.

	if err := c.initializeIfNeeded(); err != nil {
		return err
	}
	for i := 0; i < updateCount; i++ {
		c.offscreen.Clear()
		// Mipmap images should be disposed by fill.

		setDrawingSkipped(i < updateCount-1)
		if err := hooks.RunBeforeUpdateHooks(); err != nil {
			return err
		}
		if err := c.f(c.offscreen); err != nil {
			return err
		}
		afterFrameUpdate()
	}

	// This clear is needed for fullscreen mode or some mobile platforms (#622).
	c.screen.Clear()

	op := &DrawImageOptions{}

	switch vd := graphicscommand.Driver().VDirection(); vd {
	case graphicsdriver.VDownward:
		// c.screen is special: its Y axis is down to up,
		// and the origin point is lower left.
		op.GeoM.Scale(c.screenScale, -c.screenScale)
		op.GeoM.Translate(0, float64(c.screenHeight))
	case graphicsdriver.VUpward:
		op.GeoM.Scale(c.screenScale, c.screenScale)
	default:
		panic(fmt.Sprintf("ebiten: invalid v-direction: %d", vd))
	}

	op.GeoM.Translate(c.offsetX, c.offsetY)
	op.CompositeMode = CompositeModeCopy

	// filterScreen works with >=1 scale, but does not well with <1 scale.
	// Use regular FilterLinear instead so far (#669).
	if c.screenScale >= 1 {
		op.Filter = filterScreen
	} else {
		op.Filter = FilterLinear
	}
	_ = c.screen.DrawImage(c.offscreen, op)

	shareable.ResolveStaleImages()

	if err := shareable.Error(); err != nil {
		return err
	}
	return nil
}

func (c *graphicsContext) needsRestoring() (bool, error) {
	return c.offscreen.mipmap.original().IsInvalidated()
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
