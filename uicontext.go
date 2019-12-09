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
	"sync"

	"github.com/hajimehoshi/ebiten/internal/buffered"
	"github.com/hajimehoshi/ebiten/internal/clock"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/shareable"
)

func init() {
	shareable.SetGraphicsDriver(graphicsDriver())
	graphicscommand.SetGraphicsDriver(graphicsDriver())
}

func newUIContext(f func(*Image) error, width, height int, scaleForWindow float64) *uiContext {
	return &uiContext{
		f:              f,
		screenWidth:    width,
		screenHeight:   height,
		scaleForWindow: scaleForWindow,
	}
}

type uiContext struct {
	f              func(*Image) error
	offscreen      *Image
	screen         *Image
	screenWidth    int
	screenHeight   int
	screenScale    float64
	scaleForWindow float64
	offsetX        float64
	offsetY        float64

	reqWidth  int
	reqHeight int
	m         sync.Mutex
}

var theUIContext *uiContext

func (c *uiContext) resolveSize() (int, int) {
	c.m.Lock()
	defer c.m.Unlock()
	if c.reqWidth != 0 || c.reqHeight != 0 {
		c.screenWidth = c.reqWidth
		c.screenHeight = c.reqHeight
		c.reqWidth = 0
		c.reqHeight = 0
	}
	if c.offscreen != nil {
		if w, h := c.offscreen.Size(); w != c.screenWidth || h != c.screenHeight {
			// The offscreen might still be used somewhere. Do not Dispose it. Finalizer will do that.
			c.offscreen = nil
		}
	}
	if c.offscreen == nil {
		c.offscreen = newImage(c.screenWidth, c.screenHeight, FilterDefault, true)
	}
	return c.screenWidth, c.screenHeight
}

func (c *uiContext) size() (int, int) {
	return c.resolveSize()
}

func (c *uiContext) setScaleForWindow(scale float64) {
	c.scaleForWindow = scale
}

func (c *uiContext) getScaleForWindow() float64 {
	return c.scaleForWindow
}

func (c *uiContext) SetScreenSize(width, height int) {
	c.m.Lock()
	defer c.m.Unlock()

	// TODO: Use the interface Game's Layout and then update screenWidth and screenHeight, then this function
	// is no longer needed.
	c.reqWidth = width
	c.reqHeight = height
}

func (c *uiContext) Layout(outsideWidth, outsideHeight float64) {
	if c.screen != nil {
		_ = c.screen.Dispose()
		c.screen = nil
	}

	// TODO: This is duplicated with mobile/ebitenmobileview/funcs.go. Refactor this.
	d := uiDriver().DeviceScaleFactor()
	c.screen = newScreenFramebufferImage(int(outsideWidth*d), int(outsideHeight*d))

	sw, sh := c.resolveSize()
	scaleX := float64(outsideWidth) / float64(sw) * d
	scaleY := float64(outsideHeight) / float64(sh) * d
	c.screenScale = math.Min(scaleX, scaleY)
	if uiDriver().CanHaveWindow() && !uiDriver().IsFullscreen() {
		// When the UI driver cannot have a window, scaleForWindow is updated only via setScaleFowWindow.
		c.scaleForWindow = c.screenScale / d
	}

	width := float64(sw) * c.screenScale
	height := float64(sh) * c.screenScale
	c.offsetX = (float64(outsideWidth)*d - width) / 2
	c.offsetY = (float64(outsideHeight)*d - height) / 2
}

func (c *uiContext) Update(afterFrameUpdate func()) error {
	updateCount := clock.Update(MaxTPS())

	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.

	if err := buffered.BeginFrame(); err != nil {
		return err
	}

	for i := 0; i < updateCount; i++ {
		// Mipmap images should be disposed by Clear.
		c.offscreen.Clear()

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

	// This clear is needed for fullscreen mode or some mobile platforms (#622).
	c.screen.Clear()

	op := &DrawImageOptions{}

	switch vd := graphicsDriver().VDirection(); vd {
	case driver.VDownward:
		// c.screen is special: its Y axis is down to up,
		// and the origin point is lower left.
		op.GeoM.Scale(c.screenScale, -c.screenScale)
		op.GeoM.Translate(0, float64(c.screenHeight)*c.screenScale)
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

	if err := buffered.EndFrame(); err != nil {
		return err
	}
	return nil
}

func (c *uiContext) AdjustPosition(x, y float64) (float64, float64) {
	d := uiDriver().DeviceScaleFactor()
	return (x*d - c.offsetX) / c.screenScale, (y*d - c.offsetY) / c.screenScale
}
