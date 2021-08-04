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
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/buffered"
	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

type uiContext struct {
	game      Game
	offscreen *Image
	screen    *Image

	updateCalled bool

	outsideSizeUpdated bool
	outsideWidth       float64
	outsideHeight      float64

	err atomic.Value

	m sync.Mutex
}

var theUIContext = &uiContext{}

func (c *uiContext) set(game Game) {
	c.m.Lock()
	defer c.m.Unlock()
	c.game = game
}

func (c *uiContext) setError(err error) {
	c.err.Store(err)
}

func (c *uiContext) Layout(outsideWidth, outsideHeight float64) {
	// The given outside size can be 0 e.g. just after restoring from the fullscreen mode on Windows (#1589)
	// Just ignore such cases. Otherwise, creating a zero-sized framebuffer causes a panic.
	if outsideWidth == 0 || outsideHeight == 0 {
		return
	}
	c.outsideSizeUpdated = true
	c.outsideWidth = outsideWidth
	c.outsideHeight = outsideHeight
}

func (c *uiContext) updateOffscreen() {
	sw, sh := c.game.Layout(int(c.outsideWidth), int(c.outsideHeight))
	if sw <= 0 || sh <= 0 {
		panic("ebiten: Layout must return positive numbers")
	}

	if c.offscreen != nil && !c.outsideSizeUpdated {
		if w, h := c.offscreen.Size(); w == sw && h == sh {
			return
		}
	}
	c.outsideSizeUpdated = false

	if c.screen != nil {
		c.screen.Dispose()
		c.screen = nil
	}

	if c.offscreen != nil {
		if w, h := c.offscreen.Size(); w != sw || h != sh {
			c.offscreen.Dispose()
			c.offscreen = nil
		}
	}
	if c.offscreen == nil {
		c.offscreen = NewImage(sw, sh)
		c.offscreen.mipmap.SetVolatile(IsScreenClearedEveryFrame())
	}

	// TODO: This is duplicated with mobile/ebitenmobileview/funcs.go. Refactor this.
	d := uiDriver().DeviceScaleFactor()
	c.screen = newScreenFramebufferImage(int(c.outsideWidth*d), int(c.outsideHeight*d))
}

func (c *uiContext) setScreenClearedEveryFrame(cleared bool) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.offscreen != nil {
		c.offscreen.mipmap.SetVolatile(cleared)
	}
}

func (c *uiContext) screenScale(deviceScaleFactor float64) float64 {
	if c.offscreen == nil {
		return 0
	}
	sw, sh := c.offscreen.Size()
	scaleX := c.outsideWidth / float64(sw) * deviceScaleFactor
	scaleY := c.outsideHeight / float64(sh) * deviceScaleFactor
	return math.Min(scaleX, scaleY)
}

func (c *uiContext) offsets(deviceScaleFactor float64) (float64, float64) {
	if c.offscreen == nil {
		return 0, 0
	}
	sw, sh := c.offscreen.Size()
	s := c.screenScale(deviceScaleFactor)
	width := float64(sw) * s
	height := float64(sh) * s
	x := (c.outsideWidth*deviceScaleFactor - width) / 2
	y := (c.outsideHeight*deviceScaleFactor - height) / 2
	return x, y
}

func (c *uiContext) UpdateFrame() error {
	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.
	return c.updateFrame(clock.Update(MaxTPS()))
}

func (c *uiContext) ForceUpdateFrame() error {
	// ForceUpdate can be invoked even if uiContext it not initialized yet (#1591).
	if c.outsideWidth == 0 || c.outsideHeight == 0 {
		return nil
	}
	return c.updateFrame(1)
}

func (c *uiContext) updateFrame(updateCount int) error {
	if err, ok := c.err.Load().(error); ok && err != nil {
		return err
	}

	debug.Logf("----\n")

	if err := buffered.BeginFrame(); err != nil {
		return err
	}
	if err := c.updateFrameImpl(updateCount); err != nil {
		return err
	}

	// All the vertices data are consumed at the end of the frame, and the data backend can be
	// available after that. Until then, lock the vertices backend.
	return graphics.LockAndResetVertices(func() error {
		if err := buffered.EndFrame(); err != nil {
			return err
		}
		return nil
	})
}

func (c *uiContext) updateFrameImpl(updateCount int) error {
	c.updateOffscreen()

	// Ensure that Update is called once before Draw so that Update can be used for initialization.
	if !c.updateCalled && updateCount == 0 {
		updateCount = 1
		c.updateCalled = true
	}
	debug.Logf("Update count per frame: %d\n", updateCount)

	for i := 0; i < updateCount; i++ {
		if err := hooks.RunBeforeUpdateHooks(); err != nil {
			return err
		}
		if err := c.game.Update(); err != nil {
			return err
		}
		uiDriver().ResetForFrame()
	}

	// Even though updateCount == 0, the offscreen is cleared and Draw is called.
	// Draw should not update the game state and then the screen should not be updated without Update, but
	// users might want to process something at Draw with the time intervals of FPS.
	if IsScreenClearedEveryFrame() {
		c.offscreen.Clear()
	}
	c.game.Draw(c.offscreen)

	if uiDriver().Graphics().NeedsClearingScreen() {
		// This clear is needed for fullscreen mode or some mobile platforms (#622).
		c.screen.Clear()
	}

	op := &DrawImageOptions{}

	s := c.screenScale(uiDriver().DeviceScaleFactor())
	switch vd := uiDriver().Graphics().FramebufferYDirection(); vd {
	case driver.Upward:
		op.GeoM.Scale(s, -s)
		_, h := c.offscreen.Size()
		op.GeoM.Translate(0, float64(h)*s)
	case driver.Downward:
		op.GeoM.Scale(s, s)
	default:
		panic(fmt.Sprintf("ebiten: invalid v-direction: %d", vd))
	}

	op.GeoM.Translate(c.offsets(uiDriver().DeviceScaleFactor()))
	op.CompositeMode = CompositeModeCopy

	// filterScreen works with >=1 scale, but does not well with <1 scale.
	// Use regular FilterLinear instead so far (#669).
	if s >= 1 {
		op.Filter = filterScreen
	} else {
		op.Filter = FilterLinear
	}
	c.screen.DrawImage(c.offscreen, op)
	return nil
}

func (c *uiContext) AdjustPosition(x, y float64, deviceScaleFactor float64) (float64, float64) {
	ox, oy := c.offsets(deviceScaleFactor)
	s := c.screenScale(deviceScaleFactor)
	// The scale 0 indicates that the offscreen is not initialized yet.
	// As any cursor values don't make sense, just return NaN.
	if s == 0 {
		return math.NaN(), math.NaN()
	}
	return (x*deviceScaleFactor - ox) / s, (y*deviceScaleFactor - oy) / s
}
