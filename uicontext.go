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
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/shareable"
)

func init() {
	graphicscommand.SetGraphicsDriver(graphicsDriver())
}

func newUIContext(f func(*Image) error) *uiContext {
	return &uiContext{
		f: f,
	}
}

type uiContext struct {
	f            func(*Image) error
	offscreen    *Image
	screen       *Image
	screenWidth  int
	screenHeight int
	screenScale  float64
	initialized  bool
	offsetX      float64
	offsetY      float64
}

func (c *uiContext) SetSize(screenWidth, screenHeight int, screenScale float64) {
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
	px0, py0, px1, py1 := uiDriver().ScreenPadding()
	c.screen = newImageWithScreenFramebuffer(w+int(math.Ceil(px0+px1)), h+int(math.Ceil(py0+py1)))
	c.screenWidth = w
	c.screenHeight = h

	c.offsetX = px0
	c.offsetY = py0
}

func (c *uiContext) initializeIfNeeded() error {
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

func (c *uiContext) Update(afterFrameUpdate func()) error {
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

		uiDriver().Input().ResetForFrame()
		afterFrameUpdate()
	}

	// Before clearing the screen, the offscreen's pixels must be resolved.
	// After clearing the screen, resolving doesn't work. This is very hacky
	// but we could not find other way so far (#792).
	c.offscreen.resolvePixelsToSet(true)

	// This clear is needed for fullscreen mode or some mobile platforms (#622).
	c.screen.Clear()

	op := &DrawImageOptions{}

	switch vd := graphicsDriver().VDirection(); vd {
	case driver.VDownward:
		// c.screen is special: its Y axis is down to up,
		// and the origin point is lower left.
		op.GeoM.Scale(c.screenScale, -c.screenScale)
		op.GeoM.Translate(0, float64(c.screenHeight))
	case driver.VUpward:
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

func (c *uiContext) needsRestoring() (bool, error) {
	return c.offscreen.mipmap.original().IsInvalidated()
}

func (c *uiContext) restoreIfNeeded() error {
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
	return nil
}

func (c *uiContext) SuspendAudio() {
	hooks.SuspendAudio()
}

func (c *uiContext) ResumeAudio() {
	hooks.ResumeAudio()
}
